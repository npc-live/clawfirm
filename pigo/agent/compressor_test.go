package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ai-gateway/pi-go/types"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func textMsg(role, text string) types.Message {
	switch role {
	case "user":
		return &types.UserMessage{
			Role:    "user",
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
		}
	default:
		return &types.AssistantMessage{
			Role:    "assistant",
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
		}
	}
}

func bigMsg(role string, chars int) types.Message {
	return textMsg(role, strings.Repeat("x", chars))
}

func noopCompressFn(_ context.Context, _ []types.Message) (string, error) {
	return "compressed history", nil
}

func errCompressFn(_ context.Context, _ []types.Message) (string, error) {
	return "", fmt.Errorf("LLM error")
}

// ─── CompressorConfig defaults ────────────────────────────────────────────────

func TestCompressorConfig_Defaults(t *testing.T) {
	cfg := CompressorConfig{}
	if got := cfg.threshold(); got != 0.8 {
		t.Errorf("threshold() = %f, want 0.8", got)
	}
	if got := cfg.keepLastN(); got != 4 {
		t.Errorf("keepLastN() = %d, want 4", got)
	}
}

func TestCompressorConfig_ZeroThreshold_UsesDefault(t *testing.T) {
	cfg := CompressorConfig{Threshold: 0}
	if got := cfg.threshold(); got != 0.8 {
		t.Errorf("zero threshold should fall back to 0.8, got %f", got)
	}
}

func TestCompressorConfig_OverOneThreshold_UsesDefault(t *testing.T) {
	cfg := CompressorConfig{Threshold: 1.5}
	if got := cfg.threshold(); got != 0.8 {
		t.Errorf("threshold >1 should fall back to 0.8, got %f", got)
	}
}

// ─── TransformContext: no-op cases ────────────────────────────────────────────

