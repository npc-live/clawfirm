package remote

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-gateway/clawfirm/store"
	"github.com/ai-gateway/clawfirm/types"
)

// buildMux constructs the HTTP mux for the remote server.
func (s *Server) buildMux() http.Handler {
	mux := http.NewServeMux()

	// Landing page — simple connection info (exact match only).
	mux.HandleFunc("GET /remote/{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html><html><body><h1>clawfirm remote</h1><p>Use the mobile app to connect.</p></body></html>`))
	})

	// All API/WS routes require token auth.
	authed := http.NewServeMux()
	authed.HandleFunc("GET /remote/api/status", s.handleStatus)
	authed.HandleFunc("GET /remote/api/agents", s.handleAgents)
	authed.HandleFunc("GET /remote/api/agents/{name}/sessions", s.handleSessions)
	authed.HandleFunc("GET /remote/api/agents/{name}/sessions/{id}/history", s.handleHistory)
	authed.HandleFunc("GET /remote/api/canvas", s.handleCanvasList)
	authed.HandleFunc("GET /remote/api/canvas/{name}", s.handleCanvasGet)
	authed.HandleFunc("GET /remote/api/channels", s.handleChannels)
	authed.HandleFunc("GET /remote/ws/{agentName}/{sessionID}", s.handleWebSocket)

	mux.Handle("/remote/api/", s.authMiddleware(authed))
	mux.Handle("/remote/ws/", s.authMiddleware(authed))

	return mux
}

// authMiddleware checks for a valid token in the query string or header.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			token = r.Header.Get("X-Remote-Token")
		}
		if token != s.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.Status())
}

type agentInfo struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Sessions int    `json:"sessions"`
}

func (s *Server) handleAgents(w http.ResponseWriter, _ *http.Request) {
	counts := s.registry.Counts()
	var agents []agentInfo
	for _, ac := range s.cfg.Agents {
		agents = append(agents, agentInfo{
			Name:     ac.Name,
			Provider: ac.Provider,
			Model:    ac.Model,
			Sessions: counts[ac.Name],
		})
	}
	if agents == nil {
		agents = []agentInfo{}
	}
	writeJSON(w, agents)
}

type sessionInfo struct {
	SessionKey  string  `json:"sessionKey"`
	SessionID   string  `json:"sessionId"`
	ChannelID   string  `json:"channelId"`
	UserID      string  `json:"userId"`
	Subject     string  `json:"subject"`
	Model       string  `json:"model"`
	InputTokens int     `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
	CostUSD     float64 `json:"costUsd"`
	CreatedAt   int64   `json:"createdAt"`
	UpdatedAt   int64   `json:"updatedAt"`
	IsActive    bool    `json:"isActive"`
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("name")
	if s.db == nil {
		writeJSON(w, []sessionInfo{})
		return
	}

	entries, err := s.db.Sessions().ListByAgent(agentName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build active set.
	activeKeys := make(map[string]struct{})
	if mgr, ok := s.registry.Get(agentName); ok {
		for _, sess := range mgr.ActiveSessions() {
			activeKeys[sess.Key()] = struct{}{}
		}
	}

	out := make([]sessionInfo, 0, len(entries))
	for _, e := range entries {
		_, active := activeKeys[e.SessionKey]
		out = append(out, sessionInfo{
			SessionKey:   e.SessionKey,
			SessionID:    e.SessionID,
			ChannelID:    e.ChannelID,
			UserID:       e.UserID,
			Subject:      e.Subject,
			Model:        e.Model,
			InputTokens:  e.InputTokens,
			OutputTokens: e.OutputTokens,
			CostUSD:      e.EstimatedCostUSD,
			CreatedAt:    e.CreatedAt.UnixMilli(),
			UpdatedAt:    e.UpdatedAt.UnixMilli(),
			IsActive:     active,
		})
	}
	writeJSON(w, out)
}

type historyMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("name")
	sessionID := r.PathValue("id")
	if s.db == nil {
		writeJSON(w, []historyMessage{})
		return
	}

	channelID := "webchat/" + agentName
	msgs, err := s.db.Messages().ListMessages(store.QueryParams{
		ChannelID: channelID,
		UserID:    sessionID,
		Limit:     200,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Fallback to legacy "webchat" channel.
	if len(msgs) == 0 {
		msgs, err = s.db.Messages().ListMessages(store.QueryParams{
			ChannelID: "webchat",
			UserID:    sessionID,
			Limit:     200,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	out := make([]historyMessage, 0, len(msgs))
	for _, m := range msgs {
		role := m.MessageRole()
		content := extractTextContent(m)
		out = append(out, historyMessage{Role: role, Content: content})
	}
	writeJSON(w, out)
}

func (s *Server) handleCanvasList(w http.ResponseWriter, _ *http.Request) {
	entries, err := os.ReadDir(s.canvasDir)
	if err != nil {
		writeJSON(w, []string{})
		return
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".html") {
			names = append(names, strings.TrimSuffix(name, ".html"))
		}
	}
	if names == nil {
		names = []string{}
	}
	writeJSON(w, names)
}

func (s *Server) handleCanvasGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" || strings.ContainsAny(name, "/\\..") {
		http.Error(w, "invalid name", http.StatusBadRequest)
		return
	}
	path := filepath.Join(s.canvasDir, name+".html")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (s *Server) handleChannels(w http.ResponseWriter, _ *http.Request) {
	if s.channelStatusFn == nil {
		writeJSON(w, []ChannelStatus{})
		return
	}
	writeJSON(w, s.channelStatusFn())
}

// extractTextContent pulls text from a types.Message.
func extractTextContent(m types.Message) string {
	var blocks []types.ContentBlock
	switch msg := m.(type) {
	case *types.UserMessage:
		blocks = msg.Content
	case *types.AssistantMessage:
		blocks = msg.Content
	default:
		return ""
	}
	for _, b := range blocks {
		if t, ok := b.(*types.TextContent); ok {
			return t.Text
		}
	}
	return ""
}
