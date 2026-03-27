package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// Write creates or overwrites a file on disk.
type Write struct{}

func (w *Write) Name() string  { return "write" }
func (w *Write) Label() string { return "Write" }
func (w *Write) Description() string {
	return "Write content to a file, creating it and any necessary parent directories. Overwrites existing files."
}
func (w *Write) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute or relative path to the file to write.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file.",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (w *Write) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	path, _ := params["path"].(string)
	content, _ := params["content"].(string)
	if path == "" {
		return tool.ToolResult{}, fmt.Errorf("write: path is required")
	}
	path = expandHome(path)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return tool.ToolResult{}, fmt.Errorf("write: mkdir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return tool.ToolResult{}, fmt.Errorf("write: %w", err)
	}

	msg := fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)
	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: msg},
		},
	}, nil
}
