package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// ---------------------------------------------------------------------------
// SessionsList — list active agent sessions
// ---------------------------------------------------------------------------

// SessionsList lists running agent sessions. The registry is populated
// externally (e.g. by the gateway/app layer).
type SessionsList struct {
	// GetSessions returns a list of active session descriptors.
	// If nil, returns an empty list.
	GetSessions func() []string
}

func (s *SessionsList) Name() string  { return "sessions_list" }
func (s *SessionsList) Label() string { return "Sessions" }
func (s *SessionsList) Description() string {
	return "List active agent sessions."
}
func (s *SessionsList) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (s *SessionsList) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	var sessions []string
	if s.GetSessions != nil {
		sessions = s.GetSessions()
	}
	text := ""
	if len(sessions) == 0 {
		text = "No active sessions."
	} else {
		text = "Active sessions:\n" + strings.Join(sessions, "\n")
	}
	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
	}, nil
}

// ---------------------------------------------------------------------------
// Skill — invoke a skill by name
// ---------------------------------------------------------------------------

// Skill invokes a skill SKILL.md and returns its contents for the LLM to follow.
// SkillPaths is a list of directories or SKILL.md files to search.
type Skill struct {
	SkillPaths []string
}

func (s *Skill) Name() string  { return "skill" }
func (s *Skill) Label() string { return "Skill" }
func (s *Skill) Description() string {
	return "Invoke a skill by name. Reads the skill's SKILL.md and returns its contents."
}
func (s *Skill) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Skill name to invoke (matches the skill directory or SKILL.md filename prefix).",
			},
		},
		"required": []string{"name"},
	}
}

func (s *Skill) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	name, _ := params["name"].(string)
	if name == "" {
		return tool.ToolResult{}, fmt.Errorf("skill: name is required")
	}

	content, err := s.findSkill(name)
	if err != nil {
		return tool.ToolResult{}, err
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: content}},
	}, nil
}

func (s *Skill) findSkill(name string) (string, error) {
	for _, base := range s.SkillPaths {
		base = expandHome(base)

		// Check if base itself is a SKILL.md matching the name
		if strings.EqualFold(filepath.Base(base), name+".md") || strings.EqualFold(filepath.Base(base), "SKILL.md") {
			data, err := os.ReadFile(base)
			if err == nil {
				return string(data), nil
			}
		}

		// Check base/name/SKILL.md
		candidate := filepath.Join(base, name, "SKILL.md")
		data, err := os.ReadFile(candidate)
		if err == nil {
			return string(data), nil
		}

		// Check base/name.md
		candidate = filepath.Join(base, name+".md")
		data, err = os.ReadFile(candidate)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("skill %q not found in skill paths", name)
}
