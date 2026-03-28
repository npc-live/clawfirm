package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// Edit replaces an exact string in a file.
// Fails if the old_string is not found or appears more than once.
type Edit struct{}

func (e *Edit) Name() string  { return "edit" }
func (e *Edit) Label() string { return "Edit" }
func (e *Edit) Description() string {
	return "Replace an exact string in a file. The old_string must appear exactly once. Use read first to confirm the text."
}
func (e *Edit) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute or relative path to the file to edit.",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "Exact text to find and replace. Must match the file exactly.",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "Text to replace old_string with.",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}

func (e *Edit) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	path, _ := params["path"].(string)
	oldStr, _ := params["old_string"].(string)
	newStr, _ := params["new_string"].(string)
	if path == "" {
		return tool.ToolResult{}, fmt.Errorf("edit: path is required")
	}
	path = expandHome(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("edit: %w", err)
	}
	content := string(data)

	count := strings.Count(content, oldStr)
	if count == 0 {
		return tool.ToolResult{}, fmt.Errorf("edit: old_string not found in %s", path)
	}
	if count > 1 {
		return tool.ToolResult{}, fmt.Errorf("edit: old_string found %d times in %s; provide more context to make it unique", count, path)
	}

	updated := strings.Replace(content, oldStr, newStr, 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return tool.ToolResult{}, fmt.Errorf("edit: %w", err)
	}

	msg := fmt.Sprintf("Successfully edited %s", path)
	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: msg},
		},
	}, nil
}
