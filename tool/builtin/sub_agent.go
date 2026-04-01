package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

// SubAgentRequest describes a task for a sub-agent to handle autonomously.
type SubAgentRequest struct {
	Description string `json:"description"` // short (3-5 word) summary
	Prompt      string `json:"prompt"`      // detailed task instructions
	Model       string `json:"model,omitempty"`
}

// SubAgentResult is the output of a completed sub-agent run.
type SubAgentResult struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// SubAgent launches an independent agent to handle a complex subtask.
// The sub-agent shares the same tool set (minus sub_agent itself to prevent
// recursion) and runs with its own message history.
type SubAgent struct {
	// SpawnFn is injected by the application layer. It creates and runs a
	// sub-agent, blocking until completion. If nil, the tool returns an error.
	SpawnFn func(ctx context.Context, req SubAgentRequest) (SubAgentResult, error)
}

func (s *SubAgent) Name() string  { return "sub_agent" }
func (s *SubAgent) Label() string { return "Sub Agent" }

func (s *SubAgent) Description() string {
	return "Launch a sub-agent to handle a complex subtask autonomously. " +
		"The sub-agent runs in an independent context with its own message history, " +
		"shares the same tools, and returns a single result when done. " +
		"Use for: parallel research, isolated code changes, multi-step tasks " +
		"that would clutter the main conversation."
}

func (s *SubAgent) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description": map[string]any{
				"type":        "string",
				"description": "Short (3-5 word) summary of what the sub-agent will do.",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Detailed task instructions for the sub-agent.",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "Optional model override (e.g. use a faster model for simple tasks).",
			},
		},
		"required": []string{"description", "prompt"},
	}
}

func (s *SubAgent) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	desc, _ := params["description"].(string)
	prompt, _ := params["prompt"].(string)
	model, _ := params["model"].(string)

	desc = strings.TrimSpace(desc)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return tool.ToolResult{}, fmt.Errorf("sub_agent: prompt is required")
	}
	if desc == "" {
		desc = "sub-agent task"
	}

	if s.SpawnFn == nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText,
					Text: "sub_agent is not available in this environment."},
			},
		}, nil
	}

	onUpdate(tool.ToolUpdate{Details: fmt.Sprintf("Spawning sub-agent: %s", desc)})

	result, err := s.SpawnFn(ctx, SubAgentRequest{
		Description: desc,
		Prompt:      prompt,
		Model:       model,
	})
	if err != nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText,
					Text: fmt.Sprintf("Sub-agent failed: %v", err)},
			},
		}, nil
	}

	text := result.Output
	if result.Error != "" {
		text = fmt.Sprintf("Sub-agent completed with error: %s\n\nPartial output:\n%s", result.Error, result.Output)
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: text},
		},
	}, nil
}
