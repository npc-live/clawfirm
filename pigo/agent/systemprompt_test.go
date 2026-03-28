package agent

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt_None(t *testing.T) {
	p := SystemPromptParams{PromptMode: PromptModeNone}
	out := BuildSystemPrompt(p)
	if out != "You are a personal assistant running inside OpenClaw." {
		t.Errorf("none mode should return only intro, got %q", out)
	}
}

func TestBuildSystemPrompt_Full_Sections(t *testing.T) {
	p := SystemPromptParams{
		WorkspaceDir: "/workspace",
		PromptMode:   PromptModeFull,
		RuntimeInfo:  "host=test | model=claude-sonnet",
	}
	out := BuildSystemPrompt(p)

	sections := []string{
		"You are a personal assistant running inside OpenClaw.",
		"## Tooling",
		"## Tool Call Style",
		"## Safety",
		"## OpenClaw CLI Quick Reference",
		"## Workspace",
		"Your working directory is: /workspace",
		"## Runtime",
		"host=test | model=claude-sonnet",
	}
	for _, s := range sections {
		if !strings.Contains(out, s) {
			t.Errorf("expected section %q in full output", s)
		}
	}
	// No skills section when SkillsPrompt is empty
	if strings.Contains(out, "## Skills (mandatory)") {
		t.Error("skills section should be absent when SkillsPrompt is empty")
	}
	// No project context when ContextFiles is empty
	if strings.Contains(out, "# Project Context") {
		t.Error("project context should be absent when ContextFiles is empty")
	}
}

func TestBuildSystemPrompt_WithSkills(t *testing.T) {
	p := SystemPromptParams{
		PromptMode:   PromptModeFull,
		SkillsPrompt: "<available_skills><skill><name>foo</name></skill></available_skills>",
	}
	out := BuildSystemPrompt(p)
	if !strings.Contains(out, "## Skills (mandatory)") {
		t.Error("expected skills section when SkillsPrompt is non-empty")
	}
	if !strings.Contains(out, "Before replying: scan <available_skills>") {
		t.Error("expected skills preamble in output")
	}
	if !strings.Contains(out, "foo") {
		t.Error("expected skills prompt content in output")
	}
}

func TestBuildSystemPrompt_WithContextFiles(t *testing.T) {
	p := SystemPromptParams{
		PromptMode: PromptModeFull,
		ContextFiles: []ContextFile{
			{Name: "AGENTS.md", Path: "/workspace/AGENTS.md", Content: "## My project"},
		},
	}
	out := BuildSystemPrompt(p)
	if !strings.Contains(out, "# Project Context") {
		t.Error("expected project context section with context files")
	}
	if !strings.Contains(out, "## AGENTS.md") {
		t.Error("expected AGENTS.md header in project context")
	}
	if !strings.Contains(out, "## My project") {
		t.Error("expected AGENTS.md content in output")
	}
}

func TestBuildSystemPrompt_WithSOULMD(t *testing.T) {
	p := SystemPromptParams{
		PromptMode: PromptModeFull,
		ContextFiles: []ContextFile{
			{Name: "SOUL.md", Content: "Be a pirate"},
		},
	}
	out := BuildSystemPrompt(p)
	if !strings.Contains(out, "embody its persona") {
		t.Error("expected SOUL.md persona note when SOUL.md is present")
	}
}

func TestBuildSystemPrompt_WithWorkspaceNotes(t *testing.T) {
	p := SystemPromptParams{
		PromptMode:     PromptModeFull,
		WorkspaceNotes: []string{"Reminder: commit your changes in this workspace after edits."},
	}
	out := BuildSystemPrompt(p)
	if !strings.Contains(out, "Reminder: commit your changes") {
		t.Error("expected workspace note in output")
	}
}

func TestBuildSystemPrompt_WithExtraPrompt(t *testing.T) {
	p := SystemPromptParams{
		PromptMode:  PromptModeFull,
		ExtraPrompt: "You are also a pirate.",
	}
	out := BuildSystemPrompt(p)
	if !strings.Contains(out, "You are also a pirate.") {
		t.Error("expected extra prompt in output")
	}
	// Extra prompt should appear after other sections.
	idx := strings.Index(out, "## Workspace")
	idxExtra := strings.Index(out, "You are also a pirate.")
	if idxExtra < idx {
		t.Error("extra prompt should appear after workspace section")
	}
}

func TestBuildSystemPrompt_DefaultMode(t *testing.T) {
	// Empty PromptMode should default to full.
	p := SystemPromptParams{WorkspaceDir: "/test"}
	out := BuildSystemPrompt(p)
	if !strings.Contains(out, "## Tooling") {
		t.Error("default mode should include Tooling section")
	}
}

func TestBuildSystemPrompt_Minimal(t *testing.T) {
	p := SystemPromptParams{
		PromptMode:   PromptModeMinimal,
		WorkspaceDir: "/ws",
		RuntimeInfo:  "model=test",
	}
	out := BuildSystemPrompt(p)
	// Minimal mode should include all sections per plan (same as full in current impl).
	if !strings.Contains(out, "## Tooling") {
		t.Error("minimal mode should include Tooling section")
	}
}
