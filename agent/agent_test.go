package agent

import (
	"context"
	"testing"
	"time"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

func makeAgent(responses [][]types.AssistantMessageEvent) *Agent {
	prov := &fakeLLMProvider{id: "fake", responses: responses}
	return NewAgent(prov,
		WithModel(types.Model{ID: "test-model"}),
		WithLoopConfig(AgentLoopConfig{
			ToolExecution: types.ToolExecutionSequential,
		}),
	)
}

func TestAgentPromptEmitsEvents(t *testing.T) {
	a := makeAgent([][]types.AssistantMessageEvent{textDoneEvents("hi there")})

	var events []types.AgentEvent
	unsub := a.Subscribe(func(ev types.AgentEvent) {
		events = append(events, ev)
	})
	defer unsub()

	ctx := context.Background()
	if err := a.Prompt(ctx, "hello"); err != nil {
		t.Fatalf("Prompt error: %v", err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle error: %v", err)
	}

	var found bool
	for _, ev := range events {
		if ev.Type == types.EventAgentEnd {
			found = true
		}
	}
	if !found {
		t.Error("expected EventAgentEnd to be emitted")
	}
}

func TestAgentAbort(t *testing.T) {
	// Provider that blocks until cancelled
	prov := &blockingProvider{}
	a := NewAgent(prov, WithModel(types.Model{ID: "test-model"}))

	ctx := context.Background()
	if err := a.Prompt(ctx, "hello"); err != nil {
		t.Fatalf("Prompt error: %v", err)
	}

	// Give goroutine a moment to start
	time.Sleep(10 * time.Millisecond)

	a.Abort()

	idleCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := a.WaitForIdle(idleCtx); err != nil {
		t.Fatalf("WaitForIdle after Abort: %v", err)
	}
	if a.State().IsRunning {
		t.Error("expected IsRunning=false after Abort")
	}
}

func TestAgentWaitForIdle(t *testing.T) {
	a := makeAgent([][]types.AssistantMessageEvent{textDoneEvents("done")})

	ctx := context.Background()
	if err := a.Prompt(ctx, "test"); err != nil {
		t.Fatalf("Prompt error: %v", err)
	}

	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	if a.State().IsRunning {
		t.Error("expected IsRunning=false after completion")
	}
}

func TestAgentStateIsRunning(t *testing.T) {
	prov := &blockingProvider{}
	a := NewAgent(prov, WithModel(types.Model{ID: "test-model"}))

	ctx := context.Background()
	if err := a.Prompt(ctx, "hi"); err != nil {
		t.Fatalf("Prompt error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	state := a.State()
	if !state.IsRunning {
		t.Error("expected IsRunning=true while blocking provider")
	}

	a.Abort()
	idleCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = a.WaitForIdle(idleCtx)

	state = a.State()
	if state.IsRunning {
		t.Error("expected IsRunning=false after Abort")
	}
}

func TestAgentSteer(t *testing.T) {
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("first"),
			textDoneEvents("after steer"),
		},
	}
	a := NewAgent(prov, WithModel(types.Model{ID: "test-model"}))

	ctx := context.Background()
	if err := a.Prompt(ctx, "hello"); err != nil {
		t.Fatalf("Prompt error: %v", err)
	}

	// Inject steering message
	a.Steer(&types.UserMessage{
		Role:    "user",
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "steer"}},
	})

	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	// Should have called provider at least twice (once for first, once after steering)
	if prov.callIdx < 2 {
		t.Errorf("expected 2 LLM calls, got %d", prov.callIdx)
	}
}

func TestAgentFollowUp(t *testing.T) {
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("stop"),
			textDoneEvents("after followup"),
		},
	}
	a := NewAgent(prov, WithModel(types.Model{ID: "test-model"}))
	a.FollowUp(&types.UserMessage{
		Role:    "user",
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "follow up"}},
	})

	ctx := context.Background()
	if err := a.Prompt(ctx, "go"); err != nil {
		t.Fatalf("Prompt error: %v", err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	if prov.callIdx < 2 {
		t.Errorf("expected 2 LLM calls (follow-up), got %d", prov.callIdx)
	}
}

func TestAgentSubscribeUnsubscribe(t *testing.T) {
	a := makeAgent([][]types.AssistantMessageEvent{textDoneEvents("hello")})

	var count int
	unsub := a.Subscribe(func(_ types.AgentEvent) { count++ })
	unsub() // unsubscribe immediately

	ctx := context.Background()
	if err := a.Prompt(ctx, "hi"); err != nil {
		t.Fatalf("Prompt error: %v", err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 events after unsubscribe, got %d", count)
	}
}

func TestAgentSetModel(t *testing.T) {
	a := makeAgent(nil)
	m := types.Model{ID: "new-model", Provider: "test"}
	a.SetModel(m)
	if a.State().Model.ID != "new-model" {
		t.Errorf("SetModel: got %q want new-model", a.State().Model.ID)
	}
}

func TestAgentSetTools(t *testing.T) {
	a := makeAgent(nil)
	mt := &mockToolWrapper{name: "mytool", fn: nil}
	a.SetTools([]tool.AgentTool{mt})
	if len(a.State().Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(a.State().Tools))
	}
}

func TestAgentClearMessages(t *testing.T) {
	a := makeAgent(nil)
	a.AppendMessage(&types.UserMessage{Role: "user"})
	a.ClearMessages()
	if len(a.State().Messages) != 0 {
		t.Errorf("expected 0 messages after ClearMessages, got %d", len(a.State().Messages))
	}
}
