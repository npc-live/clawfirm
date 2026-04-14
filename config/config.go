// Package config loads and validates clawfirm YAML configuration.
//
// Default config path: ~/.clawfirm/config.yml
// Environment variable expansion: ${VAR} or $VAR in any string value.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// setIfEmpty sets *dst to the first non-empty env var value found,
// only when *dst is already empty. envKeys are tried in order.
func setIfEmpty(dst *string, envKeys ...string) {
	if *dst != "" {
		return
	}
	for _, k := range envKeys {
		if v := os.Getenv(k); v != "" {
			*dst = v
			return
		}
	}
}

// setBoolIfUnset sets *dst from the env var when *dst is false.
// Truthy values: "1", "true", "yes" (case-insensitive).
func setBoolIfUnset(dst *bool, envKey string) {
	if *dst {
		return
	}
	switch strings.ToLower(os.Getenv(envKey)) {
	case "1", "true", "yes":
		*dst = true
	}
}

// discoverProvidersFromEnv scans environment for CLAWFIRM_PROVIDERS_<ID>_API_KEY
// and registers any provider not already declared in cfg.Providers.
func discoverProvidersFromEnv(cfg *Config) {
	const prefix = "CLAWFIRM_PROVIDERS_"
	const keySuffix = "_API_KEY"
	for _, env := range os.Environ() {
		k, v, ok := strings.Cut(env, "=")
		if !ok || v == "" {
			continue
		}
		if !strings.HasPrefix(k, prefix) || !strings.HasSuffix(k, keySuffix) {
			continue
		}
		// Extract provider ID: CLAWFIRM_PROVIDERS_<ID>_API_KEY → <id> (lowercase)
		id := strings.ToLower(k[len(prefix) : len(k)-len(keySuffix)])
		if id == "" {
			continue
		}
		if _, exists := cfg.Providers[id]; exists {
			continue
		}
		envPrefix := prefix + strings.ToUpper(id)
		cfg.Providers[id] = ProviderConfig{
			Type:    os.Getenv(envPrefix + "_TYPE"),
			APIKey:  v,
			BaseURL: os.Getenv(envPrefix + "_BASE_URL"),
		}
	}
}

// applyEnvDefaults fills zero-value Config fields from environment variables.
// Called after YAML parsing so YAML values always take priority.
func applyEnvDefaults(cfg *Config) {
	// Simple top-level fields
	setIfEmpty(&cfg.DefaultProvider, "CLAWFIRM_DEFAULT_PROVIDER")
	setIfEmpty(&cfg.DefaultModel, "CLAWFIRM_DEFAULT_MODEL")
	setIfEmpty(&cfg.DefaultAgent, "CLAWFIRM_DEFAULT_AGENT")

	// Telegram (CLAWFIRM_ prefix first, legacy fallback second)
	setIfEmpty(&cfg.Telegram.BotToken, "CLAWFIRM_TELEGRAM_BOT_TOKEN", "TELEGRAM_BOT_TOKEN")

	// Feishu (CLAWFIRM_ prefix first, legacy fallback second)
	setIfEmpty(&cfg.Feishu.AppID, "CLAWFIRM_FEISHU_APP_ID", "FEISHU_APP_ID")
	setIfEmpty(&cfg.Feishu.AppSecret, "CLAWFIRM_FEISHU_APP_SECRET", "FEISHU_APP_SECRET")

	// WhatsApp
	setBoolIfUnset(&cfg.WhatsApp.Enabled, "CLAWFIRM_WHATSAPP_ENABLED")

	// Media
	setIfEmpty(&cfg.Media.Provider, "CLAWFIRM_MEDIA_PROVIDER")
	setIfEmpty(&cfg.Media.Model, "CLAWFIRM_MEDIA_MODEL")

	// ImageGen
	setIfEmpty(&cfg.ImageGen.Provider, "CLAWFIRM_IMAGE_GEN_PROVIDER")
	setIfEmpty(&cfg.ImageGen.Model, "CLAWFIRM_IMAGE_GEN_MODEL")

	// Dynamic provider discovery
	discoverProvidersFromEnv(cfg)
}

// ProviderConfig holds connection settings for a single LLM provider.
type ProviderConfig struct {
	// Type identifies the API protocol to use.
	// Supported values: "anthropic" (default), "openai", "gemini", "ollama".
	// Use "anthropic" for any Anthropic-compatible proxy (ZenMux, MiniMax, etc.).
	Type string `yaml:"type" json:"type"`

	// APIKey is the credential used to authenticate with the provider.
	// Supports ${ENV_VAR} expansion.
	APIKey string `yaml:"api_key" json:"api_key"`

	// BaseURL overrides the provider's default API endpoint.
	BaseURL string `yaml:"base_url" json:"base_url"`
}

