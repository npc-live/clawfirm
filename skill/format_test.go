package skill

import (
	"strings"
	"testing"
)

func TestFormatSkillsCompact(t *testing.T) {
	skills := []Skill{
		{Name: "foo", Description: "does foo", FilePath: "/skills/foo/SKILL.md"},
		{Name: "bar", Description: "does bar", FilePath: "/skills/bar/SKILL.md", DisableModelInvocation: true},
	}
	out := FormatSkillsCompact(skills)
	if !strings.Contains(out, "<name>foo</name>") {
		t.Error("expected foo in compact output")
	}
	if strings.Contains(out, "<description>") {
		t.Error("compact should not include description")
	}
	if strings.Contains(out, "bar") {
		t.Error("disabled skill should be excluded from compact output")
	}
	if !strings.Contains(out, "/skills/foo/SKILL.md") {
		t.Error("expected location in compact output")
	}
}

func TestFormatSkillsCompactEmpty(t *testing.T) {
	if out := FormatSkillsCompact(nil); out != "" {
		t.Errorf("expected empty string for no skills, got %q", out)
	}
	disabled := []Skill{{Name: "x", DisableModelInvocation: true}}
	if out := FormatSkillsCompact(disabled); out != "" {
		t.Errorf("expected empty string for all-disabled skills, got %q", out)
	}
}

func TestApplySkillsPromptLimits_Full(t *testing.T) {
	// Small set → should use full format without truncation
	skills := []Skill{
		{Name: "alpha", Description: "alpha task", FilePath: "/s/alpha/SKILL.md"},
		{Name: "beta", Description: "beta task", FilePath: "/s/beta/SKILL.md"},
	}
	prompt, truncated, compact := ApplySkillsPromptLimits(skills)
	if truncated {
		t.Error("expected truncated=false for small skill set")
	}
	if compact {
		t.Error("expected compact=false for small skill set")
	}
	if !strings.Contains(prompt, "<description>") {
		t.Error("full format should include description")
	}
}

func TestApplySkillsPromptLimits_Compact(t *testing.T) {
	// Build skills whose full format exceeds limit but compact doesn't.
	// Each skill full entry ≈ name(9) + desc(500) + path(40) + tags(~100) = ~650 chars.
	// 50 skills * 650 ≈ 32500 → over 30k.
	// Compact entry ≈ name(9) + path(40) + tags(~60) = ~110 chars → 50*110 ≈ 5500 → under 30k.
	var skills []Skill
	for i := 0; i < 50; i++ {
		skills = append(skills, Skill{
			Name:        strings.Repeat("s", 8) + string(rune('a'+i%26)),
			Description: strings.Repeat("long description padding text ", 17), // ~510 chars
			FilePath:    "/home/user/skills/skill" + string(rune('a'+i%26)) + "/SKILL.md",
		})
	}
	prompt, truncated, compact := ApplySkillsPromptLimits(skills)
	if truncated {
		t.Error("expected truncated=false when compact fits")
	}
	if !compact {
		t.Error("expected compact=true when full format exceeds limit")
	}
	if strings.Contains(prompt, "<description>") {
		t.Error("compact format should not contain description")
	}
	_ = prompt
}

func TestApplySkillsPromptLimits_Truncated(t *testing.T) {
	// Build many skills whose compact format won't fit in 30k.
	// Each compact entry ≈ name(80) + path(80) + tags(60) = ~220 chars.
	// 200 skills * 220 ≈ 44000 → over 30k.
	var skills []Skill
	for i := 0; i < 200; i++ {
		skills = append(skills, Skill{
			Name:     strings.Repeat("x", 64),
			FilePath: strings.Repeat("/path/to/some/very/long/directory/name", 2) + "/SKILL.md",
		})
	}
	prompt, truncated, compact := ApplySkillsPromptLimits(skills)
	if !truncated {
		t.Error("expected truncated=true for very large skill set")
	}
	if !compact {
		t.Error("expected compact=true for truncated output")
	}
	if !strings.Contains(prompt, "truncated") {
		t.Error("truncated output should contain truncation warning")
	}
	if len(prompt) > MaxSkillsPromptChars {
		t.Errorf("prompt length %d exceeds limit %d", len(prompt), MaxSkillsPromptChars)
	}
}

func TestCompactSkillPaths(t *testing.T) {
	home := "/Users/testuser"
	skills := []Skill{
		{Name: "x", FilePath: home + "/skills/x/SKILL.md", BaseDir: home + "/skills/x"},
		{Name: "y", FilePath: "/other/y/SKILL.md", BaseDir: "/other/y"},
	}
	// We can't directly override os.UserHomeDir, so just verify the function
	// doesn't panic and returns the same count.
	out := CompactSkillPaths(skills)
	if len(out) != len(skills) {
		t.Errorf("expected %d skills, got %d", len(skills), len(out))
	}
}
