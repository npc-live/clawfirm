package gateway

import (
	"context"
	"encoding/base64"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/store"
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// IncomingMessage is a message arriving from a channel client.
type IncomingMessage struct {
	ID        string // optional dedup ID
	ChannelID string
	UserID    string
	Content   string
	Images    []ImageData
	Files     []FileData
}

// ImageData is a base64-encoded image with MIME type.
type ImageData struct {
	Data     string // base64
	MimeType string // e.g. "image/jpeg"
}

// FileData holds a downloaded file with its metadata.
type FileData struct {
	Data        []byte // raw bytes
	MimeType    string // e.g. "image/jpeg", "audio/opus", "application/pdf"
	FileName    string // original filename (may be empty)
	Placeholder string // "<media:image>", "<media:audio>", "<media:document>", etc.
}

// EventSink receives agent events for a session.
type EventSink func(ev types.AgentEvent)

// ConversationSummarizer summarizes a conversation before it is reset.
type ConversationSummarizer interface {
	Summarize(ctx context.Context, msgs []types.Message) error
}

// Session binds a single Agent to a channel user and processes messages serially.
type Session struct {
	key       string // structured session key
	channelID string
	userID    string

	agent     *agent.Agent
	msgCh     chan IncomingMessage
	cancel    context.CancelFunc

	mu        sync.Mutex
	sinks     []sinkEntry
	lastUsed  time.Time

	entry      *store.SessionEntry     // nil if no store
	sessStore  *store.SessionStore     // nil if no store
	summarizer ConversationSummarizer  // nil if not configured
}

type sinkEntry struct {
	id uint64
	fn EventSink
}

var sinkSeq atomic.Uint64

// newSession creates and starts a Session.
func newSession(key, channelID, userID string, a *agent.Agent,
	entry *store.SessionEntry, ss *store.SessionStore,
	summarizer ConversationSummarizer) *Session {
	s := &Session{
		key:        key,
		channelID:  channelID,
		userID:     userID,
		agent:      a,
		msgCh:      make(chan IncomingMessage, 16),
		lastUsed:   time.Now(),
		entry:      entry,
		sessStore:  ss,
		summarizer: summarizer,
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

		// Track token usage per turn (EventTurnEnd carries the single assistant message).
		if ev.Type == types.EventTurnEnd && s.sessStore != nil && s.entry != nil {
			if am, ok := ev.Message.(*types.AssistantMessage); ok {
				delta := store.UsageDelta{
					InputTokens:      am.Usage.Input,
					OutputTokens:     am.Usage.Output,
					CacheRead:        am.Usage.CacheRead,
					CacheWrite:       am.Usage.CacheWrite,
					EstimatedCostUSD: am.Usage.Cost.Total,
					Model:            am.Model,
					ModelProvider:    am.Provider,
				}
				if delta.InputTokens > 0 || delta.OutputTokens > 0 {
					_ = s.sessStore.UpdateUsage(s.entry.SessionKey, delta)
				}
			}
		}
	})

	go s.run(ctx)
	return s
}

// Key returns the structured session key.
func (s *Session) Key() string { return s.key }

// Entry returns the persisted session entry (may be nil).
func (s *Session) Entry() *store.SessionEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.entry
}

// Subscribe registers an EventSink. Returns an unsubscribe function.
func (s *Session) Subscribe(fn EventSink) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := sinkSeq.Add(1)
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

