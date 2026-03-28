package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ai-gateway/pi-go/app"
	"github.com/ai-gateway/pi-go/config"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// newTestApp creates an App, points its home dir to a temp dir, calls OnStartup,
// and returns a cleanup function that calls OnShutdown + removes the temp dir.
func newTestApp(t *testing.T, cfg *config.Config) (*app.App, func()) {
	t.Helper()

	dir := t.TempDir()
	// Write config to temp dir so the app loads it.
	if cfg != nil {
		cfgPath := filepath.Join(dir, "config.yml")
		writeCfgFile(t, cfgPath, cfg)
		// Override default config path via env.
		t.Setenv("PI_GO_CONFIG", cfgPath)
	}
	// Point home to temp dir so auth + db also land there.
	t.Setenv("HOME", dir)

	a := app.New()
	ctx, cancel := context.WithCancel(context.Background())
	a.OnStartup(ctx)

	cleanup := func() {
		a.OnShutdown(ctx)
		cancel()
	}
	return a, cleanup
}

func writeCfgFile(t *testing.T, path string, cfg *config.Config) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	// Minimal YAML: only providers and agents matter for these tests.
	data := buildYAML(cfg)
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

// buildYAML produces a minimal YAML for config.Config.
// We keep it simple rather than importing yaml in tests.
func buildYAML(cfg *config.Config) string {
	out := "providers:\n"
	for id, pc := range cfg.Providers {
		out += "  " + id + ":\n"
		if pc.Type != "" {
			out += "    type: " + pc.Type + "\n"
		}
		if pc.APIKey != "" {
			out += "    api_key: " + pc.APIKey + "\n"
		}
		if pc.BaseURL != "" {
			out += "    base_url: " + pc.BaseURL + "\n"
		}
	}
	if len(cfg.Agents) > 0 {
		out += "agents:\n"
		for _, a := range cfg.Agents {
			out += "  - name: " + a.Name + "\n"
			out += "    provider: " + a.Provider + "\n"
			out += "    model: " + a.Model + "\n"
			if a.SystemPrompt != "" {
				out += "    system_prompt: \"" + a.SystemPrompt + "\"\n"
			}
		}
	}
	return out
}

// ── IsFirstRun ────────────────────────────────────────────────────────────────

func TestIsFirstRun_NoConfig(t *testing.T) {
	a, cleanup := newTestApp(t, nil)
	defer cleanup()

	if !a.IsFirstRun() {
		t.Error("expected IsFirstRun=true when no config file exists")
	}
}

func TestIsFirstRun_WithProviders(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"ollama": {Type: "ollama"},
		},
	}
	a, cleanup := newTestApp(t, cfg)
	defer cleanup()

	if a.IsFirstRun() {
		t.Error("expected IsFirstRun=false when providers are configured")
	}
}

// ── GetVersion ────────────────────────────────────────────────────────────────

func TestGetVersion(t *testing.T) {
	a, cleanup := newTestApp(t, nil)
	defer cleanup()

	v := a.GetVersion()
	if v == "" {
		t.Error("GetVersion should not be empty")
	}
}

// ── GetWebhookBaseURL (no gateway running) ────────────────────────────────────

func TestGetWebhookBaseURL_NoAgents(t *testing.T) {
	a, cleanup := newTestApp(t, nil)
	defer cleanup()

	// No agents → gateway not started → empty URL.
	if got := a.GetWebhookBaseURL(); got != "" {
		t.Errorf("expected empty URL without agents, got %q", got)
	}
}

// ── GetWebhookBaseURL (with ollama agent — no network needed) ─────────────────

func TestGetWebhookBaseURL_WithAgent(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"ollama": {Type: "ollama", BaseURL: "http://127.0.0.1:11434"},
		},
		Agents: []config.AgentConfig{
			{Name: "test", Provider: "ollama", Model: "llama3.2"},
		},
	}
	a, cleanup := newTestApp(t, cfg)
	defer cleanup()

	// Give the goroutine a moment to bind the port.
	time.Sleep(50 * time.Millisecond)

	url := a.GetWebhookBaseURL()
	if url == "" {
		t.Fatal("expected non-empty webhook URL when agent is configured")
	}
	if url[:7] != "http://" {
		t.Errorf("URL should start with http://, got %q", url)
	}
}

