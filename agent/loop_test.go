package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

// fakeLLMProvider implements provider.LLMProvider for loop tests.
type fakeLLMProvider struct {
	id        string
	responses [][]types.AssistantMessageEvent
	callIdx   int
}

func (f *fakeLLMProvider) ID() string            { return f.id }
func (f *fakeLLMProvider) Models() []types.Model { return nil }
func (f *fakeLLMProvider) Stream(_ context.Context, _ provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	if f.callIdx >= len(f.responses) {
		ch := make(chan types.AssistantMessageEvent, 1)
		ch <- types.AssistantMessageEvent{
			Type:    types.StreamEventDone,
			Message: &types.AssistantMessage{Role: "assistant", StopReason: types.StopReasonStop},
		}
		close(ch)
		return ch, nil
	}
	evts := f.responses[f.callIdx]
	f.callIdx++
	ch := make(chan types.AssistantMessageEvent, len(evts))
	for _, ev := range evts {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func textDoneEvents(text string) []types.AssistantMessageEvent {
	return []types.AssistantMessageEvent{
		{Type: types.StreamEventDone, Message: &types.AssistantMessage{
			Role:       "assistant",
			Content:    []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
			StopReason: types.StopReasonStop,
			Timestamp:  time.Now().UnixMilli(),
		}},
	}
}

func toolCallDoneEvents(toolName string, args map[string]any) []types.AssistantMessageEvent {
	tc := &types.ToolCall{
		Type:      types.ContentTypeToolCall,
		ID:        "call_" + toolName,
		Name:      toolName,
		Arguments: args,
	}
	return []types.AssistantMessageEvent{
		// Emit ToolCallEnd so the streaming executor can pick it up.
		{Type: types.StreamEventToolCallEnd, ToolCall: tc},
		{Type: types.StreamEventDone, Message: &types.AssistantMessage{
			Role:       "assistant",
			Content:    []types.ContentBlock{tc},
			StopReason: types.StopReasonToolUse,
			Timestamp:  time.Now().UnixMilli(),
		}},
	}
}

func baseConfig() AgentLoopConfig {
	return AgentLoopConfig{
		Model:         types.Model{ID: "test-model"},
		ToolExecution: types.ToolExecutionSequential,
	}
}

func noopEmitFn(_ types.AgentEvent) {}

func TestLoopSingleTextTurn(t *testing.T) {
	prov := &fakeLLMProvider{
		id:        "fake",
		responses: [][]types.AssistantMessageEvent{textDoneEvents("hello!")},
	}
	agentCtx := AgentContext{SystemPrompt: "be helpful"}

	var events []types.AgentEvent
	msgs, err := AgentLoop(
		context.Background(),
		prov,
		[]types.Message{&types.UserMessage{Role: "user", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "hi"},
		}}},
		agentCtx,
		baseConfig(),
		func(ev types.AgentEvent) { events = append(events, ev) },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}
	lastMsg, ok := msgs[len(msgs)-1].(*types.AssistantMessage)
	if !ok {
		t.Fatalf("last message is not *AssistantMessage, got %T", msgs[len(msgs)-1])
	}
	if len(lastMsg.Content) == 0 {
		t.Fatal("expected content in assistant message")
	}
	if tc, ok := lastMsg.Content[0].(*types.TextContent); !ok || tc.Text != "hello!" {
		t.Errorf("text: got %v want hello!", lastMsg.Content[0])
	}

	var gotEnd bool
	for _, ev := range events {
		if ev.Type == types.EventAgentEnd {
			gotEnd = true
		}
	}
	if !gotEnd {
		t.Error("expected agent_end event")
	}
}

