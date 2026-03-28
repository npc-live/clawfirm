package builtin

import (
	"context"

	"github.com/ai-gateway/pi-go/tool"
)

// Noop is a built-in tool that does nothing and returns an empty result.
type Noop struct{}

// Name returns "noop".
func (n *Noop) Name() string { return "noop" }

// Description describes the noop tool.
func (n *Noop) Description() string { return "Does nothing and returns an empty result." }

// Label returns the display label.
func (n *Noop) Label() string { return "Noop" }

// Schema returns an empty parameter schema.
func (n *Noop) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute returns an empty result immediately.
func (n *Noop) Execute(_ context.Context, _ string, _ map[string]any, _ func(tool.ToolUpdate)) (tool.ToolResult, error) {
	return tool.ToolResult{}, nil
}
