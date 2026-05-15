package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-gateway/clawfirm/browser"
	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

// BrowserShortcut executes a YAML browser automation shortcut via CDP.
// Shortcuts are located in ~/.clawfirm/shortcuts/ (e.g. douyin.yaml, xhs.yaml).
type BrowserShortcut struct {
	CDPPort   int // default 9222
	Provider  provider.LLMProvider
	Model     string
	IssueRepo string
}

func (b *BrowserShortcut) Name() string  { return "browser_shortcut" }
func (b *BrowserShortcut) Label() string { return "Browser Shortcut" }
func (b *BrowserShortcut) Description() string {
	base := `Execute a browser automation shortcut (YAML adapter) via Chrome DevTools Protocol.
Shortcuts are YAML files in ~/.clawfirm/shortcuts/. Requires Chrome with --remote-debugging-port=9222.`

	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, ".clawfirm", "shortcuts"),
		filepath.Join(home, ".clawfirm", "bundled", "shortcuts"),
	}
	seen := make(map[string]bool)
	var lines []string
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") || seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			adapter, err := browser.LoadAdapterYAML(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			cmds := make([]string, 0, len(adapter.Commands))
			for k, cmd := range adapter.Commands {
				if len(cmd.Args) > 0 {
					cmds = append(cmds, fmt.Sprintf("%s(%s)", k, strings.Join(cmd.Args, ",")))
				} else {
					cmds = append(cmds, k)
				}
			}
			lines = append(lines, fmt.Sprintf("- %s (%s): %s", e.Name(), adapter.Platform, strings.Join(cmds, ", ")))
		}
	}
	if len(lines) > 0 {
		base += "\n\nAvailable shortcuts:\n" + strings.Join(lines, "\n")
	}
	return base
}

func (b *BrowserShortcut) ConcurrencySafe() bool { return false }
func (b *BrowserShortcut) ShouldDefer() bool      { return false }

func (b *BrowserShortcut) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "YAML shortcut filename. See the tool description for available shortcuts.",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Command to execute (e.g. \"search\", \"like\", \"comment\", \"follow\", \"post\").",
			},
			"args": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Positional arguments for the command (e.g. [\"美食\"] for search keyword).",
			},
		},
		"required": []string{"file", "command"},
	}
}

func (b *BrowserShortcut) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	file, _ := params["file"].(string)
	command, _ := params["command"].(string)
	if file == "" || command == "" {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Text: "Error: both 'file' and 'command' are required."}},
		}, nil
	}

	// Parse args.
	var args []string
	if rawArgs, ok := params["args"]; ok {
		switch v := rawArgs.(type) {
		case []any:
			for _, a := range v {
				args = append(args, fmt.Sprintf("%v", a))
			}
		case []string:
			args = v
		}
	}

	// Resolve shortcut file path — user dir first, bundled fallback.
	home, _ := os.UserHomeDir()
	userDir := filepath.Join(home, ".clawfirm", "shortcuts")
	bDir := filepath.Join(home, ".clawfirm", "bundled", "shortcuts")
	fp := filepath.Join(userDir, file)
	if _, err := os.Stat(fp); err != nil {
		fp = filepath.Join(bDir, file)
	}
	if _, err := os.Stat(fp); err != nil {
		// List available shortcuts from both dirs for helpful error.
		seen := make(map[string]bool)
		var available []string
		for _, dir := range []string{userDir, bDir} {
			if entries, err := os.ReadDir(dir); err == nil {
				for _, e := range entries {
					if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") && !seen[e.Name()] {
						seen[e.Name()] = true
						available = append(available, e.Name())
					}
				}
			}
		}
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{
				Text: fmt.Sprintf("Error: shortcut file %q not found.\nAvailable shortcuts: %s", file, strings.Join(available, ", ")),
			}},
		}, nil
	}

	// List available commands if the requested one fails.
	adapter, err := browser.LoadAdapterYAML(fp)
	if err != nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Text: fmt.Sprintf("Error loading shortcut: %v", err)}},
		}, nil
	}

	// Verify command exists.
	if _, ok := adapter.Commands[command]; !ok {
		cmds := make([]string, 0, len(adapter.Commands))
		for k := range adapter.Commands {
			cmds = append(cmds, k)
		}
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{
				Text: fmt.Sprintf("Error: unknown command %q for %s.\nAvailable commands: %s", command, file, strings.Join(cmds, ", ")),
			}},
		}, nil
	}

	port := b.CDPPort
	if port == 0 {
		port = 9222
	}

	var healer *browser.HealerConfig
	if b.Provider != nil {
		healer = &browser.HealerConfig{
			Provider:  b.Provider,
			Model:     b.Model,
			IssueRepo: b.IssueRepo,
		}
	}

	// Wire onUpdate as progress callback so the frontend sees step-by-step progress.
	var progressFn browser.ProgressFunc
	if onUpdate != nil {
		progressFn = func(text string) {
			onUpdate(tool.ToolUpdate{
				Content: []types.ContentBlock{&types.TextContent{
					Type: types.ContentTypeText,
					Text: text,
				}},
			})
		}
	}

	// Determine target: App WKWebView (HTTP :9310) or external Chrome (CDP :port).
	targetType := "chrome"
	if strings.HasPrefix(adapter.Platform, "clawfirm") || adapter.Platform == "clawfirm_test" {
		targetType = "app"
	}

	rows, err := browser.RunYAMLCommand(ctx, fp, command, args, port, healer, progressFn, targetType)
	if err != nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Text: fmt.Sprintf("Browser shortcut error: %v", err)}},
		}, nil
	}

	if len(rows) == 0 {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Text: "Command completed with no output."}},
		}, nil
	}

	data, _ := json.MarshalIndent(rows, "", "  ")
	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Text: string(data)}},
	}, nil
}
