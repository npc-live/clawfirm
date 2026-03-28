package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ai-gateway/pi-go/config"
)

const sampleYAML = `
providers:
  zenmux:
    type: anthropic
    api_key: ${ZENMUX_API_KEY}
    base_url: https://zenmux.ai/api/anthropic
  minimax:
    type: anthropic
    api_key: ${MINIMAX_API_KEY}
    base_url: https://api.minimax.io/anthropic
  anthropic:
    api_key: sk-ant-test
  openai:
    api_key: ${OPENAI_API_KEY}

agents:
  - name: zenmux
    provider: zenmux
    model: anthropic/claude-haiku-4-5
    system_prompt: "你是一个有帮助的助手。"
  - name: minimax
    provider: minimax
    model: MiniMax-M2.7
    system_prompt: "你是一个简洁的助手。"
    max_tokens: 2048

default_agent: zenmux
default_provider: zenmux
default_model: anthropic/claude-haiku-4-5
`

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(sampleYAML), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Setenv("ZENMUX_API_KEY", "sk-ss-v1-testkey")
	t.Setenv("MINIMAX_API_KEY", "sk-mm-testkey")
	t.Setenv("OPENAI_API_KEY", "")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// ZenMux — env var expansion + type + base_url
	if cfg.ProviderAPIKey("zenmux") != "sk-ss-v1-testkey" {
		t.Errorf("zenmux api_key: got %q", cfg.ProviderAPIKey("zenmux"))
	}
	if cfg.ProviderBaseURL("zenmux") != "https://zenmux.ai/api/anthropic" {
		t.Errorf("zenmux base_url: got %q", cfg.ProviderBaseURL("zenmux"))
	}
	if cfg.ProviderType("zenmux") != "anthropic" {
		t.Errorf("zenmux type: got %q", cfg.ProviderType("zenmux"))
	}

	// MiniMax
	if cfg.ProviderAPIKey("minimax") != "sk-mm-testkey" {
		t.Errorf("minimax api_key: got %q", cfg.ProviderAPIKey("minimax"))
	}
	if cfg.ProviderType("minimax") != "anthropic" {
		t.Errorf("minimax type: got %q", cfg.ProviderType("minimax"))
	}

	// Anthropic — literal value, no type set → defaults to "anthropic"
	if cfg.ProviderAPIKey("anthropic") != "sk-ant-test" {
		t.Errorf("anthropic api_key: got %q", cfg.ProviderAPIKey("anthropic"))
	}
	if cfg.ProviderType("anthropic") != "anthropic" {
		t.Errorf("anthropic type: got %q", cfg.ProviderType("anthropic"))
	}

	// OpenAI — env var empty
	if cfg.ProviderAPIKey("openai") != "" {
		t.Errorf("openai api_key: got %q, want empty", cfg.ProviderAPIKey("openai"))
	}

	// Agents
	if len(cfg.Agents) != 2 {
		t.Fatalf("agents: want 2 got %d", len(cfg.Agents))
	}
	if cfg.Agents[0].Name != "zenmux" || cfg.Agents[0].Provider != "zenmux" {
		t.Errorf("agent[0]: %+v", cfg.Agents[0])
	}
	if cfg.Agents[1].Name != "minimax" || cfg.Agents[1].Model != "MiniMax-M2.7" {
		t.Errorf("agent[1]: %+v", cfg.Agents[1])
	}
	if cfg.Agents[1].MaxTokens != 2048 {
		t.Errorf("agent[1].max_tokens: want 2048 got %d", cfg.Agents[1].MaxTokens)
	}

	// Agent lookup
	a, ok := cfg.Agent("minimax")
	if !ok || a.SystemPrompt != "你是一个简洁的助手。" {
		t.Errorf("Agent lookup: ok=%v agent=%+v", ok, a)
	}
	if _, ok := cfg.Agent("nonexistent"); ok {
		t.Error("Agent lookup: expected false for unknown name")
	}

	// DefaultAgent
	if cfg.DefaultAgent != "zenmux" {
		t.Errorf("default_agent: got %q", cfg.DefaultAgent)
	}

	// Defaults
	if cfg.DefaultProvider != "zenmux" {
		t.Errorf("default_provider: got %q", cfg.DefaultProvider)
	}
	if cfg.DefaultModel != "anthropic/claude-haiku-4-5" {
		t.Errorf("default_model: got %q", cfg.DefaultModel)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}
	if cfg.ProviderAPIKey("zenmux") != "" {
		t.Error("expected empty key for missing config")
	}
}

func TestLoadConfigEnvFallback(t *testing.T) {
	t.Setenv("ZENMUX_API_KEY", "from-env")

	// Config with no providers section → falls back to env
	cfg, err := config.Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.ProviderAPIKey("zenmux"); got != "from-env" {
		t.Errorf("env fallback: got %q want %q", got, "from-env")
	}
}

func TestLoadConfigDollarBrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte("providers:\n  x:\n    api_key: ${MY_KEY}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MY_KEY", "expanded-value")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProviderAPIKey("x") != "expanded-value" {
		t.Errorf("got %q", cfg.ProviderAPIKey("x"))
	}
}
