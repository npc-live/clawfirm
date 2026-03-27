// Package webchat implements a WebSocket-based chat channel for pi-go.
//
// Protocol:
//
//	Client → Server: {"type":"message","content":"...","images":[{"data":"base64","mime":"image/jpeg"}]}
//	Server → Client: {"type":"delta","content":"..."}
//	                 {"type":"done","stop_reason":"stop"}
//	                 {"type":"error","content":"..."}
//
// URL patterns:
//
//	GET /ws/{agentName}/{sessionID}   — explicit agent
//	GET /ws/{sessionID}               — routes to the default agent
package webchat

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ai-gateway/pi-go/gateway"
	"github.com/ai-gateway/pi-go/types"
)

const channelID = "webchat"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true }, // allow all origins
}

// clientMessage is the JSON format clients send to the server.
type clientMessage struct {
	Type    string      `json:"type"`    // "message"
	Content string      `json:"content"` // text
	Images  []imageData `json:"images"`  // optional
}

type imageData struct {
	Data string `json:"data"` // base64
	Mime string `json:"mime"` // e.g. "image/jpeg"
}

// serverMessage is the JSON format the server sends to clients.
type serverMessage struct {
	Type       string `json:"type"`                  // "delta" | "done" | "error" | "tool_start" | "tool_update" | "tool_end"
	Content    string `json:"content,omitempty"`
	StopReason string `json:"stop_reason,omitempty"` // on "done"
	Timestamp  int64  `json:"timestamp,omitempty"`
	// Tool event fields
	ToolCallID    string `json:"tool_call_id,omitempty"`
	ToolName      string `json:"tool_name,omitempty"`
	ToolArgs      any    `json:"tool_args,omitempty"`
	ToolResult    any    `json:"tool_result,omitempty"`
	ToolIsError   bool   `json:"tool_is_error,omitempty"`
	PartialResult any    `json:"partial_result,omitempty"`
}

// Handler handles WebSocket connections for the webchat channel.
// It supports both single-agent (/ws/{sessionID}) and
// multi-agent (/ws/{agentName}/{sessionID}) URL patterns.
type Handler struct {
	registry     *gateway.AgentRegistry
	defaultAgent string
}

// NewHandler creates a webchat Handler backed by the given registry.
// defaultAgent is used when no agent name appears in the URL.
func NewHandler(registry *gateway.AgentRegistry, defaultAgent string) *Handler {
	return &Handler{registry: registry, defaultAgent: defaultAgent}
}

// ServeHTTP handles WebSocket upgrade for both URL patterns.
// agentName is read from {agentName} path value; falls back to defaultAgent.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("agentName")
	if agentName == "" {
		agentName = h.defaultAgent
	}
	sessionID := r.PathValue("sessionID")
	if sessionID == "" {
		http.Error(w, "missing sessionID", http.StatusBadRequest)
		return
	}

	mgr, ok := h.registry.Get(agentName)
	if !ok {
		http.Error(w, "unknown agent: "+agentName, http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("webchat: upgrade: %v", err)
		return
	}
	log.Printf("[%s] connected: %s/%s", agentName, channelID, sessionID)
	defer conn.Close()

	// writeMu serialises all WebSocket writes (gorilla allows one concurrent writer).
	var writeMu sync.Mutex
	write := func(v any) {
		b, err := json.Marshal(v)
		if err != nil {
			return
		}
		writeMu.Lock()
		conn.WriteMessage(websocket.TextMessage, b)
		writeMu.Unlock()
	}

	sess, err := mgr.GetOrCreate(channelID, sessionID)
	if err != nil {
		write(serverMessage{Type: "error", Content: err.Error()})
		return
	}

	// Register event sink — pushes agent events to the WebSocket.
	unsub := sess.Subscribe(func(ev types.AgentEvent) {
		handleAgentEvent(write, ev)
	})
	defer unsub()

	// Set up ping/pong keepalive.
	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Read loop
	readErrCh := make(chan error, 1)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				readErrCh <- err
				return
			}
			conn.SetReadDeadline(time.Now().Add(90 * time.Second))

			var msg clientMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				write(serverMessage{Type: "error", Content: "invalid JSON"})
				continue
			}
			if msg.Type != "message" {
				continue
			}
			log.Printf("[%s] recv msg: %s/%s: %q", agentName, channelID, sessionID, msg.Content)

			var images []gateway.ImageData
			for _, img := range msg.Images {
				images = append(images, gateway.ImageData{Data: img.Data, MimeType: img.Mime})
			}
			sess.Send(gateway.IncomingMessage{
				ChannelID: channelID,
				UserID:    sessionID,
				Content:   msg.Content,
				Images:    images,
			})
		}
	}()

	for {
		select {
		case <-pingTicker.C:
			writeMu.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			writeMu.Unlock()
			if err != nil {
				return
			}
		case <-readErrCh:
			return
		}
	}
}

// handleAgentEvent converts an AgentEvent to WebSocket messages.
func handleAgentEvent(write func(any), ev types.AgentEvent) {
	switch ev.Type {
	case types.EventMessageUpdate:
		if ev.StreamEvent == nil {
			return
		}
		switch ev.StreamEvent.Type {
		case types.StreamEventTextDelta:
			write(serverMessage{
				Type:    "delta",
				Content: ev.StreamEvent.Delta,
			})
		case types.StreamEventError:
			if ev.StreamEvent.Error != nil {
				write(serverMessage{
					Type:    "error",
					Content: ev.StreamEvent.Error.ErrorMessage,
				})
			}
		}
	case types.EventToolExecutionStart:
		write(serverMessage{
			Type:       "tool_start",
			ToolCallID: ev.ToolCallID,
			ToolName:   ev.ToolName,
			ToolArgs:   ev.ToolArgs,
			Timestamp:  time.Now().UnixMilli(),
		})
	case types.EventToolExecutionUpdate:
		write(serverMessage{
			Type:          "tool_update",
			ToolCallID:    ev.ToolCallID,
			ToolName:      ev.ToolName,
			PartialResult: ev.PartialResult,
		})
	case types.EventToolExecutionEnd:
		write(serverMessage{
			Type:        "tool_end",
			ToolCallID:  ev.ToolCallID,
			ToolName:    ev.ToolName,
			ToolResult:  ev.ToolResult,
			ToolIsError: ev.ToolIsError,
			Timestamp:   time.Now().UnixMilli(),
		})
	case types.EventAgentEnd:
		stop := "stop"
		if len(ev.Messages) > 0 {
			if am, ok := ev.Messages[len(ev.Messages)-1].(*types.AssistantMessage); ok {
				stop = string(am.StopReason)
			}
		}
		write(serverMessage{
			Type:       "done",
			StopReason: stop,
			Timestamp:  time.Now().UnixMilli(),
		})
	}
}
