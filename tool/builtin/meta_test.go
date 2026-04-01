package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ai-gateway/clawfirm/tool"
)

func TestSkillCache_HitAndExpiry(t *testing.T) {
	// Create a temp skill file.
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "greet")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("Hello v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Skill{SkillPaths: []string{dir}}

	// First call: cache miss, loads from disk.
	content, err := s.loadSkill("greet")
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if content != "Hello v1" {
		t.Fatalf("first load: got %q want %q", content, "Hello v1")
	}

	// Update file on disk.
	if err := os.WriteFile(skillFile, []byte("Hello v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second call: cache hit, should still return v1.
	content, err = s.loadSkill("greet")
	if err != nil {
		t.Fatalf("cached load: %v", err)
	}
	if content != "Hello v1" {
		t.Fatalf("cached load: got %q want %q (expected cached value)", content, "Hello v1")
	}

	// Expire the cache entry manually.
	s.mu.Lock()
	s.cache["greet"].loadedAt = time.Now().Add(-2 * skillCacheTTL)
	s.mu.Unlock()

	// Third call: expired, should reload from disk.
	content, err = s.loadSkill("greet")
	if err != nil {
		t.Fatalf("expired load: %v", err)
	}
	if content != "Hello v2" {
		t.Fatalf("expired load: got %q want %q", content, "Hello v2")
	}
}

func TestSkillCache_InvalidateCache(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Skill{SkillPaths: []string{dir}}

	// Load to populate cache.
	if _, err := s.loadSkill("test"); err != nil {
		t.Fatal(err)
	}

	// Update file.
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("updated"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Still cached.
	content, _ := s.loadSkill("test")
	if content != "original" {
		t.Fatalf("expected cached 'original', got %q", content)
	}

	// Invalidate.
	s.InvalidateCache()

	// Should reload from disk.
	content, err := s.loadSkill("test")
	if err != nil {
		t.Fatal(err)
	}
	if content != "updated" {
		t.Fatalf("after invalidate: got %q want %q", content, "updated")
	}
}

func TestSkillCache_NotFound(t *testing.T) {
	s := &Skill{SkillPaths: []string{t.TempDir()}}
	_, err := s.loadSkill("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing skill")
	}

	// Not-found should NOT be cached.
	s.mu.RLock()
	_, cached := s.cache["nonexistent"]
	s.mu.RUnlock()
	if cached {
		t.Error("not-found result should not be cached")
	}
}

func TestSkillExecute_Integration(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "hello")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Hello Skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Skill{SkillPaths: []string{dir}}
	result, err := s.Execute(context.Background(), "call1", map[string]any{"name": "hello"}, func(tool.ToolUpdate) {})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
}

func TestSkillCache_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "concurrent")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("concurrent skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Skill{SkillPaths: []string{dir}}

	// Run concurrent loads — should not panic or race.
	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = s.loadSkill("concurrent")
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}
