package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

const (
	readMaxLines = 40_000
	readMaxBytes = 30 * 1024 // 30 KB
)

// Read reads a file from disk and returns its contents as text.
// Supports offset/limit for paginating large files.
type Read struct{}

func (r *Read) Name() string  { return "read" }
func (r *Read) Label() string { return "Read" }
func (r *Read) Description() string {
	return "Read a file from the local filesystem. For large files, use offset and limit to paginate."
}
func (r *Read) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute or relative path to the file to read.",
			},
			"offset": map[string]any{
				"type":        "number",
				"description": "1-indexed line number to start reading from.",
			},
			"limit": map[string]any{
				"type":        "number",
				"description": "Maximum number of lines to return.",
			},
		},
		"required": []string{"path"},
	}
}

func (r *Read) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return tool.ToolResult{}, fmt.Errorf("read: path is required")
	}
	path = expandHome(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("read: %w", err)
	}

	allLines := strings.Split(string(data), "\n")
	totalLines := len(allLines)

	// Apply offset (1-indexed).
	startLine := 0
	if off, ok := params["offset"].(float64); ok && off > 1 {
		startLine = int(off) - 1
		if startLine >= totalLines {
			startLine = totalLines
		}
	}

	// Apply limit.
	endLine := totalLines
	if lim, ok := params["limit"].(float64); ok && lim > 0 {
		endLine = startLine + int(lim)
		if endLine > totalLines {
			endLine = totalLines
		}
	}

	lines := allLines[startLine:endLine]

	// Enforce hard caps.
	truncated := false
	suffix := ""
	if len(lines) > readMaxLines {
		lines = lines[:readMaxLines]
		truncated = true
	}
	content := strings.Join(lines, "\n")
	if len(content) > readMaxBytes {
		content = content[:readMaxBytes]
		truncated = true
	}
	if truncated {
		nextOffset := startLine + len(lines) + 1
		suffix = fmt.Sprintf("\n[Truncated. Total lines: %d. Use offset=%d to continue.]", totalLines, nextOffset)
	} else if endLine < totalLines {
		suffix = fmt.Sprintf("\n[Showing lines %d–%d of %d. Use offset=%d to continue.]",
			startLine+1, endLine, totalLines, endLine+1)
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: content + suffix},
		},
	}, nil
}