// AgentConfig defines a named agent with its own provider, model, and persona.
type AgentConfig struct {
	// Name is the unique identifier used in WebSocket URLs: /ws/{name}/{sessionID}.
	Name string `yaml:"name" json:"name"`

	// Provider references a key in the top-level providers map.
	Provider string `yaml:"provider" json:"provider"`

	// Model is the model ID to use with this provider.
	Model string `yaml:"model" json:"model"`

	// SystemPrompt is the agent's persona / instruction.
	SystemPrompt string `yaml:"system_prompt" json:"system_prompt"`

	// MaxTokens overrides the provider default (0 = use provider default).
	MaxTokens int `yaml:"max_tokens" json:"max_tokens"`

	// Tools lists the built-in tool names to enable for this agent.
	// Supported values: "read", "write", "edit", "bash", "exec", "process",
	//   "grep", "find", "ls", "fetch", "web_search", "browser_shortcut",
	//   "media_understand", "media_gen", "memory_search", "memory_get",
	//   "whipflow_run", "sub_agent", "ask_user", "skill", "get_current_time",
	//   "sessions_list".
	Tools []string `yaml:"tools" json:"tools"`

	// SkillPaths lists skill directories or SKILL.md files to load for this agent.
	// Absolute paths, ~/... paths, and paths relative to the config file directory
	// are all supported. Each entry follows the Agent Skills spec (agentskills.io).
	// Example: ["~/.clawfirm/skills", "/projects/myapp/.agents/skills"]
	SkillPaths []string `yaml:"skill_paths" json:"skill_paths"`

	// WorkspaceDir is the root directory used for file operations and bootstrap
	// context loading (AGENTS.md / CLAUDE.md). Defaults to cwd when empty.
	WorkspaceDir string `yaml:"workspace_dir" json:"workspace_dir"`

	// ResetMode controls when session history is cleared: "never" | "daily" | "idle".
	// Default is "never".
	ResetMode string `yaml:"reset_mode" json:"reset_mode"`

	// ResetAtHour is the UTC hour (0-23) at which daily reset fires (default 0 = midnight UTC).
	ResetAtHour int `yaml:"reset_at_hour" json:"reset_at_hour"`

	// IdleMinutes is the idle threshold for ResetMode "idle" (default 30).
	IdleMinutes int `yaml:"idle_minutes" json:"idle_minutes"`
}

// TelegramConfig holds credentials for the Telegram Bot channel.
type TelegramConfig struct {
	// BotToken is the Telegram Bot API token (format: 123456789:ABCdef...).
	// Supports ${ENV_VAR} expansion. Falls back to TELEGRAM_BOT_TOKEN env var.
	BotToken string `yaml:"bot_token"`
}

// WhatsAppConfig holds settings for the WhatsApp channel.
type WhatsAppConfig struct {
	// Enabled must be explicitly set to true to start the WhatsApp channel.
	Enabled bool `yaml:"enabled"`
}

// FeishuConfig holds credentials for the Feishu (Lark) channel.
type FeishuConfig struct {
	// AppID is the Feishu app ID (format: cli_xxx).
	// Supports ${ENV_VAR} expansion. Falls back to FEISHU_APP_ID env var.
	AppID string `yaml:"app_id"`

	// AppSecret is the Feishu app secret.
	// Supports ${ENV_VAR} expansion. Falls back to FEISHU_APP_SECRET env var.
	AppSecret string `yaml:"app_secret"`
}

// WhipflowCliProvider holds configuration for a CLI-based WhipFlow provider.
type WhipflowCliProvider struct {
	Name         string   `yaml:"name"          json:"name,omitempty"`
	Bin          string   `yaml:"bin"           json:"bin,omitempty"`
	PromptMode   string   `yaml:"prompt_mode"   json:"prompt_mode,omitempty"`
	Args         []string `yaml:"args"          json:"args,omitempty"`
	StdinArgs    []string `yaml:"stdin_args"    json:"stdin_args,omitempty"`
	Timeout      int64    `yaml:"timeout"       json:"timeout,omitempty"`
	OutputFormat string   `yaml:"output_format" json:"output_format,omitempty"`
	RawPrompt    bool     `yaml:"raw_prompt"    json:"raw_prompt,omitempty"`
}

// WhipflowConfig holds WhipFlow-specific settings within the clawfirm config.
type WhipflowConfig struct {
	CliProviders    map[string]WhipflowCliProvider `yaml:"cli_providers"    json:"cli_providers,omitempty"`
	DefaultProvider string                         `yaml:"default_provider" json:"default_provider,omitempty"`
	ToolsDir        string                         `yaml:"tools_dir"        json:"tools_dir,omitempty"`
	Tools           []string                       `yaml:"tools"            json:"tools,omitempty"`
}