// RunTool finds and directly executes a named tool, bypassing the LLM.
// Events (tool_start, tool_update, tool_end, agent_end) are emitted to subscribers.
func (s *Session) RunTool(ctx context.Context, callID, toolName string, args map[string]any) {
	t := s.agent.FindTool(toolName)
	if t == nil {
		s.emitEvent(types.AgentEvent{
			Type: types.EventToolExecutionEnd,
			ToolCallID: callID,
			ToolName:   toolName,
			ToolResult: "tool not found: " + toolName,
			ToolIsError: true,
		})
		s.emitEvent(types.AgentEvent{Type: types.EventAgentEnd})
		return
	}
	s.emitEvent(types.AgentEvent{
		Type:       types.EventToolExecutionStart,
		ToolCallID: callID,
		ToolName:   toolName,
		ToolArgs:   args,
	})
	result, err := t.Execute(ctx, callID, args, func(upd tool.ToolUpdate) {
		s.emitEvent(types.AgentEvent{
			Type:          types.EventToolExecutionUpdate,
			ToolCallID:    callID,
			ToolName:      toolName,
			PartialResult: upd.Details,
		})
	})
	toolResult := ""
	isErr := false
	if err != nil {
		toolResult = err.Error()
		isErr = true
	} else if len(result.Content) > 0 {
		if tc, ok := result.Content[0].(*types.TextContent); ok {
			toolResult = tc.Text
		}
	}
	s.emitEvent(types.AgentEvent{
		Type:        types.EventToolExecutionEnd,
		ToolCallID:  callID,
		ToolName:    toolName,
		ToolResult:  toolResult,
		ToolIsError: isErr,
	})
	s.emitEvent(types.AgentEvent{Type: types.EventAgentEnd})
}

// emitEvent fans out an event to all current subscribers.
func (s *Session) emitEvent(ev types.AgentEvent) {
	s.mu.Lock()
	entries := append([]sinkEntry{}, s.sinks...)
	s.mu.Unlock()
	for _, e := range entries {
		e.fn(ev)
	}
}

// SummarizeNow summarizes the current message history into memory (best-effort).
// Safe to call concurrently; returns immediately if no summarizer is configured
// or there are no messages.
func (s *Session) SummarizeNow(ctx context.Context) {
	if s.summarizer == nil {
		return
	}
	msgs := s.agent.State().Messages
	if len(msgs) == 0 {
		return
	}
	if err := s.summarizer.Summarize(ctx, msgs); err != nil {
		log.Printf("gateway: summarize session %q: %v", s.key, err)
	}
}

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
	// Check freshness before processing; summarize then reset history if stale.
	if s.entry != nil && s.sessStore != nil {
		cfg := FreshnessConfigFromEntry(s.entry)
		s.mu.Lock()
		lastUsed := s.lastUsed
		s.mu.Unlock()
		if !IsFresh(s.entry, cfg, time.Now(), lastUsed) {
			msgs := s.agent.State().Messages
			if s.summarizer != nil && len(msgs) > 0 {
				_ = s.summarizer.Summarize(ctx, msgs)
			}
			s.agent.ClearMessages()
			_ = s.sessStore.MarkReset(s.entry.SessionKey)
			now := time.Now()
			s.mu.Lock()
			s.entry.LastResetAt = &now
			s.mu.Unlock()
		}
	}

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
	for _, f := range msg.Files {
		switch f.Placeholder {
		case "<media:image>":
			blocks = append(blocks, &types.ImageContent{
				Type:     types.ContentTypeImage,
				Data:     encodeBase64(f.Data),
				MimeType: f.MimeType,
			})
		case "<media:audio>":
			blocks = append(blocks, &types.AudioContent{
				Type:     types.ContentTypeAudio,
				Data:     encodeBase64(f.Data),
				MimeType: f.MimeType,
			})
		default:
			// Try to extract text from parseable document formats.
			if text := extractDocumentText(f.Data, f.FileName); text != "" {
				name := f.FileName
				if name == "" {
					name = "文件"
				}
				blocks = append(blocks, &types.TextContent{
					Type: types.ContentTypeText,
					Text: "【" + name + "】\n" + text,
				})
			} else {
				// video / sticker / unknown binary — describe as text placeholder.
				name := f.FileName
				if name == "" {
					name = f.Placeholder
				}
				blocks = append(blocks, &types.TextContent{
					Type: types.ContentTypeText,
					Text: "[文件: " + name + "]",
				})
			}
		}
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

// encodeBase64 returns the standard base64 encoding of b.
func encodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
