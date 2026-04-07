package clawproc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"syscall"
)

// setSysProcAttr sets platform-specific process attributes so the subprocess
// runs in its own process group. This allows Close() to kill the entire group
// (claw + any whip sub-subprocesses) with a single signal.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// Config configures a claw subprocess.
type Config struct {
	BinaryPath     string   // path to the claw binary
	Model          string   // model ID (e.g. "claude-sonnet-4-6")
	PermissionMode string   // e.g. "danger-full-access"
	WorkingDir     string   // working directory for the subprocess
	Env            []string // extra environment variables (KEY=VALUE)
}

// Process manages a long-lived claw subprocess in serve mode.
type Process struct {
	cfg    Config
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr io.ReadCloser
	mu     sync.Mutex // protects stdin writes
	alive  bool       // true after successful Start
}

// NewProcess creates a Process (does not start it).
func NewProcess(cfg Config) *Process {
	return &Process{cfg: cfg}
}

// Start spawns the claw subprocess and waits for the "ready" event.
func (p *Process) Start(ctx context.Context) (*Event, error) {
	args := []string{"serve"}
	if p.cfg.Model != "" {
		args = append(args, "--model", p.cfg.Model)
	}
	if p.cfg.PermissionMode != "" {
		args = append(args, "--permission-mode", p.cfg.PermissionMode)
	}

	p.cmd = exec.CommandContext(ctx, p.cfg.BinaryPath, args...)
	if p.cfg.WorkingDir != "" {
		p.cmd.Dir = p.cfg.WorkingDir
	}
	if len(p.cfg.Env) > 0 {
		p.cmd.Env = append(p.cmd.Environ(), p.cfg.Env...)
	}
	// Run in its own process group so Close() can kill the whole tree.
	setSysProcAttr(p.cmd)

	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("clawproc: stdin pipe: %w", err)
	}

	stdoutPipe, err := p.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("clawproc: stdout pipe: %w", err)
	}
	p.stdout = bufio.NewScanner(stdoutPipe)
	// Allow large lines (up to 4MB) for tool results.
	p.stdout.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("clawproc: stderr pipe: %w", err)
	}

	if err := p.cmd.Start(); err != nil {
		return nil, fmt.Errorf("clawproc: start %q: %w", p.cfg.BinaryPath, err)
	}

	// Wait for the "ready" event.
	ev, err := p.ReadEvent()
	if err != nil {
		// Drain stderr for a useful error message.
		stderrMsg := p.drainStderr()
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
		if stderrMsg != "" {
			return nil, fmt.Errorf("clawproc: %s (stderr: %s)", err, stderrMsg)
		}
		return nil, fmt.Errorf("clawproc: waiting for ready: %w", err)
	}
	if ev.Type != "ready" {
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
		return nil, fmt.Errorf("clawproc: expected ready event, got %q", ev.Type)
	}
	p.alive = true
	return ev, nil
}

// Alive returns true if the subprocess started successfully.
func (p *Process) Alive() bool { return p.alive }

// SendPrompt writes a prompt command to the subprocess stdin.
func (p *Process) SendPrompt(text string) error {
	if !p.alive {
		return fmt.Errorf("clawproc: process not running")
	}
	return p.writeCommand(stdinCommand{Type: "prompt", Text: text})
}

// SendAbort sends SIGTERM to the subprocess to cancel the current turn.
// The process will exit; ClawAgent detects EOF on stdout and emits EventAgentEnd.
// After this call p.alive is false — the Process must be restarted before reuse.
func (p *Process) SendAbort() error {
	if !p.alive || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	p.alive = false
	return p.cmd.Process.Signal(syscall.SIGTERM)
}

// ReadEvent reads and parses one NDJSON line from stdout.
// Returns io.EOF when the subprocess has closed stdout.
func (p *Process) ReadEvent() (*Event, error) {
	if !p.stdout.Scan() {
		if err := p.stdout.Err(); err != nil {
			return nil, fmt.Errorf("clawproc: read: %w", err)
		}
		return nil, io.EOF
	}
	raw := p.stdout.Bytes()
	ev, err := ParseEvent(raw)
	if err != nil {
		log.Printf("clawproc: parse error on line: %s — %v", raw, err)
		return nil, err
	}
	// Log every event so we can trace the full stream.
	if ev.Type != "text_delta" {
		log.Printf("clawproc: event type=%q line=%.300s", ev.Type, raw)
	}
	return ev, nil
}

// Close sends shutdown, waits briefly, then kills the whole process group to
// ensure claw and any sub-processes it started (e.g. whip) are all gone.
func (p *Process) Close() error {
	if p.alive {
		_ = p.writeCommand(stdinCommand{Type: "shutdown"})
	}
	p.alive = false
	if p.stdin != nil {
		_ = p.stdin.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		// Kill the entire process group (negative PID = pgid).
		pgid := p.cmd.Process.Pid
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		return p.cmd.Wait()
	}
	return nil
}

func (p *Process) writeCommand(cmd stdinCommand) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err = p.stdin.Write(data)
	return err
}

// drainStderr reads up to 4KB from stderr (non-blocking best-effort).
func (p *Process) drainStderr() string {
	if p.stderr == nil {
		return ""
	}
	buf := make([]byte, 4096)
	n, _ := p.stderr.Read(buf)
	if n > 0 {
		return string(buf[:n])
	}
	return ""
}
