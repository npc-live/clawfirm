package memory_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ai-gateway/pi-go/memory"
	"github.com/ai-gateway/pi-go/store"
)

// openTestDB opens an in-memory SQLite DB with migrations applied.
func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestChunking(t *testing.T) {
	text := "Line one.\nLine two.\nLine three.\n\nLine five.\nLine six."
	chunks := memory.SplitIntoChunks(text, memory.ChunkOptions{MaxTokens: 5, OverlapTokens: 1})
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	for _, c := range chunks {
		if c.StartLine < 1 {
			t.Errorf("invalid StartLine %d", c.StartLine)
		}
		if c.EndLine < c.StartLine {
			t.Errorf("EndLine %d < StartLine %d", c.EndLine, c.StartLine)
		}
		if c.Content == "" {
			t.Error("empty chunk content")
		}
	}
}

func TestIndexAndSearch_BM25Only(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()

	// Write a Markdown file.
	content := "# Notes\n\nThe project uses Go and SQLite for storage.\n\nWe prefer pure-Go drivers to avoid CGO."
	if err := os.WriteFile(filepath.Join(dir, "2026-01-01.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := memory.New(db.SQL(), nil, memory.Config{MemoryDir: dir})

	ctx := context.Background()
	if err := mgr.Sync(ctx); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	results, err := mgr.Search(ctx, "SQLite storage", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one search result")
	}
	if results[0].Score <= 0 {
		t.Errorf("expected positive score, got %f", results[0].Score)
	}
}

func TestIndexAndSearch_Incremental(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")

	writeFile := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mgr := memory.New(db.SQL(), nil, memory.Config{MemoryDir: dir})
	ctx := context.Background()

	writeFile("First version of the notes.")
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	res1, _ := mgr.Search(ctx, "first version", 5)
	if len(res1) == 0 {
		t.Fatal("expected result for first version")
	}

	// Update the file.
	writeFile("Completely different content about databases.")
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	res2, _ := mgr.Search(ctx, "databases", 5)
	if len(res2) == 0 {
		t.Fatal("expected result after update")
	}
}

func TestReadLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	_ = os.WriteFile(path, []byte("line1\nline2\nline3\nline4\nline5"), 0o644)

	got, err := memory.ReadLines(path, 2, 4)
	if err != nil {
		t.Fatalf("ReadLines: %v", err)
	}
	want := "line2\nline3\nline4"
	if got != want {
		t.Errorf("ReadLines(2,4) = %q, want %q", got, want)
	}
}
