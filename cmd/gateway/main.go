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
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/ai-gateway/clawfirm/agent"
	"github.com/ai-gateway/clawfirm/channel/webchat"
	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/gateway"
	"github.com/ai-gateway/clawfirm/internal/agentbuilder"
	"github.com/ai-gateway/clawfirm/store"
	"github.com/ai-gateway/clawfirm/types"
)

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

		factory := gateway.AgentFactory(func(channelID, userID string) gateway.AgentRunner {
			ag := agent.NewAgent(prov,
				agent.WithModel(model),
			)
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
		log.Printf("agent: %s  model: %s", ac.Name, ac.Model)
	}

	if len(cfg.Agents) == 0 {
		log.Fatal("config: no agents defined — add an 'agents:' section to config.yml")
	}

	defer registry.Stop()

	// ── HTTP server ───────────────────────────────────────────────────────────
	listenAddr := *addr
	if listenAddr == "" {
		listenAddr = ":9988"
	}
	srv := gateway.NewServer(registry, gateway.ServerConfig{Addr: listenAddr})

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

	// ── Run ───────────────────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("gateway: starting on %s  default-agent: %s", listenAddr, defaultAgent)
	log.Printf("gateway: ws://localhost%s/ws/{agentName}/{sessionID}", listenAddr)
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("gateway: %v", err)
	}
	log.Println("gateway: stopped")
}
