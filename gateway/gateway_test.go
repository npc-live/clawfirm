package gateway_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ai-gateway/clawfirm/gateway"
	"github.com/ai-gateway/clawfirm/types"
)

// mockRunner is a minimal AgentRunner that emits a text delta and agent-end
// synchronously when PromptMessages is called.
type mockRunner struct {
	mu        sync.Mutex
	listeners []func(types.AgentEvent)
}

func (m *mockRunner) PromptMessages(_ context.Context, _ []types.Message) error {
	go func() {
		m.emit(types.AgentEvent{Type: types.EventMessageUpdate, StreamEvent: &types.AssistantMessageEvent{Type: types.StreamEventTextDelta, Delta: "hi"}})
		m.emit(types.AgentEvent{Type: types.EventAgentEnd})
	}()
	return nil
}
func (m *mockRunner) Abort()                               {}
func (m *mockRunner) WaitForIdle(_ context.Context) error  { return nil }
func (m *mockRunner) ClearMessages()                       {}
func (m *mockRunner) State() types.AgentState              { return types.AgentState{} }
func (m *mockRunner) Subscribe(fn func(types.AgentEvent)) func() {
	m.mu.Lock()
	m.listeners = append(m.listeners, fn)
	idx := len(m.listeners) - 1
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		m.listeners[idx] = nil
		m.mu.Unlock()
	}
}
func (m *mockRunner) emit(ev types.AgentEvent) {
	m.mu.Lock()
	ls := make([]func(types.AgentEvent), len(m.listeners))
	copy(ls, m.listeners)
	m.mu.Unlock()
	for _, fn := range ls {
		if fn != nil {
			fn(ev)
		}
	}
}

func testFactory() gateway.AgentFactory {
	return func(_, _ string) gateway.AgentRunner {
		return &mockRunner{}
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
