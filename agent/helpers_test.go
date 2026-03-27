package agent

import (
	"context"

	"github.com/ai-gateway/pi-go/tool"
)

// mockToolWrapper is a simple AgentTool adapter used across agent tests.
type mockToolWrapper struct {
	name string
	fn   func(ctx context.Context, id string, params map[string]any) (tool.ToolResult, error)
}

func (m *mockToolWrapper) Name() string              { return m.name }
func (m *mockToolWrapper) Description() string       { return "" }
func (m *mockToolWrapper) Label() string             { return m.name }
func (m *mockToolWrapper) Schema() map[string]any    { return map[string]any{} }
func (m *mockToolWrapper) Execute(ctx context.Context, id string, params map[string]any, _ func(tool.ToolUpdate)) (tool.ToolResult, error) {
	return m.fn(ctx, id, params)
}
