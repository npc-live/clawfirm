package provider

import (
	"context"

	"github.com/ai-gateway/pi-go/types"
)

// ToolSchema describes a tool that can be invoked by the LLM.
type ToolSchema struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema object
}

// LLMRequest is the unified input to an LLM provider's Stream method.
type LLMRequest struct {
	Model        types.Model
	SystemPrompt string
	Messages     []types.Message
	Tools        []ToolSchema
	Options      types.StreamOptions
}

// LLMProvider is the core interface that every LLM provider must implement.
type LLMProvider interface {
	// ID returns the provider's unique identifier, e.g. "anthropic".
	ID() string
	// Stream initiates a streaming request and returns a channel of events.
	// The channel must be closed when the stream ends or ctx is cancelled.
	// Failures are reported as StreamEventError events rather than a non-nil error.
	Stream(ctx context.Context, req LLMRequest) (<-chan types.AssistantMessageEvent, error)
	// Models returns the list of models supported by this provider.
	Models() []types.Model
}
