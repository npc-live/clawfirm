package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// SearchTool returns an AgentTool that performs semantic memory search.
// The manager is required; if nil a no-op tool is returned.
func SearchTool(m *Manager) tool.AgentTool {
	return &tool.BaseToolImpl{
		ToolName:  "memory_search",
		ToolLabel: "Search memory",
		ToolDescription: `Search long-term memory for relevant information from past sessions.
Returns the most relevant text fragments together with their source file path and line numbers.
Call this tool before answering questions about past work, decisions, or user preferences.`,
		ToolSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Natural-language search query describing what you are looking for.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return (default 5, max 20).",
					"default":     5,
				},
			},
			"required": []string{"query"},
		},
		ExecuteFn: func(ctx context.Context, _ string, params map[string]any, _ func(tool.ToolUpdate)) (tool.ToolResult, error) {
			if m == nil {
				return tool.ToolResult{Content: []types.ContentBlock{
					&types.TextContent{Type: "text", Text: "Memory search is not available."},
				}}, nil
			}

			query, _ := params["query"].(string)
			if strings.TrimSpace(query) == "" {
				return tool.ToolResult{}, fmt.Errorf("memory_search: query is required")
			}

			topK := 5
			if v, ok := params["limit"].(float64); ok && v > 0 {
				topK = int(v)
			}
			if topK > 20 {
				topK = 20
			}

			results, err := m.Search(ctx, query, topK)
			if err != nil {
				return tool.ToolResult{}, fmt.Errorf("memory_search: %w", err)
			}
			if len(results) == 0 {
				return tool.ToolResult{Content: []types.ContentBlock{
					&types.TextContent{Type: "text", Text: "No relevant memory found."},
				}}, nil
			}

			var sb strings.Builder
			for i, r := range results {
				fmt.Fprintf(&sb, "--- Result %d (score %.3f) ---\n", i+1, r.Score)
				fmt.Fprintf(&sb, "File: %s (lines %d–%d)\n", r.FilePath, r.StartLine, r.EndLine)
				sb.WriteString(r.Content)
				sb.WriteString("\n\n")
			}

			return tool.ToolResult{
				Content: []types.ContentBlock{
					&types.TextContent{Type: "text", Text: strings.TrimSpace(sb.String())},
				},
				Details: results,
			}, nil
		},
	}
}

// GetTool returns an AgentTool that reads a specific range of lines from a
// memory file.  It is typically used after memory_search to retrieve full
// context around a search hit.
func GetTool() tool.AgentTool {
	return &tool.BaseToolImpl{
		ToolName:  "memory_get",
		ToolLabel: "Get memory",
		ToolDescription: `Read a specific range of lines from a memory file.
Use after memory_search to retrieve the full text around a result.`,
		ToolSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Absolute path to the memory file (returned by memory_search).",
				},
				"start_line": map[string]any{
					"type":        "integer",
					"description": "First line to read (1-based).",
				},
				"end_line": map[string]any{
					"type":        "integer",
					"description": "Last line to read (1-based, inclusive).",
				},
			},
			"required": []string{"path", "start_line", "end_line"},
		},
		ExecuteFn: func(ctx context.Context, _ string, params map[string]any, _ func(tool.ToolUpdate)) (tool.ToolResult, error) {
			path, _ := params["path"].(string)
			startF, _ := params["start_line"].(float64)
			endF, _ := params["end_line"].(float64)

			if path == "" {
				return tool.ToolResult{}, fmt.Errorf("memory_get: path is required")
			}
			start, end := int(startF), int(endF)
			if start < 1 {
				start = 1
			}
			if end < start {
				end = start
			}

			content, err := ReadLines(path, start, end)
			if err != nil {
				return tool.ToolResult{}, fmt.Errorf("memory_get: %w", err)
			}

			return tool.ToolResult{
				Content: []types.ContentBlock{
					&types.TextContent{Type: "text", Text: content},
				},
			}, nil
		},
	}
}

// Tools returns both memory tools as a slice, ready to be passed to
// tool.Registry.Register or agent.SetTools.
func Tools(m *Manager) []tool.AgentTool {
	return []tool.AgentTool{SearchTool(m), GetTool()}
}
