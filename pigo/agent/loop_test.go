package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
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

// Avoid unused import error
var _ = fmt.Sprintf
