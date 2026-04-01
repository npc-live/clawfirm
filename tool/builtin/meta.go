package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
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

// skillCacheEntry holds a cached skill content with an expiry timestamp.
type skillCacheEntry struct {
	content string
	loadedAt time.Time
}

const skillCacheTTL = 60 * time.Second

// Skill invokes a skill SKILL.md and returns its contents for the LLM to follow.
// SkillPaths is a list of directories or SKILL.md files to search.
// Loaded skills are cached for 60 seconds to avoid repeated disk scans.
type Skill struct {
	SkillPaths []string

	mu    sync.RWMutex
	cache map[string]*skillCacheEntry // name → cached content
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

	content, err := s.loadSkill(name)
	if err != nil {
		return tool.ToolResult{}, err
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: content}},
	}, nil
}

// InvalidateCache clears the entire skill cache, forcing next access to reload from disk.
func (s *Skill) InvalidateCache() {
	s.mu.Lock()
	s.cache = nil
	s.mu.Unlock()
}

// loadSkill returns cached content if fresh, otherwise loads from disk and caches.
func (s *Skill) loadSkill(name string) (string, error) {
	now := time.Now()

	// Fast path: read lock to check cache.
	s.mu.RLock()
	if s.cache != nil {
		if entry, ok := s.cache[name]; ok && now.Sub(entry.loadedAt) < skillCacheTTL {
			s.mu.RUnlock()
			return entry.content, nil
		}
	}
	s.mu.RUnlock()

	// Cache miss or expired: load from disk.
	content, err := s.findSkill(name)
	if err != nil {
		return "", err
	}

	// Store in cache under write lock.
	s.mu.Lock()
	if s.cache == nil {
		s.cache = make(map[string]*skillCacheEntry)
	}
	s.cache[name] = &skillCacheEntry{content: content, loadedAt: now}
	s.mu.Unlock()

	return content, nil
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
