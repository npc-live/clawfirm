// Package skill implements skill discovery and loading per the Agent Skills spec
// (https://agentskills.io/specification). Skills are markdown files with YAML
// frontmatter that provide specialized instructions to the agent. Only their
// name/description are injected into the system prompt; the agent reads the
// full file on demand.
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	maxNameLen = 64
	maxDescLen = 1024
)

// Skill is a loaded skill ready to be injected into a system prompt.
type Skill struct {
	Name                   string
	Description            string
	FilePath               string // absolute path to SKILL.md (or .md file)
	BaseDir                string // parent directory of FilePath
	DisableModelInvocation bool   // if true, skip in system prompt (explicit /skill: only)
}

// Diagnostic is a non-fatal warning produced during skill loading.
type Diagnostic struct {
	Path    string
	Message string
}

// LoadResult holds the skills and any warnings from a load operation.
type LoadResult struct {
	Skills      []Skill
	Diagnostics []Diagnostic
}

// LoadOptions configures which directories are searched.
type LoadOptions struct {
	// SkillPaths lists explicit files or directories to load skills from.
	// Each entry may be an absolute path, a ~-relative path, or a path relative to Cwd.
	SkillPaths []string
	// Cwd is used to resolve relative SkillPaths (defaults to os.Getwd()).
	Cwd string
}

// Load discovers skills from all configured paths.
// Collision handling: first skill with a given name wins.
func Load(opts LoadOptions) LoadResult {
	cwd := opts.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	seen := make(map[string]bool)      // name → already loaded
	realPaths := make(map[string]bool) // resolved path → already loaded
	var skills []Skill
	var diags []Diagnostic

	add := func(r LoadResult) {
		diags = append(diags, r.Diagnostics...)
		for _, s := range r.Skills {
			real := s.FilePath
			if rp, err := filepath.EvalSymlinks(s.FilePath); err == nil {
				real = rp
			}
			if realPaths[real] {
				continue
			}
			if seen[s.Name] {
				diags = append(diags, Diagnostic{
					Path:    s.FilePath,
					Message: fmt.Sprintf("skill name %q collision, keeping first", s.Name),
				})
				continue
			}
			seen[s.Name] = true
			realPaths[real] = true
			skills = append(skills, s)
		}
	}

	for _, raw := range opts.SkillPaths {
		resolved := resolvePath(raw, cwd)
		info, err := os.Stat(resolved)
		if err != nil {
			diags = append(diags, Diagnostic{Path: resolved, Message: "skill path does not exist"})
			continue
		}
		if info.IsDir() {
			add(loadFromDir(resolved, true))
		} else if strings.HasSuffix(resolved, ".md") {
			add(loadFromFile(resolved))
		} else {
			diags = append(diags, Diagnostic{Path: resolved, Message: "skill path is not a markdown file"})
		}
	}

	return LoadResult{Skills: skills, Diagnostics: diags}
}

// LoadFromDir loads skills rooted at dir (public helper for single-dir loading).
func LoadFromDir(dir string) LoadResult {
	return loadFromDir(dir, true)
}