// Schedule defines when a cron job fires.
type Schedule struct {
	Kind     string `yaml:"kind" json:"kind"`                          // "at", "every", "cron"
	At       string `yaml:"at,omitempty" json:"at,omitempty"`          // ISO8601 for kind=at
	EveryMs  int64  `yaml:"every_ms,omitempty" json:"everyMs,omitempty"` // ms for kind=every
	AnchorMs int64  `yaml:"anchor_ms,omitempty" json:"anchorMs,omitempty"`
	Expr     string `yaml:"expr,omitempty" json:"expr,omitempty"`      // cron expression for kind=cron
	Tz       string `yaml:"tz,omitempty" json:"tz,omitempty"`          // timezone for kind=cron
}

// CronJobConfig defines a scheduled job in the config file.
type CronJobConfig struct {
	Name      string   `yaml:"name" json:"name"`
	Schedule  Schedule `yaml:"schedule" json:"schedule"`
	AgentName string   `yaml:"agent_name" json:"agent_name"`
	Prompt    string   `yaml:"prompt" json:"prompt"`
	Enabled   bool     `yaml:"enabled" json:"enabled"`
}

// MediaConfig holds settings for the dedicated multimodal media analysis provider.
type MediaConfig struct {
	// Provider references a key in the top-level providers map.
	Provider string `yaml:"provider" json:"provider"`

	// Model is the model ID to use for media analysis.
	Model string `yaml:"model" json:"model"`
}

// ImageGenConfig holds settings for the image generation tool (media_gen).
type ImageGenConfig struct {
	// Provider is the image generation backend: "openai" (default).
	Provider string `yaml:"provider" json:"provider"`

	// Model is the model ID, e.g. "dall-e-3".
	Model string `yaml:"model" json:"model"`
}

// Config is the top-level clawfirm configuration structure.
type Config struct {
	// Providers maps provider IDs to their connection settings.
	Providers map[string]ProviderConfig `yaml:"providers" json:"providers"`

	// Agents is the list of named agents exposed by the gateway.
	Agents []AgentConfig `yaml:"agents" json:"agents"`

	// DefaultAgent is the agent name used for /ws/{sessionID} (no agent prefix).
	DefaultAgent string `yaml:"default_agent" json:"default_agent"`

	// DefaultProvider and DefaultModel are kept for backwards compatibility
	// when no agents section is present.
	DefaultProvider string `yaml:"default_provider"`
	DefaultModel    string `yaml:"default_model"`

	// Feishu configures the Feishu (Lark) channel. Optional.
	Feishu FeishuConfig `yaml:"feishu"`

	// Telegram configures the Telegram Bot channel. Optional.
	Telegram TelegramConfig `yaml:"telegram"`

	// WhatsApp configures the WhatsApp channel. Optional.
	WhatsApp WhatsAppConfig `yaml:"whatsapp"`

	// Whipflow configures the WhipFlow workflow engine. Optional.
	Whipflow WhipflowConfig `yaml:"whipflow" json:"whipflow"`

	// CronJobs defines scheduled jobs that trigger agents on a timer.
	CronJobs []CronJobConfig `yaml:"cron_jobs" json:"cron_jobs"`

	// Media configures the dedicated multimodal provider for media analysis.
	Media MediaConfig `yaml:"media" json:"media"`

	// ImageGen configures the image generation tool (media_gen).
	ImageGen ImageGenConfig `yaml:"image_gen" json:"image_gen"`
}

var envVarRe = regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// expandEnv replaces ${VAR} and $VAR with values from the environment.
func expandEnv(s string) string {
	return envVarRe.ReplaceAllStringFunc(s, func(match string) string {
		sub := envVarRe.FindStringSubmatch(match)
		name := sub[1]
		if name == "" {
			name = sub[2]
		}
		return os.Getenv(name)
	})
}

// expandProviderConfig applies env expansion to all string fields.
func expandProviderConfig(p ProviderConfig) ProviderConfig {
	return ProviderConfig{
		Type:    expandEnv(p.Type),
		APIKey:  expandEnv(p.APIKey),
		BaseURL: expandEnv(p.BaseURL),
	}
}

