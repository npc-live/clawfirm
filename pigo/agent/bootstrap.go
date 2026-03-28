package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

// ContextFile holds the content of a single bootstrap context file.
type ContextFile struct {
	Name    string // e.g. "AGENTS.md"
	Path    string // absolute path
	Content string // possibly truncated
}

// BootstrapResult holds bootstrap context loaded from the workspace.
type BootstrapResult struct {
	ContextFiles   []ContextFile
	WorkspaceNotes []string
}

const (
	bootstrapFileMaxChars   = 20_000
	bootstrapTotalMaxChars  = 150_000
	bootstrapHeadFraction   = 0.70
	bootstrapTailFraction   = 0.20
)

// bootstrapFilenames lists the context files searched in order.
var bootstrapFilenames = []string{"AGENTS.md", "CLAUDE.md", "SOUL.md"}

// LoadBootstrapContext scans workspaceDir for known context files and loads them.
// If workspaceDir is empty, the current working directory is used.
func LoadBootstrapContext(workspaceDir string) BootstrapResult {
	if workspaceDir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			workspaceDir = cwd
		}
	}

	var result BootstrapResult
	totalChars := 0
	hasProjectFile := false

	for _, name := range bootstrapFilenames {
		if totalChars >= bootstrapTotalMaxChars {
			break
		}

		path := filepath.Join(workspaceDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue // file not present — skip silently
		}

		content := string(data)
		truncated := false

		if len(content) > bootstrapFileMaxChars {
			headLen := int(float64(bootstrapFileMaxChars) * bootstrapHeadFraction)
			tailLen := int(float64(bootstrapFileMaxChars) * bootstrapTailFraction)
			head := content[:headLen]
			tail := content[len(content)-tailLen:]
			content = fmt.Sprintf("%s\n...[truncated: file exceeded %d chars]...\n%s",
				head, bootstrapFileMaxChars, tail)
			truncated = true
		}

		// Respect total budget.
		remaining := bootstrapTotalMaxChars - totalChars
		if len(content) > remaining {
			content = content[:remaining] +
				fmt.Sprintf("\n...[truncated: total context budget of %d chars exceeded]...", bootstrapTotalMaxChars)
			truncated = true
		}

		_ = truncated // available for future logging

		totalChars += len(content)

		if name == "AGENTS.md" || name == "CLAUDE.md" {
			hasProjectFile = true
		}

		result.ContextFiles = append(result.ContextFiles, ContextFile{
			Name:    name,
			Path:    path,
			Content: content,
		})
	}

	if hasProjectFile {
		result.WorkspaceNotes = append(result.WorkspaceNotes,
			"Reminder: commit your changes in this workspace after edits.")
	}

	return result
}
