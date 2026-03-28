// Package app contains the Wails App struct and all bindings exposed to the frontend.
package app

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/auth"
	"github.com/ai-gateway/pi-go/channel/feishu"
	"github.com/ai-gateway/pi-go/channel/webchat"
	"github.com/ai-gateway/pi-go/channel/whatsapp"
	"github.com/ai-gateway/pi-go/config"
	picron "github.com/ai-gateway/pi-go/cron"
	"github.com/ai-gateway/pi-go/gateway"
	"github.com/ai-gateway/pi-go/internal/agentbuilder"
	"github.com/ai-gateway/pi-go/memory"
	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/skill"
	"github.com/ai-gateway/pi-go/store"
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version is set at build time via ldflags.
var Version = "dev"

// ProviderInfo is returned to the frontend.
type ProviderInfo struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	HasKey bool   `json:"hasKey"`
}

// ChannelInfo represents a configured agent for the dashboard.
type ChannelInfo struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Sessions int    `json:"sessions"`
}

// SessionInfo is a minimal session descriptor.
type SessionInfo struct {
	AgentName string `json:"agentName"`
	ChannelID string `json:"channelId"`
	UserID    string `json:"userId"`
}

// App is the Wails application struct. All exported methods become RPC calls available to the frontend.
type App struct {
	ctx      context.Context
	cancelFn context.CancelFunc

	mu             sync.RWMutex
	cfg            *config.Config
	db             *store.DB
	authStor       *auth.AuthStorage
	registry       *gateway.AgentRegistry
	srvAddr        string // "127.0.0.1:PORT" once gateway is running
	whatsappCh     *whatsapp.Channel
	whatsappCancel context.CancelFunc
	feishuCh       *feishu.Channel
	feishuCancel   context.CancelFunc
	cronScheduler  *picron.Scheduler
	memoryMgr      *memory.Manager
}

// New creates the App. Call wails.Run with a.OnStartup / a.OnShutdown.
func New() *App {
	return &App{}
}

