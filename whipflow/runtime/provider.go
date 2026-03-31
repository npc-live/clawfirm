package runtime

import (
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

	"github.com/ai-gateway/clawfirm/agent"
	"github.com/ai-gateway/clawfirm/config"
	llmprovider "github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/provider/anthropic"
	"github.com/ai-gateway/clawfirm/provider/gemini"
	"github.com/ai-gateway/clawfirm/provider/ollama"
	"github.com/ai-gateway/clawfirm/provider/openai"
	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

// Provider defines the interface for executing AI sessions.
type Provider interface {
	ExecuteSession(spec SessionSpec, config RuntimeConfig, enableTools bool, allowedTools []string, skillPrompts []string) (*SessionResult, error)
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
	"pi": {
		Name:         "pi",
		Bin:          "pi",
		PromptMode:   "stdin",
		StdinArgs:    []string{"-p", "--no-session", "--provider", "anthropic"},
		OutputFormat: "text",
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
//  2. clawfirm agents in config.yml — matched by agent name, creates NativeProvider
//  3. whipflow.cli_providers in config.yml
//  4. Built-in CLI presets (claude-code, claude, opencode, aider, pi, fetch)
//  5. "custom:bin args..." format
func ResolveProvider(name string, piCfg *config.Config, nativeRegistry map[string]Provider) (Provider, error) {
	// 1. Check native registry (injected programmatically).
	if nativeRegistry != nil {
		if p, ok := nativeRegistry[name]; ok {
			return p, nil
		}
	}

	if piCfg != nil {
		// 2. Check clawfirm agents — each agent is a native provider.
		if ac, ok := piCfg.Agent(name); ok {
			return newNativeProviderFromAgent(ac, piCfg)
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

// newNativeProviderFromAgent creates a NativeProvider from a clawfirm AgentConfig +
// the matching ProviderConfig in the top-level providers map.
func newNativeProviderFromAgent(ac config.AgentConfig, piCfg *config.Config) (*NativeProvider, error) {
	provID := ac.Provider
	pc, ok := piCfg.Providers[provID]
	if !ok {
		return nil, fmt.Errorf("agent %q references unknown provider %q", ac.Name, provID)
	}

	llmProv, err := buildLLMProvider(pc)
	if err != nil {
		return nil, fmt.Errorf("agent %q: %w", ac.Name, err)
	}

	return NewNativeProvider(ac.Name, ac.Model, llmProv,
		withMaxTokens(ac.MaxTokens),
		withSystemPromptHint(ac.SystemPrompt),
	)
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
	prompt := buildPrompt(spec, enableTools, allowedTools, skillPrompts, p.cfg.RawPrompt)

	ctx := cfg.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	output, err := runCli(ctx, p.cfg, prompt, cfg.VaultEnv)
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
// NativeProvider — in-process execution via agent.Agent + provider.LLMProvider
// ---------------------------------------------------------------------------

// NativeProvider implements the Provider interface using clawfirm's native
// agent.Agent and provider.LLMProvider, executing sessions in-process
// instead of spawning external CLI processes.
type NativeProvider struct {
	name             string
	llmProvider      llmprovider.LLMProvider
	model            types.Model
	tools            []tool.AgentTool
	systemPromptHint string // default system prompt from agent config
}

// NativeProviderOption configures a NativeProvider.
type NativeProviderOption func(*NativeProvider)

// WithNativeTools sets the tools available to the native agent.
func WithNativeTools(tools []tool.AgentTool) NativeProviderOption {
	return func(np *NativeProvider) { np.tools = tools }
}

// WithLLMProvider overrides the LLM provider.
func WithLLMProvider(p llmprovider.LLMProvider) NativeProviderOption {
	return func(np *NativeProvider) { np.llmProvider = p }
}

func withMaxTokens(n int) NativeProviderOption {
	return func(np *NativeProvider) {
		if n > 0 {
			np.model.MaxTokens = n
		}
	}
}

func withSystemPromptHint(s string) NativeProviderOption {
	return func(np *NativeProvider) { np.systemPromptHint = s }
}

// NewNativeProvider creates a NativeProvider.
// The llmProv parameter can be nil if WithLLMProvider is used.
func NewNativeProvider(name, modelID string, llmProv llmprovider.LLMProvider, opts ...NativeProviderOption) (*NativeProvider, error) {
	np := &NativeProvider{
		name:        name,
		llmProvider: llmProv,
		model:       types.Model{ID: modelID},
	}
	for _, opt := range opts {
		opt(np)
	}
	if np.llmProvider == nil {
		return nil, fmt.Errorf("native provider %q: no LLMProvider configured", name)
	}
	return np, nil
}

// ProviderName returns the display name.
func (np *NativeProvider) ProviderName() string { return np.name }

// ExecuteSession runs a session using clawfirm's agent loop in-process.
func (np *NativeProvider) ExecuteSession(spec SessionSpec, cfg RuntimeConfig, enableTools bool, allowedTools []string, skillPrompts []string) (*SessionResult, error) {
	prompt := buildPrompt(spec, enableTools, allowedTools, skillPrompts, false)

	// Build agent options.
	agentOpts := []agent.AgentOption{
		agent.WithModel(np.model),
	}

	// System prompt: prefer the one from the WhipFlow agent definition,
	// fall back to the hint from config.yml agent config.
	sysPrompt := np.systemPromptHint
	if spec.Agent != nil && spec.Agent.Prompt != "" {
		sysPrompt = spec.Agent.Prompt
	}
	if sysPrompt != "" {
		agentOpts = append(agentOpts, agent.WithSystemPrompt(sysPrompt))
	}

	// Attach tools if enabled.
	if enableTools && len(np.tools) > 0 {
		var filtered []tool.AgentTool
		allowed := make(map[string]bool, len(allowedTools))
		for _, n := range allowedTools {
			allowed[n] = true
		}
		for _, t := range np.tools {
			if len(allowed) == 0 || allowed[t.Name()] {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) > 0 {
			agentOpts = append(agentOpts, agent.WithTools(filtered))
		}
	}

	// Create the agent.
	a := agent.NewAgent(np.llmProvider, agentOpts...)

	// Determine timeout.
	timeout := cfg.SessionTimeout
	if timeout <= 0 {
		timeout = 300000 // 5 min default
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	startTime := time.Now()

	// Send the prompt and wait for completion.
	if err := a.Prompt(ctx, prompt); err != nil {
		return nil, fmt.Errorf("native provider %s: prompt failed: %w", np.name, err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		return nil, fmt.Errorf("native provider %s: %w", np.name, err)
	}

	// Extract the result from the agent's final state.
	state := a.State()
	output := extractAgentOutput(state.Messages)

	return &SessionResult{
		Output: output,
		Metadata: SessionMetadata{
			Model:    np.model.ID,
			Duration: time.Since(startTime).Milliseconds(),
		},
	}, nil
}

// extractAgentOutput extracts text output from the agent's message history.
func extractAgentOutput(messages []types.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if am, ok := messages[i].(*types.AssistantMessage); ok {
			var parts []string
			for _, block := range am.Content {
				if tc, ok := block.(*types.TextContent); ok && tc.Text != "" {
					parts = append(parts, tc.Text)
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n")
			}
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// buildLLMProvider — creates LLMProvider from clawfirm ProviderConfig
// ---------------------------------------------------------------------------

// buildLLMProvider creates an LLMProvider from a clawfirm ProviderConfig.
// Config values are already env-expanded by config.Load().
func buildLLMProvider(pc config.ProviderConfig) (llmprovider.LLMProvider, error) {
	apiKey := pc.APIKey
	baseURL := pc.BaseURL
	provType := pc.Type
	if provType == "" {
		provType = "anthropic"
	}

	switch provType {
	case "anthropic":
		if apiKey == "" {
			return nil, fmt.Errorf("api_key required for %s provider", provType)
		}
		if baseURL != "" {
			return anthropic.NewWithBaseURL(apiKey, baseURL), nil
		}
		return anthropic.New(apiKey), nil

	case "minimax":
		if apiKey == "" {
			return nil, fmt.Errorf("api_key required for minimax provider")
		}
		if baseURL == "" {
			baseURL = "https://api.minimax.io/anthropic"
		}
		return anthropic.NewWithBaseURL(apiKey, baseURL), nil

	case "openai", "deepseek", "groq", "mistral", "openrouter":
		if apiKey == "" {
			return nil, fmt.Errorf("api_key required for %s provider", provType)
		}
		if baseURL != "" {
			return openai.NewWithBaseURL(apiKey, baseURL), nil
		}
		return openai.New(apiKey), nil

	case "gemini", "google":
		if apiKey == "" {
			return nil, fmt.Errorf("api_key required for gemini provider")
		}
		return gemini.New(apiKey), nil

	case "ollama":
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return ollama.NewWithBaseURL(baseURL), nil

	default:
		return nil, fmt.Errorf(
			"unsupported provider type %q; use WithLLMProvider() to inject a pre-built provider",
			provType,
		)
	}
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
		parts = append(parts, fmt.Sprintf("## Available Tools\n%s", strings.Join(allowedTools, ", ")))
	}
	if len(skillPrompts) > 0 {
		parts = append(parts, fmt.Sprintf("## Skills & Knowledge\n%s", strings.Join(skillPrompts, "\n\n")))
	}
	parts = append(parts, fmt.Sprintf("## Task\n%s", spec.Prompt))
	if spec.Context != nil && len(spec.Context.Variables) > 0 {
		var varLines []string
		for name, value := range spec.Context.Variables {
			varLines = append(varLines, formatContextVariable(name, value))
		}
		parts = append(parts, fmt.Sprintf("## Context Variables\n%s", strings.Join(varLines, "\n")))
	}

	return strings.Join(parts, "\n\n")
}

func formatContextVariable(name string, value any) string {
	if sr, ok := IsSessionResult(value); ok {
		return fmt.Sprintf("- %s (session result): %s", name, sr.Output)
	}
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("- %s: %s", name, v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("- %s: %v", name, v)
		}
		return fmt.Sprintf("- %s: %s", name, string(data))
	}
}

// ---------------------------------------------------------------------------
// CLI execution
// ---------------------------------------------------------------------------

func runCli(parentCtx context.Context, cfg CliConfig, prompt string, vaultEnv func() map[string]string) (string, error) {
	binPath, err := resolveRealPath(cfg.Bin)
	if err != nil {
		return "", fmt.Errorf("binary not found: %s: %w", cfg.Bin, err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 1800000 // default 30 minutes
	}

	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

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

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if cfg.PromptMode == "stdin" {
		cmd.Stdin = strings.NewReader(prompt)
	}

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

	if cfg.OutputFormat == "stream-json" {
		return parseStreamJson(stdoutStr)
	}

	return stripAnsi(strings.TrimSpace(stdoutStr)), nil
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

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b\[.*?[@-~]`)

func stripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func resolveRealPath(bin string) (string, error) {
	return exec.LookPath(bin)
}
