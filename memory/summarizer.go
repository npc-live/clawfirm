package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ai-gateway/pi-go/types"
)

// SummarizeFunc is called to summarize a list of messages into a string.
// The caller is responsible for making the LLM call.
type SummarizeFunc func(ctx context.Context, msgs []types.Message) (string, error)

// SummarizerConfig holds options for the Summarizer.
type SummarizerConfig struct {
	// Interval between auto-summarization runs (default 30 minutes).
	Interval time.Duration
	// MemoryDir is where summary .md files are written (defaults to Manager's dir).
	MemoryDir string
	// FilenamePrefix is the prefix for summary files (default "summary").
	FilenamePrefix string
	// MinMessages is the minimum number of messages required before summarizing (default 4).
	MinMessages int
}

func (c SummarizerConfig) interval() time.Duration {
	if c.Interval <= 0 {
		return 30 * time.Minute
	}
	return c.Interval
}

func (c SummarizerConfig) filenamePrefix() string {
	if c.FilenamePrefix != "" {
		return c.FilenamePrefix
	}
	return "summary"
}

func (c SummarizerConfig) minMessages() int {
	if c.MinMessages <= 0 {
		return 4
	}
	return c.MinMessages
}

// Summarizer periodically summarizes conversation history into memory files.
// It integrates with the memory.Manager to re-index the written summary.
type Summarizer struct {
	cfg     SummarizerConfig
	manager *Manager
	sumFn   SummarizeFunc

	mu          sync.Mutex
	lastSummary time.Time
	stopCh      chan struct{}
	done        chan struct{}
}

// NewSummarizer creates a Summarizer that writes summaries to the Manager's memory dir.
func NewSummarizer(mgr *Manager, sumFn SummarizeFunc, cfg SummarizerConfig) *Summarizer {
	if cfg.MemoryDir == "" {
		cfg.MemoryDir = mgr.cfg.memoryDir()
	}
	return &Summarizer{
		cfg:     cfg,
		manager: mgr,
		sumFn:   sumFn,
		stopCh:  make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// Start begins the background auto-summarization loop.
// The loop fires every cfg.Interval; call Stop to shut it down.
func (s *Summarizer) Start(ctx context.Context, getMessages func() []types.Message) {
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.cfg.interval())
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				msgs := getMessages()
				_ = s.Summarize(ctx, msgs) // best-effort; errors are silently dropped
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop shuts down the background loop and waits for it to exit.
func (s *Summarizer) Stop() {
	close(s.stopCh)
	<-s.done
}

// Summarize immediately summarizes msgs and writes the result to a memory file.
// It skips summarization when there are fewer than MinMessages messages.
// Returns the path of the written file, or "" if skipped.
func (s *Summarizer) Summarize(ctx context.Context, msgs []types.Message) error {
	if len(msgs) < s.cfg.minMessages() {
		return nil
	}

	text, err := s.sumFn(ctx, msgs)
	if err != nil {
		return fmt.Errorf("memory: summarize: %w", err)
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}

	path, err := s.writeFile(text)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.lastSummary = time.Now()
	s.mu.Unlock()

	// Re-index the new file so it's immediately searchable.
	return s.manager.IndexFile(ctx, path)
}

// LastSummaryTime returns when the most recent summary was written.
func (s *Summarizer) LastSummaryTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSummary
}

// writeFile appends the summary text to today's daily .md file in the memory dir.
// File name: {prefix}-YYYYMMDD.md  — one file per calendar day, UTC.
// Each append is separated by a timestamp header so the file remains readable.
func (s *Summarizer) writeFile(text string) (string, error) {
	if err := os.MkdirAll(s.cfg.MemoryDir, 0o700); err != nil {
		return "", fmt.Errorf("memory: mkdir: %w", err)
	}
	date := time.Now().UTC().Format("20060102")
	name := fmt.Sprintf("%s-%s.md", s.cfg.filenamePrefix(), date)
	path := filepath.Join(s.cfg.MemoryDir, name)

	// Build the section to append: a time-stamped header + body.
	ts := time.Now().UTC().Format("15:04 UTC")
	section := fmt.Sprintf("\n\n## %s\n\n%s", ts, strings.TrimSpace(text))

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("memory: open summary: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(section); err != nil {
		return "", fmt.Errorf("memory: write summary: %w", err)
	}
	return path, nil
}

// BuildSummarizePrompt converts a message slice into a plain-text transcript
// suitable for feeding to an LLM summarization prompt.
func BuildSummarizePrompt(msgs []types.Message) string {
	var sb strings.Builder
	sb.WriteString("Please summarize the following conversation in concise Markdown.\n")
	sb.WriteString("Focus on decisions made, key information, and action items.\n\n")
	sb.WriteString("---\n\n")
	for _, m := range msgs {
		switch msg := m.(type) {
		case *types.UserMessage:
			sb.WriteString("**User:** ")
			sb.WriteString(extractText(msg.Content))
			sb.WriteString("\n\n")
		case *types.AssistantMessage:
			sb.WriteString("**Assistant:** ")
			sb.WriteString(extractText(msg.Content))
			sb.WriteString("\n\n")
		case *types.ToolResultMessage:
			sb.WriteString("**Tool [")
			sb.WriteString(msg.ToolName)
			sb.WriteString("]:** ")
			sb.WriteString(extractText(msg.Content))
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

// extractText pulls plain text from a ContentBlock slice.
func extractText(blocks []types.ContentBlock) string {
	var parts []string
	for _, b := range blocks {
		if t, ok := b.(*types.TextContent); ok {
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, " ")
}