// applySystemProxy reads macOS system proxy settings via scutil --proxy and
// sets HTTPS_PROXY / HTTP_PROXY env vars so net/http.ProxyFromEnvironment picks
// them up even when the app is launched from Finder/Dock (no shell env).
func applySystemProxy() {
	if runtime.GOOS != "darwin" {
		return
	}
	// Already set — nothing to do.
	if os.Getenv("HTTPS_PROXY") != "" || os.Getenv("HTTP_PROXY") != "" {
		return
	}
	out, err := exec.Command("scutil", "--proxy").Output()
	if err != nil {
		return
	}
	vals := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, " : ", 2)
		if len(parts) == 2 {
			vals[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	if vals["HTTPSEnable"] == "1" && vals["HTTPSProxy"] != "" {
		proxy := "http://" + vals["HTTPSProxy"] + ":" + vals["HTTPSPort"]
		_ = os.Setenv("HTTPS_PROXY", proxy)
		_ = os.Setenv("https_proxy", proxy)
		log.Printf("app: system HTTPS proxy applied: %s", proxy)
	}
	if vals["HTTPEnable"] == "1" && vals["HTTPProxy"] != "" {
		proxy := "http://" + vals["HTTPProxy"] + ":" + vals["HTTPPort"]
		_ = os.Setenv("HTTP_PROXY", proxy)
		_ = os.Setenv("http_proxy", proxy)
		log.Printf("app: system HTTP proxy applied: %s", proxy)
	}
	if vals["SOCKSEnable"] == "1" && vals["SOCKSProxy"] != "" {
		proxy := "socks5://" + vals["SOCKSProxy"] + ":" + vals["SOCKSPort"]
		_ = os.Setenv("ALL_PROXY", proxy)
		_ = os.Setenv("all_proxy", proxy)
		log.Printf("app: system SOCKS proxy applied: %s", proxy)
	}
}

// initUserDirs creates ~/.pi-go and its subdirectories on first run,
// and writes a minimal default config.yml if one does not exist yet.
// It also extracts bundled binaries (e.g. func) to ~/.pi-go/bin/.
func initUserDirs() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	base := filepath.Join(home, ".pi-go")
	dirs := []string{
		base,
		filepath.Join(base, "skills"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "workflows"),
		filepath.Join(base, "canvas"),
		filepath.Join(base, "bin"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Printf("app: mkdir %s: %v", d, err)
		}
	}

	// Extract bundled `func` binary — only if it's a real binary (> 4 KB).
	if len(embeddedFunc) > 4096 {
		funcPath := filepath.Join(base, "bin", "func")
		if err := os.WriteFile(funcPath, embeddedFunc, 0o755); err != nil {
			log.Printf("app: write func binary: %v", err)
		} else {
			log.Printf("app: extracted func binary to %s", funcPath)
		}
	}

	cfgPath := filepath.Join(base, "config.yml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		defaultCfg := `# pi-go configuration
# Docs: https://github.com/ai-gateway/pi-go
providers:
    # anthropic:
    #     type: anthropic
    #     api_key: ${ANTHROPIC_API_KEY}
    # openai:
    #     type: openai
    #     api_key: ${OPENAI_API_KEY}

agents:
    - name: default
      provider: anthropic
      model: claude-sonnet-4-6
      system_prompt: ""
      tools:
        - read
        - write
        - edit
        - bash
        - memory_search
        - memory_get
        - whipflow_run
      skill_paths:
        - ~/.pi-go/skills/

default_agent: ""

feishu:
    app_id: ""
    app_secret: ""

whatsapp:
    enabled: false

whipflow:
    default_provider: ""

cron_jobs: []
`
		if err := os.WriteFile(cfgPath, []byte(defaultCfg), 0o644); err != nil {
			log.Printf("app: write default config: %v", err)
		} else {
			log.Printf("app: created default config at %s", cfgPath)
		}
	}
}

// OnStartup is called by Wails once the frontend webview is ready.
func (a *App) OnStartup(ctx context.Context) {
	initUserDirs()
	applySystemProxy()
	a.ctx, a.cancelFn = context.WithCancel(ctx)

	// Load config (tolerates missing file).
	// PI_GO_CONFIG env var overrides the default path (used in tests).
	cfg, err := config.Load(os.Getenv("PI_GO_CONFIG"))
	if err != nil {
		log.Printf("app: config load: %v", err)
		cfg = &config.Config{Providers: make(map[string]config.ProviderConfig)}
	}
	a.cfg = cfg

	// Open auth storage.
	a.authStor = auth.NewAuthStorage("")
	if err := a.authStor.Load(); err != nil {
		log.Printf("app: auth load: %v", err)
	}

	// Open SQLite store.
	db, err := store.Open("")
	if err != nil {
		log.Printf("app: store open: %v", err)
	} else {
		a.db = db
	}

	// Initialise memory manager before gateway so tools have a valid manager.
	if a.db != nil {
		embedProvider, _ := memory.NewAutoProvider()
		a.memoryMgr = memory.New(a.db.SQL(), embedProvider, memory.Config{})
		go func() {
			if err := a.memoryMgr.Sync(a.ctx); err != nil {
				log.Printf("app: memory sync: %v", err)
			}
		}()
	}

	// Start gateway only if agents are configured.
	if len(cfg.Agents) > 0 {
		if err := a.startGateway(); err != nil {
			log.Printf("app: gateway start: %v", err)
		}
	}

	// Start cron scheduler if DB is available.
	if a.db != nil {
		a.syncCronConfigToDB()
		sched := picron.New(a.db.CronJobs(), a.buildAgentForCron)
		if err := sched.Start(a.ctx); err != nil {
			log.Printf("app: cron scheduler start: %v", err)
		}
		a.cronScheduler = sched
	}
}

// OnDomReady is called after the DOM is ready.
func (a *App) OnDomReady(_ context.Context) {}

// OnShutdown is called when the application is shutting down.
func (a *App) OnShutdown(_ context.Context) {
	a.mu.Lock()
	if a.whatsappCancel != nil {
		a.whatsappCancel()
		a.whatsappCancel = nil
	}
	if a.feishuCancel != nil {
		a.feishuCancel()
		a.feishuCancel = nil
	}
	a.mu.Unlock()
	if a.cancelFn != nil {
		a.cancelFn()
	}
	a.mu.Lock()
	if a.cronScheduler != nil {
		a.cronScheduler.Stop()
	}
	if a.registry != nil {
		a.registry.Stop()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
	a.mu.Unlock()
}

// startGateway builds providers and agents, then starts the HTTP server.
// Must be called while holding no locks (it acquires a.mu.Lock briefly).
func (a *App) startGateway() error {
	a.mu.Lock()
	cfg := a.cfg
	db := a.db
	memMgr := a.memoryMgr
	a.mu.Unlock()

	providerMap, err := buildProviders(cfg)
	if err != nil {
		return fmt.Errorf("startGateway: providers: %w", err)
	}

	registry := gateway.NewAgentRegistry()
	var msgStore *store.MessageStore
	if db != nil {
		msgStore = db.Messages()
	}

	for _, ac := range cfg.Agents {
		prov, ok := providerMap[ac.Provider]
		if !ok {
			return fmt.Errorf("startGateway: agent %q: provider %q not found", ac.Name, ac.Provider)
		}
		maxTokens := ac.MaxTokens
		if maxTokens == 0 {
			maxTokens = 4096
		}
		model := types.Model{ID: ac.Model, Provider: ac.Provider, MaxTokens: maxTokens}
		agentName := ac.Name
		tools := buildTools(ac.Tools, memMgr, cfg, db)

		// Load skills and build the skills prompt with size limits.
		skillResult := skill.Load(skill.LoadOptions{SkillPaths: ac.SkillPaths})
		for _, d := range skillResult.Diagnostics {
			log.Printf("app: agent %s: skill warning: %s: %s", agentName, d.Path, d.Message)
		}
		if len(skillResult.Skills) > 0 {
			names := make([]string, len(skillResult.Skills))
			for i, s := range skillResult.Skills {
				names[i] = s.Name
			}
			log.Printf("app: agent %s: loaded %d skill(s): %s", agentName, len(skillResult.Skills), strings.Join(names, ", "))
		}
		compacted := skill.CompactSkillPaths(skillResult.Skills)
		skillsPrompt, truncated, compact := skill.ApplySkillsPromptLimits(compacted)
		if truncated {
			log.Printf("app: agent %s: skills prompt truncated (compact=%v)", agentName, compact)
		} else if compact {
			log.Printf("app: agent %s: skills prompt using compact format", agentName)
		}

		// Load bootstrap context files (AGENTS.md / CLAUDE.md / SOUL.md).
		bootstrap := agent.LoadBootstrapContext(ac.WorkspaceDir)

		// Build full system prompt.
		systemPrompt := agent.BuildSystemPrompt(agent.SystemPromptParams{
			WorkspaceDir:   ac.WorkspaceDir,
			SkillsPrompt:   skillsPrompt,
			ContextFiles:   bootstrap.ContextFiles,
			WorkspaceNotes: bootstrap.WorkspaceNotes,
			PromptMode:     agent.PromptModeFull,
			RuntimeInfo:    buildRuntimeInfo(ac),
			ExtraPrompt:    ac.SystemPrompt,
		})

		temporal := agent.NewTemporalInjector(0)
		loopCfg := agent.AgentLoopConfig{
			TransformContext: temporal.TransformContext,
		}

		factory := gateway.AgentFactory(func(channelID, userID string) *agent.Agent {
			opts := []agent.AgentOption{
				agent.WithModel(model),
				agent.WithSystemPrompt(systemPrompt),
				agent.WithLoopConfig(loopCfg),
			}
			if len(tools) > 0 {
				opts = append(opts, agent.WithTools(tools))
			}
			a := agent.NewAgent(prov, opts...)
			savedCount := 0
			// Use "webchat/<agentName>" as channel_id so each agent's history
			// is stored separately and can be filtered by agent name.
			storeChannelID := "webchat/" + agentName
			if msgStore != nil {
				if history, err := msgStore.ListMessages(store.QueryParams{
					ChannelID: storeChannelID,
					UserID:    userID,
				}); err == nil && len(history) > 0 {
					a.ReplaceMessages(history)
					savedCount = len(history)
					log.Printf("[%s] session %s/%s: restored %d messages", agentName, storeChannelID, userID, savedCount)
				}
			}
			a.Subscribe(func(ev types.AgentEvent) {
				if ev.Type != types.EventAgentEnd || msgStore == nil {
					return
				}
				msgs := ev.Messages
				for i := savedCount; i < len(msgs); i++ {
					if err := msgStore.SaveMessage(storeChannelID, userID, msgs[i]); err != nil {
						log.Printf("[%s] save message[%d]: %v", agentName, i, err)
					}
				}
				savedCount = len(msgs)
			})
			return a
		})

		mgr := gateway.NewSessionManager(factory, gateway.ManagerConfig{})
		registry.Register(ac.Name, mgr)
		log.Printf("app: agent %s  provider: %s  model: %s", ac.Name, ac.Provider, ac.Model)
	}

	// Random loopback port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("startGateway: listen: %w", err)
	}
	addr := ln.Addr().String()
	ln.Close() // gateway.Server will re-listen on this addr
	// Note: there is a brief TOCTOU window here; acceptable for desktop use.

	srv := gateway.NewServer(registry, gateway.ServerConfig{Addr: addr})

	defaultAgent := cfg.DefaultAgent
	if defaultAgent == "" && len(cfg.Agents) > 0 {
		defaultAgent = cfg.Agents[0].Name
	}
	handler := webchat.NewHandler(registry, defaultAgent)
	srv.Handle("GET /ws/{agentName}/{sessionID}", handler)
	srv.Handle("GET /ws/{sessionID}", handler)

	a.mu.Lock()
	a.registry = registry
	a.srvAddr = addr
	a.mu.Unlock()

	go func() {
		if err := srv.Start(a.ctx); err != nil {
			log.Printf("app: gateway stopped: %v", err)
		}
	}()

	// Start WhatsApp channel only when explicitly enabled in config.
	if defaultAgent != "" && cfg.WhatsApp.Enabled {
		waCh := whatsapp.New(registry, defaultAgent)
		waCtx, waCancel := context.WithCancel(a.ctx)
		a.mu.Lock()
		a.whatsappCh = waCh
		a.whatsappCancel = waCancel
		a.mu.Unlock()
		go func() {
			if err := waCh.Start(waCtx); err != nil {
				log.Printf("app: whatsapp: %v", err)
			}
		}()
	}

	// Start Feishu channel if credentials are configured.
	if defaultAgent != "" && cfg.Feishu.AppID != "" && cfg.Feishu.AppSecret != "" {
		fsCh := feishu.New(cfg.Feishu.AppID, cfg.Feishu.AppSecret, registry, defaultAgent)
		fsCtx, fsCancel := context.WithCancel(a.ctx)
		a.mu.Lock()
		a.feishuCh = fsCh
		a.feishuCancel = fsCancel
		a.mu.Unlock()
		go func() {
			if err := fsCh.Start(fsCtx); err != nil {
				log.Printf("app: feishu: %v", err)
			}
		}()
	}

	log.Printf("app: gateway started on %s", addr)
	return nil
}

// stopGateway stops the running gateway, registry, and all channels (no-op if not running).
func (a *App) stopGateway() {
	a.mu.Lock()
	reg := a.registry
	a.registry = nil
	a.srvAddr = ""
	waCancel := a.whatsappCancel
	a.whatsappCh = nil
	a.whatsappCancel = nil
	fsCancel := a.feishuCancel
	a.feishuCh = nil
	a.feishuCancel = nil
	a.mu.Unlock()
	if waCancel != nil {
		waCancel()
	}
	if fsCancel != nil {
		fsCancel()
	}
	if reg != nil {
		reg.Stop()
	}
}

// restartGateway stops any running gateway and starts a fresh one.
func (a *App) restartGateway() error {
	a.stopGateway()
	if len(a.cfg.Agents) == 0 {
		return nil
	}
	err := a.startGateway()
	// Reload cron scheduler after gateway restart (agent config may have changed).
	if a.cronScheduler != nil {
		if rerr := a.cronScheduler.Reload(); rerr != nil {
			log.Printf("app: cron reload: %v", rerr)
		}
	}
	return err
}

// ─── Frontend API ─────────────────────────────────────────────────────────────

// IsFirstRun returns true when no providers or agents are configured.
func (a *App) IsFirstRun() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.cfg.Providers) == 0 && len(a.cfg.Agents) == 0
}

// GetConfig returns the current configuration.
func (a *App) GetConfig() *config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

// ─── Vault ────────────────────────────────────────────────────────────────────

// GetVault returns all vault entries (keys + values).
func (a *App) GetVault() ([]store.VaultEntry, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil, nil
	}
	return db.Vault().List()
}

// SetVaultEntry upserts a vault key-value pair.
func (a *App) SetVaultEntry(key, value string) error {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return fmt.Errorf("vault: store not ready")
	}
	return db.Vault().Set(key, value)
}

// DeleteVaultEntry removes a vault entry by key.
func (a *App) DeleteVaultEntry(key string) error {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return fmt.Errorf("vault: store not ready")
	}
	return db.Vault().Delete(key)
}

// ReadCanvasFile reads ~/.pi-go/canvas/{name}.html and returns its content.
// Returns empty string if the file does not exist yet.
func (a *App) ReadCanvasFile(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "/\\..") {
		return "", fmt.Errorf("invalid canvas name: %q", name)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".pi-go", "canvas", name+".html")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteCanvasFile writes content to ~/.pi-go/canvas/{name}.html.
func (a *App) WriteCanvasFile(name, content string) error {
	if name == "" || strings.ContainsAny(name, "/\\..") {
		return fmt.Errorf("invalid canvas name: %q", name)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".pi-go", "canvas")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name+".html"), []byte(content), 0o644)
}

// SaveConfig writes the config to ~/.pi-go/config.yml and restarts the gateway.
func (a *App) SaveConfig(cfg *config.Config) error {
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	return a.restartGateway()
}

// GetProviders returns all configured providers with a hasKey flag.
func (a *App) GetProviders() []ProviderInfo {
	a.mu.RLock()
	cfg := a.cfg
	authStor := a.authStor
	a.mu.RUnlock()

	out := make([]ProviderInfo, 0, len(cfg.Providers))
	for id, pc := range cfg.Providers {
		hasKey := pc.APIKey != ""
		if !hasKey && authStor != nil {
			_, hasKey = authStor.GetAPIKey(id)
		}
		out = append(out, ProviderInfo{ID: id, Type: pc.Type, HasKey: hasKey})
	}
	return out
}

// SaveAPIKey stores an API key for the given provider and updates config.
func (a *App) SaveAPIKey(providerID, key string) error {
	a.mu.RLock()
	authStor := a.authStor
	cfg := a.cfg
	a.mu.RUnlock()

	if authStor != nil {
		if err := authStor.SetAPIKey(providerID, key); err != nil {
			return err
		}
	}

	// Also patch config so the provider exists with the right type if absent.
	if _, ok := cfg.Providers[providerID]; !ok {
		a.mu.Lock()
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]config.ProviderConfig)
		}
		cfg.Providers[providerID] = config.ProviderConfig{Type: providerID, APIKey: key}
		a.mu.Unlock()
		if err := saveConfig(cfg); err != nil {
			return err
		}
	}
	return nil
}

// StartOAuthLogin opens the system browser for OAuth flow and returns immediately.
// The frontend should listen for the "oauth:callback" Wails event.
func (a *App) StartOAuthLogin(providerID string) error {
	// Build the OAuth URL based on provider.
	// Currently a placeholder — each provider would need its own redirect URL.
	oauthURL := fmt.Sprintf("https://auth.%s.example.com/oauth/authorize", providerID)
	return openBrowser(oauthURL)
}

// GetModels returns a static list of known models for a provider type.
// For Ollama it returns an empty list (user should type model name).
func (a *App) GetModels(providerID string) []string {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()

	pc, ok := cfg.Providers[providerID]
	if !ok {
		return nil
	}
	t := pc.Type
	if t == "" {
		t = providerID
	}
	switch t {
	case "anthropic":
		return []string{
			"claude-opus-4-5-20251101",
			"claude-sonnet-4-5-20251001",
			"claude-haiku-4-5-20251001",
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
		}
	case "openai":
		return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "o1-mini", "o3-mini"}
	case "gemini":
		return []string{"gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash", "gemini-2.0-flash-lite"}
	case "zenmux":
		return []string{
			"anthropic/claude-sonnet-4-5", "anthropic/claude-haiku-4-5",
			"openai/gpt-4o", "openai/gpt-4o-mini",
			"google/gemini-2.0-flash",
		}
	case "ollama", "sglang", "vllm", "litellm":
		return []string{} // dynamic; user types model name
	case "deepseek":
		return []string{"deepseek-chat", "deepseek-reasoner"}
	case "minimax":
		return []string{"MiniMax-M2.7", "MiniMax-M2.7-highspeed", "MiniMax-M2.5", "MiniMax-M2.5-highspeed", "MiniMax-M2.1", "MiniMax-M2"}
	case "moonshot":
		return []string{"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"}
	case "volcengine":
		return []string{"doubao-pro-32k", "doubao-pro-4k", "doubao-lite-32k", "doubao-lite-4k"}
	case "modelstudio":
		return []string{"qwen-max", "qwen-plus", "qwen-turbo", "qwen-long", "qwen2.5-72b-instruct"}
	case "glm", "zai":
		return []string{"glm-4-flash", "glm-4-air", "glm-4", "glm-4-plus", "glm-z1-flash"}
	case "groq":
		return []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768", "gemma2-9b-it"}
	case "openrouter":
		return []string{
			"anthropic/claude-3.5-sonnet", "openai/gpt-4o",
			"google/gemini-flash-1.5", "meta-llama/llama-3.1-70b-instruct",
		}
	case "together":
		return []string{
			"meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
			"mistralai/Mixtral-8x7B-Instruct-v0.1",
			"Qwen/Qwen2.5-72B-Instruct-Turbo",
		}
	case "mistral":
		return []string{"mistral-large-latest", "mistral-small-latest", "codestral-latest", "open-mistral-nemo"}
	case "xai":
		return []string{"grok-3", "grok-3-mini", "grok-2", "grok-beta"}
	case "nvidia":
		return []string{"nvidia/llama-3.1-nemotron-70b-instruct", "meta/llama-3.1-405b-instruct", "mistralai/mistral-large-2-instruct"}
	case "xiaomi":
		return []string{"mimo-v2-flash", "mimo-v2-pro", "mimo-v2-omni"}
	case "venice":
		return []string{"llama-3.3-70b", "mistral-31-24b", "qwen-2.5-qwq-32b"}
	case "huggingface":
		return []string{"meta-llama/Llama-3.3-70B-Instruct", "Qwen/Qwen2.5-72B-Instruct", "mistralai/Mistral-7B-Instruct-v0.3"}
	case "perplexity":
		return []string{"sonar-pro", "sonar", "sonar-reasoning-pro", "sonar-reasoning"}
	default:
		return nil
	}
}

// TestProviderConnection sends a minimal request to verify provider credentials.
// Returns true if the connection succeeds.
func (a *App) TestProviderConnection(providerID string) bool {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()

	prov, err := buildProvider(providerID, cfg.Providers[providerID])
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15_000_000_000) // 15s
	defer cancel()
	// Use a dummy agent with zero tools to send a minimal prompt.
	a2 := agent.NewAgent(prov,
		agent.WithModel(types.Model{ID: defaultModelForProvider(providerID), Provider: providerID, MaxTokens: 16}),
	)
	return a2.Prompt(ctx, "ping") == nil
}

// GetChannels returns the list of configured agents with live session counts.
func (a *App) GetChannels() []ChannelInfo {
	a.mu.RLock()
	cfg := a.cfg
	registry := a.registry
	a.mu.RUnlock()

	var counts map[string]int
	if registry != nil {
		counts = registry.Counts()
	}

	out := make([]ChannelInfo, 0, len(cfg.Agents))
	for _, ac := range cfg.Agents {
		ci := ChannelInfo{
			Name:     ac.Name,
			Provider: ac.Provider,
			Model:    ac.Model,
			Sessions: counts[ac.Name],
		}
		out = append(out, ci)
	}
	return out
}

// SaveChannelConfig upserts an agent config and restarts the gateway.
func (a *App) SaveChannelConfig(ac config.AgentConfig) error {
	a.mu.Lock()
	updated := false
	for i, existing := range a.cfg.Agents {
		if existing.Name == ac.Name {
			a.cfg.Agents[i] = ac
			updated = true
			break
		}
	}
	if !updated {
		a.cfg.Agents = append(a.cfg.Agents, ac)
	}
	cfg := a.cfg
	a.mu.Unlock()

	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// DeleteChannelConfig removes an agent by name and restarts the gateway.
func (a *App) DeleteChannelConfig(name string) error {
	a.mu.Lock()
	agents := make([]config.AgentConfig, 0, len(a.cfg.Agents))
	for _, ac := range a.cfg.Agents {
		if ac.Name != name {
			agents = append(agents, ac)
		}
	}
	a.cfg.Agents = agents
	cfg := a.cfg
	a.mu.Unlock()

	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// TestChannelConnection checks if the named agent's session manager is registered.
func (a *App) TestChannelConnection(name string) bool {
	a.mu.RLock()
	registry := a.registry
	a.mu.RUnlock()
	if registry == nil {
		return false
	}
	_, ok := registry.Get(name)
	return ok
}

// SendMessage enqueues a user message to the named agent/session.
func (a *App) SendMessage(agentName, sessionID, content string) error {
	a.mu.RLock()
	registry := a.registry
	a.mu.RUnlock()
	if registry == nil {
		return fmt.Errorf("gateway not running")
	}
	mgr, ok := registry.Get(agentName)
	if !ok {
		return fmt.Errorf("agent %q not found", agentName)
	}
	sess, err := mgr.GetOrCreate("desktop", sessionID)
	if err != nil {
		return err
	}
	if !sess.Send(gateway.IncomingMessage{ChannelID: "desktop", UserID: sessionID, Content: content}) {
		return fmt.Errorf("session queue full")
	}
	return nil
}

// AbortCurrentTurn cancels the in-progress agent turn for the given session.
func (a *App) AbortCurrentTurn(agentName, sessionID string) {
	a.mu.RLock()
	registry := a.registry
	a.mu.RUnlock()
	if registry == nil {
		return
	}
	mgr, ok := registry.Get(agentName)
	if !ok {
		return
	}
	sess, err := mgr.GetOrCreate("desktop", sessionID)
	if err != nil {
		return
	}
	sess.Abort()
}

// ToolExecutionInfo is a summary of one tool call + result for the frontend.
type ToolExecutionInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Args      any    `json:"args,omitempty"`
	Result    string `json:"result,omitempty"`
	IsError   bool   `json:"isError"`
	Timestamp int64  `json:"timestamp"`
}

// GetToolExecutions returns tool call summaries extracted from stored messages.
func (a *App) GetToolExecutions(channelID, userID string) ([]ToolExecutionInfo, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil, nil
	}
	msgs, err := db.Messages().ListMessages(store.QueryParams{
		ChannelID: channelID,
		UserID:    userID,
		Limit:     200,
	})
	if err != nil {
		return nil, err
	}

	// Build id → ToolExecutionInfo from AssistantMessage tool calls.
	byID := make(map[string]*ToolExecutionInfo)
	var order []string
	for _, m := range msgs {
		switch msg := m.(type) {
		case *types.AssistantMessage:
			for _, b := range msg.Content {
				tc, ok := b.(*types.ToolCall)
				if !ok {
					continue
				}
				info := &ToolExecutionInfo{
					ID:        tc.ID,
					Name:      tc.Name,
					Args:      tc.Arguments,
					Timestamp: msg.Timestamp,
				}
				byID[tc.ID] = info
				order = append(order, tc.ID)
			}
		case *types.ToolResultMessage:
			if info, ok := byID[msg.ToolCallID]; ok {
				info.IsError = msg.IsError
				for _, b := range msg.Content {
					if t, ok := b.(*types.TextContent); ok {
						info.Result = t.Text
						break
					}
				}
			}
		}
	}

	out := make([]ToolExecutionInfo, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}

// GetHistory returns the last N messages for a channel+user.
func (a *App) GetHistory(channelID, userID string) ([]map[string]string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil, nil
	}
	msgs, err := db.Messages().ListMessages(store.QueryParams{
		ChannelID: channelID,
		UserID:    userID,
		Limit:     50,
	})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]string, 0, len(msgs))
	for _, m := range msgs {
		role := "user"
		if _, ok := m.(*types.AssistantMessage); ok {
			role = "assistant"
		}
		// Flatten to role+text for simple frontend rendering.
		// Skip tool_use / tool_result messages (empty text).
		text := extractText(m)
		if text == "" {
			continue
		}
		out = append(out, map[string]string{"role": role, "content": text})
	}
	return out, nil
}

// GetSessions returns a list of active session keys from all registered agents.
func (a *App) GetSessions() []SessionInfo {
	a.mu.RLock()
	registry := a.registry
	cfg := a.cfg
	a.mu.RUnlock()
	if registry == nil {
		return nil
	}

	var out []SessionInfo
	for _, agentName := range registry.Names() {
		mgr, ok := registry.Get(agentName)
		if !ok {
			continue
		}
		_ = mgr
		_ = cfg
		out = append(out, SessionInfo{AgentName: agentName})
	}
	return out
}

// GetChatSessions returns distinct session IDs (user_ids) that have messages
// stored for a given agent, ordered by most recent activity.
func (a *App) GetChatSessions(agentName string) ([]string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil, nil
	}
	return db.Messages().ListUserIDs("webchat/" + agentName)
}

// GetConfigRaw returns the raw YAML content of config.yml and its file path.
func (a *App) GetConfigRaw() (map[string]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(home, ".pi-go", "config.yml")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return map[string]string{"path": p, "content": string(data)}, nil
}

// SaveConfigRaw writes raw YAML to config.yml and reloads the gateway.
func (a *App) SaveConfigRaw(content string) error {
	cfg, err := config.ParseYAML([]byte(content))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	return a.restartGateway()
}

// GetVersion returns the application version string.
func (a *App) GetVersion() string { return Version }

// OpenLogsFolder opens the ~/.pi-go directory in the system file manager.
func (a *App) OpenLogsFolder() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".pi-go")
	return openBrowser(dir)
}

// GetWebhookBaseURL returns the HTTP base URL of the embedded gateway.
func (a *App) GetWebhookBaseURL() string {
	a.mu.RLock()
	addr := a.srvAddr
	a.mu.RUnlock()
	if addr == "" {
		return ""
	}
	return "http://" + addr
}

// GetWhatsAppStatus returns the WhatsApp channel connection status.
// Returns "disabled" when not enabled in config.
func (a *App) GetWhatsAppStatus() string {
	a.mu.RLock()
	ch := a.whatsappCh
	enabled := a.cfg != nil && a.cfg.WhatsApp.Enabled
	a.mu.RUnlock()
	if !enabled {
		return "disabled"
	}
	if ch == nil {
		return whatsapp.StatusDisconnected
	}
	return ch.GetStatus()
}

// GetWhatsAppQR returns the current QR code data URL, or "" if not in qr_pending state.
func (a *App) GetWhatsAppQR() string {
	a.mu.RLock()
	ch := a.whatsappCh
	a.mu.RUnlock()
	if ch == nil {
		return ""
	}
	return ch.GetQR()
}

// GetFeishuConfig returns the current Feishu credentials (AppSecret is masked).
func (a *App) GetFeishuConfig() map[string]string {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()
	appID := cfg.Feishu.AppID
	hasSecret := cfg.Feishu.AppSecret != ""
	masked := ""
	if hasSecret {
		masked = "••••••••"
	}
	return map[string]string{"appId": appID, "appSecretMasked": masked}
}

// SaveFeishuConfig persists Feishu credentials and restarts the gateway.
func (a *App) SaveFeishuConfig(appID, appSecret string) error {
	a.mu.Lock()
	a.cfg.Feishu.AppID = appID
	a.cfg.Feishu.AppSecret = appSecret
	cfg := a.cfg
	a.mu.Unlock()
	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// LogoutWhatsApp logs the WhatsApp session out.
func (a *App) LogoutWhatsApp() error {
	a.mu.RLock()
	ch := a.whatsappCh
	a.mu.RUnlock()
	if ch == nil {
		return nil
	}
	return ch.Logout(context.Background())
}

// ─── Skills RPC ───────────────────────────────────────────────────────────────

// SkillInfo is returned to the frontend to describe a loaded skill.
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	FilePath    string `json:"filePath"`
	Source      string `json:"source"` // the skill_path entry it came from
}

// GetSkillContent returns the raw markdown content of a skill file.
func (a *App) GetSkillContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read skill: %w", err)
	}
	return string(data), nil
}

// GetAllSkills returns every skill across all agents plus the default ~/.pi-go/skills/ directory.
func (a *App) GetAllSkills() []SkillInfo {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()

	seen := make(map[string]bool)
	var out []SkillInfo

	addFromPaths := func(paths []string) {
		result := skill.Load(skill.LoadOptions{SkillPaths: paths})
		for _, s := range result.Skills {
			if seen[s.FilePath] {
				continue
			}
			seen[s.FilePath] = true
			out = append(out, SkillInfo{
				Name:        s.Name,
				Description: s.Description,
				FilePath:    s.FilePath,
			})
		}
	}

	// Always scan the default skill directory.
	addFromPaths([]string{"~/.pi-go/skills/"})

	// Then scan each agent's configured skill_paths.
	for _, ac := range cfg.Agents {
		addFromPaths(ac.SkillPaths)
	}

	return out
}

// GetAgentSkills returns all currently loaded skills for the given agent.
func (a *App) GetAgentSkills(agentName string) []SkillInfo {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()

	ac, ok := cfg.Agent(agentName)
	if !ok {
		return nil
	}
	result := skill.Load(skill.LoadOptions{SkillPaths: ac.SkillPaths})
	out := make([]SkillInfo, 0, len(result.Skills))
	for _, s := range result.Skills {
		// Map each skill back to the skill_path entry it came from
		src := ""
		for _, p := range ac.SkillPaths {
			resolved := skill.ResolvePath(p, "")
			if strings.HasPrefix(s.FilePath, resolved) || s.FilePath == resolved {
				src = p
				break
			}
		}
		out = append(out, SkillInfo{
			Name:        s.Name,
			Description: s.Description,
			FilePath:    s.FilePath,
			Source:      src,
		})
	}
	return out
}

// AddSkillPath appends a skill path to the agent's config and restarts the gateway.
func (a *App) AddSkillPath(agentName, path string) error {
	a.mu.Lock()
	cfg := a.cfg
	for i, ac := range cfg.Agents {
		if ac.Name == agentName {
			// Avoid duplicates
			for _, p := range ac.SkillPaths {
				if p == path {
					a.mu.Unlock()
					return nil
				}
			}
			cfg.Agents[i].SkillPaths = append(cfg.Agents[i].SkillPaths, path)
			break
		}
	}
	a.mu.Unlock()
	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// RemoveSkillPath removes a skill path from the agent's config and restarts.
func (a *App) RemoveSkillPath(agentName, path string) error {
	a.mu.Lock()
	cfg := a.cfg
	for i, ac := range cfg.Agents {
		if ac.Name == agentName {
			paths := make([]string, 0, len(ac.SkillPaths))
			for _, p := range ac.SkillPaths {
				if p != path {
					paths = append(paths, p)
				}
			}
			cfg.Agents[i].SkillPaths = paths
			break
		}
	}
	a.mu.Unlock()
	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// EmitChannelStatus sends a channel:status event to the frontend.
func (a *App) EmitChannelStatus(agentName, status string) {
	wailsruntime.EventsEmit(a.ctx, "channel:status", map[string]string{
		"agent":  agentName,
		"status": status,
	})
}

// ─── Memory ───────────────────────────────────────────────────────────────────

// MemoryFile is the frontend representation of an indexed Markdown file.
type MemoryFile struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Hash      string `json:"hash"`
	ModifiedAt int64  `json:"modifiedAt"`
	IndexedAt  int64  `json:"indexedAt"`
	ChunkCount int    `json:"chunkCount"`
}

// MemorySearchResult is a single search hit returned to the frontend.
type MemorySearchResult struct {
	FilePath  string  `json:"filePath"`
	StartLine int     `json:"startLine"`
	EndLine   int     `json:"endLine"`
	Content   string  `json:"content"`
	Score     float32 `json:"score"`
}

// ListMemoryFiles returns all indexed memory files with chunk counts.
func (a *App) ListMemoryFiles() ([]MemoryFile, error) {
	if a.db == nil {
		return nil, nil
	}
	rows, err := a.db.SQL().QueryContext(a.ctx, `
		SELECT mf.path, mf.hash, mf.modified_at, mf.indexed_at,
		       COUNT(mc.id) AS chunk_count
		FROM memory_files mf
		LEFT JOIN memory_chunks mc ON mc.file_id = mf.id
		GROUP BY mf.id
		ORDER BY mf.indexed_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []MemoryFile
	for rows.Next() {
		var f MemoryFile
		if err := rows.Scan(&f.Path, &f.Hash, &f.ModifiedAt, &f.IndexedAt, &f.ChunkCount); err != nil {
			continue
		}
		f.Name = filepath.Base(f.Path)
		files = append(files, f)
	}
	return files, rows.Err()
}

// GetMemoryFileContent returns the raw Markdown content of a memory file.
func (a *App) GetMemoryFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveMemoryFileContent writes new content to a memory file and re-indexes it.
func (a *App) SaveMemoryFileContent(path string, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	if a.memoryMgr != nil {
		return a.memoryMgr.IndexFile(a.ctx, path)
	}
	return nil
}

// DeleteMemoryFile removes a memory file from disk and its index.
func (a *App) DeleteMemoryFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if a.memoryMgr != nil {
		return a.memoryMgr.DeleteFile(a.ctx, path)
	}
	return nil
}

