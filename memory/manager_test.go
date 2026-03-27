package memory

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/ai-gateway/pi-go/store"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func openDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// mockEmbedder returns deterministic unit vectors based on text length.
// Identical texts → identical vectors; longer texts → larger first component.
type mockEmbedder struct {
	dims  int
	calls int // how many Embed calls were made
}

func (m *mockEmbedder) Name() string  { return "mock" }
func (m *mockEmbedder) Model() string { return "mock-v1" }
func (m *mockEmbedder) Dims() int     { return m.dims }

func (m *mockEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	m.calls++
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, m.dims)
		// First component encodes text length (normalised later).
		v[0] = float32(len(t)%100) + 1
		v[1] = 1.0
		// Normalise to unit vector so cosine similarity is well-defined.
		norm := float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1])))
		v[0] /= norm
		v[1] /= norm
		out[i] = v
	}
	return out, nil
}

// ─── encodeFloat32s / decodeFloat32s ─────────────────────────────────────────

func TestEncodeDecodeFloat32s_RoundTrip(t *testing.T) {
	in := []float32{0, 1, -1, 0.5, 1e-6, -1e6, math.MaxFloat32, math.SmallestNonzeroFloat32}
	b := encodeFloat32s(in)
	out := decodeFloat32s(b)
	if len(out) != len(in) {
		t.Fatalf("len mismatch: %d vs %d", len(out), len(in))
	}
	for i := range in {
		if in[i] != out[i] {
			t.Errorf("[%d] encode/decode mismatch: in=%v out=%v", i, in[i], out[i])
		}
	}
}

func TestDecodeFloat32s_BadLength(t *testing.T) {
	if got := decodeFloat32s([]byte{1, 2, 3}); got != nil {
		t.Errorf("expected nil for odd-length input, got %v", got)
	}
}

func TestEncodeDecodeFloat32s_Empty(t *testing.T) {
	b := encodeFloat32s(nil)
	if len(b) != 0 {
		t.Error("expected empty bytes for nil input")
	}
	out := decodeFloat32s(b)
	if len(out) != 0 {
		t.Error("expected empty slice for empty bytes")
	}
}

// ─── cosine ───────────────────────────────────────────────────────────────────

func TestCosine_IdenticalVectors(t *testing.T) {
	v := []float32{1, 0, 0}
	got := cosine(v, v)
	if math.Abs(float64(got-1)) > 1e-5 {
		t.Errorf("cosine(v,v) = %f, want 1.0", got)
	}
}

func TestCosine_Orthogonal(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0, 1}
	got := cosine(a, b)
	if math.Abs(float64(got)) > 1e-5 {
		t.Errorf("cosine(orthogonal) = %f, want 0.0", got)
	}
}

func TestCosine_Opposite(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{-1, 0}
	got := cosine(a, b)
	if math.Abs(float64(got+1)) > 1e-5 {
		t.Errorf("cosine(opposite) = %f, want -1.0", got)
	}
}

func TestCosine_ZeroVector(t *testing.T) {
	a := []float32{0, 0}
	b := []float32{1, 1}
	if got := cosine(a, b); got != 0 {
		t.Errorf("cosine(zero, v) = %f, want 0", got)
	}
}

func TestCosine_LengthMismatch(t *testing.T) {
	if got := cosine([]float32{1}, []float32{1, 2}); got != 0 {
		t.Errorf("cosine(len mismatch) = %f, want 0", got)
	}
}

func TestCosine_Empty(t *testing.T) {
	if got := cosine(nil, nil); got != 0 {
		t.Errorf("cosine(nil, nil) = %f, want 0", got)
	}
}

// ─── indexFile: hash-based deduplication ────────────────────────────────────

