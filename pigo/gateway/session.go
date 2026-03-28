package gateway

import (
	"context"
	"sync"
	"time"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/types"
)

// IncomingMessage is a message arriving from a channel client.
type IncomingMessage struct {
	ID        string // optional dedup ID
	ChannelID string
	UserID    string
	Content   string
	Images    []ImageData
}

// ImageData is a base64-encoded image with MIME type.
type ImageData struct {
	Data     string // base64
	MimeType string // e.g. "image/jpeg"
}

// EventSink receives agent events for a session.
type EventSink func(ev types.AgentEvent)

// Session binds a single Agent to a channel user and processes messages serially.
type Session struct {
	channelID string
	userID    string

	agent  *agent.Agent
	msgCh  chan IncomingMessage
	cancel context.CancelFunc

	mu        sync.Mutex
	sinks     []sinkEntry
	lastUsed  time.Time
}

type sinkEntry struct {
	id uint64
	fn EventSink
}

var sinkSeq uint64

// newSession creates and starts a Session.
func newSession(channelID, userID string, a *agent.Agent) *Session {
	s := &Session{
		channelID: channelID,
		userID:    userID,
		agent:     a,
		msgCh:     make(chan IncomingMessage, 16),
		lastUsed:  time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// Subscribe to agent events and fan out to all registered sinks.
	a.Subscribe(func(ev types.AgentEvent) {
		s.mu.Lock()
		entries := append([]sinkEntry{}, s.sinks...)
		s.mu.Unlock()
		for _, e := range entries {
			e.fn(ev)
		}
	})

	go s.run(ctx)
	return s
}

// Subscribe registers an EventSink. Returns an unsubscribe function.
func (s *Session) Subscribe(fn EventSink) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	sinkSeq++
	id := sinkSeq
	s.sinks = append(s.sinks, sinkEntry{id: id, fn: fn})
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, e := range s.sinks {
			if e.id == id {
				s.sinks = append(s.sinks[:i], s.sinks[i+1:]...)
				return
			}
		}
	}
}

// Send enqueues an incoming message for processing.
// Returns false if the queue is full.
func (s *Session) Send(msg IncomingMessage) bool {
	s.mu.Lock()
	s.lastUsed = time.Now()
	s.mu.Unlock()
	select {
	case s.msgCh <- msg:
		return true
	default:
		return false
	}
}

// Abort cancels the agent's current in-progress turn without stopping the session.
func (s *Session) Abort() { s.agent.Abort() }

// Stop shuts down the session's processing goroutine.
func (s *Session) Stop() {
	s.cancel()
}

// LastUsed returns the time the session last received a message.
func (s *Session) LastUsed() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastUsed
}

// run processes incoming messages serially.
func (s *Session) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-s.msgCh:
			s.process(ctx, msg)
		}
	}
}

// process sends one message through the agent.
func (s *Session) process(ctx context.Context, msg IncomingMessage) {
	// Build content blocks
	var blocks []types.ContentBlock
	if msg.Content != "" {
		blocks = append(blocks, &types.TextContent{
			Type: types.ContentTypeText,
			Text: msg.Content,
		})
	}
	for _, img := range msg.Images {
		blocks = append(blocks, &types.ImageContent{
			Type:     types.ContentTypeImage,
			Data:     img.Data,
			MimeType: img.MimeType,
		})
	}
	if len(blocks) == 0 {
		return
	}

	userMsg := &types.UserMessage{Role: "user", Content: blocks}
	if err := s.agent.PromptMessages(ctx, []types.Message{userMsg}); err != nil {
		return
	}
	_ = s.agent.WaitForIdle(ctx)
}
