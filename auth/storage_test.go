package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func tempStoragePath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "auth.json")
}

func TestSetGetAPIKey(t *testing.T) {
	s := NewAuthStorage(tempStoragePath(t))
	if err := s.SetAPIKey("anthropic", "test-key-123"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}
	got, ok := s.GetAPIKey("anthropic")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "test-key-123" {
		t.Errorf("GetAPIKey: got %q want test-key-123", got)
	}
}

func TestRuntimeKeyPriority(t *testing.T) {
	s := NewAuthStorage(tempStoragePath(t))
	if err := s.SetAPIKey("openai", "stored-key"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}
	s.SetRuntimeKey("openai", "runtime-key")
	got, ok := s.GetAPIKey("openai")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "runtime-key" {
		t.Errorf("expected runtime-key, got %q", got)
	}
}

func TestMultipleProvidersIndependent(t *testing.T) {
	s := NewAuthStorage(tempStoragePath(t))
	if err := s.SetAPIKey("anthropic", "ant-key"); err != nil {
		t.Fatalf("SetAPIKey anthropic: %v", err)
	}
	if err := s.SetAPIKey("openai", "oai-key"); err != nil {
		t.Fatalf("SetAPIKey openai: %v", err)
	}
	ant, _ := s.GetAPIKey("anthropic")
	oai, _ := s.GetAPIKey("openai")
	if ant != "ant-key" {
		t.Errorf("anthropic: got %q want ant-key", ant)
	}
	if oai != "oai-key" {
		t.Errorf("openai: got %q want oai-key", oai)
	}
}

func TestPersistAndReload(t *testing.T) {
	path := tempStoragePath(t)
	s1 := NewAuthStorage(path)
	if err := s1.SetAPIKey("gemini", "gem-key"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}

	// Load in a fresh instance
	s2 := NewAuthStorage(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, ok := s2.GetAPIKey("gemini")
	if !ok {
		t.Fatal("expected ok=true after reload")
	}
	if got != "gem-key" {
		t.Errorf("reloaded key: got %q want gem-key", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	s := NewAuthStorage(path)
	if err := s.Load(); err != nil {
		t.Errorf("Load on missing file should not error: %v", err)
	}
}

func TestGetAPIKeyMissing(t *testing.T) {
	s := NewAuthStorage(tempStoragePath(t))
	_, ok := s.GetAPIKey("unknown")
	if ok {
		t.Error("expected ok=false for missing key")
	}
}

func TestSetGetOAuth(t *testing.T) {
	path := tempStoragePath(t)
	s := NewAuthStorage(path)
	creds := OAuthCredentials{
		Refresh: "refresh-tok",
		Access:  "access-tok",
	}
	if err := s.SetOAuth("google", creds); err != nil {
		t.Fatalf("SetOAuth: %v", err)
	}
	got, ok := s.GetOAuth("google")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.Refresh != "refresh-tok" {
		t.Errorf("Refresh: got %q want refresh-tok", got.Refresh)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "auth.json")
	s := NewAuthStorage(path)
	if err := s.SetAPIKey("test", "val"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}
