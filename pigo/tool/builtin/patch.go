package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// ApplyPatch applies a unified diff / multi-file patch to the filesystem.
// Format: each file block starts with "*** path/to/file" then hunks of
// "<<<<<<< ORIGINAL ... ======= ... >>>>>>> UPDATED".
type ApplyPatch struct{}

func (a *ApplyPatch) Name() string  { return "apply_patch" }
func (a *ApplyPatch) Label() string { return "Apply Patch" }
func (a *ApplyPatch) Description() string {
	return `Apply a multi-file patch. Each file block uses the format:
*** path/to/file
<<<<<<< ORIGINAL
old content
=======
new content
>>>>>>> UPDATED
Multiple file blocks can appear in a single patch.`
}
func (a *ApplyPatch) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"patch": map[string]any{
				"type": "string",
				"description": `Patch content. Each file block: "*** path\n<<<<<<< ORIGINAL\n...\n=======\n...\n>>>>>>> UPDATED"`,
			},
		},
		"required": []string{"patch"},
	}
}

func (a *ApplyPatch) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	patch, _ := params["patch"].(string)
	if patch == "" {
		return tool.ToolResult{}, fmt.Errorf("apply_patch: patch is required")
	}

	applied, err := applyPatch(patch)
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("apply_patch: %w", err)
	}

	text := fmt.Sprintf("Applied patch to %d file(s):\n%s", len(applied), strings.Join(applied, "\n"))
	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
	}, nil
}

// applyPatch parses and applies the patch, returning a list of modified files.
func applyPatch(patch string) ([]string, error) {
	const (
		markerFile     = "*** "
		markerOriginal = "<<<<<<< ORIGINAL"
		markerSep      = "======="
		markerUpdated  = ">>>>>>> UPDATED"
	)

	lines := strings.Split(patch, "\n")
	var modified []string
	var currentFile string
	var origLines, newLines []string
	state := "scan" // scan | original | sep | updated

	flush := func() error {
		if currentFile == "" {
			return nil
		}
		path := expandHome(currentFile)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", path, err)
		}
		content := string(data)
		old := strings.Join(origLines, "\n")
		new_ := strings.Join(newLines, "\n")

		count := strings.Count(content, old)
		if count == 0 {
			return fmt.Errorf("original text not found in %s", path)
		}
		if count > 1 {
			return fmt.Errorf("original text found %d times in %s; provide more context", count, path)
		}
		updated := strings.Replace(content, old, new_, 1)
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		modified = append(modified, path)
		origLines = nil
		newLines = nil
		return nil
	}

	for _, line := range lines {
		switch state {
		case "scan":
			if strings.HasPrefix(line, markerFile) {
				if err := flush(); err != nil {
					return nil, err
				}
				currentFile = strings.TrimSpace(strings.TrimPrefix(line, markerFile))
				state = "scan"
			} else if line == markerOriginal {
				state = "original"
			}
		case "original":
			if line == markerSep {
				state = "updated"
			} else {
				origLines = append(origLines, line)
			}
		case "updated":
			if line == markerUpdated {
				if err := flush(); err != nil {
					return nil, err
				}
				state = "scan"
			} else {
				newLines = append(newLines, line)
			}
		}
	}

	return modified, nil
}
