package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// ---------------------------------------------------------------------------
// Exec — run a shell command (alias for bash with explicit env support)
// ---------------------------------------------------------------------------

// Exec runs a shell command. Supports working directory and env overrides.
type Exec struct {
	// VaultEnv, if set, is called to obtain extra env vars from the vault.
	VaultEnv func() map[string]string
}

func (e *Exec) Name() string  { return "exec" }
func (e *Exec) Label() string { return "Exec" }
func (e *Exec) Description() string {
	return "Run a shell command. Supports working directory, environment variables, and timeout."
}
func (e *Exec) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to run.",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory for the command.",
			},
			"env": map[string]any{
				"type":        "object",
				"description": "Additional environment variables as key-value pairs.",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds (default 60).",
			},
		},
		"required": []string{"command"},
	}
}

func (e *Exec) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	command, _ := params["command"].(string)
	if command == "" {
		return tool.ToolResult{}, fmt.Errorf("exec: command is required")
	}
	cwd, _ := params["cwd"].(string)
	if cwd != "" {
		cwd = expandHome(cwd)
	}

	timeoutSecs := 60.0
	if t, ok := params["timeout"].(float64); ok && t > 0 {
		timeoutSecs = t
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs*float64(time.Second)))
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
	if cwd != "" {
		cmd.Dir = cwd
	}

	// Inherit current env, overlay vault secrets, then overlay caller extras.
	cmd.Env = os.Environ()
	if e.VaultEnv != nil {
		for k, v := range e.VaultEnv() {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	if envMap, ok := params["env"].(map[string]any); ok {
		for k, v := range envMap {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", k, v))
		}
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()

	output := buf.String()
	// Truncate
	if len(output) > bashMaxBytes {
		output = output[len(output)-bashMaxBytes:]
	}

	suffix := ""
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			suffix = fmt.Sprintf("\n[Timed out after %.0fs]", timeoutSecs)
		} else {
			suffix = fmt.Sprintf("\n[Exit: %v]", err)
		}
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: output + suffix}},
	}, nil
}

// ---------------------------------------------------------------------------
// Process — manage background processes
// ---------------------------------------------------------------------------

// processRegistry is a global store of background processes keyed by name.
var processRegistry = &procRegistry{procs: make(map[string]*bgProcess)}

type bgProcess struct {
	cmd    *exec.Cmd
	buf    *syncBuf
	done   chan struct{}
	err    error
	cancel context.CancelFunc
}

type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

type procRegistry struct {
	mu    sync.Mutex
	procs map[string]*bgProcess
}

func (r *procRegistry) start(name, command, cwd string, envMap map[string]any, vaultEnv map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.procs[name]; exists {
		return fmt.Errorf("process %q already running; stop it first", name)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
	if cwd != "" {
		cmd.Dir = expandHome(cwd)
	}
	cmd.Env = os.Environ()
	for k, v := range vaultEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", k, v))
	}

	buf := &syncBuf{}
	cmd.Stdout = buf
	cmd.Stderr = buf

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("process start: %w", err)
	}

	done := make(chan struct{})
	bp := &bgProcess{cmd: cmd, buf: buf, done: done, cancel: cancel}
	r.procs[name] = bp

	go func() {
		bp.err = cmd.Wait()
		close(done)
		r.mu.Lock()
		delete(r.procs, name)
		r.mu.Unlock()
	}()

	return nil
}

func (r *procRegistry) stop(name string) error {
	r.mu.Lock()
	bp, ok := r.procs[name]
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("no process named %q", name)
	}
	bp.cancel()
	select {
	case <-bp.done:
	case <-time.After(5 * time.Second):
		_ = bp.cmd.Process.Kill()
	}
	return nil
}

func (r *procRegistry) poll(name string, timeoutMs float64) (string, bool) {
	r.mu.Lock()
	bp, ok := r.procs[name]
	r.mu.Unlock()
	if !ok {
		return "(process not running)", false
	}
	if timeoutMs > 0 {
		select {
		case <-bp.done:
		case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		}
	}
	output := bp.buf.String()
	if len(output) > bashMaxBytes {
		output = output[len(output)-bashMaxBytes:]
	}
	select {
	case <-bp.done:
		return output, false
	default:
		return output, true
	}
}

func (r *procRegistry) list() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, 0, len(r.procs))
	for n := range r.procs {
		names = append(names, n)
	}
	return names
}

// Process manages background processes.
type Process struct {
	VaultEnv func() map[string]string
}

func (p *Process) Name() string  { return "process" }
func (p *Process) Label() string { return "Process" }
func (p *Process) Description() string {
	return "Manage background processes: start, stop, poll output, or list running processes."
}
func (p *Process) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"start", "stop", "poll", "list"},
				"description": "Action to perform: start a new process, stop it, poll its output, or list all.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Name to identify the process (required for start/stop/poll).",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to run (required for start).",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory (for start).",
			},
			"env": map[string]any{
				"type":        "object",
				"description": "Environment variables (for start).",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "For poll: max wait in milliseconds before returning current output.",
			},
		},
		"required": []string{"action"},
	}
}

func (p *Process) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	action, _ := params["action"].(string)
	name, _ := params["name"].(string)

	text := ""
	switch action {
	case "start":
		command, _ := params["command"].(string)
		if command == "" {
			return tool.ToolResult{}, fmt.Errorf("process start: command is required")
		}
		if name == "" {
			return tool.ToolResult{}, fmt.Errorf("process start: name is required")
		}
		cwd, _ := params["cwd"].(string)
		envMap, _ := params["env"].(map[string]any)
		if envMap == nil {
			envMap = map[string]any{}
		}
		var vaultEnv map[string]string
		if p.VaultEnv != nil {
			vaultEnv = p.VaultEnv()
		}
		if err := processRegistry.start(name, command, cwd, envMap, vaultEnv); err != nil {
			return tool.ToolResult{}, err
		}
		text = fmt.Sprintf("Process %q started.", name)

	case "stop":
		if name == "" {
			return tool.ToolResult{}, fmt.Errorf("process stop: name is required")
		}
		if err := processRegistry.stop(name); err != nil {
			return tool.ToolResult{}, err
		}
		text = fmt.Sprintf("Process %q stopped.", name)

	case "poll":
		if name == "" {
			return tool.ToolResult{}, fmt.Errorf("process poll: name is required")
		}
		timeoutMs, _ := params["timeout"].(float64)
		output, running := processRegistry.poll(name, timeoutMs)
		status := "finished"
		if running {
			status = "running"
		}
		text = fmt.Sprintf("[%s | %s]\n%s", name, status, output)

	case "list":
		names := processRegistry.list()
		if len(names) == 0 {
			text = "No background processes running."
		} else {
			text = "Running processes:\n" + strings.Join(names, "\n")
		}

	default:
		return tool.ToolResult{}, fmt.Errorf("process: unknown action %q (use start/stop/poll/list)", action)
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
	}, nil
}