func TestLoopToolCallAndResult(t *testing.T) {
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			toolCallDoneEvents("echo", map[string]any{"text": "test"}),
			textDoneEvents("done"),
		},
	}

	echoTool := &mockToolWrapper{
		name: "echo",
		fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
			text, _ := params["text"].(string)
			return tool.ToolResult{
				Content: []types.ContentBlock{
					&types.TextContent{Type: types.ContentTypeText, Text: "echoed: " + text},
				},
			}, nil
		},
	}

	agentCtx := AgentContext{Tools: []tool.AgentTool{echoTool}}
	msgs, err := AgentLoop(
		context.Background(),
		prov,
		[]types.Message{&types.UserMessage{Role: "user", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "echo test"},
		}}},
		agentCtx,
		baseConfig(),
		noopEmitFn,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// user + assistant(toolcall) + tool_result + assistant(text) = 4
	if len(msgs) < 4 {
		t.Fatalf("expected at least 4 messages, got %d: %v", len(msgs), msgRoles(msgs))
	}
	// Verify there is a tool result message
	var foundTool bool
	for _, m := range msgs {
		if m.MessageRole() == "tool" {
			foundTool = true
		}
	}
	if !foundTool {
		t.Error("expected a tool result message")
	}
}

func TestLoopBeforeToolCallBlock(t *testing.T) {
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			toolCallDoneEvents("blocked_tool", map[string]any{}),
			textDoneEvents("ok"),
		},
	}

	agentCtx := AgentContext{}
	config := baseConfig()
	config.BeforeToolCall = func(ctx BeforeToolCallCtx) (BeforeToolCallResult, error) {
		return BeforeToolCallResult{Block: true, Reason: "not allowed in test"}, nil
	}

	msgs, err := AgentLoop(
		context.Background(),
		prov,
		[]types.Message{&types.UserMessage{Role: "user"}},
		agentCtx,
		config,
		noopEmitFn,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var toolResults []*types.ToolResultMessage
	for _, m := range msgs {
		if trm, ok := m.(*types.ToolResultMessage); ok {
			toolResults = append(toolResults, trm)
		}
	}
	if len(toolResults) == 0 {
		t.Error("expected at least one tool result message (blocked)")
	}
	if !toolResults[0].IsError {
		t.Error("expected blocked tool result to have IsError=true")
	}
}

func TestLoopAfterToolCallOverride(t *testing.T) {
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			toolCallDoneEvents("mytool", map[string]any{}),
			textDoneEvents("done"),
		},
	}

	myTool := &mockToolWrapper{
		name: "mytool",
		fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
			return tool.ToolResult{
				Content: []types.ContentBlock{
					&types.TextContent{Type: types.ContentTypeText, Text: "original"},
				},
			}, nil
		},
	}

	agentCtx := AgentContext{Tools: []tool.AgentTool{myTool}}
	config := baseConfig()
	config.AfterToolCall = func(ctx AfterToolCallCtx) (AfterToolCallResult, error) {
		return AfterToolCallResult{
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "overridden"},
			},
		}, nil
	}

	msgs, err := AgentLoop(
		context.Background(),
		prov,
		[]types.Message{&types.UserMessage{Role: "user"}},
		agentCtx,
		config,
		noopEmitFn,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the tool result and verify it was overridden
	for _, m := range msgs {
		if trm, ok := m.(*types.ToolResultMessage); ok {
			if len(trm.Content) > 0 {
				if tc, ok := trm.Content[0].(*types.TextContent); ok {
					if tc.Text != "overridden" {
						t.Errorf("tool result text: got %q want overridden", tc.Text)
					}
				}
			}
			return
		}
	}
	t.Error("no tool result message found")
}

func TestLoopSteeringMessages(t *testing.T) {
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("first"),
			textDoneEvents("second"),
		},
	}

	steeringCalled := false
	config := baseConfig()
	config.GetSteeringMessages = func() ([]types.Message, error) {
		if !steeringCalled {
			steeringCalled = true
			return []types.Message{
				&types.UserMessage{Role: "user", Content: []types.ContentBlock{
					&types.TextContent{Type: types.ContentTypeText, Text: "steering msg"},
				}},
			}, nil
		}
		return nil, nil
	}

	_, err := AgentLoop(
		context.Background(),
		prov,
		[]types.Message{&types.UserMessage{Role: "user"}},
		AgentContext{},
		config,
		noopEmitFn,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov.callIdx < 2 {
		t.Errorf("expected 2 LLM calls (steering), got %d", prov.callIdx)
	}
}

