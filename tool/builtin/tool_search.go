package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

// ToolInfo holds a tool's name and description for search purposes.
type ToolInfo struct {
	Name        string
	Description string
}

// ToolSearch is a meta-tool that lets the LLM discover deferred tools.
// When the LLM calls tool_search, matching tools are returned and their
// names are recorded in ActivatedTools for the agent loop to include
// their schemas in subsequent turns.
type ToolSearch struct {
	// AllTools is the complete list of available tools (including deferred ones).
	AllTools []ToolInfo
	// ActivatedTools tracks tool names activated by previous searches.
	// The agent loop reads this after each tool_search execution.
	ActivatedTools map[string]bool
}

func (t *ToolSearch) Name() string        { return "tool_search" }
func (t *ToolSearch) Label() string       { return "Search tools" }
func (t *ToolSearch) ConcurrencySafe() bool { return true }
func (t *ToolSearch) ShouldDefer() bool   { return false } // tool_search itself is always available

func (t *ToolSearch) Description() string {
	return "Search for available tools by keyword. Use this when you need a capability " +
		"that isn't in your current tool list. Returns matching tool names and descriptions, " +
		"and activates them for use in subsequent turns."
}

func (t *ToolSearch) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Keyword or description of the capability you need (e.g. 'http', 'time', 'patch').",
			},
		},
		"required": []string{"query"},
	}
}

func (t *ToolSearch) Execute(_ context.Context, _ string, params map[string]any, _ func(tool.ToolUpdate)) (tool.ToolResult, error) {
	query, _ := params["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return tool.ToolResult{}, fmt.Errorf("tool_search: query is required")
	}

	queryLower := strings.ToLower(query)
	keywords := strings.Fields(queryLower)

	var matches []ToolInfo
	for _, ti := range t.AllTools {
		nameLower := strings.ToLower(ti.Name)
		descLower := strings.ToLower(ti.Description)
		for _, kw := range keywords {
			if strings.Contains(nameLower, kw) || strings.Contains(descLower, kw) {
				matches = append(matches, ti)
				break
			}
		}
	}

	if len(matches) == 0 {
		return tool.ToolResult{
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "No tools found matching: " + query},
			},
		}, nil
	}

	// Activate matched tools and build response.
	var sb strings.Builder
	activated := make([]string, 0, len(matches))
	fmt.Fprintf(&sb, "Found %d tool(s) matching %q:\n\n", len(matches), query)
	for _, m := range matches {
		fmt.Fprintf(&sb, "- **%s**: %s\n", m.Name, m.Description)
		if t.ActivatedTools != nil {
			t.ActivatedTools[m.Name] = true
		}
		activated = append(activated, m.Name)
	}
	sb.WriteString("\nThese tools are now available for use.")

	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: sb.String()},
		},
		Details: activated, // agent loop reads this to update activatedTools
	}, nil
}
