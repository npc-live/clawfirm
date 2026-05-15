package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

// Write creates or overwrites a file on disk.
type Write struct{}

func (w *Write) Name() string  { return "write" }
func (w *Write) Label() string { return "Write" }
func (w *Write) Description() string {
	return "Write content to a file, creating it and any necessary parent directories. Use append=true for additional chunks when writing content longer than the content maxLength."
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
				"description": "Content to write to the file. Keep each call at or below maxLength; split larger files into chunks.",
				"maxLength":   4000,
			},
			"append": map[string]any{
				"type":        "boolean",
				"description": "Append content instead of overwriting. For large files, first call with append=false, then continue with append=true chunks.",
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
	appendMode, _ := params["append"].(bool)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return tool.ToolResult{}, fmt.Errorf("write: mkdir: %w", err)
	}
	if appendMode {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return tool.ToolResult{}, fmt.Errorf("write: open append: %w", err)
		}
		if _, err := f.WriteString(content); err != nil {
			_ = f.Close()
			return tool.ToolResult{}, fmt.Errorf("write: append: %w", err)
		}
		if err := f.Close(); err != nil {
			return tool.ToolResult{}, fmt.Errorf("write: close: %w", err)
		}
	} else {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return tool.ToolResult{}, fmt.Errorf("write: %w", err)
		}
	}

	action := "wrote"
	if appendMode {
		action = "appended"
	}
	msg := fmt.Sprintf("Successfully %s %d bytes to %s", action, len(content), path)
	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: msg},
		},
	}, nil
}