func TestTransformContext_NoopWhenContextWindowZero(t *testing.T) {
	comp := NewContextCompressor(noopCompressFn, CompressorConfig{ContextWindow: 0})
	msgs := []types.Message{textMsg("user", "hello"), textMsg("assistant", "hi")}

	got, err := comp.TransformContext(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(msgs) {
		t.Errorf("expected no change when ContextWindow=0, got %d messages", len(got))
	}
}

func TestTransformContext_NoopWhenEmpty(t *testing.T) {
	comp := NewContextCompressor(noopCompressFn, CompressorConfig{ContextWindow: 1000})
	got, err := comp.TransformContext(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result for empty input, got %d", len(got))
	}
}

func TestTransformContext_NoopBelowThreshold(t *testing.T) {
	// ContextWindow=10000, threshold=0.8 → limit=8000 tokens.
	// Each message is ~10 chars → ~3 tokens. 5 messages ≈ 15 tokens → well below.
	comp := NewContextCompressor(noopCompressFn, CompressorConfig{
		ContextWindow: 10000,
		Threshold:     0.8,
	})

	msgs := make([]types.Message, 5)
	for i := range msgs {
		msgs[i] = textMsg("user", "short msg")
	}

	got, err := comp.TransformContext(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(msgs) {
		t.Errorf("expected no compression below threshold, got %d messages (want %d)", len(got), len(msgs))
	}
}

// ─── TransformContext: compression triggered ──────────────────────────────────

func TestTransformContext_CompressesWhenOverThreshold(t *testing.T) {
	// ContextWindow=100 tokens, threshold=0.8 → limit=80.
	// Each big message is 200 chars → ~50 tokens. 3 messages ≈ 150 tokens → over limit.
	comp := NewContextCompressor(noopCompressFn, CompressorConfig{
		ContextWindow: 100,
		Threshold:     0.8,
		KeepLastN:     2,
	})

	msgs := []types.Message{
		bigMsg("user", 200),
		bigMsg("assistant", 200),
		bigMsg("user", 200),
		bigMsg("assistant", 200),
	}

	got, err := comp.TransformContext(context.Background(), msgs)
	if err != nil {
		t.Fatalf("TransformContext: %v", err)
	}

	// Result: 1 summary message + 2 kept messages.
	if len(got) != 3 {
		t.Errorf("expected 3 messages (1 summary + 2 kept), got %d", len(got))
	}
}

func TestTransformContext_SummaryMessageIsUser(t *testing.T) {
	comp := NewContextCompressor(noopCompressFn, CompressorConfig{
		ContextWindow: 100,
		Threshold:     0.8,
		KeepLastN:     1,
	})

	msgs := []types.Message{
		bigMsg("user", 400),
		bigMsg("user", 400),
	}

	got, err := comp.TransformContext(context.Background(), msgs)
	if err != nil {
		t.Fatalf("TransformContext: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one message")
	}
	if got[0].MessageRole() != "user" {
		t.Errorf("summary message role = %q, want user", got[0].MessageRole())
	}
}

func TestTransformContext_SummaryContainsLabel(t *testing.T) {
	comp := NewContextCompressor(noopCompressFn, CompressorConfig{
		ContextWindow: 100,
		Threshold:     0.8,
		KeepLastN:     1,
	})

	msgs := []types.Message{
		bigMsg("user", 400),
		bigMsg("user", 400),
	}

	got, err := comp.TransformContext(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}

	summaryMsg := got[0].(*types.UserMessage)
	text := summaryMsg.Content[0].(*types.TextContent).Text
	if !strings.Contains(text, "[Context Summary") {
		t.Errorf("summary message text %q should contain [Context Summary", text)
	}
	if !strings.Contains(text, "compressed history") {
		t.Errorf("summary message text should contain the summarized content")
	}
}

func TestTransformContext_TailMessagesPreserved(t *testing.T) {
	comp := NewContextCompressor(noopCompressFn, CompressorConfig{
		ContextWindow: 100,
		Threshold:     0.8,
		KeepLastN:     2,
	})

	tail1 := bigMsg("user", 200)
	tail2 := bigMsg("assistant", 200)
	msgs := []types.Message{
		bigMsg("user", 200),
		bigMsg("assistant", 200),
		tail1,
		tail2,
	}

	got, err := comp.TransformContext(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) < 3 {
		t.Fatalf("expected ≥3 messages, got %d", len(got))
	}
	// Last two must be the original tail messages.
	if got[len(got)-2] != tail1 {
		t.Error("second-to-last message should be tail1")
	}
	if got[len(got)-1] != tail2 {
		t.Error("last message should be tail2")
	}
}

func TestTransformContext_KeepLastNExceedsLen_NoCompression(t *testing.T) {
	comp := NewContextCompressor(noopCompressFn, CompressorConfig{
		ContextWindow: 10,  // very small → would normally trigger
		Threshold:     0.1, // very low threshold
		KeepLastN:     100, // larger than message count
	})

	msgs := []types.Message{textMsg("user", "a"), textMsg("assistant", "b")}
	got, err := comp.TransformContext(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No head left to summarize → return original.
	if len(got) != len(msgs) {
		t.Errorf("expected original messages when keepLastN >= len(msgs), got %d", len(got))
	}
}

// ─── TransformContext: error handling ────────────────────────────────────────

func TestTransformContext_ReturnsOriginalOnSumFnError(t *testing.T) {
	comp := NewContextCompressor(errCompressFn, CompressorConfig{
		ContextWindow: 100,
		Threshold:     0.8,
		KeepLastN:     1,
	})

	msgs := []types.Message{
		bigMsg("user", 400),
		bigMsg("user", 400),
	}

	got, err := comp.TransformContext(context.Background(), msgs)
	// Should return an error but also the original messages (non-fatal).
	if err == nil {
		t.Fatal("expected error when sumFn fails")
	}
	if len(got) != len(msgs) {
		t.Errorf("on sumFn error, expected original %d messages, got %d", len(msgs), len(got))
	}
}

// ─── estimateMessagesTokens ───────────────────────────────────────────────────

func TestEstimateMessagesTokens_Empty(t *testing.T) {
	if got := estimateMessagesTokens(nil); got != 0 {
		t.Errorf("estimateMessagesTokens(nil) = %d, want 0", got)
	}
}

func TestEstimateMessagesTokens_UserMessage(t *testing.T) {
	// "hello" = 5 chars → ceil(5/4) = 2 tokens
	msg := textMsg("user", "hello")
	got := estimateMessageTokens(msg)
	if got < 1 {
		t.Errorf("estimateMessageTokens for 'hello' = %d, want ≥1", got)
	}
}

func TestEstimateMessagesTokens_AssistantWithUsage(t *testing.T) {
	// Assistant message with Usage populated should include usage tokens.
	msg := &types.AssistantMessage{
		Role:    "assistant",
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "hi"}},
		Usage:   types.Usage{Input: 10, Output: 20},
	}
	got := estimateMessageTokens(msg)
	// Should be at least Usage.Input + Usage.Output = 30.
	if got < 30 {
		t.Errorf("estimateMessageTokens for assistant with Usage{30} = %d, want ≥30", got)
	}
}

func TestEstimateMessagesTokens_GrowsWithTextLength(t *testing.T) {
	short := estimateMessageTokens(textMsg("user", strings.Repeat("a", 40)))
	long := estimateMessageTokens(textMsg("user", strings.Repeat("a", 400)))
	if long <= short {
		t.Errorf("longer text should have more tokens: short=%d long=%d", short, long)
	}
}

func TestEstimateMessagesTokens_MediaPlaceholder(t *testing.T) {
	msg := &types.UserMessage{
		Role:    "user",
		Content: []types.ContentBlock{&types.ImageContent{Type: types.ContentTypeImage, MimeType: "image/png"}},
	}
	got := estimateMessageTokens(msg)
	if got != 256 {
		t.Errorf("image block should contribute 256 placeholder tokens, got %d", got)
	}
}

// ─── BuildSummarizeContextPrompt ─────────────────────────────────────────────

func TestBuildSummarizeContextPrompt_ContainsRoles(t *testing.T) {
	msgs := []types.Message{
		textMsg("user", "deploy the service"),
		textMsg("assistant", "deploying now"),
		&types.ToolResultMessage{
			Role:     "tool",
			ToolName: "kubectl",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "deployed successfully"},
			},
		},
	}

	prompt := BuildSummarizeContextPrompt(msgs)

	for _, want := range []string{
		"deploy the service",
		"deploying now",
		"kubectl",
		"deployed successfully",
		"User:",
		"Assistant:",
		"Tool [kubectl]:",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildSummarizeContextPrompt_Empty(t *testing.T) {
	prompt := BuildSummarizeContextPrompt(nil)
	if !strings.Contains(prompt, "---") {
		t.Error("prompt should include separator even for empty input")
	}
}

// ─── Integration: compressor inside AgentLoop ─────────────────────────────────

func TestCompressor_IntegrationWithAgentLoop(t *testing.T) {
	// Build a compressor that fires at low threshold.
	// 3 messages × 400 chars ≈ 100 tokens each = 300 total.
	// ContextWindow=200, threshold=0.8 → limit=160. 300 > 160 → triggers.
	// KeepLastN=1: head=2 messages summarized, tail=1 preserved.
	var compressCalled bool
	compFn := func(_ context.Context, msgs []types.Message) (string, error) {
		compressCalled = true
		return "summary of old messages", nil
	}
	comp := NewContextCompressor(compFn, CompressorConfig{
		ContextWindow: 200,
		Threshold:     0.8,
		KeepLastN:     1,
	})

	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("final answer"),
		},
	}

	// Pre-populate history that is already over threshold.
	existing := []types.Message{
		bigMsg("user", 400),
		bigMsg("assistant", 400),
		bigMsg("user", 400),
	}

	cfg := baseConfig()
	cfg.TransformContext = comp.TransformContext

	ctx := context.Background()
	agentCtx := AgentContext{Messages: existing}
	_, err := AgentLoop(ctx, prov, nil, agentCtx, cfg, noopEmitFn)
	if err != nil {
		t.Fatalf("AgentLoop: %v", err)
	}

	if !compressCalled {
		t.Error("expected compressor to be called during AgentLoop")
	}
}
