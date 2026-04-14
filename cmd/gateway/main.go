// Command gateway starts the clawfirm gateway server.
//
// Usage:
//
//	go run ./cmd/gateway                      # uses ~/.clawfirm/config.yml
//	go run ./cmd/gateway -config ./config.yml # explicit config path
//	go run ./cmd/gateway -addr :8080          # override listen address
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ai-gateway/clawfirm/agent"
	"github.com/ai-gateway/clawfirm/channel/telegram"
	"github.com/ai-gateway/clawfirm/channel/webchat"
	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/gateway"
	"github.com/ai-gateway/clawfirm/internal/agentbuilder"
	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/skill"
	"github.com/ai-gateway/clawfirm/store"
	"github.com/ai-gateway/clawfirm/types"
)

//go:embed all:frontend/dist
var frontendAssets embed.FS

var version = "dev"

func main() {
	cfgPath := flag.String("config", "", "path to config.yml (default: ~/.clawfirm/config.yml)")
	addr := flag.String("addr", "", "listen address (default: :9988)")
	dbPath := flag.String("db", "", "SQLite path (default: ~/.clawfirm/data.db)")
	flag.Parse()

	// ── Config ───────────────────────────────────────────────────────────────
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// ── Store ────────────────────────────────────────────────────────────────
	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer db.Close()
	log.Println("store: SQLite ready")

	// ── Providers ────────────────────────────────────────────────────────────
	providerMap, err := agentbuilder.BuildProviders(cfg)
	if err != nil {
		log.Fatalf("providers: %v", err)
	}

	// ── Media provider ──────────────────────────────────────────────────────
	var mediaProvider provider.LLMProvider
	if cfg.Media.Provider != "" {
		mediaProvider = providerMap[cfg.Media.Provider]
	}

	// ── Agent registry ────────────────────────────────────────────────────────
	msgStore := db.Messages()
	registry := gateway.NewAgentRegistry()

	for _, ac := range cfg.Agents {
		prov, ok := providerMap[ac.Provider]
		if !ok {
			log.Fatalf("agent %q: provider %q not found", ac.Name, ac.Provider)
		}
		agentName := ac.Name
		maxTokens := ac.MaxTokens
		if maxTokens == 0 {
			maxTokens = 16384
		}
		model := types.Model{ID: ac.Model, Provider: ac.Provider, MaxTokens: maxTokens}

		// Build tools (same as app.go).
		tools := agentbuilder.BuildTools(ac.Tools, nil, cfg, nil, mediaProvider, agentbuilder.AgentRef{Provider: prov, Model: ac.Model})

		// Load skills.
		skillResult := skill.Load(skill.LoadOptions{SkillPaths: ac.SkillPaths})
		for _, d := range skillResult.Diagnostics {
			log.Printf("agent %s: skill warning: %s: %s", agentName, d.Path, d.Message)
		}
		if len(skillResult.Skills) > 0 {
			names := make([]string, len(skillResult.Skills))
			for i, s := range skillResult.Skills {
				names[i] = s.Name
			}
			log.Printf("agent %s: loaded %d skill(s): %s", agentName, len(skillResult.Skills), strings.Join(names, ", "))
		}
		compacted := skill.CompactSkillPaths(skillResult.Skills)
		skillsPrompt, _, _ := skill.ApplySkillsPromptLimits(compacted)

		// Build system prompt.
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

		temporal := agent.NewTemporalInjector(0)
		loopCfg := agent.AgentLoopConfig{
			TransformContext: temporal.TransformContext,
		}

		factory := gateway.AgentFactory(func(channelID, userID string) gateway.AgentRunner {
			opts := []agent.AgentOption{
				agent.WithModel(model),
				agent.WithSystemPrompt(systemPrompt),
				agent.WithLoopConfig(loopCfg),
			}
			if len(tools) > 0 {
				opts = append(opts, agent.WithTools(tools))
			}
			ag := agent.NewAgent(prov, opts...)
			// Restore message history from store.
			storeChannelID := "webchat/" + agentName
			if history, err := msgStore.ListMessages(store.QueryParams{
				ChannelID: storeChannelID,
				UserID:    userID,
			}); err == nil && len(history) > 0 {
				ag.ReplaceMessages(history)
				log.Printf("[%s] session %s/%s: restored %d messages", agentName, storeChannelID, userID, len(history))
			}
			ag.Subscribe(func(ev types.AgentEvent) {
				switch ev.Type {
				case types.EventMessageEnd:
					if ev.AssistantMsg != nil {
						if err := msgStore.SaveMessage(channelID, userID, ev.AssistantMsg); err != nil {
							log.Printf("[%s] session %s/%s: save assistant message: %v", agentName, channelID, userID, err)
						}
					}
				}
			})
			return ag
		})

		mgr := gateway.NewSessionManager(factory, gateway.ManagerConfig{
			OnUserMessage: func(channelID, userID string, msg types.Message) error {
				if err := msgStore.SaveMessage(channelID, userID, msg); err != nil {
					log.Printf("[%s] save user message: %v", agentName, err)
					return err
				}
				return nil
			},
		})
		registry.Register(ac.Name, mgr)
		log.Printf("agent: %s  model: %s  tools: %d", ac.Name, ac.Model, len(tools))
	}

	if len(cfg.Agents) == 0 {
		log.Fatal("config: no agents defined — add an 'agents:' section to config.yml")
	}

	defer registry.Stop()

	// ── Embedded frontend ────────────────────────────────────────────────────
	frontendFS, _ := fs.Sub(frontendAssets, "frontend/dist")

	// ── HTTP server ───────────────────────────────────────────────────────────
	listenAddr := *addr
	if listenAddr == "" {
		listenAddr = ":9988"
	}
	srv := gateway.NewServer(registry, gateway.ServerConfig{
		Addr:     listenAddr,
		Frontend: frontendFS,
	})

	defaultAgent := cfg.DefaultAgent
	if defaultAgent == "" {
		defaultAgent = cfg.Agents[0].Name
	}
	handler := webchat.NewHandler(registry, defaultAgent)
	// Multi-agent: /ws/{agentName}/{sessionID}
	// Backward-compat: /ws/{sessionID} → routes to defaultAgent
	// Note: register the more-specific pattern first.
	srv.Handle("GET /ws/{agentName}/{sessionID}", handler)
	srv.Handle("GET /ws/{sessionID}", handler)

	// ── Minimal REST API (for Web UI) ─────────────────────────────────────────
	registerAPI(srv, cfg, db, registry, listenAddr)

	// ── Run ───────────────────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── Telegram channel ─────────────────────────────────────────────────────
	if defaultAgent != "" && cfg.Telegram.BotToken != "" {
		tgCh := telegram.New(cfg.Telegram.BotToken, registry, defaultAgent)
		tgCtx, tgCancel := context.WithCancel(ctx)
		defer tgCancel()
		go func() {
			if err := tgCh.Start(tgCtx); err != nil {
				log.Printf("telegram: %v", err)
			}
		}()
		log.Printf("telegram: starting bot")
	}

	log.Printf("gateway: starting on %s  default-agent: %s", listenAddr, defaultAgent)
	log.Printf("gateway: http://localhost%s  (Web UI)", listenAddr)
	log.Printf("gateway: ws://localhost%s/ws/{agentName}/{sessionID}", listenAddr)
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("gateway: %v", err)
	}
	log.Println("gateway: stopped")
}