// loadFromDir recursively discovers skills under dir.
// If includeRootFiles is true, .md files directly in dir are also considered skills.
func loadFromDir(dir string, includeRootFiles bool) LoadResult {
	var skills []Skill
	var diags []Diagnostic

	entries, err := os.ReadDir(dir)
	if err != nil {
		return LoadResult{}
	}

	// Check for a skill entry file in this directory — if found, this IS the skill root.
	// Recognized entry filenames (in priority order): SKILL.md, skills.md
	for _, entryName := range []string{"SKILL.md", "skills.md"} {
		for _, e := range entries {
			if e.Name() != entryName {
				continue
			}
			fp := filepath.Join(dir, entryName)
			// Resolve symlink
			if e.Type()&os.ModeSymlink != 0 {
				if info, err := os.Stat(fp); err != nil || !info.Mode().IsRegular() {
					continue
				}
			}
			r := loadFromFile(fp)
			skills = append(skills, r.Skills...)
			diags = append(diags, r.Diagnostics...)
			// Do not recurse further — this directory is itself a skill.
			return LoadResult{Skills: skills, Diagnostics: diags}
		}
	}

	// No SKILL.md here: recurse into subdirectories and optionally load root .md files.
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") || e.Name() == "node_modules" {
			continue
		}
		fp := filepath.Join(dir, e.Name())

		isDir := e.IsDir()
		isFile := e.Type().IsRegular()
		if e.Type()&os.ModeSymlink != 0 {
			info, err := os.Stat(fp)
			if err != nil {
				continue
			}
			isDir = info.IsDir()
			isFile = info.Mode().IsRegular()
		}

		if isDir {
			r := loadFromDir(fp, false)
			skills = append(skills, r.Skills...)
			diags = append(diags, r.Diagnostics...)
			continue
		}
		if isFile && includeRootFiles && strings.HasSuffix(e.Name(), ".md") {
			r := loadFromFile(fp)
			skills = append(skills, r.Skills...)
			diags = append(diags, r.Diagnostics...)
		}
	}

	return LoadResult{Skills: skills, Diagnostics: diags}
}

// loadFromFile loads a single .md skill file.
func loadFromFile(path string) LoadResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return LoadResult{Diagnostics: []Diagnostic{{Path: path, Message: err.Error()}}}
	}

	fm := parseFrontmatter(string(data))
	baseDir := filepath.Dir(path)
	parentDir := filepath.Base(baseDir)

	description, _ := fm["description"].(string)
	name, _ := fm["name"].(string)
	if name == "" {
		name = parentDir
	}

	var diags []Diagnostic

	// Validate description — required
	if strings.TrimSpace(description) == "" {
		diags = append(diags, Diagnostic{Path: path, Message: "description is required"})
		return LoadResult{Diagnostics: diags}
	}

	// Warn if name doesn't match parent dir
	if name != parentDir {
		diags = append(diags, Diagnostic{Path: path, Message: fmt.Sprintf("name %q does not match parent directory %q", name, parentDir)})
	}
	if len(name) > maxNameLen {
		diags = append(diags, Diagnostic{Path: path, Message: fmt.Sprintf("name exceeds %d characters", maxNameLen)})
	}
	if len(description) > maxDescLen {
		diags = append(diags, Diagnostic{Path: path, Message: fmt.Sprintf("description exceeds %d characters", maxDescLen)})
	}

	disable := false
	if v, ok := fm["disable-model-invocation"]; ok {
		if b, ok := v.(bool); ok {
			disable = b
		}
	}

	return LoadResult{
		Skills: []Skill{{
			Name:                   name,
			Description:            description,
			FilePath:               path,
			BaseDir:                baseDir,
			DisableModelInvocation: disable,
		}},
		Diagnostics: diags,
	}
}

// parseFrontmatter extracts YAML frontmatter from a markdown string using
// the standard gopkg.in/yaml.v3 parser, supporting all YAML syntax including
// multi-line folded (>-) and literal (|) block scalars.
// Format: document starts with "---\n...\n---".
func parseFrontmatter(content string) map[string]any {
	result := make(map[string]any)
	if !strings.HasPrefix(content, "---") {
		return result
	}
	rest := content[3:]
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return result
	}
	block := rest[:end]
	if err := yaml.Unmarshal([]byte(block), &result); err != nil {
		return make(map[string]any)
	}
	return result
}

// ResolvePath expands ~ and resolves relative paths against cwd (public helper).
func ResolvePath(p, cwd string) string { return resolvePath(p, cwd) }

// resolvePath expands ~ and resolves relative paths against cwd.
func resolvePath(p, cwd string) string {
	p = strings.TrimSpace(p)
	if p == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(cwd, p)
}