func TestLoopFollowUpMessages(t *testing.T) {
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			textDoneEvents("first stop"),
			textDoneEvents("after followup"),
		},
	}

	followUpCalled := false
	config := baseConfig()
	config.GetFollowUpMessages = func() ([]types.Message, error) {
		if !followUpCalled {
			followUpCalled = true
			return []types.Message{
				&types.UserMessage{Role: "user", Content: []types.ContentBlock{
					&types.TextContent{Type: types.ContentTypeText, Text: "follow up"},
				}},
			}, nil
		}
		return nil, nil
	}

	_, err := AgentLoop(
		context.Background(),
		prov,
		[]types.Message{&types.UserMessage{Role: "user"}},
		AgentContext{},
		config,
		noopEmitFn,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !followUpCalled {
		t.Error("expected follow-up messages to be requested")
	}
	if prov.callIdx < 2 {
		t.Errorf("expected 2 LLM calls for follow-up, got %d", prov.callIdx)
	}
}

func TestLoopCtxCancel(t *testing.T) {
	blockProv := &blockingProvider{}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = AgentLoop(
			ctx,
			blockProv,
			[]types.Message{&types.UserMessage{Role: "user"}},
			AgentContext{},
			baseConfig(),
			noopEmitFn,
		)
	}()

	cancel()
	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Error("AgentLoop did not terminate after ctx cancel")
	}
}

// blockingProvider returns a channel that never sends until ctx is cancelled.
type blockingProvider struct{}

func (b *blockingProvider) ID() string            { return "blocking" }
func (b *blockingProvider) Models() []types.Model { return nil }
func (b *blockingProvider) Stream(ctx context.Context, _ provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	ch := make(chan types.AssistantMessageEvent)
	go func() {
		defer close(ch)
		<-ctx.Done()
	}()
	return ch, nil
}

func msgRoles(msgs []types.Message) []string {
	roles := make([]string, len(msgs))
	for i, m := range msgs {
		roles[i] = m.MessageRole()
	}
	return roles
}

func TestLoopReactiveCompaction(t *testing.T) {
	// First call returns a "prompt too long" error; after compaction,
	// second call succeeds with a normal text response.
	callCount := 0
	prov := &promptTooLongProvider{
		failCount:   1,
		successEvts: textDoneEvents("compacted ok"),
		callCount:   &callCount,
	}

	// Build a compressor that just returns a fixed summary.
	compressor := NewContextCompressor(
		func(_ context.Context, msgs []types.Message) (string, error) {
			return fmt.Sprintf("summary of %d messages", len(msgs)), nil
		},
		CompressorConfig{ContextWindow: 100000, KeepLastN: 2},
	)

	// Seed initial messages so there's something to compress.
	agentCtx := AgentContext{
		SystemPrompt: "be helpful",
		Messages: []types.Message{
			&types.UserMessage{Role: "user", Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "first message"},
			}},
			&types.AssistantMessage{Role: "assistant", Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "first response"},
			}, StopReason: types.StopReasonStop},
			&types.UserMessage{Role: "user", Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "second message"},
			}},
			&types.AssistantMessage{Role: "assistant", Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "second response"},
			}, StopReason: types.StopReasonStop},
		},
	}

	config := baseConfig()
	config.Compressor = compressor
	config.RetryConfig = RetryConfig{MaxRetries: 1, BaseDelay: time.Millisecond}

	msgs, err := AgentLoop(
		context.Background(),
		prov,
		[]types.Message{&types.UserMessage{Role: "user", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "trigger compaction"},
		}}},
		agentCtx,
		config,
		noopEmitFn,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have succeeded after compaction.
	if callCount < 2 {
		t.Errorf("expected at least 2 provider calls (1 fail + 1 success), got %d", callCount)
	}

	// Last message should be the successful assistant response.
	lastMsg, ok := msgs[len(msgs)-1].(*types.AssistantMessage)
	if !ok {
		t.Fatalf("last message is not *AssistantMessage, got %T", msgs[len(msgs)-1])
	}
	if len(lastMsg.Content) == 0 {
		t.Fatal("expected content in assistant message")
	}
	if tc, ok := lastMsg.Content[0].(*types.TextContent); !ok || tc.Text != "compacted ok" {
		t.Errorf("text: got %v want 'compacted ok'", lastMsg.Content[0])
	}

	// Verify compaction happened — first message should be a summary.
	if len(msgs) < 2 {
		t.Fatal("expected at least 2 messages")
	}
	firstMsg, ok := msgs[0].(*types.UserMessage)
	if !ok {
		t.Fatalf("first message after compaction should be UserMessage, got %T", msgs[0])
	}
	if len(firstMsg.Content) > 0 {
		if tc, ok := firstMsg.Content[0].(*types.TextContent); ok {
			if !contains(tc.Text, "Context Summary") {
				t.Errorf("expected compacted summary message, got %q", tc.Text)
			}
		}
	}
}

