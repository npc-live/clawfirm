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
	return `Execute a browser automation shortcut (YAML adapter) via Chrome DevTools Protocol.
Shortcuts are YAML files in ~/.clawfirm/shortcuts/ (e.g. douyin.yaml, xhs.yaml, bilibili.yaml).
Each shortcut defines commands like "search", "like", "comment", "follow", "post".
Requires Chrome running with --remote-debugging-port=9222.

Example: to search Douyin for "美食", use file="douyin.yaml", command="search", args=["美食"].`
}

func (b *BrowserShortcut) ConcurrencySafe() bool { return false }
func (b *BrowserShortcut) ShouldDefer() bool      { return false }

func (b *BrowserShortcut) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "YAML shortcut filename (e.g. \"douyin.yaml\", \"xhs.yaml\"). Located in ~/.clawfirm/shortcuts/.",
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

	// Resolve shortcut file path.
	home, _ := os.UserHomeDir()
	fp := filepath.Join(home, ".clawfirm", "shortcuts", file)
	if _, err := os.Stat(fp); err != nil {
		// List available shortcuts for helpful error.
		var available []string
		dir := filepath.Join(home, ".clawfirm", "shortcuts")
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
					available = append(available, e.Name())
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

	rows, err := browser.RunYAMLCommand(ctx, fp, command, args, port, healer, progressFn)
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
