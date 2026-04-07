package clawproc

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ai-gateway/clawfirm/types"
)

func TestExtractTextMultipleBlocks(t *testing.T) {
	// Simulate a user message with text + file attachment hint (two TextContent blocks).
	msgs := []types.Message{
		&types.UserMessage{
			Role: "user",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "运行 media.whip 帮我做个视频"},
				&types.TextContent{Type: types.ContentTypeText, Text: "[User attached 2 file(s): /tmp/upload-123.jpg, /tmp/upload-456.mp3]\nYou can read these files using the Read tool."},
			},
		},
	}

	got := extractText(msgs)
	if got == "" {
		t.Fatal("extractText returned empty string")
	}
	// Must contain both the user text AND the file hint.
	if !strings.Contains(got, "运行 media.whip") {
		t.Error("missing user text")
	}
	if !strings.Contains(got, "/tmp/upload-123.jpg") {
		t.Error("missing file attachment hint")
	}
	t.Logf("extractText result:\n%s", got)
}

func TestExtractTextSingleBlock(t *testing.T) {
	msgs := []types.Message{
		&types.UserMessage{
			Role: "user",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "hello world"},
			},
		},
	}
	got := extractText(msgs)
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func findClawBinary(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"../claw-code/rust/target/debug/claw",
		"../claw-code/rust/target/release/claw",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath("claw"); err == nil {
		return p
	}
	t.Skip("claw binary not found, skipping integration test")
	return ""
}

func TestProcessStartAndShutdown(t *testing.T) {
	bin := findClawBinary(t)
	proc := NewProcess(Config{
		BinaryPath:     bin,
		Model:          "claude-haiku-4-5-20251001",
		PermissionMode: "danger-full-access",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ready, err := proc.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if ready.Type != "ready" {
		t.Fatalf("expected ready event, got %q", ready.Type)
	}
	if ready.SessionID == "" {
		t.Error("ready event missing session_id")
	}
	t.Logf("ready: session=%s model=%s", ready.SessionID, ready.Model)

	if err := proc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestClawAgentPromptAndEvents(t *testing.T) {
	bin := findClawBinary(t)

	proc := NewProcess(Config{
		BinaryPath:     bin,
		Model:          "claude-haiku-4-5-20251001",
		PermissionMode: "danger-full-access",
	})
	agent := NewClawAgent(proc)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := agent.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer agent.Close()

	// Collect events.
	var mu sync.Mutex
	var events []types.AgentEvent
	agent.Subscribe(func(ev types.AgentEvent) {
		mu.Lock()
		events = append(events, ev)
		mu.Unlock()
	})

	// Send a simple prompt.
	err := agent.PromptMessages(ctx, []types.Message{
		&types.UserMessage{
			Role: "user",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "Say exactly: HELLO_TEST_OK"},
			},
		},
	})
	if err != nil {
		t.Fatalf("PromptMessages: %v", err)
	}

	// Wait for completion.
	if err := agent.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) == 0 {
		t.Fatal("no events received")
	}

	// Verify we got text_delta and agent_end events.
	var hasTextDelta, hasAgentEnd bool
	var textAccum string
	for _, ev := range events {
		t.Logf("event: %s", ev.Type)
		if ev.Type == types.EventMessageUpdate && ev.StreamEvent != nil && ev.StreamEvent.Type == types.StreamEventTextDelta {
			hasTextDelta = true
			textAccum += ev.StreamEvent.Delta
		}
		if ev.Type == types.EventAgentEnd {
			hasAgentEnd = true
		}
	}

	if !hasTextDelta {
		t.Error("missing text_delta event")
	}
	if !hasAgentEnd {
		t.Error("missing agent_end event")
	}
	t.Logf("response text: %s", textAccum)
}

func TestClawAgentStartFailure(t *testing.T) {
	// Use a non-existent binary to simulate start failure.
	proc := NewProcess(Config{
		BinaryPath: "/nonexistent/claw",
	})
	agent := NewClawAgent(proc)

	err := agent.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start to fail with non-existent binary")
	}
	t.Logf("expected error: %v", err)

	// PromptMessages should emit an error event, not panic.
	var mu sync.Mutex
	var events []types.AgentEvent
	agent.Subscribe(func(ev types.AgentEvent) {
		mu.Lock()
		events = append(events, ev)
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = agent.PromptMessages(ctx, []types.Message{
		&types.UserMessage{
			Role: "user",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "hello"},
			},
		},
	})
	_ = agent.WaitForIdle(ctx)

	mu.Lock()
	defer mu.Unlock()

	if len(events) == 0 {
		t.Fatal("expected error events from PromptMessages on dead agent")
	}

	// Should have a visible error text and agent_end.
	var hasError bool
	for _, ev := range events {
		if ev.Type == types.EventMessageUpdate && ev.StreamEvent != nil {
			if ev.StreamEvent.Delta != "" {
				hasError = true
				t.Logf("error message shown to user: %s", ev.StreamEvent.Delta)
			}
		}
	}
	if !hasError {
		t.Error("expected visible error message in events")
	}
}