func TestIndexFile_UnchangedSkipped(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	_ = os.WriteFile(path, []byte("some content"), 0o644)

	emb := &mockEmbedder{dims: 4}
	mgr := New(db.SQL(), emb, Config{MemoryDir: dir})
	ctx := context.Background()

	if err := mgr.IndexFile(ctx, path); err != nil {
		t.Fatalf("first index: %v", err)
	}
	callsAfterFirst := emb.calls

	// Index the same file again — must be skipped (no new embed calls).
	if err := mgr.IndexFile(ctx, path); err != nil {
		t.Fatalf("second index: %v", err)
	}
	if emb.calls != callsAfterFirst {
		t.Errorf("embed called %d times on unchanged file, want %d", emb.calls, callsAfterFirst)
	}
}

func TestIndexFile_ContentChangeTriggersReindex(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")

	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})
	ctx := context.Background()

	_ = os.WriteFile(path, []byte("version one content"), 0o644)
	if err := mgr.IndexFile(ctx, path); err != nil {
		t.Fatal(err)
	}

	r1, _ := mgr.Search(ctx, "version one", 5)
	if len(r1) == 0 {
		t.Fatal("expected result for version one")
	}

	_ = os.WriteFile(path, []byte("completely new text about widgets"), 0o644)
	if err := mgr.IndexFile(ctx, path); err != nil {
		t.Fatal(err)
	}

	r2, _ := mgr.Search(ctx, "widgets", 5)
	if len(r2) == 0 {
		t.Fatal("expected result for updated content")
	}
}

// ─── DeleteFile ───────────────────────────────────────────────────────────────

func TestDeleteFile_RemovesFromIndex(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "to_delete.md")
	_ = os.WriteFile(path, []byte("unique keyword zephyr"), 0o644)

	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})
	ctx := context.Background()

	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	r1, _ := mgr.Search(ctx, "zephyr", 5)
	if len(r1) == 0 {
		t.Fatal("expected result before delete")
	}

	if err := mgr.DeleteFile(ctx, path); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}

	r2, _ := mgr.Search(ctx, "zephyr", 5)
	if len(r2) != 0 {
		t.Errorf("expected no results after delete, got %d", len(r2))
	}
}

// ─── embedding cache ─────────────────────────────────────────────────────────

func TestEmbedWithCache_CacheHit(t *testing.T) {
	db := openDB(t)
	emb := &mockEmbedder{dims: 4}
	mgr := New(db.SQL(), emb, Config{})
	ctx := context.Background()

	texts := []string{"hello", "world"}

	_, err := mgr.embedWithCache(ctx, texts)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	callsAfterFirst := emb.calls

	// Second call must be fully served from cache.
	_, err = mgr.embedWithCache(ctx, texts)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if emb.calls != callsAfterFirst {
		t.Errorf("embed called %d extra times on cache hit, want 0", emb.calls-callsAfterFirst)
	}
}

func TestEmbedWithCache_PartialCacheMiss(t *testing.T) {
	db := openDB(t)
	emb := &mockEmbedder{dims: 4}
	mgr := New(db.SQL(), emb, Config{})
	ctx := context.Background()

	// Warm cache with one text.
	_, _ = mgr.embedWithCache(ctx, []string{"cached text"})
	callsAfterWarm := emb.calls

	// Request the cached + one new text.
	res, err := mgr.embedWithCache(ctx, []string{"cached text", "new text"})
	if err != nil {
		t.Fatalf("partial miss: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 embeddings, got %d", len(res))
	}
	if emb.calls == callsAfterWarm {
		t.Error("expected at least one new embed call for new text")
	}
}

// ─── hybrid rerank ────────────────────────────────────────────────────────────

