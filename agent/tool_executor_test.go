package agent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

func makeRegistry(tools ...tool.AgentTool) *tool.Registry {
	r := tool.NewRegistry()
	for _, t := range tools {
		r.Register(t)
	}
	return r
}

func noOpEmit(_ types.AgentEvent) {}

func TestExecuteSequentialOrder(t *testing.T) {
	var order []string

	mt1 := &mockToolWrapper{name: "tool1", fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
		order = append(order, "tool1")
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "result1"}},
		}, nil
	}}
	mt2 := &mockToolWrapper{name: "tool2", fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
		order = append(order, "tool2")
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "result2"}},
		}, nil
	}}

	reg := makeRegistry(mt1, mt2)
	calls := []types.ToolCall{
		{ID: "c1", Name: "tool1", Type: types.ContentTypeToolCall, Arguments: map[string]any{}},
		{ID: "c2", Name: "tool2", Type: types.ContentTypeToolCall, Arguments: map[string]any{}},
	}

	results, err := ExecuteSequential(context.Background(), calls, reg, nil, nil, nil, noOpEmit)
	if err != nil {
		t.Fatalf("ExecuteSequential error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if order[0] != "tool1" || order[1] != "tool2" {
		t.Errorf("order: got %v want [tool1 tool2]", order)
	}
	if results[0].ToolName != "tool1" || results[1].ToolName != "tool2" {
		t.Errorf("result order wrong: %v %v", results[0].ToolName, results[1].ToolName)
	}
}

func TestExecuteParallelOrder(t *testing.T) {
	var callCount int64

	mt1 := &mockToolWrapper{name: "slow1", fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(30 * time.Millisecond)
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "r1"}},
		}, nil
	}}
	mt2 := &mockToolWrapper{name: "slow2", fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(30 * time.Millisecond)
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "r2"}},
		}, nil
	}}

	reg := makeRegistry(mt1, mt2)
	calls := []types.ToolCall{
		{ID: "c1", Name: "slow1", Type: types.ContentTypeToolCall, Arguments: map[string]any{}},
		{ID: "c2", Name: "slow2", Type: types.ContentTypeToolCall, Arguments: map[string]any{}},
	}

	start := time.Now()
	results, err := ExecuteParallel(context.Background(), calls, reg, nil, nil, nil, noOpEmit)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ExecuteParallel error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Results must be in original order
	if results[0].ToolName != "slow1" || results[1].ToolName != "slow2" {
		t.Errorf("result order wrong: %v %v", results[0].ToolName, results[1].ToolName)
	}
	// Parallel execution should take ~30ms, not ~60ms
	if elapsed > 55*time.Millisecond {
		t.Errorf("parallel execution too slow (%v); likely not running concurrently", elapsed)
	}
}

func TestExecuteSequentialToolError(t *testing.T) {
	mt1 := &mockToolWrapper{name: "failing", fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
		return tool.ToolResult{}, errors.New("tool blew up")
	}}
	mt2 := &mockToolWrapper{name: "ok", fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "ok result"}},
		}, nil
	}}

	reg := makeRegistry(mt1, mt2)
	calls := []types.ToolCall{
		{ID: "c1", Name: "failing", Type: types.ContentTypeToolCall, Arguments: map[string]any{}},
		{ID: "c2", Name: "ok", Type: types.ContentTypeToolCall, Arguments: map[string]any{}},
	}

	results, err := ExecuteSequential(context.Background(), calls, reg, nil, nil, nil, noOpEmit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].IsError {
		t.Error("expected first result to be an error")
	}
	if results[1].IsError {
		t.Error("expected second result to be ok")
	}
}

func TestBeforeToolCallBlock(t *testing.T) {
	mt := &mockToolWrapper{name: "blocked", fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
		t.Error("blocked tool should not execute")
		return tool.ToolResult{}, nil
	}}
	reg := makeRegistry(mt)
	calls := []types.ToolCall{
		{ID: "c1", Name: "blocked", Type: types.ContentTypeToolCall, Arguments: map[string]any{}},
	}

	before := func(ctx BeforeToolCallCtx) (BeforeToolCallResult, error) {
		return BeforeToolCallResult{Block: true, Reason: "not allowed"}, nil
	}

	results, err := ExecuteSequential(context.Background(), calls, reg, nil, before, nil, noOpEmit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsError {
		t.Error("expected blocked result to have IsError=true")
	}
}

func TestAfterToolCallOverrideContent(t *testing.T) {
	mt := &mockToolWrapper{name: "t1", fn: func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error) {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "original"}},
		}, nil
	}}
	reg := makeRegistry(mt)
	calls := []types.ToolCall{
		{ID: "c1", Name: "t1", Type: types.ContentTypeToolCall, Arguments: map[string]any{}},
	}

	after := func(ctx AfterToolCallCtx) (AfterToolCallResult, error) {
		return AfterToolCallResult{
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "overridden"}},
		}, nil
	}

	results, err := ExecuteSequential(context.Background(), calls, reg, nil, nil, after, noOpEmit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result")
	}
	tc, ok := results[0].Content[0].(*types.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if tc.Text != "overridden" {
		t.Errorf("expected overridden, got %q", tc.Text)
	}
}