// promptTooLongProvider returns a 400 "prompt too long" error for the first
// failCount calls, then returns successEvts.
type promptTooLongProvider struct {
	failCount   int
	successEvts []types.AssistantMessageEvent
	callCount   *int
}

func (p *promptTooLongProvider) ID() string            { return "prompt-too-long" }
func (p *promptTooLongProvider) Models() []types.Model { return nil }
func (p *promptTooLongProvider) Stream(_ context.Context, _ provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	*p.callCount++
	if *p.callCount <= p.failCount {
		return nil, &provider.APIError{
			StatusCode: 400,
			Body:       `{"error": {"message": "prompt is too long: 200000 tokens > 100000 maximum"}}`,
			Provider:   "test",
		}
	}
	ch := make(chan types.AssistantMessageEvent, len(p.successEvts))
	for _, ev := range p.successEvts {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// textDoneEventsWithUsage creates a text response with explicit token usage.
func textDoneEventsWithUsage(text string, inputTokens, outputTokens int) []types.AssistantMessageEvent {
	return []types.AssistantMessageEvent{
		{Type: types.StreamEventDone, Message: &types.AssistantMessage{
			Role:       "assistant",
			Content:    []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
			StopReason: types.StopReasonStop,
			Timestamp:  time.Now().UnixMilli(),
			Usage:      types.Usage{Input: inputTokens, Output: outputTokens},
		}},
	}
}

func TestLoopCumulativeUsage(t *testing.T) {
	// Two turns: tool call → text response, each with known usage.
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			// Turn 1: tool call with usage 100/50
			{
				{Type: types.StreamEventToolCallEnd, ToolCall: &types.ToolCall{
					Type: types.ContentTypeToolCall, ID: "c1", Name: "echo", Arguments: map[string]any{"text": "x"},
				}},
				{Type: types.StreamEventDone, Message: &types.AssistantMessage{
					Role:       "assistant",
					Content:    []types.ContentBlock{&types.ToolCall{Type: types.ContentTypeToolCall, ID: "c1", Name: "echo", Arguments: map[string]any{"text": "x"}}},
					StopReason: types.StopReasonToolUse,
					Timestamp:  time.Now().UnixMilli(),
					Usage:      types.Usage{Input: 100, Output: 50},
				}},
			},
			// Turn 2: text response with usage 200/80
			textDoneEventsWithUsage("done", 200, 80),
		},
	}

	echoTool := &mockToolWrapper{
		name: "echo",
		fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
			return tool.ToolResult{
				Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "ok"}},
			}, nil
		},
	}

	var turnEndEvents []types.AgentEvent
	var agentEndEvent types.AgentEvent
	emit := func(ev types.AgentEvent) {
		if ev.Type == types.EventTurnEnd {
			turnEndEvents = append(turnEndEvents, ev)
		}
		if ev.Type == types.EventAgentEnd {
			agentEndEvent = ev
		}
	}

	agentCtx := AgentContext{Tools: []tool.AgentTool{echoTool}}
	_, err := AgentLoop(
		context.Background(), prov,
		[]types.Message{&types.UserMessage{Role: "user"}},
		agentCtx, baseConfig(), emit,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify turn 1 cumulative usage.
	if len(turnEndEvents) < 2 {
		t.Fatalf("expected 2 turn_end events, got %d", len(turnEndEvents))
	}
	cu1 := turnEndEvents[0].CumulativeUsage
	if cu1 == nil {
		t.Fatal("turn 1: CumulativeUsage is nil")
	}
	if cu1.TotalInput != 100 || cu1.TotalOutput != 50 || cu1.TurnCount != 1 {
		t.Errorf("turn 1 cumulative: got in=%d out=%d turns=%d, want in=100 out=50 turns=1",
			cu1.TotalInput, cu1.TotalOutput, cu1.TurnCount)
	}

	// Verify turn 2 cumulative usage (accumulated).
	cu2 := turnEndEvents[1].CumulativeUsage
	if cu2 == nil {
		t.Fatal("turn 2: CumulativeUsage is nil")
	}
	if cu2.TotalInput != 300 || cu2.TotalOutput != 130 || cu2.TurnCount != 2 {
		t.Errorf("turn 2 cumulative: got in=%d out=%d turns=%d, want in=300 out=130 turns=2",
			cu2.TotalInput, cu2.TotalOutput, cu2.TurnCount)
	}

	// Verify agent_end has same cumulative usage.
	cuEnd := agentEndEvent.CumulativeUsage
	if cuEnd == nil {
		t.Fatal("agent_end: CumulativeUsage is nil")
	}
	if cuEnd.TotalInput != 300 || cuEnd.TotalOutput != 130 {
		t.Errorf("agent_end cumulative: got in=%d out=%d, want in=300 out=130",
			cuEnd.TotalInput, cuEnd.TotalOutput)
	}
}