// CreateMemoryFile creates a new Markdown file in the memory directory and indexes it.
func (a *App) CreateMemoryFile(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".pi-go", "memory")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// SearchMemory performs a semantic search over all indexed memory files.
func (a *App) SearchMemory(query string, limit int) ([]MemorySearchResult, error) {
	if a.memoryMgr == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	results, err := a.memoryMgr.Search(a.ctx, query, limit)
	if err != nil {
		return nil, err
	}
	out := make([]MemorySearchResult, len(results))
	for i, r := range results {
		out[i] = MemorySearchResult{
			FilePath:  r.FilePath,
			StartLine: r.StartLine,
			EndLine:   r.EndLine,
			Content:   r.Content,
			Score:     r.Score,
		}
	}
	return out, nil
}

// SyncMemory re-indexes all files in the memory directory.
func (a *App) SyncMemory() error {
	if a.memoryMgr == nil {
		return nil
	}
	return a.memoryMgr.Sync(a.ctx)
}

// GetMemoryDir returns the path to the memory directory.
func (a *App) GetMemoryDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pi-go", "memory")
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func saveConfig(cfg *config.Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".pi-go")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return writeConfigYAML(filepath.Join(dir, "config.yml"), cfg)
}

func buildProviders(cfg *config.Config) (map[string]provider.LLMProvider, error) {
	return agentbuilder.BuildProviders(cfg)
}

