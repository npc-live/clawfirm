package testutil

import (
	"context"
	"sync"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// MockToolCall records a single Execute invocation.
type MockToolCall struct {
	ID     string
	Params map[string]any
}

// MockTool is a test double for tool.AgentTool that records calls and returns preset results.
type MockTool struct {
	name        string
	description string
	schema      map[string]any
	result      tool.ToolResult
	err         error
	calls       []MockToolCall
	mu          sync.Mutex
	// optional hook to run before returning
	ExecuteHook func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error)
}

// NewMockTool creates a MockTool with the given name and a default empty result.
func NewMockTool(name string) *MockTool {
	return &MockTool{
		name:   name,
		schema: map[string]any{},
	}
}

// SetResult sets the result returned by Execute.
func (m *MockTool) SetResult(content []types.ContentBlock, details any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.result = tool.ToolResult{Content: content, Details: details}
}

// SetError sets the error returned by Execute.
func (m *MockTool) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Calls returns the recorded invocations.
func (m *MockTool) Calls() []MockToolCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MockToolCall, len(m.calls))
	copy(out, m.calls)
	return out
}

// Name returns the tool's name.
func (m *MockTool) Name() string { return m.name }

// Description returns the tool's description.
func (m *MockTool) Description() string { return m.description }

// Label returns the tool's display label.
func (m *MockTool) Label() string { return m.name }

// Schema returns the tool's parameter schema.
func (m *MockTool) Schema() map[string]any { return m.schema }

// Execute records the call and returns the preset result or error.
func (m *MockTool) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	m.mu.Lock()
	m.calls = append(m.calls, MockToolCall{ID: id, Params: params})
	hook := m.ExecuteHook
	result := m.result
	err := m.err
	m.mu.Unlock()

	if hook != nil {
		return hook(ctx, id, params)
	}
	return result, err
}
