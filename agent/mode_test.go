package agent

import (
	"context"
	"testing"

	"github.com/ai-gateway/pi-go/types"
)

// ── A. SteeringMode / FollowUpMode ──────────────────────────────────────────

func TestSteeringModeAll(t *testing.T) {
	// Default mode = all: both queued steering messages should be delivered in
	// a single batch after the first turn ends.
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("turn1"),
			textDoneEvents("turn2"), // second turn after steering batch
		},
	}
	a := NewAgent(prov,
		WithModel(types.Model{ID: "m"}),
		WithSteeringMode(types.SteeringModeAll),
	)

	a.Steer(&types.UserMessage{Role: "user", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: "steer1"},
	}})
	a.Steer(&types.UserMessage{Role: "user", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: "steer2"},
	}})

	ctx := context.Background()
	if err := a.Prompt(ctx, "go"); err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	// Both steering messages consumed → 2 LLM calls total.
	if prov.callIdx != 2 {
		t.Errorf("SteeringModeAll: want 2 LLM calls, got %d", prov.callIdx)
	}
	// Steering queue should be empty after consumption.
	if a.steeringQ.Len() != 0 {
		t.Errorf("steeringQ not empty: len=%d", a.steeringQ.Len())
	}
}

func TestSteeringModeOneAtATime(t *testing.T) {
	// one-at-a-time: 2 queued steering messages → 3 LLM calls (initial + 1 per steer)
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("turn1"),
			textDoneEvents("after-steer1"),
			textDoneEvents("after-steer2"),
		},
	}
	a := NewAgent(prov,
		WithModel(types.Model{ID: "m"}),
		WithSteeringMode(types.SteeringModeOneAtATime),
	)

	a.Steer(&types.UserMessage{Role: "user", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: "steer1"},
	}})
	a.Steer(&types.UserMessage{Role: "user", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: "steer2"},
	}})

	ctx := context.Background()
	if err := a.Prompt(ctx, "go"); err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	if prov.callIdx != 3 {
		t.Errorf("SteeringModeOneAtATime: want 3 LLM calls, got %d", prov.callIdx)
	}
}

func TestFollowUpModeAll(t *testing.T) {
	// all: both follow-up messages delivered at once → 2 LLM calls total.
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("initial"),
			textDoneEvents("after-followups"),
		},
	}
	a := NewAgent(prov,
		WithModel(types.Model{ID: "m"}),
		WithFollowUpMode(types.FollowUpModeAll),
	)

	a.FollowUp(&types.UserMessage{Role: "user", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: "fu1"},
	}})
	a.FollowUp(&types.UserMessage{Role: "user", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: "fu2"},
	}})

	ctx := context.Background()
	if err := a.Prompt(ctx, "go"); err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	if prov.callIdx != 2 {
		t.Errorf("FollowUpModeAll: want 2 LLM calls, got %d", prov.callIdx)
	}
}

func TestFollowUpModeOneAtATime(t *testing.T) {
	// one-at-a-time: 2 follow-up messages → 3 LLM calls.
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("initial"),
			textDoneEvents("after-fu1"),
			textDoneEvents("after-fu2"),
		},
	}
	a := NewAgent(prov,
		WithModel(types.Model{ID: "m"}),
		WithFollowUpMode(types.FollowUpModeOneAtATime),
	)

	a.FollowUp(&types.UserMessage{Role: "user", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: "fu1"},
	}})
	a.FollowUp(&types.UserMessage{Role: "user", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: "fu2"},
	}})

	ctx := context.Background()
	if err := a.Prompt(ctx, "go"); err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	if prov.callIdx != 3 {
		t.Errorf("FollowUpModeOneAtATime: want 3 LLM calls, got %d", prov.callIdx)
	}
}

