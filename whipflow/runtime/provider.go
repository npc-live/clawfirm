package runtime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/ai-gateway/clawfirm/config"
)

// Provider defines the interface for executing AI sessions.
type Provider interface {
	ExecuteSession(spec SessionSpec, config RuntimeConfig, enableTools bool, allowedTools []string, skillPrompts []string) (*SessionResult, error)
	// ExecuteSessionStream is like ExecuteSession but calls onStream for each
	// incremental update (tool_use events, text deltas) during execution.
	// If onStream is nil it behaves identically to ExecuteSession.
	ExecuteSessionStream(spec SessionSpec, config RuntimeConfig, enableTools bool, allowedTools []string, skillPrompts []string, onStream func(delta string)) (*SessionResult, error)
	ProviderName() string
}

// ---------------------------------------------------------------------------
// CLI Provider
// ---------------------------------------------------------------------------

// CliConfig holds configuration for a CLI-based AI provider.
// In config.yml this lives under whipflow.cli_providers.
type CliConfig struct {
	Name         string
	Bin          string
	PromptMode   string   // "stdin" or "arg"
	Args         []string // args for arg mode
	StdinArgs    []string // args for stdin mode
	Timeout      int64    // ms, 0 = no timeout, default 1800000 (30 min)
	OutputFormat string   // "text" or "stream-json"
	RawPrompt    bool     // pass prompt without wrapping
}

// cliConfigFromYAML converts a config.WhipflowCliProvider to a CliConfig.
func cliConfigFromYAML(name string, wc config.WhipflowCliProvider) CliConfig {
	cfg := CliConfig{
		Name:         wc.Name,
		Bin:          wc.Bin,
		PromptMode:   wc.PromptMode,
		Args:         wc.Args,
		StdinArgs:    wc.StdinArgs,
		Timeout:      wc.Timeout,
		OutputFormat: wc.OutputFormat,
		RawPrompt:    wc.RawPrompt,
	}
	if cfg.Name == "" {
		cfg.Name = name
	}
	return cfg
}

// builtinPresets contains the default CLI provider configurations.
var builtinPresets = map[string]CliConfig{
	"claw": {
		Name:         "claw",
		Bin:          findClawBin(),
		PromptMode:   "arg",
		Args:         []string{"--output-format", "ndjson", "--permission-mode", "danger-full-access", "prompt"},
		OutputFormat: "ndjson",
	},
	"claude-code": {
		Name:         "claude-code",
		Bin:          "claude",
		PromptMode:   "stdin",
		StdinArgs:    []string{"--output-format", "stream-json", "--verbose", "--dangerously-skip-permissions"},
		OutputFormat: "stream-json",
	},
	"claude": {
		Name:         "claude",
		Bin:          "claude",
		PromptMode:   "stdin",
		StdinArgs:    []string{"--output-format", "stream-json", "--verbose", "--dangerously-skip-permissions"},
		OutputFormat: "stream-json",
	},
	"opencode": {
		Name:       "opencode",
		Bin:        "opencode",
		PromptMode: "arg",
		Args:       []string{"run"},
	},
	"aider": {
		Name:       "aider",
		Bin:        "aider",
		PromptMode: "arg",
		Args:       []string{"--message"},
	},
	"fetch": {
		Name:         "fetch",
		Bin:          "curl",
		PromptMode:   "arg",
		Args:         []string{"-sL", "--max-time", "30", "-A", "Mozilla/5.0"},
		OutputFormat: "text",
		RawPrompt:    true,
	},
}