func buildProvider(id string, pc config.ProviderConfig) (provider.LLMProvider, error) {
	return agentbuilder.BuildProvider(id, pc)
}

func providerEnvVar(id string) string {
	return agentbuilder.ProviderEnvVar(id)
}

func defaultModelForProvider(providerID string) string {
	return agentbuilder.DefaultModelForProvider(providerID)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// buildRuntimeInfo returns a one-line runtime summary for the system prompt.
func buildRuntimeInfo(ac config.AgentConfig) string {
	host, _ := os.Hostname()
	info := "model=" + ac.Model
	if host != "" {
		info = "host=" + host + " | " + info
	}
	if ac.WorkspaceDir != "" {
		info += " | workspace=" + ac.WorkspaceDir
	}
	return info
}

// buildTools resolves a list of tool names to AgentTool instances.
// db may be nil, in which case vault injection is skipped.
func buildTools(names []string, memMgr *memory.Manager, cfg *config.Config, db *store.DB) []tool.AgentTool {
	var vaultEnv func() map[string]string
	if db != nil {
		v := db.Vault()
		vaultEnv = func() map[string]string {
			m, _ := v.Map()
			return m
		}
	}
	return agentbuilder.BuildTools(names, memMgr, cfg, vaultEnv)
}

// ─── Cron Scheduler ──────────────────────────────────────────────────────────

// buildAgentForCron creates a fresh Agent for the named agent config.
// Reuses the same provider/model/tools/skills/system-prompt logic as startGateway.
func (a *App) buildAgentForCron(agentName string) (*agent.Agent, error) {
	a.mu.RLock()
	cfg := a.cfg
	memMgr := a.memoryMgr
	a.mu.RUnlock()

	ac, ok := cfg.Agent(agentName)
	if !ok {
		return nil, fmt.Errorf("cron: agent %q not found in config", agentName)
	}

	providerMap, err := buildProviders(cfg)
	if err != nil {
		return nil, fmt.Errorf("cron: providers: %w", err)
	}
	prov, ok := providerMap[ac.Provider]
	if !ok {
		return nil, fmt.Errorf("cron: provider %q not found for agent %q", ac.Provider, agentName)
	}

	maxTokens := ac.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	model := types.Model{ID: ac.Model, Provider: ac.Provider, MaxTokens: maxTokens}
	tools := buildTools(ac.Tools, memMgr, cfg, a.db)

	skillResult := skill.Load(skill.LoadOptions{SkillPaths: ac.SkillPaths})
	compacted := skill.CompactSkillPaths(skillResult.Skills)
	skillsPrompt, _, _ := skill.ApplySkillsPromptLimits(compacted)

	bootstrap := agent.LoadBootstrapContext(ac.WorkspaceDir)
	systemPrompt := agent.BuildSystemPrompt(agent.SystemPromptParams{
		WorkspaceDir:   ac.WorkspaceDir,
		SkillsPrompt:   skillsPrompt,
		ContextFiles:   bootstrap.ContextFiles,
		WorkspaceNotes: bootstrap.WorkspaceNotes,
		PromptMode:     agent.PromptModeFull,
		RuntimeInfo:    buildRuntimeInfo(ac),
		ExtraPrompt:    ac.SystemPrompt,
	})

	opts := []agent.AgentOption{
		agent.WithModel(model),
		agent.WithSystemPrompt(systemPrompt),
	}
	if len(tools) > 0 {
		opts = append(opts, agent.WithTools(tools))
	}
	return agent.NewAgent(prov, opts...), nil
}

// syncCronConfigToDB seeds the database with cron jobs from config.yml (by name, no duplicates).
func (a *App) syncCronConfigToDB() {
	a.mu.RLock()
	cfg := a.cfg
	db := a.db
	a.mu.RUnlock()

	if db == nil || len(cfg.CronJobs) == 0 {
		return
	}
	cronStore := db.CronJobs()
	for _, cj := range cfg.CronJobs {
		if _, found, err := cronStore.FindByName(cj.Name); err != nil {
			log.Printf("app: sync cron job %s: %v", cj.Name, err)
			continue
		} else if found {
			continue // already exists
		}
		job := &store.CronJob{
			ID:           uuid.New().String(),
			Name:         cj.Name,
			ScheduleKind: cj.Schedule.Kind,
			Schedule: store.ScheduleData{
				At:       cj.Schedule.At,
				EveryMs:  cj.Schedule.EveryMs,
				AnchorMs: cj.Schedule.AnchorMs,
				Expr:     cj.Schedule.Expr,
				Tz:       cj.Schedule.Tz,
			},
			AgentName: cj.AgentName,
			Prompt:    cj.Prompt,
			Enabled:   cj.Enabled,
		}
		if err := cronStore.Create(job); err != nil {
			log.Printf("app: sync cron job %s: create: %v", cj.Name, err)
		} else {
			log.Printf("app: seeded cron job %s from config", cj.Name)
		}
	}
}

// ListCronJobs returns all cron jobs.
func (a *App) ListCronJobs() ([]store.CronJob, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.db.CronJobs().List()
}

// AddCronJob creates a new cron job and schedules it.
func (a *App) AddCronJob(job store.CronJob) error {
	if a.db == nil {
		return fmt.Errorf("database not available")
	}
	if job.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if job.ScheduleKind == "at" && job.Schedule.At == "" {
		return fmt.Errorf("'at' schedule requires a date/time")
	}
	if job.ScheduleKind == "every" && job.Schedule.EveryMs <= 0 {
		return fmt.Errorf("'every' schedule requires a positive interval")
	}
	if job.ScheduleKind == "cron" && job.Schedule.Expr == "" {
		return fmt.Errorf("'cron' schedule requires an expression")
	}
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if a.cronScheduler != nil {
		return a.cronScheduler.AddJob(&job)
	}
	return a.db.CronJobs().Create(&job)
}

// UpdateCronJob updates an existing cron job and reschedules it.
func (a *App) UpdateCronJob(job store.CronJob) error {
	if a.db == nil {
		return fmt.Errorf("database not available")
	}
	if a.cronScheduler != nil {
		return a.cronScheduler.UpdateJob(&job)
	}
	return a.db.CronJobs().Update(&job)
}

// DeleteCronJob removes a cron job.
func (a *App) DeleteCronJob(jobID string) error {
	if a.db == nil {
		return fmt.Errorf("database not available")
	}
	if a.cronScheduler != nil {
		return a.cronScheduler.RemoveJob(jobID)
	}
	return a.db.CronJobs().Delete(jobID)
}

// ToggleCronJob enables or disables a cron job.
func (a *App) ToggleCronJob(jobID string, enabled bool) error {
	if a.db == nil {
		return fmt.Errorf("database not available")
	}
	if a.cronScheduler != nil {
		return a.cronScheduler.ToggleJob(jobID, enabled)
	}
	return a.db.CronJobs().SetEnabled(jobID, enabled)
}

// GetCronJobHistory returns execution history for a specific job.
func (a *App) GetCronJobHistory(jobID string, limit int) ([]store.CronJobHistory, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.db.CronJobs().ListHistory(jobID, limit)
}

// GetCronJobHistoryAll returns execution history across all jobs.
func (a *App) GetCronJobHistoryAll(limit int) ([]store.CronJobHistory, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.db.CronJobs().ListHistoryAll(limit)
}

// TriggerCronJob runs a job immediately, regardless of its schedule.
func (a *App) TriggerCronJob(jobID string) error {
	a.mu.RLock()
	sched := a.cronScheduler
	a.mu.RUnlock()
	if sched == nil {
		return fmt.Errorf("cron scheduler not running")
	}
	return sched.TriggerNow(jobID)
}

// extractText returns a plain-text preview of a message for history display.
func extractText(m types.Message) string {
	switch msg := m.(type) {
	case *types.UserMessage:
		for _, b := range msg.Content {
			if t, ok := b.(*types.TextContent); ok {
				return t.Text
			}
		}
	case *types.AssistantMessage:
		for _, b := range msg.Content {
			if t, ok := b.(*types.TextContent); ok {
				return t.Text
			}
		}
	}
	return ""
}
