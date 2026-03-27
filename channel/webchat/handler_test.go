package webchat_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/channel/webchat"
	"github.com/ai-gateway/pi-go/gateway"
	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/types"
)

// fakeProvider returns one delta + done for every stream call.
type fakeProvider struct{ reply string }

func (f *fakeProvider) ID() string            { return "fake" }
func (f *fakeProvider) Models() []types.Model { return nil }
func (f *fakeProvider) Stream(_ context.Context, _ provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	ch := make(chan types.AssistantMessageEvent, 3)
	ch <- types.AssistantMessageEvent{Type: types.StreamEventTextDelta, Delta: f.reply}
	ch <- types.AssistantMessageEvent{
		Type:    types.StreamEventDone,
		Message: &types.AssistantMessage{Role: "assistant", StopReason: types.StopReasonStop},
	}
	close(ch)
	return ch, nil
}

func newTestRegistry(reply string) *gateway.AgentRegistry {
	factory := gateway.AgentFactory(func(_, _ string) *agent.Agent {
		return agent.NewAgent(&fakeProvider{reply: reply}, agent.WithModel(types.Model{ID: "m"}))
	})
	mgr := gateway.NewSessionManager(factory, gateway.ManagerConfig{})
	reg := gateway.NewAgentRegistry()
	reg.Register("test", mgr)
	return reg
}

// newTestServer creates an httptest.Server with the webchat handler mounted.
func newTestServer(t *testing.T, reply string) (*httptest.Server, *gateway.AgentRegistry) {
	t.Helper()
	reg := newTestRegistry(reply)
	handler := webchat.NewHandler(reg, "test")
	mux := http.NewServeMux()
	mux.Handle("/ws/{agentName}/{sessionID}", handler)
	mux.Handle("/ws/{sessionID}", handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
		reg.Stop()
	})
	return srv, reg
}

// wsURL converts an http:// test server URL to ws://.
func wsURL(srv *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + path
}

// dialDirect dials a WebSocket URL bypassing any env proxy.
func dialDirect(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	dialer := websocket.Dialer{
		Proxy:            nil, // explicitly no proxy
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	return conn
}

func TestHandlerSingleTurn(t *testing.T) {
	srv, _ := newTestServer(t, "pong")
	conn := dialDirect(t, wsURL(srv, "/ws/test/alice"))
	defer conn.Close()

	if err := conn.WriteJSON(map[string]string{"type": "message", "content": "ping"}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var got []map[string]any
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var m map[string]any
		json.Unmarshal(data, &m)
		got = append(got, m)
		if m["type"] == "done" {
			break
		}
	}

	if len(got) < 2 {
		t.Fatalf("want at least 2 messages (delta + done), got %d", len(got))
	}
	if got[0]["type"] != "delta" || got[0]["content"] != "pong" {
		t.Errorf("first message: want delta/pong, got %v", got[0])
	}
	if got[len(got)-1]["type"] != "done" {
		t.Errorf("last message: want done, got %v", got[len(got)-1])
	}
}

func TestHandlerMultiTurn(t *testing.T) {
	srv, _ := newTestServer(t, "ok")
	conn := dialDirect(t, wsURL(srv, "/ws/test/multi"))
	defer conn.Close()

	for i := range 2 {
		if err := conn.WriteJSON(map[string]string{"type": "message", "content": "msg"}); err != nil {
			t.Fatalf("turn %d write: %v", i, err)
		}
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("turn %d read: %v", i, err)
			}
			var m map[string]any
			json.Unmarshal(data, &m)
			if m["type"] == "done" {
				break
			}
		}
	}
}

func TestHandlerDefaultAgent(t *testing.T) {
	srv, _ := newTestServer(t, "hi")
	// Use the short URL that routes to defaultAgent
	conn := dialDirect(t, wsURL(srv, "/ws/sess1"))
	defer conn.Close()

	conn.WriteJSON(map[string]string{"type": "message", "content": "hello"})
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var m map[string]any
		json.Unmarshal(data, &m)
		if m["type"] == "done" {
			break
		}
	}
}

func TestHandlerUnknownAgent(t *testing.T) {
	srv, _ := newTestServer(t, "")
	resp, err := http.Get(srv.URL + "/ws/unknown-agent/sess")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", resp.StatusCode)
	}
}