// ResolveProvider resolves a provider by name, returning a Provider interface.
//
// Resolution order:
//  1. nativeRegistry (injected programmatically, e.g. from app integration)
//  2. clawfirm agents in config.yml — matched by agent name, routes to claw CLI binary
//  3. whipflow.cli_providers in config.yml
//  4. Built-in CLI presets (claude-code, claude, opencode, aider, fetch)
//  5. "custom:bin args..." format
func ResolveProvider(name string, piCfg *config.Config, nativeRegistry map[string]Provider) (Provider, error) {
	// 1. Check native registry (injected programmatically).
	if nativeRegistry != nil {
		if p, ok := nativeRegistry[name]; ok {
			return p, nil
		}
	}

	if piCfg != nil {
		// 2. Check clawfirm agents — route to claw CLI binary so the
		//    session gets claw-code's full tool set (40+ built-in tools).
		if ac, ok := piCfg.Agent(name); ok {
			return newClawCliProvider(ac), nil
		}

		// 3. Check whipflow.cli_providers.
		if wc, ok := piCfg.Whipflow.CliProviders[name]; ok {
			return &CliProvider{cfg: cliConfigFromYAML(name, wc)}, nil
		}
	}

	// 4. Built-in CLI presets.
	if preset, ok := builtinPresets[name]; ok {
		copy := preset
		return &CliProvider{cfg: copy}, nil
	}

	// 5. Handle "custom:bin args..." format.
	if strings.HasPrefix(name, "custom:") {
		rest := strings.TrimPrefix(name, "custom:")
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			cfg := CliConfig{
				Name:       "custom",
				Bin:        parts[0],
				PromptMode: "stdin",
			}
			if len(parts) > 1 {
				cfg.StdinArgs = parts[1:]
			}
			return &CliProvider{cfg: cfg}, nil
		}
	}

	return nil, fmt.Errorf("unknown provider: %s", name)
}

// newClawCliProvider creates a CliProvider that invokes the claw binary for
// a clawfirm agent. The agent's model is passed via --model so claw uses the
// correct model. This replaces the old NativeProvider (Go agent loop) and gives
// whipflow sessions access to claw-code's full 40+ built-in tools.
func newClawCliProvider(ac config.AgentConfig) *CliProvider {
	bin := findClawBin()
	args := []string{
		"--output-format", "ndjson",
		"--permission-mode", "danger-full-access",
	}
	if ac.Model != "" {
		args = append(args, "--model", ac.Model)
	}
	args = append(args, "prompt")
	return &CliProvider{cfg: CliConfig{
		Name:         ac.Name,
		Bin:          bin,
		PromptMode:   "arg",
		Args:         args,
		OutputFormat: "ndjson",
		Timeout:      1800000, // 30 min
	}}
}

