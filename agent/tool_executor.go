package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// BeforeToolCallCtx is passed to the BeforeToolCall hook.
type BeforeToolCallCtx struct {
	AssistantMsg *types.AssistantMessage
	ToolCall     types.ToolCall
	Args         map[string]any
}

// BeforeToolCallResult is returned by the BeforeToolCall hook.
type BeforeToolCallResult struct {
	// Block set to true prevents the tool from being executed.
	Block  bool
	Reason string
}

// AfterToolCallCtx is passed to the AfterToolCall hook.
type AfterToolCallCtx struct {
	AssistantMsg *types.AssistantMessage
	ToolCall     types.ToolCall
	Args         map[string]any
	Result       ToolResult
	IsError      bool
}

// AfterToolCallResult is returned by the AfterToolCall hook.
// Nil fields mean "keep original".
type AfterToolCallResult struct {
	Content []types.ContentBlock // nil = keep original
	Details any                  // nil = keep original
	IsError *bool                // nil = keep original
}

// ToolResult is the outcome of a tool execution.
type ToolResult struct {
	Content []types.ContentBlock
	Details any
}

// ExecuteSequential runs tool calls in order, returning results in the same order.
func ExecuteSequential(
	ctx context.Context,
	calls []types.ToolCall,
	registry *tool.Registry,
	assistantMsg *types.AssistantMessage,
	beforeHook func(BeforeToolCallCtx) (BeforeToolCallResult, error),
	afterHook func(AfterToolCallCtx) (AfterToolCallResult, error),
	emit func(types.AgentEvent),
) ([]types.ToolResultMessage, error) {
	results := make([]types.ToolResultMessage, 0, len(calls))
	for _, tc := range calls {
		result, err := executeOne(ctx, tc, registry, assistantMsg, beforeHook, afterHook, emit)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// ExecuteParallel runs all tool calls concurrently, returning results in original order.
func ExecuteParallel(
	ctx context.Context,
	calls []types.ToolCall,
	registry *tool.Registry,
	assistantMsg *types.AssistantMessage,
	beforeHook func(BeforeToolCallCtx) (BeforeToolCallResult, error),
	afterHook func(AfterToolCallCtx) (AfterToolCallResult, error),
	emit func(types.AgentEvent),
) ([]types.ToolResultMessage, error) {
	results := make([]types.ToolResultMessage, len(calls))
	errs := make([]error, len(calls))
	var wg sync.WaitGroup

	for i, tc := range calls {
		wg.Add(1)
		go func(idx int, call types.ToolCall) {
			defer wg.Done()
			r, err := executeOne(ctx, call, registry, assistantMsg, beforeHook, afterHook, emit)
			results[idx] = r
			errs[idx] = err
		}(i, tc)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

// executeOne handles a single tool call, applying hooks and returning a ToolResultMessage.
func executeOne(
	ctx context.Context,
	tc types.ToolCall,
	registry *tool.Registry,
	assistantMsg *types.AssistantMessage,
	beforeHook func(BeforeToolCallCtx) (BeforeToolCallResult, error),
	afterHook func(AfterToolCallCtx) (AfterToolCallResult, error),
	emit func(types.AgentEvent),
) (types.ToolResultMessage, error) {
	emit(types.AgentEvent{
		Type:       types.EventToolExecutionStart,
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		ToolArgs:   tc.Arguments,
	})

	// Before hook
	if beforeHook != nil {
		res, err := beforeHook(BeforeToolCallCtx{
			AssistantMsg: assistantMsg,
			ToolCall:     tc,
			Args:         tc.Arguments,
		})
		if err != nil {
			return errorToolResult(tc, fmt.Sprintf("before hook error: %s", err)), nil
		}
		if res.Block {
			reason := res.Reason
			if reason == "" {
				reason = "tool execution blocked"
			}
			errResult := errorToolResult(tc, reason)
			emit(types.AgentEvent{
				Type:        types.EventToolExecutionEnd,
				ToolCallID:  tc.ID,
				ToolName:    tc.Name,
				ToolResult:  reason,
				ToolIsError: true,
			})
			return errResult, nil
		}
	}

	// Look up tool
	t, ok := registry.Get(tc.Name)
	if !ok {
		errMsg := fmt.Sprintf("tool not found: %s", tc.Name)
		errResult := errorToolResult(tc, errMsg)
		emit(types.AgentEvent{
			Type:        types.EventToolExecutionEnd,
			ToolCallID:  tc.ID,
			ToolName:    tc.Name,
			ToolResult:  errMsg,
			ToolIsError: true,
		})
		return errResult, nil
	}

	// Execute
	onUpdate := func(u tool.ToolUpdate) {
		emit(types.AgentEvent{
			Type:          types.EventToolExecutionUpdate,
			ToolCallID:    tc.ID,
			ToolName:      tc.Name,
			PartialResult: u.Details,
		})
	}

	toolResult, execErr := t.Execute(ctx, tc.ID, tc.Arguments, onUpdate)
	isError := execErr != nil
	var content []types.ContentBlock
	var details any
	if execErr != nil {
		content = []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: execErr.Error()},
		}
	} else {
		content = toolResult.Content
		details = toolResult.Details
	}

	result := ToolResult{Content: content, Details: details}

	// After hook
	if afterHook != nil {
		afterRes, err := afterHook(AfterToolCallCtx{
			AssistantMsg: assistantMsg,
			ToolCall:     tc,
			Args:         tc.Arguments,
			Result:       result,
			IsError:      isError,
		})
		if err == nil {
			if afterRes.Content != nil {
				content = afterRes.Content
			}
			if afterRes.Details != nil {
				details = afterRes.Details
			}
			if afterRes.IsError != nil {
				isError = *afterRes.IsError
			}
		}
	}

	emit(types.AgentEvent{
		Type:        types.EventToolExecutionEnd,
		ToolCallID:  tc.ID,
		ToolName:    tc.Name,
		ToolResult:  details,
		ToolIsError: isError,
	})

	return types.ToolResultMessage{
		Role:       "tool",
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Content:    content,
		Details:    details,
		IsError:    isError,
		Timestamp:  time.Now().UnixMilli(),
	}, nil
}

// errorToolResult builds a ToolResultMessage representing a tool error.
func errorToolResult(tc types.ToolCall, errMsg string) types.ToolResultMessage {
	return types.ToolResultMessage{
		Role:       "tool",
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: errMsg},
		},
		IsError:   true,
		Timestamp: time.Now().UnixMilli(),
	}
}
