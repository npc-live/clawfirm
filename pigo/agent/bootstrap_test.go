package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBootstrapContext_NoFiles(t *testing.T) {
	dir := t.TempDir()
	result := LoadBootstrapContext(dir)
	if len(result.ContextFiles) != 0 {
		t.Errorf("expected 0 context files, got %d", len(result.ContextFiles))
	}
	if len(result.WorkspaceNotes) != 0 {
		t.Errorf("expected 0 workspace notes, got %d", len(result.WorkspaceNotes))
	}
}

func TestLoadBootstrapContext_WithAgentsMD(t *testing.T) {
	dir := t.TempDir()
	content := "# My Project\n\nThis is the agents file."
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadBootstrapContext(dir)
	if len(result.ContextFiles) != 1 {
		t.Fatalf("expected 1 context file, got %d", len(result.ContextFiles))
	}
	if result.ContextFiles[0].Name != "AGENTS.md" {
		t.Errorf("expected AGENTS.md, got %q", result.ContextFiles[0].Name)
	}
	if result.ContextFiles[0].Content != content {
		t.Errorf("content mismatch: got %q", result.ContextFiles[0].Content)
	}
	if len(result.WorkspaceNotes) == 0 {
		t.Error("expected workspace note for AGENTS.md presence")
	}
	if !strings.Contains(result.WorkspaceNotes[0], "commit") {
		t.Errorf("workspace note should mention commit, got %q", result.WorkspaceNotes[0])
	}
}

func TestLoadBootstrapContext_WithClaudeMD(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Claude instructions"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadBootstrapContext(dir)
	if len(result.ContextFiles) != 1 {
		t.Fatalf("expected 1 context file, got %d", len(result.ContextFiles))
	}
	if result.ContextFiles[0].Name != "CLAUDE.md" {
		t.Errorf("expected CLAUDE.md, got %q", result.ContextFiles[0].Name)
	}
	if len(result.WorkspaceNotes) == 0 {
		t.Error("expected workspace note for CLAUDE.md presence")
	}
}

func TestLoadBootstrapContext_WithSOULMD(t *testing.T) {
	dir := t.TempDir()
	// SOUL.md alone should NOT generate workspace notes.
	if err := os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("# Soul"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadBootstrapContext(dir)
	if len(result.ContextFiles) != 1 {
		t.Fatalf("expected 1 context file, got %d", len(result.ContextFiles))
	}
	if result.ContextFiles[0].Name != "SOUL.md" {
		t.Errorf("expected SOUL.md, got %q", result.ContextFiles[0].Name)
	}
	// SOUL.md alone does not trigger workspace note.
	if len(result.WorkspaceNotes) != 0 {
		t.Errorf("expected 0 workspace notes for SOUL.md only, got %d", len(result.WorkspaceNotes))
	}
}

func TestLoadBootstrapContext_AllFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"AGENTS.md": "agents content",
		"CLAUDE.md": "claude content",
		"SOUL.md":   "soul content",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result := LoadBootstrapContext(dir)
	if len(result.ContextFiles) != 3 {
		t.Fatalf("expected 3 context files, got %d", len(result.ContextFiles))
	}
	// Should appear in order: AGENTS.md, CLAUDE.md, SOUL.md
	if result.ContextFiles[0].Name != "AGENTS.md" {
		t.Errorf("first file should be AGENTS.md, got %q", result.ContextFiles[0].Name)
	}
}

func TestLoadBootstrapContext_Truncation(t *testing.T) {
	dir := t.TempDir()
	// Write a file larger than bootstrapFileMaxChars (20000 chars).
	big := strings.Repeat("x", bootstrapFileMaxChars+5000)
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadBootstrapContext(dir)
	if len(result.ContextFiles) != 1 {
		t.Fatalf("expected 1 context file, got %d", len(result.ContextFiles))
	}
	content := result.ContextFiles[0].Content
	if len(content) >= len(big) {
		t.Error("expected truncated content to be shorter than original")
	}
	if !strings.Contains(content, "truncated") {
		t.Error("truncated content should contain truncation notice")
	}
}

func TestLoadBootstrapContext_EmptyDir(t *testing.T) {
	result := LoadBootstrapContext("")
	// Should not panic; empty dir = use cwd
	_ = result
}