// findClawBin locates the claw binary for whipflow CLI providers.
func findClawBin() string {
	// 1. ~/.clawfirm/bin/claw (extracted from app bundle).
	if home, err := os.UserHomeDir(); err == nil {
		p := home + "/.clawfirm/bin/claw"
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// 2. PATH.
	if p, err := exec.LookPath("claw"); err == nil {
		return p
	}
	return "claw"
}

// CliProvider implements the Provider interface using CLI-based AI tools.
type CliProvider struct {
	cfg CliConfig
}

// NewCliProvider creates a new CliProvider from a CliConfig.
func NewCliProvider(cfg CliConfig) *CliProvider {
	return &CliProvider{cfg: cfg}
}

// ProviderName returns the display name of this provider.
func (p *CliProvider) ProviderName() string {
	return p.cfg.Name
}

// ExecuteSession executes an AI session using the configured CLI tool.
func (p *CliProvider) ExecuteSession(spec SessionSpec, cfg RuntimeConfig, enableTools bool, allowedTools []string, skillPrompts []string) (*SessionResult, error) {
	return p.ExecuteSessionStream(spec, cfg, enableTools, allowedTools, skillPrompts, nil)
}

// ExecuteSessionStream executes an AI session and calls onStream for each
// tool_use event or text delta emitted by the provider during execution.
func (p *CliProvider) ExecuteSessionStream(spec SessionSpec, cfg RuntimeConfig, enableTools bool, allowedTools []string, skillPrompts []string, onStream func(delta string)) (*SessionResult, error) {
	prompt := buildPrompt(spec, enableTools, allowedTools, skillPrompts, p.cfg.RawPrompt)

	ctx := cfg.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	output, err := runCliStream(ctx, p.cfg, prompt, cfg.VaultEnv, onStream)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.cfg.Name, err)
	}

	return &SessionResult{
		Output: output,
		Metadata: SessionMetadata{
			Model: p.cfg.Name,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Prompt building
// ---------------------------------------------------------------------------

// buildPrompt constructs the full prompt from a SessionSpec.
func buildPrompt(spec SessionSpec, enableTools bool, allowedTools []string, skillPrompts []string, rawPrompt bool) string {
	if rawPrompt {
		return spec.Prompt
	}

	var parts []string

	if spec.Agent != nil && spec.Agent.Prompt != "" {
		parts = append(parts, fmt.Sprintf("## System\n%s", spec.Agent.Prompt))
	}
	if enableTools && len(allowedTools) > 0 {
		parts = append(parts, fmt.Sprintf("## Available Tools\n%s\nIMPORTANT: These tools are pre-approved and available. Do NOT use ToolSearch to verify them — call them directly.", strings.Join(allowedTools, ", ")))
	}
	if len(skillPrompts) > 0 {
		parts = append(parts, fmt.Sprintf("## Skills & Knowledge\n%s", strings.Join(skillPrompts, "\n\n")))
	}
	parts = append(parts, spec.Prompt)

	return strings.Join(parts, "\n\n")
}


// ---------------------------------------------------------------------------
// CLI execution
// ---------------------------------------------------------------------------

func runCliStream(parentCtx context.Context, cfg CliConfig, prompt string, vaultEnv func() map[string]string, onStream func(delta string)) (string, error) {
	binPath, err := resolveRealPath(cfg.Bin)
	if err != nil {
		return "", fmt.Errorf("binary not found: %s: %w", cfg.Bin, err)
	}

	timeout := cfg.Timeout
	ctx := parentCtx
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(parentCtx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
	}

	var args []string
	if cfg.PromptMode == "stdin" {
		args = append(args, cfg.StdinArgs...)
	} else {
		args = append(args, cfg.Args...)
		args = append(args, prompt)
	}

	cmd := exec.CommandContext(ctx, binPath, args...)

	// Strip CLAUDECODE so that nested claude invocations are not rejected
	// when clawfirm itself is running inside a Claude Code terminal session.
	// Also inject vault secrets as environment variables.
	filtered := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "CLAUDECODE=") {
			filtered = append(filtered, kv)
		}
	}
	if vaultEnv != nil {
		for k, v := range vaultEnv() {
			filtered = append(filtered, fmt.Sprintf("%s=%s", k, v))
		}
	}
	cmd.Env = filtered

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if cfg.PromptMode == "stdin" {
		cmd.Stdin = strings.NewReader(prompt)
	}

	// For ndjson or claw-json format with a stream callback, read stdout
	// line-by-line so we can emit text deltas in real time.
	if (cfg.OutputFormat == "ndjson" || cfg.OutputFormat == "claw-json") && onStream != nil {
		stdoutPipe, pipeErr := cmd.StdoutPipe()
		if pipeErr != nil {
			return "", fmt.Errorf("stdout pipe: %w", pipeErr)
		}
		if startErr := cmd.Start(); startErr != nil {
			return "", fmt.Errorf("failed to start %s: %w", cfg.Bin, startErr)
		}

		var allLines []string
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 256*1024), 256*1024)
		for scanner.Scan() {
			line := scanner.Text()
			allLines = append(allLines, line)
			emitClawStreamDelta(line, onStream)
		}

		runErr := cmd.Wait()
		if runErr != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return "", fmt.Errorf("command timed out after %dms", timeout)
			}
			if _, ok := runErr.(*exec.ExitError); !ok {
				return "", fmt.Errorf("failed to execute %s: %w", cfg.Bin, runErr)
			}
			log.Printf("provider %s exited with error: %v, stderr: %s", cfg.Name, runErr, stderr.String())
		}

		combined := strings.Join(allLines, "\n")
		if cfg.OutputFormat == "ndjson" {
			return parseNdjson(combined)
		}
		return parseClawJson(combined)
	}

	// Default: buffer all stdout then parse.
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out after %dms", timeout)
		}
		if _, ok := err.(*exec.ExitError); !ok {
			return "", fmt.Errorf("failed to execute %s: %w", cfg.Bin, err)
		}
		log.Printf("provider %s exited with error: %v, stderr: %s", cfg.Name, err, stderr.String())
	}

	stdoutStr := stdout.String()

	switch cfg.OutputFormat {
	case "stream-json":
		return parseStreamJson(stdoutStr)
	case "ndjson":
		return parseNdjson(stdoutStr)
	case "claw-json":
		return parseClawJson(stdoutStr)
	}

	return stripAnsi(strings.TrimSpace(stdoutStr)), nil
}

