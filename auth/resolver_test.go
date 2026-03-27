package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolverRuntimeKeyPriority(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	s := NewAuthStorage(path)
	if err := s.SetAPIKey("anthropic", "stored-key"); err != nil {
		t.Fatal(err)
	}
	s.SetRuntimeKey("anthropic", "runtime-key")

	r := NewAuthResolver(s)
	got, err := r.ResolveAPIKey(context.Background(), "anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "runtime-key" {
		t.Errorf("expected runtime-key, got %q", got)
	}
}

func TestResolverEnvVarPriority(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	s := NewAuthStorage(path)
	// No stored key, no runtime key
	os.Setenv("ANTHROPIC_API_KEY", "env-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	r := NewAuthResolver(s)
	got, err := r.ResolveAPIKey(context.Background(), "anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "env-key" {
		t.Errorf("expected env-key, got %q", got)
	}
}

func TestResolverStoredKeyOverEnvVar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	s := NewAuthStorage(path)
	if err := s.SetAPIKey("openai", "stored-oai"); err != nil {
		t.Fatal(err)
	}
	os.Setenv("OPENAI_API_KEY", "env-oai")
	defer os.Unsetenv("OPENAI_API_KEY")

	r := NewAuthResolver(s)
	got, err := r.ResolveAPIKey(context.Background(), "openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// stored key wins over env var (runtime > storage > env > oauth > keychain)
	// But our GetAPIKey returns stored if runtime is absent; env is checked after storage misses
	// Expect stored-oai since storage has it
	if got != "stored-oai" {
		t.Errorf("expected stored-oai, got %q", got)
	}
}

func TestResolverEnvVarFallsBackWhenStoredMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	s := NewAuthStorage(path)
	// No stored key
	os.Setenv("OPENAI_API_KEY", "env-fallback")
	defer os.Unsetenv("OPENAI_API_KEY")

	r := NewAuthResolver(s)
	got, err := r.ResolveAPIKey(context.Background(), "openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "env-fallback" {
		t.Errorf("expected env-fallback, got %q", got)
	}
}

func TestResolverEnvVarMapping(t *testing.T) {
	cases := []struct {
		provider string
		envVar   string
	}{
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"google", "GOOGLE_API_KEY"},
	}
	for _, c := range cases {
		path := filepath.Join(t.TempDir(), "auth.json")
		s := NewAuthStorage(path)
		os.Setenv(c.envVar, "test-val-"+c.provider)
		r := NewAuthResolver(s)
		got, err := r.ResolveAPIKey(context.Background(), c.provider)
		os.Unsetenv(c.envVar)
		if err != nil {
			t.Errorf("provider %q: unexpected error: %v", c.provider, err)
			continue
		}
		if got != "test-val-"+c.provider {
			t.Errorf("provider %q: got %q want test-val-%s", c.provider, got, c.provider)
		}
	}
}

func TestResolverNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	s := NewAuthStorage(path)
	r := NewAuthResolver(s)
	_, err := r.ResolveAPIKey(context.Background(), "unknown-provider-xyz")
	if err == nil {
		t.Error("expected error for unknown provider with no key")
	}
}
