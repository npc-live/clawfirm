package builtin

import (
	"context"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// Echo is a built-in tool that returns its input text unchanged.
type Echo struct{}

// Name returns "echo".
func (e *Echo) Name() string { return "echo" }

// Description describes the echo tool.
func (e *Echo) Description() string { return "Echoes the input text back as output." }

// Label returns the display label.
func (e *Echo) Label() string { return "Echo" }

// Schema returns the JSON schema for echo parameters.
func (e *Echo) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Text to echo back",
			},
		},
		"required": []string{"text"},
	}
}

// Execute returns the input text as a TextContent result.
func (e *Echo) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	text, _ := params["text"].(string)
	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: text},
		},
	}, nil
}
