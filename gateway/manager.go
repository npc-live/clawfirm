package gateway

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ai-gateway/clawfirm/store"
	"github.com/ai-gateway/clawfirm/types"
)

const (
	defaultIdleTimeout = 30 * time.Minute
	defaultMaxSessions = 100
)

// AgentFactory creates a new AgentRunner for the given channelID+userID.
type AgentFactory func(channelID, userID string) AgentRunner

// ManagerConfig configures a SessionManager.
type ManagerConfig struct {
	IdleTimeout  time.Duration
	MaxSessions  int
	AgentName    string              // used for structured session keys
	SessionStore *store.SessionStore // nil disables persistence
	Summarizer   ConversationSummarizer // nil disables summarization on reset

	// Default session reset policy (applied to newly created SessionEntry records).
	DefaultResetMode  store.ResetMode // "" → store.ResetModeNever
	DefaultResetHour  int             // UTC hour for daily reset (0–23)
	DefaultIdleMinutes int            // idle threshold in minutes (0 → 30)

	// OnUserMessage is called immediately before a user message is sent to the agent.
	// Use this to persist the user message before the agent processes it.
	// Returning a non-nil error signals a save failure; the session may emit
	// an EventSaveError to connected sinks.
	OnUserMessage func(channelID, userID string, msg types.Message) error

	// OnAgentEvent is called for every agent event emitted during a session.
	// Use this to persist assistant messages and tool results.
	// Returning a non-nil error causes the session to emit an EventSaveError.
	OnAgentEvent func(channelID, userID string, ev types.AgentEvent) error
}

// SessionManager creates, caches, and expires Sessions.
type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]*Session
	factory  AgentFactory
	cfg      ManagerConfig
	stopCh   chan struct{}
}

// NewSessionManager creates a SessionManager using the given AgentFactory.
func NewSessionManager(factory AgentFactory, cfg ManagerConfig) *SessionManager {
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}
	if cfg.MaxSessions == 0 {
		cfg.MaxSessions = defaultMaxSessions
	}
	m := &SessionManager{
		sessions: make(map[string]*Session),
		factory:  factory,
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}
	go m.cleanupLoop()
	return m
}

// buildKey returns the structured session key for a channel+user pair.
func (m *SessionManager) buildKey(channelID, userID string) string {
	if m.cfg.AgentName == "" {
		return channelID + "/" + userID // legacy fallback
	}
	if userID == "" {
		return SessionKeyMain(m.cfg.AgentName)
	}
	return SessionKeyDM(m.cfg.AgentName, channelID, userID)
}

// GetOrCreate returns an existing session or creates a new one.
func (m *SessionManager) GetOrCreate(channelID, userID string) (*Session, error) {
	key := m.buildKey(channelID, userID)
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[key]; ok {
		return s, nil
	}
	if len(m.sessions) >= m.cfg.MaxSessions {
		return nil, fmt.Errorf("session manager: max sessions (%d) reached", m.cfg.MaxSessions)
	}

	// Load or create persistent entry.
	var entry *store.SessionEntry
	if m.cfg.SessionStore != nil {
		e, err := m.cfg.SessionStore.Get(key)
		if err != nil {
			log.Printf("gateway: session store get %q: %v", key, err)
		} else if e != nil {
			entry = e
		}
		if entry == nil {
			resetMode := m.cfg.DefaultResetMode
			if resetMode == "" {
				resetMode = store.ResetModeNever
			}
			idleMinutes := m.cfg.DefaultIdleMinutes
			if idleMinutes <= 0 {
				idleMinutes = 30
			}
			entry = &store.SessionEntry{
				SessionKey:  key,
				ChannelID:   channelID,
				UserID:      userID,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				ResetMode:   resetMode,
				ResetAtHour: m.cfg.DefaultResetHour,
				IdleMinutes: idleMinutes,
			}
			if err := m.cfg.SessionStore.Upsert(entry); err != nil {
				log.Printf("gateway: session store upsert %q: %v", key, err)
				entry = nil // degrade gracefully
			}
		}
	}

	var onUser func(types.Message) error
	if m.cfg.OnUserMessage != nil {
		onUser = func(msg types.Message) error { return m.cfg.OnUserMessage(channelID, userID, msg) }
	}
	var onEvent func(types.AgentEvent) error
	if m.cfg.OnAgentEvent != nil {
		onEvent = func(ev types.AgentEvent) error { return m.cfg.OnAgentEvent(channelID, userID, ev) }
	}
	s := newSession(key, channelID, userID, m.factory(channelID, userID), entry, m.cfg.SessionStore, m.cfg.Summarizer, onUser, onEvent)
	m.sessions[key] = s
	return s, nil
}

// Remove stops and removes a session by channelID+userID.
func (m *SessionManager) Remove(channelID, userID string) {
	key := m.buildKey(channelID, userID)
	m.removeByKey(key)
}

// RemoveByKey stops and removes a session by its structured key.
func (m *SessionManager) RemoveByKey(sessionKey string) {
	m.removeByKey(sessionKey)
}

func (m *SessionManager) removeByKey(key string) {
	m.mu.Lock()
	s, ok := m.sessions[key]
	if ok {
		delete(m.sessions, key)
	}
	m.mu.Unlock()
	if ok {
		s.Stop()
		if m.cfg.SessionStore != nil {
			if err := m.cfg.SessionStore.MarkEnded(key); err != nil {
				log.Printf("gateway: mark session ended %q: %v", key, err)
			}
		}
	}
}

// Count returns the number of active sessions.
func (m *SessionManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

// ActiveSessions returns a snapshot of all currently active sessions.
func (m *SessionManager) ActiveSessions() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out
}

// ListEntries returns all persisted session entries for this agent.
func (m *SessionManager) ListEntries() ([]store.SessionEntry, error) {
	if m.cfg.SessionStore == nil {
		return nil, nil
	}
	if m.cfg.AgentName == "" {
		return nil, nil
	}
	return m.cfg.SessionStore.ListByAgent(m.cfg.AgentName)
}

// Stop shuts down the cleanup goroutine and all sessions.
func (m *SessionManager) Stop() {
	close(m.stopCh)
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()
	ctx := context.Background()
	for _, s := range sessions {
		s.SummarizeNow(ctx)
		s.Stop()
		if m.cfg.SessionStore != nil && s.key != "" {
			_ = m.cfg.SessionStore.MarkEnded(s.key)
		}
	}
}

// cleanupLoop periodically removes idle sessions.
func (m *SessionManager) cleanupLoop() {
	tick := time.NewTicker(m.cfg.IdleTimeout / 2)
	defer tick.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-tick.C:
			m.evictIdle()
		}
	}
}

func (m *SessionManager) evictIdle() {
	threshold := time.Now().Add(-m.cfg.IdleTimeout)
	m.mu.Lock()
	var evict []*Session
	for key, s := range m.sessions {
		if s.LastUsed().Before(threshold) {
			evict = append(evict, s)
			delete(m.sessions, key)
		}
	}
	m.mu.Unlock()
	ctx := context.Background()
	for _, s := range evict {
		// Summarize into memory before the session disappears from RAM.
		s.SummarizeNow(ctx)
		s.Stop()
		if m.cfg.SessionStore != nil && s.key != "" {
			_ = m.cfg.SessionStore.MarkEnded(s.key)
		}
	}
}
