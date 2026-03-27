// Package agentbuilder provides shared helpers for constructing LLM providers,
// tool sets, and agents from pi-go config. It is the single source of truth
// used by cmd/gateway, cmd/pi, app, and the whipflow runtime.
package agentbuilder

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ai-gateway/pi-go/config"
	"github.com/ai-gateway/pi-go/memory"
	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/provider/anthropic"
	"github.com/ai-gateway/pi-go/provider/gemini"
	"github.com/ai-gateway/pi-go/provider/ollama"
	"github.com/ai-gateway/pi-go/provider/openai"
	"github.com/ai-gateway/pi-go/provider/zenmux"
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/tool/builtin"
)

// BuildProviders constructs one LLMProvider per entry in cfg.Providers.
func BuildProviders(cfg *config.Config) (map[string]provider.LLMProvider, error) {
	out := make(map[string]provider.LLMProvider, len(cfg.Providers))
	for id, pc := range cfg.Providers {
		prov, err := BuildProvider(id, pc)
		if err != nil {
			return nil, err
		}
		out[id] = prov
	}
	return out, nil
}

// BuildProvider creates an LLMProvider from a single ProviderConfig.
func BuildProvider(id string, pc config.ProviderConfig) (provider.LLMProvider, error) {
	key := pc.APIKey
	if key == "" {
		key = os.Getenv(ProviderEnvVar(id))
	}
	t := pc.Type
	if t == "" {
		t = "anthropic"
	}
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
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return openai.NewWithBaseURL(key, base), nil

	case "zenmux":
		base := pc.BaseURL
		if base == "" {
			base = zenmux.DefaultBaseURL
		}
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
		}
		return zenmux.NewWithBaseURL(key, base), nil

	case "gemini":
		if key == "" {
			return nil, fmt.Errorf("provider %q: no api_key", id)
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
			base = "https://openrouter.ai/api/v1"
		}
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
		"zenmux":      "anthropic/claude-haiku-4-5",
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
func BuildTools(names []string, memMgr *memory.Manager, cfg *config.Config, vaultEnv func() map[string]string) []tool.AgentTool {
	if len(names) == 0 {
		return nil
	}

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
		"fetch": &builtin.Fetch{},
		// Workflows
		"whipflow_run": &builtin.WhipflowRun{PiConfig: cfg, VaultEnv: vaultEnv},
		// Memory
		"memory_search": memory.SearchTool(memMgr),
		"memory_get":    memory.GetTool(),
		// Meta
		"sessions_list":    &builtin.SessionsList{},
		"skill":            &builtin.Skill{},
		"get_current_time": &builtin.GetCurrentTime{},
	}

	out := make([]tool.AgentTool, 0, len(names))
	for _, n := range names {
		if t, ok := available[n]; ok {
			out = append(out, t)
		} else {
			log.Printf("agentbuilder: unknown tool %q (ignored)", n)
		}
	}
	return out
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
