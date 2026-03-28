package gateway

import (
	"fmt"
	"sync"
	"time"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/types"
)

const (
	defaultIdleTimeout = 30 * time.Minute
	defaultMaxSessions = 100
)

// AgentFactory creates a new Agent for the given channelID+userID.
type AgentFactory func(channelID, userID string) *agent.Agent

// ManagerConfig configures a SessionManager.
type ManagerConfig struct {
	IdleTimeout time.Duration
	MaxSessions int
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

// SimpleAgentFactory returns an AgentFactory that creates agents from the given provider+model.
func SimpleAgentFactory(prov provider.LLMProvider, model types.Model, systemPrompt string) AgentFactory {
	return func(_, _ string) *agent.Agent {
		return agent.NewAgent(prov,
			agent.WithModel(model),
			agent.WithSystemPrompt(systemPrompt),
		)
	}
}

// GetOrCreate returns an existing session or creates a new one.
func (m *SessionManager) GetOrCreate(channelID, userID string) (*Session, error) {
	key := channelID + "/" + userID
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[key]; ok {
		return s, nil
	}
	if len(m.sessions) >= m.cfg.MaxSessions {
		return nil, fmt.Errorf("session manager: max sessions (%d) reached", m.cfg.MaxSessions)
	}

	s := newSession(channelID, userID, m.factory(channelID, userID))
	m.sessions[key] = s
	return s, nil
}

// Remove stops and removes a session.
func (m *SessionManager) Remove(channelID, userID string) {
	key := channelID + "/" + userID
	m.mu.Lock()
	s, ok := m.sessions[key]
	if ok {
		delete(m.sessions, key)
	}
	m.mu.Unlock()
	if ok {
		s.Stop()
	}
}

// Count returns the number of active sessions.
func (m *SessionManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
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
	for _, s := range sessions {
		s.Stop()
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
	for _, s := range evict {
		s.Stop()
	}
}