// expandAgentConfig applies env expansion to all string fields.
func expandAgentConfig(a AgentConfig) AgentConfig {
	expanded := AgentConfig{
		Name:         expandEnv(a.Name),
		Provider:     expandEnv(a.Provider),
		Model:        expandEnv(a.Model),
		SystemPrompt: expandEnv(a.SystemPrompt),
		MaxTokens:    a.MaxTokens,
		Tools:        a.Tools,
		SkillPaths:   make([]string, len(a.SkillPaths)),
		WorkspaceDir: expandEnv(a.WorkspaceDir),
		ResetMode:    a.ResetMode,
		ResetAtHour:  a.ResetAtHour,
		IdleMinutes:  a.IdleMinutes,
	}
	for i, p := range a.SkillPaths {
		expanded.SkillPaths[i] = expandEnv(p)
	}
	return expanded
}

// ParseYAML parses raw YAML bytes into a Config.
func ParseYAML(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}
	return &cfg, nil
}

// Load reads and parses a YAML config file.
// If path is empty it falls back to ~/.clawfirm/config.yml.
// Returns an empty Config (not an error) if the file does not exist.
func Load(path string) (*Config, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("config: home dir: %w", err)
		}
		path = filepath.Join(home, ".clawfirm", "config.yml")
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := &Config{Providers: make(map[string]ProviderConfig)}
		applyEnvDefaults(cfg)
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}

	// Expand env vars in all provider and agent fields.
	for id, p := range cfg.Providers {
		cfg.Providers[id] = expandProviderConfig(p)
	}
	for i, a := range cfg.Agents {
		cfg.Agents[i] = expandAgentConfig(a)
	}
	cfg.DefaultProvider = expandEnv(cfg.DefaultProvider)
	cfg.DefaultModel = expandEnv(cfg.DefaultModel)
	cfg.DefaultAgent = expandEnv(cfg.DefaultAgent)
	cfg.Feishu.AppID = expandEnv(cfg.Feishu.AppID)
	cfg.Feishu.AppSecret = expandEnv(cfg.Feishu.AppSecret)
	cfg.Telegram.BotToken = expandEnv(cfg.Telegram.BotToken)

	// Expand env vars in media config.
	cfg.Media.Provider = expandEnv(cfg.Media.Provider)
	cfg.Media.Model = expandEnv(cfg.Media.Model)

	// Expand env vars in image gen config.
	cfg.ImageGen.Provider = expandEnv(cfg.ImageGen.Provider)
	cfg.ImageGen.Model = expandEnv(cfg.ImageGen.Model)

	// Expand env vars in cron jobs.
	for i, cj := range cfg.CronJobs {
		cfg.CronJobs[i].Name = expandEnv(cj.Name)
		cfg.CronJobs[i].AgentName = expandEnv(cj.AgentName)
		cfg.CronJobs[i].Prompt = expandEnv(cj.Prompt)
		cfg.CronJobs[i].Schedule.At = expandEnv(cj.Schedule.At)
		cfg.CronJobs[i].Schedule.Expr = expandEnv(cj.Schedule.Expr)
		cfg.CronJobs[i].Schedule.Tz = expandEnv(cj.Schedule.Tz)
	}

	// Fill zero-value fields from environment variables (YAML > env).
	applyEnvDefaults(&cfg)

	return &cfg, nil
}

// ProviderType returns the API type for the given provider ID (default: "anthropic").
func (c *Config) ProviderType(providerID string) string {
	if p, ok := c.Providers[providerID]; ok && p.Type != "" {
		return p.Type
	}
	return "anthropic"
}

// Agent returns the AgentConfig with the given name, or false if not found.
func (c *Config) Agent(name string) (AgentConfig, bool) {
	for _, a := range c.Agents {
		if a.Name == name {
			return a, true
		}
	}
	return AgentConfig{}, false
}

// ProviderAPIKey returns the API key for the given provider ID,
// falling back to well-known environment variables when the config has no key.
func (c *Config) ProviderAPIKey(providerID string) string {
	if p, ok := c.Providers[providerID]; ok && p.APIKey != "" {
		return p.APIKey
	}
	// Fall back to conventional env vars.
	return os.Getenv(defaultEnvVar(providerID))
}

// ProviderBaseURL returns the configured base URL for the given provider ID,
// or empty string if not set (caller should use the provider's own default).
func (c *Config) ProviderBaseURL(providerID string) string {
	if p, ok := c.Providers[providerID]; ok {
		return p.BaseURL
	}
	return ""
}

// defaultEnvVar maps provider IDs to canonical environment variable names.
func defaultEnvVar(providerID string) string {
	m := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
		"gemini":    "GEMINI_API_KEY",
		"google":    "GOOGLE_API_KEY",
		"zenmux":    "ZENMUX_API_KEY",
		"ollama":    "", // no key needed
	}
	if v, ok := m[providerID]; ok {
		return v
	}
	// Generic fallback: UPPER_PROVIDER_API_KEY
	return strings.ToUpper(providerID) + "_API_KEY"
}
