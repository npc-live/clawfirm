package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ai-gateway/clawfirm/config"
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

func TestApplyEnvDefaults_SimpleFields(t *testing.T) {
	t.Setenv("CLAWFIRM_DEFAULT_PROVIDER", "env-provider")
	t.Setenv("CLAWFIRM_DEFAULT_MODEL", "env-model")
	t.Setenv("CLAWFIRM_DEFAULT_AGENT", "env-agent")

	cfg, err := config.Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultProvider != "env-provider" {
		t.Errorf("DefaultProvider: got %q want %q", cfg.DefaultProvider, "env-provider")
	}
	if cfg.DefaultModel != "env-model" {
		t.Errorf("DefaultModel: got %q want %q", cfg.DefaultModel, "env-model")
	}
	if cfg.DefaultAgent != "env-agent" {
		t.Errorf("DefaultAgent: got %q want %q", cfg.DefaultAgent, "env-agent")
	}
}

func TestApplyEnvDefaults_YAMLTakesPriority(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte("default_agent: from-yaml\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAWFIRM_DEFAULT_AGENT", "from-env")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultAgent != "from-yaml" {
		t.Errorf("DefaultAgent: got %q want %q (YAML should win)", cfg.DefaultAgent, "from-yaml")
	}
}

func TestApplyEnvDefaults_TelegramLegacy(t *testing.T) {
	// Legacy env var should work
	t.Setenv("TELEGRAM_BOT_TOKEN", "legacy-token")

	cfg, err := config.Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Telegram.BotToken != "legacy-token" {
		t.Errorf("Telegram.BotToken: got %q want %q", cfg.Telegram.BotToken, "legacy-token")
	}
}

func TestApplyEnvDefaults_TelegramPrefixWins(t *testing.T) {
	// CLAWFIRM_ prefix takes priority over legacy
	t.Setenv("CLAWFIRM_TELEGRAM_BOT_TOKEN", "new-token")
	t.Setenv("TELEGRAM_BOT_TOKEN", "legacy-token")

	cfg, err := config.Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Telegram.BotToken != "new-token" {
		t.Errorf("Telegram.BotToken: got %q want %q (CLAWFIRM_ should win)", cfg.Telegram.BotToken, "new-token")
	}
}

func TestApplyEnvDefaults_FeishuLegacy(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "legacy-id")
	t.Setenv("FEISHU_APP_SECRET", "legacy-secret")

	cfg, err := config.Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Feishu.AppID != "legacy-id" {
		t.Errorf("Feishu.AppID: got %q want %q", cfg.Feishu.AppID, "legacy-id")
	}
	if cfg.Feishu.AppSecret != "legacy-secret" {
		t.Errorf("Feishu.AppSecret: got %q want %q", cfg.Feishu.AppSecret, "legacy-secret")
	}
}

func TestApplyEnvDefaults_WhatsAppEnabled(t *testing.T) {
	t.Setenv("CLAWFIRM_WHATSAPP_ENABLED", "true")

	cfg, err := config.Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.WhatsApp.Enabled {
		t.Error("WhatsApp.Enabled: want true")
	}
}

func TestApplyEnvDefaults_MediaAndImageGen(t *testing.T) {
	t.Setenv("CLAWFIRM_MEDIA_PROVIDER", "gemini")
	t.Setenv("CLAWFIRM_MEDIA_MODEL", "gemini-2.0")
	t.Setenv("CLAWFIRM_IMAGE_GEN_PROVIDER", "openai")
	t.Setenv("CLAWFIRM_IMAGE_GEN_MODEL", "dall-e-3")

	cfg, err := config.Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Media.Provider != "gemini" {
		t.Errorf("Media.Provider: got %q", cfg.Media.Provider)
	}
	if cfg.Media.Model != "gemini-2.0" {
		t.Errorf("Media.Model: got %q", cfg.Media.Model)
	}
	if cfg.ImageGen.Provider != "openai" {
		t.Errorf("ImageGen.Provider: got %q", cfg.ImageGen.Provider)
	}
	if cfg.ImageGen.Model != "dall-e-3" {
		t.Errorf("ImageGen.Model: got %q", cfg.ImageGen.Model)
	}
}

func TestApplyEnvDefaults_DiscoverProviders(t *testing.T) {
	t.Setenv("CLAWFIRM_PROVIDERS_MYCLOUD_API_KEY", "sk-mycloud")
	t.Setenv("CLAWFIRM_PROVIDERS_MYCLOUD_TYPE", "openai")
	t.Setenv("CLAWFIRM_PROVIDERS_MYCLOUD_BASE_URL", "https://mycloud.example.com/v1")

	cfg, err := config.Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ProviderAPIKey("mycloud") != "sk-mycloud" {
		t.Errorf("mycloud api_key: got %q", cfg.ProviderAPIKey("mycloud"))
	}
	if cfg.ProviderType("mycloud") != "openai" {
		t.Errorf("mycloud type: got %q", cfg.ProviderType("mycloud"))
	}
	if cfg.ProviderBaseURL("mycloud") != "https://mycloud.example.com/v1" {
		t.Errorf("mycloud base_url: got %q", cfg.ProviderBaseURL("mycloud"))
	}
}

func TestApplyEnvDefaults_DiscoverSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	yaml := `providers:
  existing:
    api_key: yaml-key
    type: anthropic
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAWFIRM_PROVIDERS_EXISTING_API_KEY", "env-key")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ProviderAPIKey("existing") != "yaml-key" {
		t.Errorf("existing api_key: got %q want %q (YAML should win)", cfg.ProviderAPIKey("existing"), "yaml-key")
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
