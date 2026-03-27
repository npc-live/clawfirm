// Package config loads and validates pi-go YAML configuration.
//
// Default config path: ~/.pi-go/config.yml
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
	// Supported values: "read", "write", "edit", "bash".
	// Use ["read","write","edit","bash"] to enable all coding tools.
	Tools []string `yaml:"tools" json:"tools"`

	// SkillPaths lists skill directories or SKILL.md files to load for this agent.
	// Absolute paths, ~/... paths, and paths relative to the config file directory
	// are all supported. Each entry follows the Agent Skills spec (agentskills.io).
	// Example: ["~/.pi-go/skills", "/projects/myapp/.agents/skills"]
	SkillPaths []string `yaml:"skill_paths" json:"skill_paths"`

	// WorkspaceDir is the root directory used for file operations and bootstrap
	// context loading (AGENTS.md / CLAUDE.md). Defaults to cwd when empty.
	WorkspaceDir string `yaml:"workspace_dir" json:"workspace_dir"`
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

// WhipflowConfig holds WhipFlow-specific settings within the pi-go config.
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

// Config is the top-level pi-go configuration structure.
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

	// WhatsApp configures the WhatsApp channel. Optional.
	WhatsApp WhatsAppConfig `yaml:"whatsapp"`

	// Whipflow configures the WhipFlow workflow engine. Optional.
	Whipflow WhipflowConfig `yaml:"whipflow" json:"whipflow"`

	// CronJobs defines scheduled jobs that trigger agents on a timer.
	CronJobs []CronJobConfig `yaml:"cron_jobs" json:"cron_jobs"`
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
// If path is empty it falls back to ~/.pi-go/config.yml.
// Returns an empty Config (not an error) if the file does not exist.
func Load(path string) (*Config, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("config: home dir: %w", err)
		}
		path = filepath.Join(home, ".pi-go", "config.yml")
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{Providers: make(map[string]ProviderConfig)}, nil
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
	// Fall back to environment variables when not set in config.
	if cfg.Feishu.AppID == "" {
		cfg.Feishu.AppID = os.Getenv("FEISHU_APP_ID")
	}
	if cfg.Feishu.AppSecret == "" {
		cfg.Feishu.AppSecret = os.Getenv("FEISHU_APP_SECRET")
	}

	// Expand env vars in cron jobs.
	for i, cj := range cfg.CronJobs {
		cfg.CronJobs[i].Name = expandEnv(cj.Name)
		cfg.CronJobs[i].AgentName = expandEnv(cj.AgentName)
		cfg.CronJobs[i].Prompt = expandEnv(cj.Prompt)
		cfg.CronJobs[i].Schedule.At = expandEnv(cj.Schedule.At)
		cfg.CronJobs[i].Schedule.Expr = expandEnv(cj.Schedule.Expr)
		cfg.CronJobs[i].Schedule.Tz = expandEnv(cj.Schedule.Tz)
	}

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
