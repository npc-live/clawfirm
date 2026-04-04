package remote

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ai-gateway/clawfirm/gateway"
	"github.com/ai-gateway/clawfirm/types"
)

const remoteChannelID = "remote"

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// clientMessage is the JSON format clients send to the server.
type clientMessage struct {
	Type     string         `json:"type"`      // "message" | "run_tool"
	Content  string         `json:"content"`   // text (for "message")
	Images   []imageData    `json:"images"`    // optional
	ToolName string         `json:"tool_name"` // (for "run_tool")
	ToolID   string         `json:"tool_id"`   // call ID
	ToolArgs map[string]any `json:"tool_args"` // (for "run_tool")
}

type imageData struct {
	Data string `json:"data"` // base64
	Mime string `json:"mime"` // e.g. "image/jpeg"
}

// serverMessage is the JSON format the server sends to clients.
type serverMessage struct {
	Type          string `json:"type"`
	Content       string `json:"content,omitempty"`
	StopReason    string `json:"stop_reason,omitempty"`
	Timestamp     int64  `json:"timestamp,omitempty"`
	ToolCallID    string `json:"tool_call_id,omitempty"`
	ToolName      string `json:"tool_name,omitempty"`
	ToolArgs      any    `json:"tool_args,omitempty"`
	ToolResult    any    `json:"tool_result,omitempty"`
	ToolIsError   bool   `json:"tool_is_error,omitempty"`
	PartialResult any    `json:"partial_result,omitempty"`
}

// handleWebSocket handles WebSocket connections for remote clients.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("agentName")
	sessionID := r.PathValue("sessionID")
	if sessionID == "" {
		http.Error(w, "missing sessionID", http.StatusBadRequest)
		return
	}

	mgr, ok := s.registry.Get(agentName)
	if !ok {
		http.Error(w, "unknown agent: "+agentName, http.StatusNotFound)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("remote: ws upgrade: %v", err)
		return
	}
	s.clients.Add(1)
	defer func() {
		s.clients.Add(-1)
		conn.Close()
	}()
	log.Printf("remote: [%s] ws connected: %s/%s (clients=%d)", agentName, remoteChannelID, sessionID, s.clients.Load())

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

	channelID := remoteChannelID + "/" + agentName
	sess, err := mgr.GetOrCreate(channelID, sessionID)
	if err != nil {
		write(serverMessage{Type: "error", Content: err.Error()})
		return
	}

	unsub := sess.Subscribe(func(ev types.AgentEvent) {
		handleAgentEvent(write, ev)
	})
	defer unsub()

	// Ping/pong keepalive.
	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Read loop.
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
			switch msg.Type {
			case "message":
				log.Printf("remote: [%s] msg: %s/%s: %q", agentName, channelID, sessionID, msg.Content)
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
			case "run_tool":
				callID := msg.ToolID
				if callID == "" {
					callID = "direct-" + msg.ToolName
				}
				go sess.RunTool(r.Context(), callID, msg.ToolName, msg.ToolArgs)
			}
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
