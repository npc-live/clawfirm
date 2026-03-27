package skill

import (
	"os"
	"strings"
)

const (
	// MaxSkillsPromptChars is the character budget for the skills XML block.
	MaxSkillsPromptChars = 30_000
	// MaxSkillsInPrompt is the maximum number of skills injected into the prompt.
	MaxSkillsInPrompt = 150
)

// FormatForPrompt returns an XML block listing all visible skills for injection
// into the agent's system prompt. Skills with DisableModelInvocation=true are
// excluded — they can only be used via explicit /skill: commands.
//
// Format follows the Agent Skills spec: https://agentskills.io/integrate-skills
func FormatForPrompt(skills []Skill, skillDir string) string {
	var visible []Skill
	for _, s := range skills {
		if !s.DisableModelInvocation {
			visible = append(visible, s)
		}
	}
	if skillDir == "" {
		skillDir = "~/.pi-go/skills/"
	}

	var b strings.Builder

	if len(visible) > 0 {
		b.WriteString("\n\nThe following skills provide specialized instructions for specific tasks.")
		b.WriteString("\nUse the read tool to load a skill's file when the task matches its description.")
		b.WriteString("\nWhen a skill file references a relative path, resolve it against the skill directory (parent of SKILL.md / dirname of the path) and use that absolute path in tool commands.")
		b.WriteString("\n\n<available_skills>")
		for _, s := range visible {
			b.WriteString("\n  <skill>")
			b.WriteString("\n    <name>" + xmlEscape(s.Name) + "</name>")
			b.WriteString("\n    <description>" + xmlEscape(s.Description) + "</description>")
			b.WriteString("\n    <location>" + xmlEscape(s.FilePath) + "</location>")
			b.WriteString("\n  </skill>")
		}
		b.WriteString("\n</available_skills>")
	}

	// Remote skill installation guidance — always injected.
	{
		b.WriteString("\n\n<remote_skill_install>")
		b.WriteString("\nWhen a user sends a URL ending in .md (e.g. SKILL.md or any-skill-name.md),")
		b.WriteString("\nit is likely a remote skill file. To install it:")
		b.WriteString("\n1. Derive the filename from the last path segment of the URL (e.g. arena-skills.md).")
		b.WriteString("\n2. Use the bash tool to download and save it directly in the skill directory (no subdirectory): curl -fsSL \"<url>\" -o \"" + skillDir + "<filename>\"")
		b.WriteString("\n3. Use the read tool to load the saved file immediately so the skill is active right now.")
		b.WriteString("\n4. Confirm to the user that the skill is now active in this session.")
		b.WriteString("\n</remote_skill_install>")
	}

	return b.String()
}

// FormatSkillsCompact returns a compact XML block with only name and location
// (no description). Skills with DisableModelInvocation=true are excluded.
func FormatSkillsCompact(skills []Skill) string {
	var visible []Skill
	for _, s := range skills {
		if !s.DisableModelInvocation {
			visible = append(visible, s)
		}
	}
	if len(visible) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<available_skills>")
	for _, s := range visible {
		b.WriteString("\n  <skill>")
		b.WriteString("\n    <name>" + xmlEscape(s.Name) + "</name>")
		b.WriteString("\n    <location>" + xmlEscape(s.FilePath) + "</location>")
		b.WriteString("\n  </skill>")
	}
	b.WriteString("\n</available_skills>")
	return b.String()
}

// CompactSkillPaths returns a copy of skills with home-dir prefixes replaced by ~/
// to reduce prompt character count.
func CompactSkillPaths(skills []Skill) []Skill {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return skills
	}
	out := make([]Skill, len(skills))
	copy(out, skills)
	for i, s := range out {
		if strings.HasPrefix(s.FilePath, home+"/") {
			out[i].FilePath = "~/" + s.FilePath[len(home)+1:]
		}
		if strings.HasPrefix(s.BaseDir, home+"/") {
			out[i].BaseDir = "~/" + s.BaseDir[len(home)+1:]
		}
	}
	return out
}

// ApplySkillsPromptLimits applies the three-level degradation strategy:
//  1. Truncate to MaxSkillsInPrompt skills.
//  2. Try full format (FormatForPrompt); return if ≤ MaxSkillsPromptChars.
//  3. Try compact format; return if ≤ MaxSkillsPromptChars.
//  4. Binary-search for the largest N where compact(skills[:N]) fits, prepend warning.
//
// Returns the prompt string, whether it was truncated (fewer skills than input),
// and whether the compact format was used.
func ApplySkillsPromptLimits(skills []Skill) (prompt string, truncated bool, compact bool) {
	// Step 1: cap at MaxSkillsInPrompt
	capped := skills
	if len(capped) > MaxSkillsInPrompt {
		capped = capped[:MaxSkillsInPrompt]
	}

	// Build full-format XML (without remote_skill_install preamble text)
	fullXML := formatSkillsXML(capped)
	if len(fullXML) <= MaxSkillsPromptChars {
		return fullXML, len(skills) > len(capped), false
	}

	// Step 3: compact
	compactXML := FormatSkillsCompact(capped)
	if len(compactXML) <= MaxSkillsPromptChars {
		return compactXML, len(skills) > len(capped), true
	}

	// Step 4: binary search for largest N where compact fits with 150-char headroom
	budget := MaxSkillsPromptChars - 150
	lo, hi := 1, len(capped)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if len(FormatSkillsCompact(capped[:mid])) <= budget {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	final := FormatSkillsCompact(capped[:lo])
	warning := "[Note: skill list truncated to fit prompt limits. Some skills may not be shown.]\n"
	return warning + final, true, true
}

// formatSkillsXML renders the raw <available_skills> XML block in full format
// (with descriptions). Used by ApplySkillsPromptLimits for size checks.
func formatSkillsXML(skills []Skill) string {
	var visible []Skill
	for _, s := range skills {
		if !s.DisableModelInvocation {
			visible = append(visible, s)
		}
	}
	if len(visible) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<available_skills>")
	for _, s := range visible {
		b.WriteString("\n  <skill>")
		b.WriteString("\n    <name>" + xmlEscape(s.Name) + "</name>")
		b.WriteString("\n    <description>" + xmlEscape(s.Description) + "</description>")
		b.WriteString("\n    <location>" + xmlEscape(s.FilePath) + "</location>")
		b.WriteString("\n  </skill>")
	}
	b.WriteString("\n</available_skills>")
	return b.String()
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
