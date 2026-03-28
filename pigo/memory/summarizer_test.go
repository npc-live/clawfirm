package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ai-gateway/pi-go/types"
)


// ─── helpers ──────────────────────────────────────────────────────────────────

func makeMessages(n int) []types.Message {
	msgs := make([]types.Message, n)
	for i := range n {
		if i%2 == 0 {
			msgs[i] = &types.UserMessage{
				Role: "user",
				Content: []types.ContentBlock{
					&types.TextContent{Type: types.ContentTypeText, Text: fmt.Sprintf("user message %d", i)},
				},
			}
		} else {
			msgs[i] = &types.AssistantMessage{
				Role: "assistant",
				Content: []types.ContentBlock{
					&types.TextContent{Type: types.ContentTypeText, Text: fmt.Sprintf("assistant reply %d", i)},
				},
				StopReason: types.StopReasonStop,
			}
		}
	}
	return msgs
}

func noopSumFn(_ context.Context, _ []types.Message) (string, error) {
	return "# Summary\n\nThis is a test summary.", nil
}

func errSumFn(_ context.Context, _ []types.Message) (string, error) {
	return "", fmt.Errorf("summarize: LLM unavailable")
}

// ─── SummarizerConfig defaults ────────────────────────────────────────────────

func TestSummarizerConfig_Defaults(t *testing.T) {
	cfg := SummarizerConfig{}
	if got := cfg.interval(); got != 30*time.Minute {
		t.Errorf("interval() = %v, want 30m", got)
	}
	if got := cfg.filenamePrefix(); got != "summary" {
		t.Errorf("filenamePrefix() = %q, want %q", got, "summary")
	}
	if got := cfg.minMessages(); got != 4 {
		t.Errorf("minMessages() = %d, want 4", got)
	}
}

func TestSummarizerConfig_Custom(t *testing.T) {
	cfg := SummarizerConfig{
		Interval:       5 * time.Minute,
		FilenamePrefix: "chat",
		MinMessages:    10,
	}
	if got := cfg.interval(); got != 5*time.Minute {
		t.Errorf("interval() = %v, want 5m", got)
	}
	if got := cfg.filenamePrefix(); got != "chat" {
		t.Errorf("filenamePrefix() = %q, want %q", got, "chat")
	}
	if got := cfg.minMessages(); got != 10 {
		t.Errorf("minMessages() = %d, want 10", got)
	}
}

// ─── Summarize: skip when too few messages ────────────────────────────────────

func TestSummarize_SkipsWhenTooFewMessages(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	called := false
	sumFn := func(_ context.Context, _ []types.Message) (string, error) {
		called = true
		return "should not be called", nil
	}

	s := NewSummarizer(mgr, sumFn, SummarizerConfig{MinMessages: 4})
	err := s.Summarize(context.Background(), makeMessages(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("summarize function should not be called when messages < MinMessages")
	}
}

func TestSummarize_SkipsOnExactMinMessages_Boundary(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	var calls int
	sumFn := func(_ context.Context, _ []types.Message) (string, error) {
		calls++
		return "# ok", nil
	}

	s := NewSummarizer(mgr, sumFn, SummarizerConfig{MinMessages: 4})

	// Exactly 4 — should fire.
	if err := s.Summarize(context.Background(), makeMessages(4)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call with exactly MinMessages messages, got %d", calls)
	}
}

// ─── Summarize: writes file ───────────────────────────────────────────────────

func TestSummarize_WritesMarkdownFile(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	s := NewSummarizer(mgr, noopSumFn, SummarizerConfig{MemoryDir: dir})
	if err := s.Summarize(context.Background(), makeMessages(6)); err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var mdFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if len(mdFiles) == 0 {
		t.Fatal("expected at least one .md file to be written")
	}
	for _, f := range mdFiles {
		if !strings.HasPrefix(f, "summary-") {
			t.Errorf("unexpected filename %q: should start with summary-", f)
		}
	}
}

func TestSummarize_FileContentMatchesSummary(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	wantContent := "# Decision\n\nWe chose Go."
	sumFn := func(_ context.Context, _ []types.Message) (string, error) {
		return wantContent, nil
	}

	s := NewSummarizer(mgr, sumFn, SummarizerConfig{MemoryDir: dir})
	if err := s.Summarize(context.Background(), makeMessages(5)); err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		t.Fatal("no file written")
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != wantContent {
		t.Errorf("file content = %q, want %q", string(data), wantContent)
	}
}

func TestSummarize_CustomFilenamePrefix(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	s := NewSummarizer(mgr, noopSumFn, SummarizerConfig{
		MemoryDir:      dir,
		FilenamePrefix: "session",
	})
	if err := s.Summarize(context.Background(), makeMessages(5)); err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		t.Fatal("no file written")
	}
	if !strings.HasPrefix(entries[0].Name(), "session-") {
		t.Errorf("expected file to start with session-, got %q", entries[0].Name())
	}
}