// ── GetProviders ──────────────────────────────────────────────────────────────

func TestGetProviders(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"anthropic": {Type: "anthropic", APIKey: "sk-test"},
			"ollama":    {Type: "ollama"},
		},
	}
	a, cleanup := newTestApp(t, cfg)
	defer cleanup()

	providers := a.GetProviders()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}

	byID := make(map[string]app.ProviderInfo)
	for _, p := range providers {
		byID[p.ID] = p
	}

	if !byID["anthropic"].HasKey {
		t.Error("anthropic should have key")
	}
	if byID["ollama"].HasKey {
		t.Error("ollama should not have key")
	}
}

// ── GetChannels ───────────────────────────────────────────────────────────────

func TestGetChannels_Empty(t *testing.T) {
	a, cleanup := newTestApp(t, nil)
	defer cleanup()

	channels := a.GetChannels()
	if len(channels) != 0 {
		t.Errorf("expected 0 channels, got %d", len(channels))
	}
}

func TestGetChannels_Configured(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"ollama": {Type: "ollama", BaseURL: "http://127.0.0.1:11434"},
		},
		Agents: []config.AgentConfig{
			{Name: "chat", Provider: "ollama", Model: "llama3.2"},
			{Name: "code", Provider: "ollama", Model: "codellama"},
		},
	}
	a, cleanup := newTestApp(t, cfg)
	defer cleanup()

	channels := a.GetChannels()
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}
	names := map[string]bool{}
	for _, ch := range channels {
		names[ch.Name] = true
	}
	if !names["chat"] || !names["code"] {
		t.Errorf("unexpected channels: %v", channels)
	}
}

// ── SaveAPIKey + GetProviders reflects new key ────────────────────────────────

func TestSaveAPIKey(t *testing.T) {
	a, cleanup := newTestApp(t, &config.Config{
		Providers: map[string]config.ProviderConfig{
			"anthropic": {Type: "anthropic"},
		},
	})
	defer cleanup()

	// Before: no key.
	for _, p := range a.GetProviders() {
		if p.ID == "anthropic" && p.HasKey {
			t.Fatal("should not have key before SaveAPIKey")
		}
	}

	if err := a.SaveAPIKey("anthropic", "sk-test-123"); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}

	// After: key present.
	for _, p := range a.GetProviders() {
		if p.ID == "anthropic" {
			if !p.HasKey {
				t.Error("expected hasKey=true after SaveAPIKey")
			}
			return
		}
	}
	t.Error("anthropic provider not found after SaveAPIKey")
}

// ── GetModels ─────────────────────────────────────────────────────────────────

func TestGetModels_Anthropic(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"anthropic": {Type: "anthropic"},
		},
	}
	a, cleanup := newTestApp(t, cfg)
	defer cleanup()

	models := a.GetModels("anthropic")
	if len(models) == 0 {
		t.Error("expected non-empty model list for anthropic")
	}
	// Spot check: at least one claude model.
	found := false
	for _, m := range models {
		if len(m) > 5 && m[:6] == "claude" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected at least one claude model, got %v", models)
	}
}

func TestGetModels_Unknown(t *testing.T) {
	a, cleanup := newTestApp(t, nil)
	defer cleanup()

	if models := a.GetModels("nonexistent"); len(models) != 0 {
		t.Errorf("expected empty list for unknown provider, got %v", models)
	}
}

// ── Session.Abort via App.AbortCurrentTurn ────────────────────────────────────

func TestAbortCurrentTurn_NoGateway(t *testing.T) {
	a, cleanup := newTestApp(t, nil)
	defer cleanup()

	// Should not panic when gateway is not running.
	a.AbortCurrentTurn("nonexistent", "session1")
}

// ── GetHistory empty ─────────────────────────────────────────────────────────

func TestGetHistory_Empty(t *testing.T) {
	a, cleanup := newTestApp(t, nil)
	defer cleanup()

	msgs, err := a.GetHistory("webchat", "user1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected empty history, got %d messages", len(msgs))
	}
}