// emitClawStreamDelta parses a single claw JSON output line and calls onStream
// with a human-readable delta if the line is a tool_use event or text delta.
func emitClawStreamDelta(line string, onStream func(delta string)) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] != '{' {
		return
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return
	}
	// claw --output-format json emits lines like:
	//   {"type":"tool_use","name":"bash","input":"..."}
	//   {"type":"text","text":"..."}
	evType, _ := obj["type"].(string)
	switch evType {
	case "tool_use":
		name, _ := obj["name"].(string)
		input, _ := obj["input"].(string)
		if name != "" {
			// Trim input to a short preview for readability.
			preview := strings.ReplaceAll(input, "\n", " ")
			if len(preview) > 80 {
				preview = preview[:80] + "…"
			}
			if preview != "" {
				onStream(fmt.Sprintf("🔧 %s  %s", name, preview))
			} else {
				onStream(fmt.Sprintf("🔧 %s", name))
			}
		}
	case "text", "text_delta": // claw --output-format json uses "text"; ndjson uses "text_delta"
		text, _ := obj["text"].(string)
		if text != "" {
			onStream(text)
		}
	}
}

func parseStreamJson(stdout string) (string, error) {
	lines := strings.Split(stdout, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		msgType, _ := msg["type"].(string)
		if msgType != "result" {
			continue
		}

		if isError, ok := msg["is_error"].(bool); ok && isError {
			errMsg := "session returned an error"
			if result, ok := msg["result"].(string); ok {
				errMsg = result
			}
			return "", fmt.Errorf("%s", errMsg)
		}

		if result, ok := msg["result"].(string); ok {
			return result, nil
		}

		if result, ok := msg["result"]; ok {
			data, err := json.Marshal(result)
			if err != nil {
				return fmt.Sprintf("%v", result), nil
			}
			return string(data), nil
		}
	}

	return stripAnsi(strings.TrimSpace(stdout)), nil
}

// parseClawJson parses claw's non-interactive JSON output format.
// claw --output-format json prompt "..." writes {"message":"..."} as the last
// JSON line on stdout. Earlier lines may contain TUI startup text, session
// banners, or ANSI-decorated progress — we scan from the end to find the JSON.
func parseClawJson(stdout string) (string, error) {
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		return "", nil
	}
	// Scan lines in reverse: the last valid JSON line with a "message" field wins.
	lines := strings.Split(stdout, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || line[0] != '{' {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		if msg, ok := obj["message"].(string); ok && msg != "" {
			return strings.TrimSpace(msg), nil
		}
	}
	// Fallback: strip ANSI and return the raw text.
	return stripAnsi(stdout), nil
}

// parseNdjson parses claw's ndjson session format.
// Each line is a JSON object; we find the last assistant "message" event and
// extract its text blocks as the final answer.
func parseNdjson(stdout string) (string, error) {
	lines := strings.Split(stdout, "\n")

	var lastAssistantText string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		msgType, _ := msg["type"].(string)
		switch msgType {
		case "message":
			message, ok := msg["message"].(map[string]any)
			if !ok {
				continue
			}
			role, _ := message["role"].(string)
			if role != "assistant" {
				continue
			}
			blocks, ok := message["blocks"].([]any)
			if !ok {
				continue
			}
			var parts []string
			for _, b := range blocks {
				block, ok := b.(map[string]any)
				if !ok {
					continue
				}
				if block["type"] == "text" {
					if text, ok := block["text"].(string); ok && text != "" {
						parts = append(parts, text)
					}
				}
			}
			if len(parts) > 0 {
				lastAssistantText = strings.Join(parts, "")
			}
		case "error":
			errMsg := "session returned an error"
			if message, ok := msg["message"].(string); ok {
				errMsg = message
			}
			return "", fmt.Errorf("%s", errMsg)
		}
	}

	if lastAssistantText != "" {
		return strings.TrimSpace(lastAssistantText), nil
	}
	return stripAnsi(strings.TrimSpace(stdout)), nil
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b\[.*?[@-~]`)

func stripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func resolveRealPath(bin string) (string, error) {
	return exec.LookPath(bin)
}