func TestSetSteeringMode(t *testing.T) {
	a := NewAgent(&fakeLLMProvider{id: "f"}, WithModel(types.Model{ID: "m"}))
	if a.steeringMode != types.SteeringModeAll {
		t.Errorf("default steeringMode: want %q got %q", types.SteeringModeAll, a.steeringMode)
	}
	a.SetSteeringMode(types.SteeringModeOneAtATime)
	if a.steeringMode != types.SteeringModeOneAtATime {
		t.Errorf("after SetSteeringMode: want %q got %q", types.SteeringModeOneAtATime, a.steeringMode)
	}
}

func TestSetFollowUpMode(t *testing.T) {
	a := NewAgent(&fakeLLMProvider{id: "f"}, WithModel(types.Model{ID: "m"}))
	if a.followUpMode != types.FollowUpModeAll {
		t.Errorf("default followUpMode: want %q got %q", types.FollowUpModeAll, a.followUpMode)
	}
	a.SetFollowUpMode(types.FollowUpModeOneAtATime)
	if a.followUpMode != types.FollowUpModeOneAtATime {
		t.Errorf("after SetFollowUpMode: want %q got %q", types.FollowUpModeOneAtATime, a.followUpMode)
	}
}

// ── B. AgentLoopContinue ────────────────────────────────────────────────────

func TestAgentLoopContinue(t *testing.T) {
	// AgentLoopContinue should run the loop from the pre-loaded context
	// without prepending any new messages.
	prov := &fakeLLMProvider{
		id:        "fake",
		responses: [][]types.AssistantMessageEvent{textDoneEvents("resumed")},
	}

	agentCtx := AgentContext{
		SystemPrompt: "you are helpful",
		Messages: []types.Message{
			&types.UserMessage{
				Role: "user",
				Content: []types.ContentBlock{
					&types.TextContent{Type: types.ContentTypeText, Text: "existing prompt"},
				},
			},
		},
	}
	cfg := AgentLoopConfig{
		Model:         types.Model{ID: "test-model"},
		ToolExecution: types.ToolExecutionSequential,
	}

	var gotEnd bool
	emit := func(ev types.AgentEvent) {
		if ev.Type == types.EventAgentEnd {
			gotEnd = true
		}
	}

	msgs, err := AgentLoopContinue(context.Background(), prov, agentCtx, cfg, emit)
	if err != nil {
		t.Fatalf("AgentLoopContinue: %v", err)
	}
	if !gotEnd {
		t.Error("expected EventAgentEnd")
	}
	// Original message + assistant reply = 2 messages
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
	// Provider called exactly once (the existing context message was the prompt)
	if prov.callIdx != 1 {
		t.Errorf("expected 1 LLM call, got %d", prov.callIdx)
	}
}

func TestAgentContinue(t *testing.T) {
	// Agent.Continue should resume from existing messages without a new prompt.
	prov := &fakeLLMProvider{
		id:        "fake",
		responses: [][]types.AssistantMessageEvent{textDoneEvents("continued")},
	}
	a := NewAgent(prov, WithModel(types.Model{ID: "m"}))

	// Pre-load a message into history so Continue has something to send.
	a.AppendMessage(&types.UserMessage{
		Role: "user",
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "pre-loaded"},
		},
	})

	var gotEnd bool
	a.Subscribe(func(ev types.AgentEvent) {
		if ev.Type == types.EventAgentEnd {
			gotEnd = true
		}
	})

	ctx := context.Background()
	if err := a.Continue(ctx); err != nil {
		t.Fatalf("Continue: %v", err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	if !gotEnd {
		t.Error("Continue: expected EventAgentEnd")
	}
	// Provider should have been called once with the pre-loaded message.
	if prov.callIdx != 1 {
		t.Errorf("Continue: expected 1 LLM call, got %d", prov.callIdx)
	}
	// History: pre-loaded user msg + assistant reply
	if n := len(a.State().Messages); n != 2 {
		t.Errorf("Continue: expected 2 messages in history, got %d", n)
	}
}
