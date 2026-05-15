// Package agentbuilder provides shared helpers for constructing LLM providers,
// tool sets, and agents from clawfirm config. It is the single source of truth
// used by cmd/gateway, cmd/pi, app, and the whipflow runtime.
package agentbuilder

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/memory"
	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/provider/anthropic"
	"github.com/ai-gateway/clawfirm/provider/gemini"
	"github.com/ai-gateway/clawfirm/provider/ollama"
	"github.com/ai-gateway/clawfirm/provider/openai"
"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/tool/builtin"
)

// BuildProviders constructs one LLMProvider per entry in cfg.Providers.
func BuildProviders(cfg *config.Config) (map[string]provider.LLMProvider, error) {
	out := make(map[string]provider.LLMProvider, len(cfg.Providers))
	for id, pc := range cfg.Providers {
		prov, err := BuildProvider(id, pc)
		if err != nil {
			log.Printf("agentbuilder: skipping provider %q: %v", id, err)
			continue
		}
		out[id] = prov
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no usable providers configured")
	}
	return out, nil
}

// BuildProvider creates an LLMProvider from a single ProviderConfig.
func BuildProvider(id string, pc config.ProviderConfig) (provider.LLMProvider, error) {
	key := pc.APIKey
	if key == "" {
		key = os.Getenv(ProviderEnvVar(id))
	}
	t := pc.ResolvedProtocol()
	switch t {
	case "anthropic":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.anthropic.com"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return anthropic.NewWithBaseURL(key, base), nil

	case "openai":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.openai.com/v1"
		}
		if key == "" && !isLocalURL(base) {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "gemini":
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		if pc.BaseURL != "" {
			return gemini.NewWithBaseURL(key, pc.BaseURL), nil
		}
		return gemini.New(key), nil

	case "ollama":
		base := pc.BaseURL
		if base == "" {
			base = "http://localhost:11434"
		}
		return ollama.NewWithBaseURL(base), nil

	// ── OpenAI-compatible providers ──────────────────────────────────────────

	case "deepseek":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.deepseek.com/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "minimax":
		// MiniMax Anthropic-compatible endpoint.
		// Doc: https://platform.minimax.io/docs/guides/text-generation
		base := pc.BaseURL
		if base == "" {
			base = "https://api.minimax.io/anthropic"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return anthropic.NewWithBaseURL(key, base), nil

	case "moonshot":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.moonshot.cn/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "volcengine":
		base := pc.BaseURL
		if base == "" {
			base = "https://ark.cn-beijing.volces.com/api/v3"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "modelstudio":
		// Alibaba Cloud Model Studio (DashScope OpenAI-compat).
		base := pc.BaseURL
		if base == "" {
			base = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "glm", "zai":
		// Z.AI / GLM (Zhipu AI) — OpenAI-compatible v4 endpoint.
		base := pc.BaseURL
		if base == "" {
			base = "https://open.bigmodel.cn/api/paas/v4"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "groq":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.groq.com/openai/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "openrouter":
		base := pc.BaseURL
		if base == "" {
			base = "https://openrouter.ai/api"
		}
		// Normalize: the openai provider appends /v1/chat/completions,
		// so strip trailing /v1 if the user supplied the OpenRouter default.
		base = strings.TrimSuffix(base, "/v1")
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "together":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.together.xyz/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "mistral":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.mistral.ai/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "xai":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.x.ai/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "nvidia":
		base := pc.BaseURL
		if base == "" {
			base = "https://integrate.api.nvidia.com/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "xiaomi":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.xiaomimimo.com/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "venice":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.venice.ai/api/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "huggingface":
		base := pc.BaseURL
		if base == "" {
			base = "https://router.huggingface.co/v1"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "perplexity":
		base := pc.BaseURL
		if base == "" {
			base = "https://api.perplexity.ai"
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "litellm":
		base := pc.BaseURL
		if base == "" {
			base = "http://localhost:4000/v1"
		}
		// LiteLLM is often deployed without a key; allow empty.
		return openai.NewWithBaseURL(key, base), nil

	case "sglang":
		base := pc.BaseURL
		if base == "" {
			base = "http://127.0.0.1:30000/v1"
		}
		return openai.NewWithBaseURL(key, base), nil

	case "vllm":
		base := pc.BaseURL
		if base == "" {
			base = "http://127.0.0.1:8000/v1"
		}
		return openai.NewWithBaseURL(key, base), nil

	default:
		return nil, fmt.Errorf("provider %q: unknown type %q", id, t)
	}
}

// DefaultModelForProvider returns the default model ID for a provider type.
func DefaultModelForProvider(providerID string) string {
	defaults := map[string]string{
		"anthropic":   "claude-haiku-4-5-20251001",
		"openai":      "gpt-4o-mini",
		"gemini":      "gemini-2.0-flash",
		"ollama":      "llama3.2",
		"deepseek":    "deepseek-chat",
		"minimax":     "MiniMax-M2.7",
		"moonshot":    "moonshot-v1-8k",
		"volcengine":  "doubao-lite-4k",
		"modelstudio": "qwen-turbo",
		"glm":         "glm-4-flash",
		"zai":         "glm-4-flash",
		"groq":        "llama-3.1-8b-instant",
		"openrouter":  "openai/gpt-4o-mini",
		"together":    "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
		"mistral":     "mistral-small-latest",
		"xai":         "grok-3-mini",
		"nvidia":      "meta/llama-3.1-405b-instruct",
		"xiaomi":      "mimo-v2-flash",
		"venice":      "llama-3.3-70b",
		"huggingface": "meta-llama/Llama-3.3-70B-Instruct",
		"perplexity":  "sonar",
		"litellm":     "",
		"sglang":      "",
		"vllm":        "",
	}
	if m, ok := defaults[providerID]; ok {
		return m
	}
	return ""
}

// BuildTools resolves tool names to AgentTool instances.
// memMgr may be nil — in that case memory_search/memory_get are unavailable.
// vaultEnv may be nil — in that case the vault is not injected into exec/bash/process.
// providerMap may be nil — tools that need an LLM provider will be unavailable.
//
// The returned slice always includes a tool_search meta-tool as the last element
// (unless names is empty). tool_search lets the LLM discover deferred tools.
// AgentRef identifies the calling agent's provider and model so that tools
// like browser_shortcut can reuse the same LLM for sub-tasks (e.g. healing).
type AgentRef struct {
	Provider provider.LLMProvider
	Model    string
}

func BuildTools(names []string, memMgr *memory.Manager, cfg *config.Config, vaultEnv func() map[string]string, providerMap map[string]provider.LLMProvider, agentRef ...AgentRef) []tool.AgentTool {
	if len(names) == 0 {
		return nil
	}

	// Build shared Skill loader with paths from all agents.
	// Bundled skills dir is appended last as fallback (user paths take priority).
	var skillPaths []string
	if cfg != nil {
		for _, ac := range cfg.Agents {
			skillPaths = append(skillPaths, ac.SkillPaths...)
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		skillPaths = append(skillPaths, filepath.Join(home, ".clawfirm", "bundled", "skills"))
	}
	skillTool := &builtin.Skill{SkillPaths: skillPaths}

	available := map[string]tool.AgentTool{
		// File system
		"read":        &builtin.Read{},
		"write":       &builtin.Write{},
		"edit":        &builtin.Edit{},
		"apply_patch": &builtin.ApplyPatch{},
		"grep":        &builtin.Grep{},
		"find":        &builtin.Find{},
		"ls":          &builtin.Ls{},
		// Execution — vault secrets injected as env vars
		"bash":    &builtin.Bash{VaultEnv: vaultEnv},
		"exec":    &builtin.Exec{VaultEnv: vaultEnv},
		"process": &builtin.Process{VaultEnv: vaultEnv},
		// Network
		"fetch":      &builtin.Fetch{},
		"web_search": &builtin.WebSearch{APIKey: toolAPIKey(cfg, "web_search")},
		// Browser automation — auto-healing uses the agent's own provider.
		"browser_shortcut": buildBrowserShortcut(agentRef),
		// Workflows
		"whipflow_run": &builtin.WhipflowRun{PiConfig: cfg, VaultEnv: vaultEnv, SkillLoader: skillTool},
		// Memory
		"memory_search": memory.SearchTool(memMgr),
		"memory_get":    memory.GetTool(),
		// Meta
		"sessions_list":    &builtin.SessionsList{},
		"skill":            skillTool,
		"get_current_time": &builtin.GetCurrentTime{},
		// Interactive / delegation
		"ask_user":  &builtin.AskUser{},
		"sub_agent": &builtin.SubAgent{},
		// Media analysis
		"media_understand": &builtin.MediaUnderstand{
			Provider: toolLLMProvider(cfg, providerMap, "media_understand"),
			Model:    toolModel(cfg, "media_understand"),
		},
		// Image generation
		"media_gen": &builtin.MediaGen{
			Platform: toolPlatform(cfg, "media_gen"),
			Protocol: toolProtocol(cfg, "media_gen"),
			Model:    toolModel(cfg, "media_gen"),
			APIKey:   toolAPIKey(cfg, "media_gen"),
			BaseURL:  toolBaseURL(cfg, "media_gen"),
		},
	}

	out := make([]tool.AgentTool, 0, len(names)+1)
	for _, n := range names {
		if t, ok := available[n]; ok {
			out = append(out, t)
		} else {
			log.Printf("agentbuilder: unknown tool %q (ignored)", n)
		}
	}

	// Inject resolved tools into WhipflowRun so its NativeProvider sessions
	// can use real tools (bash, read, write, etc.) instead of text stubs.
	for _, t := range out {
		if wr, ok := t.(*builtin.WhipflowRun); ok {
			wr.Tools = out
			break
		}
	}

	// Append tool_search meta-tool with catalog of all resolved tools.
	allToolInfos := make([]builtin.ToolInfo, 0, len(out))
	for _, t := range out {
		allToolInfos = append(allToolInfos, builtin.ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
		})
	}
	out = append(out, &builtin.ToolSearch{
		AllTools:       allToolInfos,
		ActivatedTools: make(map[string]bool),
	})

	return out
}

// toolAPIKey returns the API key for a named tool.
// Priority: ToolConfig.APIKey > Providers[tc.Provider].APIKey > env fallback.
func toolAPIKey(cfg *config.Config, toolName string) string {
	if cfg == nil {
		return ""
	}
	tc, ok := cfg.Tools[toolName]
	if !ok {
		return ""
	}
	if tc.APIKey != "" {
		return tc.APIKey
	}
	if tc.Provider == "" {
		return ""
	}
	if pc, ok := cfg.Providers[tc.Provider]; ok && pc.APIKey != "" {
		return pc.APIKey
	}
	return os.Getenv(ProviderEnvVar(tc.Provider))
}

// toolProviderType returns the provider protocol string (e.g. "gemini", "openai")
// for a named tool, falling back to the provider ID if unset.
func toolProviderType(cfg *config.Config, toolName string) string {
	if cfg == nil {
		return ""
	}
	tc, ok := cfg.Tools[toolName]
	if !ok || tc.Provider == "" {
		return ""
	}
	if pc, ok := cfg.Providers[tc.Provider]; ok {
		return pc.ResolvedProtocol()
	}
	return tc.Provider
}

// toolLLMProvider looks up a full LLMProvider for a named tool from the provider map.
func toolLLMProvider(cfg *config.Config, providerMap map[string]provider.LLMProvider, toolName string) provider.LLMProvider {
	if cfg == nil || providerMap == nil {
		return nil
	}
	tc, ok := cfg.Tools[toolName]
	if !ok || tc.Provider == "" {
		return nil
	}
	return providerMap[tc.Provider]
}

// toolPlatform returns the hosting platform for a named tool.
// Priority: ProviderConfig.Platform > provider key name.
func toolPlatform(cfg *config.Config, toolName string) string {
	if cfg == nil {
		return ""
	}
	tc, ok := cfg.Tools[toolName]
	if !ok || tc.Provider == "" {
		return ""
	}
	if pc, ok := cfg.Providers[tc.Provider]; ok && pc.Platform != "" {
		return pc.Platform
	}
	return tc.Provider
}

// toolProtocol returns the explicit protocol override for a named tool.
func toolProtocol(cfg *config.Config, toolName string) string {
	if cfg == nil {
		return ""
	}
	if tc, ok := cfg.Tools[toolName]; ok {
		return tc.Protocol
	}
	return ""
}

// toolModel returns the configured model for a named tool.
func toolModel(cfg *config.Config, toolName string) string {
	if cfg == nil {
		return ""
	}
	if tc, ok := cfg.Tools[toolName]; ok {
		return tc.Model
	}
	return ""
}

// toolBaseURL returns the base URL for a named tool's configured provider.
func toolBaseURL(cfg *config.Config, toolName string) string {
	if cfg == nil {
		return ""
	}
	tc, ok := cfg.Tools[toolName]
	if !ok || tc.Provider == "" {
		return ""
	}
	if pc, ok := cfg.Providers[tc.Provider]; ok {
		return pc.BaseURL
	}
	return ""
}

// buildBrowserShortcut creates a BrowserShortcut tool, optionally wired
// with the calling agent's provider for auto-healing.
func buildBrowserShortcut(refs []AgentRef) *builtin.BrowserShortcut {
	if len(refs) > 0 && refs[0].Provider != nil {
		return &builtin.BrowserShortcut{
			Provider: refs[0].Provider,
			Model:    refs[0].Model,
		}
	}
	return &builtin.BrowserShortcut{}
}

// isLocalURL returns true if the URL points to a local, loopback, or
// private-network address (RFC 1918 / link-local).
func isLocalURL(u string) bool {
	return strings.Contains(u, "://localhost") ||
		strings.Contains(u, "://127.") ||
		strings.Contains(u, "://0.0.0.0") ||
		strings.Contains(u, "://[::1]") ||
		strings.Contains(u, "://10.") ||
		strings.Contains(u, "://192.168.") ||
		strings.Contains(u, "://172.16.") ||
		strings.Contains(u, "://172.17.") ||
		strings.Contains(u, "://172.18.") ||
		strings.Contains(u, "://172.19.") ||
		strings.Contains(u, "://172.20.") ||
		strings.Contains(u, "://172.21.") ||
		strings.Contains(u, "://172.22.") ||
		strings.Contains(u, "://172.23.") ||
		strings.Contains(u, "://172.24.") ||
		strings.Contains(u, "://172.25.") ||
		strings.Contains(u, "://172.26.") ||
		strings.Contains(u, "://172.27.") ||
		strings.Contains(u, "://172.28.") ||
		strings.Contains(u, "://172.29.") ||
		strings.Contains(u, "://172.30.") ||
		strings.Contains(u, "://172.31.")
}

// ProviderEnvVar returns the conventional env var name for a provider ID.
// e.g. "minimax" → "MINIMAX_API_KEY", "my-provider" → "MY_PROVIDER_API_KEY"
func ProviderEnvVar(id string) string {
	var sb strings.Builder
	for _, c := range id {
		if c == '-' {
			sb.WriteByte('_')
		} else if c >= 'a' && c <= 'z' {
			sb.WriteByte(byte(c - 32))
		} else {
			sb.WriteRune(c)
		}
	}
	sb.WriteString("_API_KEY")
	return sb.String()
}