// registerAPI adds the minimal REST endpoints needed by the Web UI.
func registerAPI(srv *gateway.Server, cfg *config.Config, db *store.DB, registry *gateway.AgentRegistry, listenAddr string) {
	msgStore := db.Messages()

	// GET /api/system/first-run — true if no providers and no agents configured.
	srv.HandleFunc("GET /api/system/first-run", func(w http.ResponseWriter, _ *http.Request) {
		first := len(cfg.Providers) == 0 && len(cfg.Agents) == 0
		writeJSON(w, map[string]bool{"firstRun": first})
	})

	// GET /api/system/version
	srv.HandleFunc("GET /api/system/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"version": version})
	})

	// GET /api/channels — list configured agents with session counts.
	srv.HandleFunc("GET /api/channels", func(w http.ResponseWriter, _ *http.Request) {
		counts := registry.Counts()
		type channelInfo struct {
			Name     string `json:"name"`
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Sessions int    `json:"sessions"`
		}
		out := make([]channelInfo, 0, len(cfg.Agents))
		for _, ac := range cfg.Agents {
			out = append(out, channelInfo{
				Name:     ac.Name,
				Provider: ac.Provider,
				Model:    ac.Model,
				Sessions: counts[ac.Name],
			})
		}
		writeJSON(w, out)
	})

	// GET /api/channels/{name}/sessions — list session IDs for an agent.
	srv.HandleFunc("GET /api/channels/{name}/sessions", func(w http.ResponseWriter, r *http.Request) {
		agentName := r.PathValue("name")
		ids1, err := msgStore.ListUserIDs("webchat/" + agentName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ids2, err := msgStore.ListUserIDs("webchat")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		seen := make(map[string]struct{}, len(ids1))
		for _, id := range ids1 {
			seen[id] = struct{}{}
		}
		for _, id := range ids2 {
			if _, ok := seen[id]; !ok {
				ids1 = append(ids1, id)
			}
		}
		if ids1 == nil {
			ids1 = []string{}
		}
		writeJSON(w, ids1)
	})

	// GET /api/channels/{name}/sessions/{id}/history — message history.
	srv.HandleFunc("GET /api/channels/{name}/sessions/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		agentName := r.PathValue("name")
		sessionID := r.PathValue("id")
		channelID := "webchat/" + agentName

		msgs, err := msgStore.ListMessages(store.QueryParams{
			ChannelID: channelID,
			UserID:    sessionID,
			Limit:     50,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Fall back to legacy "webchat" channel.
		if len(msgs) == 0 {
			msgs, err = msgStore.ListMessages(store.QueryParams{
				ChannelID: "webchat",
				UserID:    sessionID,
				Limit:     50,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		type histMsg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		out := make([]histMsg, 0, len(msgs))
		for _, m := range msgs {
			role := "user"
			if _, ok := m.(*types.AssistantMessage); ok {
				role = "assistant"
			}
			text := extractText(m)
			if text == "" {
				continue
			}
			out = append(out, histMsg{Role: role, Content: text})
		}
		writeJSON(w, out)
	})

	// GET /api/webhook/base-url — returns the WebSocket base URL.
	srv.HandleFunc("GET /api/webhook/base-url", func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if host == "" {
			host = "localhost" + listenAddr
		}
		scheme := "ws"
		if r.TLS != nil {
			scheme = "wss"
		}
		writeJSON(w, map[string]string{"url": scheme + "://" + host})
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func buildRuntimeInfo(ac config.AgentConfig) string {
	info := "model=" + ac.Model
	if ac.WorkspaceDir != "" {
		info += " | workspace=" + ac.WorkspaceDir
	}
	return info
}

// extractText returns the first text content from a message.
func extractText(m types.Message) string {
	switch msg := m.(type) {
	case *types.UserMessage:
		for _, b := range msg.Content {
			if t, ok := b.(*types.TextContent); ok {
				return t.Text
			}
		}
	case *types.AssistantMessage:
		var parts []string
		for _, b := range msg.Content {
			if t, ok := b.(*types.TextContent); ok {
				parts = append(parts, t.Text)
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}