func TestHybridRerank_ScoreOrder(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()

	// Write two files with clearly different content.
	_ = os.WriteFile(filepath.Join(dir, "go.md"),
		[]byte("Go programming language is compiled and statically typed."), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "python.md"),
		[]byte("Python is an interpreted dynamically typed language."), 0o644)

	emb := &mockEmbedder{dims: 4}
	mgr := New(db.SQL(), emb, Config{MemoryDir: dir})
	ctx := context.Background()

	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	results, err := mgr.Search(ctx, "compiled statically typed language", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	// All returned scores must be ≥ 0.
	for i, r := range results {
		if r.Score < 0 {
			t.Errorf("result[%d] has negative score %f", i, r.Score)
		}
	}
	// Scores must be sorted descending.
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted: [%d]=%.3f > [%d]=%.3f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

// ─── Search edge cases ────────────────────────────────────────────────────────

func TestSearch_EmptyIndex(t *testing.T) {
	db := openDB(t)
	mgr := New(db.SQL(), nil, Config{MemoryDir: t.TempDir()})
	ctx := context.Background()

	results, err := mgr.Search(ctx, "anything", 5)
	if err != nil {
		t.Fatalf("unexpected error on empty index: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results on empty index, got %d", len(results))
	}
}

func TestSearch_TopKTruncates(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()

	// Write 5 files each containing the word "alpha".
	for i := range 5 {
		content := "alpha content number " + string(rune('A'+i)) +
			" with extra padding so each chunk is distinct and searchable"
		_ = os.WriteFile(filepath.Join(dir, string(rune('a'+i))+".md"), []byte(content), 0o644)
	}

	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})
	ctx := context.Background()
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	results, err := mgr.Search(ctx, "alpha", 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) > 2 {
		t.Errorf("expected at most 2 results (topK=2), got %d", len(results))
	}
}

func TestSearch_DefaultTopK(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "x.md"), []byte("beta gamma delta"), 0o644)

	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})
	ctx := context.Background()
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	// topK=0 should default to 5 without panicking.
	_, err := mgr.Search(ctx, "beta", 0)
	if err != nil {
		t.Fatalf("Search with topK=0: %v", err)
	}
}

// ─── Sync: ignores non-.md files ─────────────────────────────────────────────

func TestSync_IgnoresNonMarkdown(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("should not be indexed"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"key":"value"}`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "real.md"), []byte("this is markdown"), 0o644)

	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})
	ctx := context.Background()
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	var count int
	mgr.db.QueryRow(`SELECT COUNT(*) FROM memory_files`).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 indexed file (only .md), got %d", count)
	}
}

// ─── timeDecayFactor ─────────────────────────────────────────────────────────

func TestTimeDecayFactor_RecentIsNearOne(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "new.md")
	_ = os.WriteFile(path, []byte("fresh content"), 0o644)

	mgr := New(db.SQL(), nil, Config{MemoryDir: dir})
	ctx := context.Background()
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	factor := timeDecayFactor(path, mgr.db)
	// Just indexed → age ≈ 0 → factor ≈ 1.
	if factor < 0.99 {
		t.Errorf("decay factor for brand-new file = %f, want ≥0.99", factor)
	}
}

func TestTimeDecayFactor_MissingFile(t *testing.T) {
	db := openDB(t)
	// File not in DB → should return 1.0 (neutral).
	factor := timeDecayFactor("/nonexistent/path.md", db.SQL())
	if factor != 1.0 {
		t.Errorf("timeDecayFactor for missing file = %f, want 1.0", factor)
	}
}

// ─── ReadLines ────────────────────────────────────────────────────────────────

func TestReadLines_MissingFile(t *testing.T) {
	_, err := ReadLines("/nonexistent/path.md", 1, 1)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestReadLines_BeyondEOF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "short.md")
	_ = os.WriteFile(path, []byte("only one line"), 0o644)

	got, err := ReadLines(path, 1, 100)
	if err != nil {
		t.Fatalf("ReadLines: %v", err)
	}
	if got != "only one line" {
		t.Errorf("got %q, want %q", got, "only one line")
	}
}

func TestReadLines_StartBeyondEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.md")
	_ = os.WriteFile(path, []byte("a\nb\nc"), 0o644)

	got, err := ReadLines(path, 5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for out-of-range, got %q", got)
	}
}

func TestReadLines_ZeroStartLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.md")
	_ = os.WriteFile(path, []byte("first\nsecond"), 0o644)

	// startLine=0 should clamp to 1.
	got, err := ReadLines(path, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != "first" {
		t.Errorf("got %q, want %q", got, "first")
	}
}
