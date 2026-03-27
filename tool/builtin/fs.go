package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// ---------------------------------------------------------------------------
// Grep — ripgrep-style file content search
// ---------------------------------------------------------------------------

// Grep searches file contents for a pattern using filepath.WalkDir + strings.Contains.
type Grep struct{}

func (g *Grep) Name() string  { return "grep" }
func (g *Grep) Label() string { return "Grep" }
func (g *Grep) Description() string {
	return "Search file contents for a pattern. Returns matching lines with file path and line number."
}
func (g *Grep) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The string or substring to search for.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory to search in. Defaults to current directory.",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "Optional glob pattern to filter files (e.g. '*.go', '*.ts').",
			},
			"case_insensitive": map[string]any{
				"type":        "boolean",
				"description": "Case-insensitive search (default false).",
			},
			"max_results": map[string]any{
				"type":        "number",
				"description": "Maximum number of matching lines to return (default 200).",
			},
		},
		"required": []string{"pattern"},
	}
}

func (g *Grep) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	pattern, _ := params["pattern"].(string)
	if pattern == "" {
		return tool.ToolResult{}, fmt.Errorf("grep: pattern is required")
	}
	searchPath, _ := params["path"].(string)
	if searchPath == "" {
		searchPath = "."
	}
	searchPath = expandHome(searchPath)

	globPat, _ := params["glob"].(string)
	caseInsensitive, _ := params["case_insensitive"].(bool)
	maxResults := 200
	if mr, ok := params["max_results"].(float64); ok && mr > 0 {
		maxResults = int(mr)
	}

	needle := pattern
	if caseInsensitive {
		needle = strings.ToLower(pattern)
	}

	var results []string
	totalMatches := 0

	err := filepath.WalkDir(searchPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		// Skip hidden dirs
		if strings.Contains(p, "/.") {
			return nil
		}
		// Filter by glob
		if globPat != "" {
			matched, _ := filepath.Match(globPat, filepath.Base(p))
			if !matched {
				return nil
			}
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			haystack := line
			if caseInsensitive {
				haystack = strings.ToLower(line)
			}
			if strings.Contains(haystack, needle) {
				totalMatches++
				if len(results) < maxResults {
					results = append(results, fmt.Sprintf("%s:%d: %s", p, i+1, line))
				}
			}
		}
		return nil
	})
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("grep: %w", err)
	}

	text := strings.Join(results, "\n")
	if totalMatches > maxResults {
		text += fmt.Sprintf("\n[Showing %d of %d matches. Use max_results to increase limit.]", maxResults, totalMatches)
	} else if totalMatches == 0 {
		text = fmt.Sprintf("No matches for %q in %s", pattern, searchPath)
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
	}, nil
}

// ---------------------------------------------------------------------------
// Find — find files by glob pattern
// ---------------------------------------------------------------------------

// Find finds files matching a glob pattern.
type Find struct{}

func (f *Find) Name() string  { return "find" }
func (f *Find) Label() string { return "Find" }
func (f *Find) Description() string {
	return "Find files by name glob pattern. Returns matching file paths."
}
func (f *Find) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern to match filenames (e.g. '*.go', 'main.*'). Use ** for recursive.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Directory to search in. Defaults to current directory.",
			},
			"max_results": map[string]any{
				"type":        "number",
				"description": "Maximum number of results (default 500).",
			},
		},
		"required": []string{"pattern"},
	}
}

func (f *Find) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	pattern, _ := params["pattern"].(string)
	if pattern == "" {
		return tool.ToolResult{}, fmt.Errorf("find: pattern is required")
	}
	searchPath, _ := params["path"].(string)
	if searchPath == "" {
		searchPath = "."
	}
	searchPath = expandHome(searchPath)

	maxResults := 500
	if mr, ok := params["max_results"].(float64); ok && mr > 0 {
		maxResults = int(mr)
	}

	var matches []string
	err := filepath.WalkDir(searchPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip .git and node_modules
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == ".DS_Store" {
				return filepath.SkipDir
			}
			return nil
		}
		matched, _ := filepath.Match(pattern, filepath.Base(p))
		if matched {
			matches = append(matches, p)
		}
		return nil
	})
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("find: %w", err)
	}

	text := ""
	if len(matches) == 0 {
		text = fmt.Sprintf("No files matching %q found in %s", pattern, searchPath)
	} else {
		shown := matches
		suffix := ""
		if len(shown) > maxResults {
			shown = shown[:maxResults]
			suffix = fmt.Sprintf("\n[Showing %d of %d matches.]", maxResults, len(matches))
		}
		text = strings.Join(shown, "\n") + suffix
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
	}, nil
}

// ---------------------------------------------------------------------------
// Ls — list directory contents
// ---------------------------------------------------------------------------

// Ls lists the contents of a directory.
type Ls struct{}

func (l *Ls) Name() string  { return "ls" }
func (l *Ls) Label() string { return "Ls" }
func (l *Ls) Description() string {
	return "List the contents of a directory."
}
func (l *Ls) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Directory to list. Defaults to current directory.",
			},
			"all": map[string]any{
				"type":        "boolean",
				"description": "Include hidden files (starting with .). Default false.",
			},
		},
	}
}

func (l *Ls) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	dir, _ := params["path"].(string)
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return tool.ToolResult{}, fmt.Errorf("ls: %w", err)
		}
	}
	dir = expandHome(dir)

	showAll, _ := params["all"].(bool)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("ls: %w", err)
	}

	var lines []string
	for _, e := range entries {
		name := e.Name()
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			lines = append(lines, name+"/")
		} else {
			info, _ := e.Info()
			if info != nil {
				lines = append(lines, fmt.Sprintf("%-40s %d", name, info.Size()))
			} else {
				lines = append(lines, name)
			}
		}
	}

	text := dir + ":\n" + strings.Join(lines, "\n")
	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
	}, nil
}
