package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

// AskUserQuestion is a structured question presented to the user.
type AskUserQuestion struct {
	Question    string           `json:"question"`
	Options     []AskUserOption  `json:"options,omitempty"`
	MultiSelect bool             `json:"multi_select,omitempty"`
}

// AskUserOption is a single choice in a structured question.
type AskUserOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// AskUserAnswer is the user's response to a structured question.
type AskUserAnswer struct {
	Selected []string `json:"selected"`       // selected option labels
	Text     string   `json:"text,omitempty"` // free-text (when user picks "Other")
}

// AskUser presents a structured question to the user via a callback.
// If OnQuestion is nil, the tool returns an error guiding the LLM
// to ask the question in plain text instead.
type AskUser struct {
	// OnQuestion is injected by the frontend. It blocks until the user answers.
	OnQuestion func(ctx context.Context, question AskUserQuestion) (AskUserAnswer, error)
}

func (a *AskUser) Name() string  { return "ask_user" }
func (a *AskUser) Label() string { return "Ask User" }

func (a *AskUser) Description() string {
	return "Ask the user a question with optional structured choices. " +
		"Use when you need to clarify requirements, choose between approaches, or get user preferences. " +
		"The user will see the question with clickable options in the UI."
}

func (a *AskUser) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "The question to ask the user.",
			},
			"options": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"label":       map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
					},
					"required": []string{"label"},
				},
				"description": "Optional choices for the user to select from (2-5 options).",
			},
			"multi_select": map[string]any{
				"type":        "boolean",
				"description": "Allow selecting multiple options (default false).",
			},
		},
		"required": []string{"question"},
	}
}

func (a *AskUser) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	question, _ := params["question"].(string)
	question = strings.TrimSpace(question)
	if question == "" {
		return tool.ToolResult{}, fmt.Errorf("ask_user: question is required")
	}

	q := AskUserQuestion{Question: question}

	// Parse options.
	if optionsRaw, ok := params["options"]; ok {
		data, _ := json.Marshal(optionsRaw)
		_ = json.Unmarshal(data, &q.Options)
	}

	if ms, ok := params["multi_select"].(bool); ok {
		q.MultiSelect = ms
	}

	if a.OnQuestion == nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText,
					Text: "ask_user is not available in this environment. Ask the user directly in your response text instead."},
			},
		}, nil
	}

	answer, err := a.OnQuestion(ctx, q)
	if err != nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText,
					Text: fmt.Sprintf("User interaction failed: %v", err)},
			},
		}, nil
	}

	// Format the answer as readable text.
	var sb strings.Builder
	if len(answer.Selected) > 0 {
		sb.WriteString("User selected: ")
		sb.WriteString(strings.Join(answer.Selected, ", "))
	}
	if answer.Text != "" {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("User said: ")
		sb.WriteString(answer.Text)
	}
	if sb.Len() == 0 {
		sb.WriteString("User provided no answer.")
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: sb.String()},
		},
	}, nil
}
