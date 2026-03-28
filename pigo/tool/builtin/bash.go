package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

const (
	bashMaxLines = 5_000
	bashMaxBytes = 30 * 1024 // 30 KB
)

// Bash executes a shell command and returns combined stdout+stderr.
type Bash struct {
	VaultEnv func() map[string]string
}

func (b *Bash) Name() string  { return "bash" }
func (b *Bash) Label() string { return "Bash" }
func (b *Bash) Description() string {
	return "Execute a bash command and return the combined stdout and stderr. Output is truncated to 30 KB."
}
func (b *Bash) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The bash command to execute.",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds (default: 60).",
			},
		},
		"required": []string{"command"},
	}
}

func (b *Bash) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	command, _ := params["command"].(string)
	if command == "" {
		return tool.ToolResult{}, fmt.Errorf("bash: command is required")
	}

	timeoutSecs := 60.0
	if t, ok := params["timeout"].(float64); ok && t > 0 {
		timeoutSecs = t
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs*float64(time.Second)))
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
	cmd.Env = os.Environ()
	if b.VaultEnv != nil {
		for k, v := range b.VaultEnv() {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()

	output := buf.String()

	// Truncate output to hard cap.
	truncated := false
	lines := strings.Split(output, "\n")
	if len(lines) > bashMaxLines {
		lines = lines[len(lines)-bashMaxLines:]
		truncated = true
	}
	output = strings.Join(lines, "\n")
	if len(output) > bashMaxBytes {
		output = output[len(output)-bashMaxBytes:]
		truncated = true
	}

	suffix := ""
	if truncated {
		suffix = "\n[Output truncated to last 30 KB / 5000 lines.]"
	}

	// Append exit error if any.
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			suffix += fmt.Sprintf("\n[Command timed out after %.0f seconds.]", timeoutSecs)
		} else {
			suffix += fmt.Sprintf("\n[Exit error: %v]", err)
		}
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: output + suffix},
		},
	}, nil
}
