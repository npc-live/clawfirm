package gateway

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ai-gateway/clawfirm/store"
	"github.com/ai-gateway/clawfirm/types"
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

	agent     AgentRunner
	msgCh     chan IncomingMessage
	cancel    context.CancelFunc

	mu        sync.Mutex
	sinks     []sinkEntry
	lastUsed  time.Time

	entry         *store.SessionEntry    // nil if no store
	sessStore     *store.SessionStore    // nil if no store
	summarizer    ConversationSummarizer // nil if not configured
	onUserMessage func(msg types.Message)
}

type sinkEntry struct {
	id uint64
	fn EventSink
}

var sinkSeq atomic.Uint64

// newSession creates and starts a Session.
func newSession(key, channelID, userID string, a AgentRunner,
	entry *store.SessionEntry, ss *store.SessionStore,
	summarizer ConversationSummarizer, onUserMessage func(types.Message),
	onAgentEvent func(types.AgentEvent)) *Session {
	s := &Session{
		key:           key,
		channelID:     channelID,
		userID:        userID,
		agent:         a,
		msgCh:         make(chan IncomingMessage, 16),
		lastUsed:      time.Now(),
		entry:         entry,
		sessStore:     ss,
		summarizer:    summarizer,
		onUserMessage: onUserMessage,
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

		// Forward to the persistence callback (if configured).
		if onAgentEvent != nil {
			onAgentEvent(ev)
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

// Stop shuts down the session's processing goroutine and the underlying agent process.
func (s *Session) Stop() {
	s.cancel()
	if c, ok := s.agent.(interface{ Close() error }); ok {
		_ = c.Close()
	}
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
	// Save attached media to temp files and add text hints so the agent
	// can use the media_understand tool. This works regardless of whether
	// the main LLM supports vision.
	if len(msg.Images) > 0 {
		var filePaths []string
		for _, img := range msg.Images {
			path, err := saveMediaToTemp(img.Data, img.MimeType)
			if err != nil {
				log.Printf("gateway: save media: %v", err)
				continue
			}
			log.Printf("gateway: saved attached file → %s (%s, %d bytes)", path, img.MimeType, len(img.Data))
			filePaths = append(filePaths, path)
		}
		if len(filePaths) > 0 {
			hint := fmt.Sprintf("[User attached %d file(s): %s]\nYou can read these files using the Read tool (images) or process them with Bash (audio/video).",
				len(filePaths), strings.Join(filePaths, ", "))
			blocks = append(blocks, &types.TextContent{
				Type: types.ContentTypeText,
				Text: hint,
			})
		}
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
	if s.onUserMessage != nil {
		s.onUserMessage(userMsg)
	}
	if err := s.agent.PromptMessages(ctx, []types.Message{userMsg}); err != nil {
		return
	}
	_ = s.agent.WaitForIdle(ctx)
}

// encodeBase64 returns the standard base64 encoding of b.
func encodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// mimeToExt maps common MIME types to file extensions.
var mimeToExt = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"video/mp4":       ".mp4",
	"video/webm":      ".webm",
	"video/quicktime": ".mov",
	"audio/mpeg":      ".mp3",
	"audio/mp4":       ".m4a",
	"audio/wav":       ".wav",
	"audio/ogg":       ".ogg",
	"audio/aac":       ".aac",
	"audio/flac":      ".flac",
	"audio/webm":      ".weba",
}

// saveMediaToTemp decodes base64 media data and writes it to a temp file.
func saveMediaToTemp(b64Data, mimeType string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}
	ext := mimeToExt[mimeType]
	if ext == "" {
		ext = ".bin"
	}
	dir := filepath.Join(os.TempDir(), "clawfirm-media")
	os.MkdirAll(dir, 0o755)
	f, err := os.CreateTemp(dir, "upload-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return "", fmt.Errorf("write temp: %w", err)
	}
	return f.Name(), nil
}
