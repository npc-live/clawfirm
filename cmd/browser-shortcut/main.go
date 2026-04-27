// Command browser-shortcut is a standalone binary for the claw-code plugin system.
// It reads JSON input from stdin, executes a YAML browser automation shortcut
// via Chrome DevTools Protocol, and writes JSON output to stdout.
//
// Input (stdin JSON):
//
//	{"file":"douyin.yaml","command":"search","args":["美食"]}
//
// Output (stdout JSON):
//
//	[{"title":"...","url":"..."}]  — on success
//	{"error":"..."}                — on failure
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-gateway/clawfirm/browser"
)

type input struct {
	File    string   `json:"file"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func main() {
	// Read input: prefer CLAWD_TOOL_INPUT env var, fall back to stdin.
	var raw []byte
	if env := os.Getenv("CLAWD_TOOL_INPUT"); env != "" {
		raw = []byte(env)
	} else {
		var err error
		raw, err = io.ReadAll(os.Stdin)
		if err != nil {
			fatal("failed to read stdin: %v", err)
		}
	}

	var in input
	if err := json.Unmarshal(raw, &in); err != nil {
		fatal("invalid JSON input: %v", err)
	}

	if in.File == "" || in.Command == "" {
		fatal("both 'file' and 'command' are required")
	}

	// Resolve shortcut file path — user dir first, bundled fallback.
	home, _ := os.UserHomeDir()
	userDir := filepath.Join(home, ".clawfirm", "shortcuts")
	bDir := filepath.Join(home, ".clawfirm", "bundled", "shortcuts")
	fp := filepath.Join(userDir, in.File)
	if _, err := os.Stat(fp); err != nil {
		fp = filepath.Join(bDir, in.File)
	}
	if _, err := os.Stat(fp); err != nil {
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
		fatal("shortcut file %q not found.\nAvailable shortcuts: %s", in.File, strings.Join(available, ", "))
	}

	// Load adapter to validate command.
	adapter, err := browser.LoadAdapterYAML(fp)
	if err != nil {
		fatal("error loading shortcut: %v", err)
	}
	if _, ok := adapter.Commands[in.Command]; !ok {
		cmds := make([]string, 0, len(adapter.Commands))
		for k := range adapter.Commands {
			cmds = append(cmds, k)
		}
		fatal("unknown command %q for %s.\nAvailable commands: %s", in.Command, in.File, strings.Join(cmds, ", "))
	}

	// Execute.
	targetType := "chrome"
	if strings.HasPrefix(adapter.Platform, "clawfirm") || adapter.Platform == "clawfirm_test" {
		targetType = "app"
	}
	rows, err := browser.RunYAMLCommand(context.Background(), fp, in.Command, in.Args, 9222, nil, nil, targetType)
	if err != nil {
		fatal("browser shortcut error: %v", err)
	}

	if len(rows) == 0 {
		fmt.Println(`"Command completed with no output."`)
		return
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rows); err != nil {
		fatal("failed to encode output: %v", err)
	}
}

func fatal(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	out, _ := json.Marshal(map[string]string{"error": msg})
	os.Stdout.Write(out)
	os.Stdout.Write([]byte("\n"))
	os.Exit(1)
}