func TestLoopTokenBudget(t *testing.T) {
	// Provider returns 3 tool-call turns, each using 500 input + 200 output tokens.
	// Budget is 1000 tokens — should stop after turn 1 or 2 (700 after turn 1, 1400 after turn 2).
	prov := &fakeLLMProvider{
		id: "fake",
		responses: [][]types.AssistantMessageEvent{
			// Turn 1: tool call, 500+200=700 tokens
			{
				{Type: types.StreamEventToolCallEnd, ToolCall: &types.ToolCall{
					Type: types.ContentTypeToolCall, ID: "c1", Name: "echo", Arguments: map[string]any{"text": "1"},
				}},
				{Type: types.StreamEventDone, Message: &types.AssistantMessage{
					Role:       "assistant",
					Content:    []types.ContentBlock{&types.ToolCall{Type: types.ContentTypeToolCall, ID: "c1", Name: "echo", Arguments: map[string]any{"text": "1"}}},
					StopReason: types.StopReasonToolUse,
					Timestamp:  time.Now().UnixMilli(),
					Usage:      types.Usage{Input: 500, Output: 200},
				}},
			},
			// Turn 2: tool call, another 500+200=700 tokens (cumulative 1400)
			{
				{Type: types.StreamEventToolCallEnd, ToolCall: &types.ToolCall{
					Type: types.ContentTypeToolCall, ID: "c2", Name: "echo", Arguments: map[string]any{"text": "2"},
				}},
				{Type: types.StreamEventDone, Message: &types.AssistantMessage{
					Role:       "assistant",
					Content:    []types.ContentBlock{&types.ToolCall{Type: types.ContentTypeToolCall, ID: "c2", Name: "echo", Arguments: map[string]any{"text": "2"}}},
					StopReason: types.StopReasonToolUse,
					Timestamp:  time.Now().UnixMilli(),
					Usage:      types.Usage{Input: 500, Output: 200},
				}},
			},
			// Turn 3: should never be reached
			textDoneEventsWithUsage("unreachable", 500, 200),
		},
	}

	echoTool := &mockToolWrapper{
		name: "echo",
		fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
			return tool.ToolResult{
				Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "ok"}},
			}, nil
		},
	}

	config := baseConfig()
	config.TokenBudget = 1000 // Stop when total >= 1000

	agentCtx := AgentContext{Tools: []tool.AgentTool{echoTool}}
	_, err := AgentLoop(
		context.Background(), prov,
		[]types.Message{&types.UserMessage{Role: "user"}},
		agentCtx, config, noopEmitFn,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Provider should have been called at most 2 times (budget exceeded after turn 2).
	// Turn 1: 700 < 1000 → continue. Turn 2: 1400 >= 1000 → break.
	if prov.callIdx > 2 {
		t.Errorf("expected at most 2 LLM calls (budget), got %d", prov.callIdx)
	}
	if prov.callIdx < 2 {
		t.Errorf("expected at least 2 LLM calls before budget exceeded, got %d", prov.callIdx)
	}
}

// Avoid unused import error
var _ = fmt.Sprintf
