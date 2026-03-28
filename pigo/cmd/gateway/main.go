// Command gateway starts the pi-go gateway server.
//
// Usage:
//
//	go run ./cmd/gateway                      # uses ~/.pi-go/config.yml
//	go run ./cmd/gateway -config ./config.yml # explicit config path
//	go run ./cmd/gateway -addr :8080          # override listen address
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/channel/webchat"
	"github.com/ai-gateway/pi-go/config"
	"github.com/ai-gateway/pi-go/gateway"
	"github.com/ai-gateway/pi-go/internal/agentbuilder"
	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/store"
	"github.com/ai-gateway/pi-go/types"
)

func main() {
	cfgPath := flag.String("config", "", "path to config.yml (default: ~/.pi-go/config.yml)")
	addr := flag.String("addr", "", "listen address (default: :9988)")
	dbPath := flag.String("db", "", "SQLite path (default: ~/.pi-go/data.db)")
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

	// ── Provider instances ────────────────────────────────────────────────────
	providerMap, err := buildProviders(cfg)
	if err != nil {
		log.Fatalf("providers: %v", err)
	}

	// ── Agent registry ────────────────────────────────────────────────────────
	msgStore := db.Messages()
	registry := gateway.NewAgentRegistry()

	for _, ac := range cfg.Agents {
		prov, ok := providerMap[ac.Provider]
		if !ok {
			log.Fatalf("agent %q: provider %q not found in config", ac.Name, ac.Provider)
		}
		maxTokens := ac.MaxTokens
		if maxTokens == 0 {
			maxTokens = 4096
		}
		model := types.Model{
			ID:        ac.Model,
			Provider:  ac.Provider,
			MaxTokens: maxTokens,
		}
		systemPrompt := ac.SystemPrompt
		agentName := ac.Name

		factory := gateway.AgentFactory(func(channelID, userID string) *agent.Agent {
			a := agent.NewAgent(prov,
				agent.WithModel(model),
				agent.WithSystemPrompt(systemPrompt),
			)
			savedCount := 0
			if history, err := msgStore.ListMessages(store.QueryParams{
				ChannelID: channelID,
				UserID:    userID,
			}); err == nil && len(history) > 0 {
				a.ReplaceMessages(history)
				savedCount = len(history)
				log.Printf("[%s] session %s/%s: restored %d messages", agentName, channelID, userID, len(history))
			}
			a.Subscribe(func(ev types.AgentEvent) {
				if ev.Type != types.EventAgentEnd {
					return
				}
				msgs := ev.Messages
				for i := savedCount; i < len(msgs); i++ {
					if err := msgStore.SaveMessage(channelID, userID, msgs[i]); err != nil {
						log.Printf("[%s] session %s/%s: save message[%d]: %v", agentName, channelID, userID, i, err)
					}
				}
				savedCount = len(msgs)
			})
			return a
		})

		mgr := gateway.NewSessionManager(factory, gateway.ManagerConfig{})
		registry.Register(ac.Name, mgr)
		log.Printf("agent: %s  provider: %s  model: %s", ac.Name, ac.Provider, ac.Model)
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("gateway: starting on %s  default-agent: %s", listenAddr, defaultAgent)
	log.Printf("gateway: ws://localhost%s/ws/{agentName}/{sessionID}", listenAddr)
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("gateway: %v", err)
	}
	log.Println("gateway: stopped")
}

// buildProviders constructs one LLMProvider per entry in cfg.Providers.
func buildProviders(cfg *config.Config) (map[string]provider.LLMProvider, error) {
	return agentbuilder.BuildProviders(cfg)
}

// buildProvider creates an LLMProvider from a single ProviderConfig.
func buildProvider(id string, pc config.ProviderConfig) (provider.LLMProvider, error) {
	return agentbuilder.BuildProvider(id, pc)
}

func providerEnvVar(id string) string {
	return agentbuilder.ProviderEnvVar(id)
}