// ─── Summarize: indexing ──────────────────────────────────────────────────────

func TestSummarize_IndexesWrittenFile(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	uniqueWord := "zephyranite"
	sumFn := func(_ context.Context, _ []types.Message) (string, error) {
		return "# Summary\n\nThe project uses " + uniqueWord + " architecture.", nil
	}

	s := NewSummarizer(mgr, sumFn, SummarizerConfig{MemoryDir: dir})
	if err := s.Summarize(context.Background(), makeMessages(5)); err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	// The written file should now be searchable.
	results, err := mgr.Search(context.Background(), uniqueWord, 3)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected summary content to be searchable after indexing")
	}
}

// ─── Summarize: error handling ────────────────────────────────────────────────

func TestSummarize_ReturnsErrorOnSumFnFailure(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	s := NewSummarizer(mgr, errSumFn, SummarizerConfig{MemoryDir: dir})
	err := s.Summarize(context.Background(), makeMessages(5))
	if err == nil {
		t.Fatal("expected error when sumFn fails")
	}

	// No file should be written on failure.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected no files on sumFn failure, got %d", len(entries))
	}
}

func TestSummarize_SkipsOnEmptySummary(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	emptySumFn := func(_ context.Context, _ []types.Message) (string, error) {
		return "   ", nil // whitespace only
	}

	s := NewSummarizer(mgr, emptySumFn, SummarizerConfig{MemoryDir: dir})
	if err := s.Summarize(context.Background(), makeMessages(5)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected no file written for empty summary, got %d", len(entries))
	}
}

// ─── LastSummaryTime ──────────────────────────────────────────────────────────

func TestSummarizer_LastSummaryTimeUpdated(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	s := NewSummarizer(mgr, noopSumFn, SummarizerConfig{MemoryDir: dir})

	if !s.LastSummaryTime().IsZero() {
		t.Error("LastSummaryTime should be zero before first summary")
	}

	before := time.Now()
	if err := s.Summarize(context.Background(), makeMessages(5)); err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	after := time.Now()

	got := s.LastSummaryTime()
	if got.Before(before) || got.After(after) {
		t.Errorf("LastSummaryTime %v not in [%v, %v]", got, before, after)
	}
}

// ─── Background timer ─────────────────────────────────────────────────────────

func TestSummarizer_StartStop_FiresOnTick(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	var calls atomic.Int32
	sumFn := func(_ context.Context, _ []types.Message) (string, error) {
		calls.Add(1)
		return "# tick", nil
	}

	s := NewSummarizer(mgr, sumFn, SummarizerConfig{
		MemoryDir: dir,
		Interval:  30 * time.Millisecond, // very short for tests
	})

	msgs := makeMessages(6)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Start(ctx, func() []types.Message { return msgs })
	time.Sleep(120 * time.Millisecond) // allow ~3 ticks
	s.Stop()

	if n := calls.Load(); n < 2 {
		t.Errorf("expected ≥2 summary calls in 120ms with 30ms interval, got %d", n)
	}
}

func TestSummarizer_StopIdempotent(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	s := NewSummarizer(mgr, noopSumFn, SummarizerConfig{
		MemoryDir: dir,
		Interval:  time.Hour, // won't fire during test
	})
	msgs := makeMessages(5)
	ctx := context.Background()
	s.Start(ctx, func() []types.Message { return msgs })
	s.Stop() // should not panic or block
}

func TestSummarizer_StopOnContextCancel(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})

	s := NewSummarizer(mgr, noopSumFn, SummarizerConfig{
		MemoryDir: dir,
		Interval:  time.Hour,
	})
	msgs := makeMessages(5)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx, func() []types.Message { return msgs })
	cancel() // cancelling ctx should stop the loop
	// done channel should be closed; verify no goroutine leak by joining
	select {
	case <-s.done:
		// good
	case <-time.After(500 * time.Millisecond):
		t.Error("goroutine did not stop after context cancel")
	}
}

// ─── BuildSummarizePrompt ─────────────────────────────────────────────────────

func TestBuildSummarizePrompt_ContainsMessages(t *testing.T) {
	msgs := []types.Message{
		&types.UserMessage{
			Role: "user",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "what is 2+2?"},
			},
		},
		&types.AssistantMessage{
			Role: "assistant",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "It is 4."},
			},
		},
		&types.ToolResultMessage{
			Role:     "tool",
			ToolName: "calculator",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "result=4"},
			},
		},
	}

	prompt := BuildSummarizePrompt(msgs)

	for _, want := range []string{"what is 2+2?", "It is 4.", "calculator", "result=4"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildSummarizePrompt_Empty(t *testing.T) {
	prompt := BuildSummarizePrompt(nil)
	if !strings.Contains(prompt, "---") {
		t.Error("prompt should still include the separator even with no messages")
	}
}
