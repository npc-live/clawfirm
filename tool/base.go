package tool

import (
	"context"

	"github.com/ai-gateway/pi-go/types"
)

// ToolUpdate carries an incremental update from a tool during execution.
type ToolUpdate struct {
	Content []types.ContentBlock
	Details any
}

// ToolResult is the final output of a tool invocation.
type ToolResult struct {
	Content []types.ContentBlock
	Details any
}

// AgentTool is the interface that all tools registered with an Agent must implement.
type AgentTool interface {
	// Name returns the unique identifier used in tool call requests.
	Name() string
	// Description returns a human-readable explanation for the LLM.
	Description() string
	// Label returns a short display label for UI surfaces.
	Label() string
	// Schema returns the JSON Schema object describing the tool's parameters.
	Schema() map[string]any
	// Execute runs the tool and may emit incremental updates via onUpdate.
	Execute(ctx context.Context, id string, params map[string]any, onUpdate func(ToolUpdate)) (ToolResult, error)
}

// BaseToolImpl is a convenience struct for implementing AgentTool via function fields.
type BaseToolImpl struct {
	ToolName        string
	ToolDescription string
	ToolLabel       string
	ToolSchema      map[string]any
	ExecuteFn       func(ctx context.Context, id string, params map[string]any, onUpdate func(ToolUpdate)) (ToolResult, error)
}

// Name returns the tool's name.
func (b *BaseToolImpl) Name() string { return b.ToolName }

// Description returns the tool's description.
func (b *BaseToolImpl) Description() string { return b.ToolDescription }

// Label returns the tool's display label.
func (b *BaseToolImpl) Label() string { return b.ToolLabel }

// Schema returns the tool's parameter JSON schema.
func (b *BaseToolImpl) Schema() map[string]any { return b.ToolSchema }

// Execute calls the underlying ExecuteFn.
func (b *BaseToolImpl) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(ToolUpdate)) (ToolResult, error) {
	if b.ExecuteFn != nil {
		return b.ExecuteFn(ctx, id, params, onUpdate)
	}
	return ToolResult{}, nil
}
