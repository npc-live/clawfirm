package gateway_test

import (
	"context"
	"testing"
	"time"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/gateway"
	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/types"
)

// fakeProvider returns a single done event for every call.
type fakeProvider struct{}

func (f *fakeProvider) ID() string            { return "fake" }
func (f *fakeProvider) Models() []types.Model { return nil }
func (f *fakeProvider) Stream(_ context.Context, _ provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	ch := make(chan types.AssistantMessageEvent, 2)
	ch <- types.AssistantMessageEvent{Type: types.StreamEventTextDelta, Delta: "hi"}
	ch <- types.AssistantMessageEvent{
		Type:    types.StreamEventDone,
		Message: &types.AssistantMessage{Role: "assistant", StopReason: types.StopReasonStop},
	}
	close(ch)
	return ch, nil
}

func testFactory() gateway.AgentFactory {
	return func(_, _ string) *agent.Agent {
		return agent.NewAgent(&fakeProvider{}, agent.WithModel(types.Model{ID: "m"}))
	}
}

// ── SessionManager ───────────────────────────────────────────────────────────

func TestManagerGetOrCreate(t *testing.T) {
	mgr := gateway.NewSessionManager(testFactory(), gateway.ManagerConfig{})
	defer mgr.Stop()

	s1, err := mgr.GetOrCreate("webchat", "user1")
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	s2, err := mgr.GetOrCreate("webchat", "user1")
	if err != nil {
		t.Fatalf("GetOrCreate 2nd: %v", err)
	}
	if s1 != s2 {
		t.Error("expected same session for same channel+user")
	}
}

func TestManagerDifferentUsers(t *testing.T) {
	mgr := gateway.NewSessionManager(testFactory(), gateway.ManagerConfig{})
	defer mgr.Stop()

	s1, _ := mgr.GetOrCreate("webchat", "alice")
	s2, _ := mgr.GetOrCreate("webchat", "bob")
	if s1 == s2 {
		t.Error("expected different sessions for different users")
	}
	if mgr.Count() != 2 {
		t.Errorf("Count: want 2 got %d", mgr.Count())
	}
}

func TestManagerMaxSessions(t *testing.T) {
	mgr := gateway.NewSessionManager(testFactory(), gateway.ManagerConfig{MaxSessions: 2})
	defer mgr.Stop()

	if _, err := mgr.GetOrCreate("ch", "u1"); err != nil {
		t.Fatalf("u1: %v", err)
	}
	if _, err := mgr.GetOrCreate("ch", "u2"); err != nil {
		t.Fatalf("u2: %v", err)
	}
	if _, err := mgr.GetOrCreate("ch", "u3"); err == nil {
		t.Error("expected error when max sessions reached")
	}
}

func TestManagerRemove(t *testing.T) {
	mgr := gateway.NewSessionManager(testFactory(), gateway.ManagerConfig{})
	defer mgr.Stop()

	mgr.GetOrCreate("ch", "u")
	mgr.Remove("ch", "u")
	if mgr.Count() != 0 {
		t.Errorf("Count after Remove: want 0 got %d", mgr.Count())
	}
}

func TestManagerIdleEviction(t *testing.T) {
	mgr := gateway.NewSessionManager(testFactory(), gateway.ManagerConfig{
		IdleTimeout: 50 * time.Millisecond,
	})
	defer mgr.Stop()

	mgr.GetOrCreate("ch", "u")
	if mgr.Count() != 1 {
		t.Fatalf("expected 1 session")
	}
	// Wait for eviction (cleanup runs at IdleTimeout/2 = 25ms)
	time.Sleep(200 * time.Millisecond)
	if mgr.Count() != 0 {
		t.Errorf("expected 0 sessions after idle eviction, got %d", mgr.Count())
	}
}

// ── Session ───────────────────────────────────────────────────────────────────

func TestSessionSendReceivesEvent(t *testing.T) {
	mgr := gateway.NewSessionManager(testFactory(), gateway.ManagerConfig{})
	defer mgr.Stop()

	sess, _ := mgr.GetOrCreate("webchat", "testuser")

	gotEnd := make(chan struct{}, 1)
	unsub := sess.Subscribe(func(ev types.AgentEvent) {
		if ev.Type == types.EventAgentEnd {
			gotEnd <- struct{}{}
		}
	})
	defer unsub()

	sess.Send(gateway.IncomingMessage{
		ChannelID: "webchat",
		UserID:    "testuser",
		Content:   "hello",
	})

	select {
	case <-gotEnd:
		// pass
	case <-time.After(3 * time.Second):
		t.Error("timeout waiting for EventAgentEnd")
	}
}

func TestSessionSubscribeUnsubscribe(t *testing.T) {
	mgr := gateway.NewSessionManager(testFactory(), gateway.ManagerConfig{})
	defer mgr.Stop()

	sess, _ := mgr.GetOrCreate("ch", "u")

	var count int
	unsub := sess.Subscribe(func(_ types.AgentEvent) { count++ })
	unsub() // immediately unsubscribe

	sess.Send(gateway.IncomingMessage{ChannelID: "ch", UserID: "u", Content: "hi"})
	time.Sleep(200 * time.Millisecond)

	if count != 0 {
		t.Errorf("expected 0 events after unsub, got %d", count)
	}
}

// ── DedupCache (via Server) ───────────────────────────────────────────────────

func TestServerDedup(t *testing.T) {
	mgr := gateway.NewSessionManager(testFactory(), gateway.ManagerConfig{})
	defer mgr.Stop()

	reg := gateway.NewAgentRegistry()
	reg.Register("test", mgr)
	srv := gateway.NewServer(reg, gateway.ServerConfig{Addr: ":0"})

	if srv.IsDuplicate("msg-1") {
		t.Error("first time should not be duplicate")
	}
	if !srv.IsDuplicate("msg-1") {
		t.Error("second time should be duplicate")
	}
	if srv.IsDuplicate("msg-2") {
		t.Error("different ID should not be duplicate")
	}
}
