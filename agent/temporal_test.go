package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ai-gateway/pi-go/types"
)

// makeUserMsg creates a UserMessage with the given timestamp (ms).
func makeUserMsg(text string, tsMs int64) *types.UserMessage {
	return &types.UserMessage{
		Role:      "user",
		Content:   []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
		Timestamp: tsMs,
	}
}

func makeAssistantMsg(text string, tsMs int64) *types.AssistantMessage {
	return &types.AssistantMessage{
		Role:       "assistant",
		Content:    []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
		StopReason: types.StopReasonStop,
		Timestamp:  tsMs,
	}
}

// ─── shouldInject ─────────────────────────────────────────────────────────────

func TestTemporalInjector_NoInjectOnEmpty(t *testing.T) {
	ti := NewTemporalInjector(30 * time.Minute)
	if ti.shouldInject(nil) {
		t.Error("should not inject for empty message slice")
	}
	if ti.shouldInject([]types.Message{}) {
		t.Error("should not inject for empty message slice")
	}
}

func TestTemporalInjector_InjectsOnFirstMessage(t *testing.T) {
	ti := NewTemporalInjector(30 * time.Minute)
	// Only a user message, no assistant reply yet → first message of session.
	msgs := []types.Message{makeUserMsg("hello", time.Now().UnixMilli())}
	if !ti.shouldInject(msgs) {
		t.Error("should inject on first user message (no assistant reply yet)")
	}
}

func TestTemporalInjector_NoInjectDuringActiveSession(t *testing.T) {
	ti := NewTemporalInjector(30 * time.Minute)
	now := time.Now().UnixMilli()
	msgs := []types.Message{
		makeUserMsg("hi", now-int64(5*time.Minute/time.Millisecond)),
		makeAssistantMsg("hello", now-int64(4*time.Minute/time.Millisecond)),
		makeUserMsg("follow-up", now), // gap = 4min < 30min
	}
	if ti.shouldInject(msgs) {
		t.Error("should not inject mid-session with gap < 30min")
	}
}

func TestTemporalInjector_InjectsAfterSessionGap(t *testing.T) {
	ti := NewTemporalInjector(30 * time.Minute)
	old := time.Now().Add(-45 * time.Minute).UnixMilli()
	msgs := []types.Message{
		makeUserMsg("earlier msg", old),
		makeAssistantMsg("response", old+1000),
	}
	if !ti.shouldInject(msgs) {
		t.Error("should inject after 45min gap (> 30min threshold)")
	}
}

func TestTemporalInjector_NoInjectJustUnderThreshold(t *testing.T) {
	ti := NewTemporalInjector(30 * time.Minute)
	recent := time.Now().Add(-29 * time.Minute).UnixMilli()
	msgs := []types.Message{
		makeUserMsg("msg", recent),
		makeAssistantMsg("reply", recent+1000),
	}
	if ti.shouldInject(msgs) {
		t.Error("should not inject with gap just under 30min")
	}
}

func TestTemporalInjector_InjectsWithNoTimestamp(t *testing.T) {
	ti := NewTemporalInjector(30 * time.Minute)
	// Messages with no timestamp → treated as first message.
	msgs := []types.Message{
		&types.UserMessage{Role: "user", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "hi"},
		}},
	}
	if !ti.shouldInject(msgs) {
		t.Error("should inject when no timestamps are present")
	}
}

func TestTemporalInjector_CustomThreshold(t *testing.T) {
	ti := NewTemporalInjector(5 * time.Minute)
	old := time.Now().Add(-6 * time.Minute).UnixMilli()
	msgs := []types.Message{
		makeUserMsg("msg", old),
		makeAssistantMsg("reply", old+1000),
	}
	if !ti.shouldInject(msgs) {
		t.Error("should inject after 6min with 5min threshold")
	}
}

func TestTemporalInjector_DefaultThreshold(t *testing.T) {
	ti := NewTemporalInjector(0) // 0 → default 30min
	if ti.GapThreshold != 30*time.Minute {
		t.Errorf("default threshold = %v, want 30m", ti.GapThreshold)
	}
}

// ─── TransformContext ─────────────────────────────────────────────────────────

func TestTemporalInjector_TransformContext_PrependsAnchor(t *testing.T) {
	ti := NewTemporalInjector(30 * time.Minute)
	// First message — should inject.
	msgs := []types.Message{makeUserMsg("what day is it?", time.Now().UnixMilli())}

	got, err := ti.TransformContext(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 messages (anchor + original), got %d", len(got))
	}

	anchor, ok := got[0].(*types.UserMessage)
	if !ok {
		t.Fatalf("first message should be UserMessage, got %T", got[0])
	}
	text := anchor.Content[0].(*types.TextContent).Text
	if !strings.Contains(text, "[Temporal context:") {
		t.Errorf("anchor text %q should contain [Temporal context:", text)
	}
	if !strings.Contains(text, time.Now().Format("2006-01-02")) {
		t.Errorf("anchor text %q should contain today's date", text)
	}

	// Original message preserved.
	if got[1] != msgs[0] {
		t.Error("original message should be preserved at index 1")
	}
}

func TestTemporalInjector_TransformContext_NoOpDuringSession(t *testing.T) {
	ti := NewTemporalInjector(30 * time.Minute)
	now := time.Now().UnixMilli()
	msgs := []types.Message{
		makeUserMsg("hi", now-int64(2*time.Minute/time.Millisecond)),
		makeAssistantMsg("hello", now-int64(1*time.Minute/time.Millisecond)),
		makeUserMsg("next question", now),
	}

	got, err := ti.TransformContext(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(msgs) {
		t.Errorf("expected no injection during active session, got %d messages (want %d)", len(got), len(msgs))
	}
}

func TestTemporalInjector_TransformContext_EmptyNoOp(t *testing.T) {
	ti := NewTemporalInjector(30 * time.Minute)
	got, err := ti.TransformContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result for empty input, got %d", len(got))
	}
}

// ─── Integration with AgentLoop ───────────────────────────────────────────────

func TestTemporalInjector_IntegrationFirstMessage(t *testing.T) {
	// Verify the anchor is visible to the LLM on a brand-new conversation.
	var capturedMsgs []types.Message
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("ok"),
		},
	}

	ti := NewTemporalInjector(30 * time.Minute)
	cfg := baseConfig()
	cfg.TransformContext = func(ctx context.Context, msgs []types.Message) ([]types.Message, error) {
		transformed, err := ti.TransformContext(ctx, msgs)
		capturedMsgs = transformed
		return transformed, err
	}

	ctx := context.Background()
	agentCtx := AgentContext{} // empty history = first message
	prompt := &types.UserMessage{
		Role:      "user",
		Content:   []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "what time is it?"}},
		Timestamp: time.Now().UnixMilli(),
	}
	_, err := AgentLoop(ctx, prov, []types.Message{prompt}, agentCtx, cfg, noopEmitFn)
	if err != nil {
		t.Fatal(err)
	}

	if len(capturedMsgs) == 0 {
		t.Fatal("no messages captured")
	}
	first, ok := capturedMsgs[0].(*types.UserMessage)
	if !ok {
		t.Fatalf("first message = %T, want *UserMessage", capturedMsgs[0])
	}
	text := first.Content[0].(*types.TextContent).Text
	if !strings.Contains(text, "[Temporal context:") {
		t.Errorf("LLM did not receive temporal anchor; first msg = %q", text)
	}
}
