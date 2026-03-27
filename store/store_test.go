package store_test

import (
	"path/filepath"
	"testing"

	"github.com/ai-gateway/pi-go/store"
	"github.com/ai-gateway/pi-go/types"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ── KVStore ──────────────────────────────────────────────────────────────────

func TestKVSetGet(t *testing.T) {
	kv := openTestDB(t).KV()

	if err := kv.Set("foo", "bar"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var got string
	if err := kv.Get("foo", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "bar" {
		t.Errorf("Get: want %q got %q", "bar", got)
	}
}

func TestKVOverwrite(t *testing.T) {
	kv := openTestDB(t).KV()
	_ = kv.Set("k", 1)
	_ = kv.Set("k", 2)
	var got int
	_ = kv.Get("k", &got)
	if got != 2 {
		t.Errorf("overwrite: want 2 got %d", got)
	}
}

func TestKVNotFound(t *testing.T) {
	kv := openTestDB(t).KV()
	var dst string
	if err := kv.Get("missing", &dst); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestKVDelete(t *testing.T) {
	kv := openTestDB(t).KV()
	_ = kv.Set("del", "v")
	_ = kv.Delete("del")
	var dst string
	if err := kv.Get("del", &dst); err != store.ErrNotFound {
		t.Errorf("after Delete: expected ErrNotFound, got %v", err)
	}
}

func TestKVStruct(t *testing.T) {
	type cfg struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}
	kv := openTestDB(t).KV()
	_ = kv.Set("config", cfg{Name: "pi-go", Port: 9988})

	var got cfg
	if err := kv.Get("config", &got); err != nil {
		t.Fatalf("Get struct: %v", err)
	}
	if got.Name != "pi-go" || got.Port != 9988 {
		t.Errorf("struct: got %+v", got)
	}
}

// ── MessageStore ─────────────────────────────────────────────────────────────

func userMsg(text string) *types.UserMessage {
	return &types.UserMessage{Role: "user", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: text},
	}}
}

func assistantMsg(text string) *types.AssistantMessage {
	return &types.AssistantMessage{Role: "assistant", Content: []types.ContentBlock{
		&types.TextContent{Type: types.ContentTypeText, Text: text},
	}}
}

func TestMessageSaveList(t *testing.T) {
	msgs := openTestDB(t).Messages()

	_ = msgs.SaveMessage("webchat", "user1", userMsg("hello"))
	_ = msgs.SaveMessage("webchat", "user1", assistantMsg("world"))
	_ = msgs.SaveMessage("webchat", "user2", userMsg("other user"))

	// user1 should have 2 messages
	out, err := msgs.ListMessages(store.QueryParams{ChannelID: "webchat", UserID: "user1"})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("ListMessages: want 2 got %d", len(out))
	}
	if _, ok := out[0].(*types.UserMessage); !ok {
		t.Errorf("want UserMessage at [0], got %T", out[0])
	}
	if _, ok := out[1].(*types.AssistantMessage); !ok {
		t.Errorf("want AssistantMessage at [1], got %T", out[1])
	}
}

func TestMessageRoundtrip(t *testing.T) {
	msgs := openTestDB(t).Messages()

	original := userMsg("round-trip test")
	_ = msgs.SaveMessage("ch", "u", original)

	out, err := msgs.ListMessages(store.QueryParams{ChannelID: "ch", UserID: "u"})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 message, got %d", len(out))
	}
	um, ok := out[0].(*types.UserMessage)
	if !ok {
		t.Fatalf("want *types.UserMessage, got %T", out[0])
	}
	tc, ok := um.Content[0].(*types.TextContent)
	if !ok || tc.Text != "round-trip test" {
		t.Errorf("content mismatch: %+v", um.Content)
	}
}

func TestMessagePagination(t *testing.T) {
	msgs := openTestDB(t).Messages()
	for i := range 5 {
		_ = msgs.SaveMessage("ch", "u", userMsg(string(rune('a'+i))))
	}

	page1, _ := msgs.ListMessages(store.QueryParams{ChannelID: "ch", UserID: "u", Limit: 3, Offset: 0})
	page2, _ := msgs.ListMessages(store.QueryParams{ChannelID: "ch", UserID: "u", Limit: 3, Offset: 3})
	if len(page1) != 3 {
		t.Errorf("page1: want 3 got %d", len(page1))
	}
	if len(page2) != 2 {
		t.Errorf("page2: want 2 got %d", len(page2))
	}
}

func TestMessageCount(t *testing.T) {
	msgs := openTestDB(t).Messages()
	_ = msgs.SaveMessage("ch", "u1", userMsg("a"))
	_ = msgs.SaveMessage("ch", "u2", userMsg("b"))
	_ = msgs.SaveMessage("other", "u1", userMsg("c"))

	n, err := msgs.CountByChannel("ch")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 2 {
		t.Errorf("Count: want 2 got %d", n)
	}
}

func TestMigrationIdempotent(t *testing.T) {
	// Opening twice should not fail (migrations are idempotent).
	path := filepath.Join(t.TempDir(), "idem.db")
	db1, err := store.Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db1.Close()

	db2, err := store.Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	db2.Close()
}
