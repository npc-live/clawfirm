// Package memory implements a semantic memory system backed by SQLite FTS5 and
// dense vector embeddings. Markdown files under a watched directory are chunked,
// embedded, and indexed. Retrieval uses a hybrid BM25 + cosine-similarity score.
//
// Data flow: Markdown files → chunks → embeddings → SQLite → hybrid search.
package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config holds options for the Manager.
type Config struct {
	// MemoryDir is the directory containing Markdown memory files.
	// Defaults to ~/.pi-go/memory/.
	MemoryDir string
	// ChunkTokens is the target chunk size (default 400).
	ChunkTokens int
	// ChunkOverlapTokens is the overlap between consecutive chunks (default 80).
	ChunkOverlapTokens int
	// HybridAlpha is the weight given to vector similarity (0–1, default 0.7).
	// The remainder (1-alpha) is given to BM25.
	HybridAlpha float64
	// TimeDecay enables exponential decay that boosts more-recent chunks.
	// Half-life is 30 days.
	TimeDecay bool
}

func (c Config) memoryDir() string {
	if c.MemoryDir != "" {
		return c.MemoryDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pi-go", "memory")
}

func (c Config) hybridAlpha() float64 {
	if c.HybridAlpha <= 0 || c.HybridAlpha > 1 {
		return 0.7
	}
	return c.HybridAlpha
}

// SearchResult is a single hit returned by Search.
type SearchResult struct {
	FilePath  string
	StartLine int
	EndLine   int
	Content   string
	Score     float32
}

// Manager indexes Markdown files and answers semantic search queries.
type Manager struct {
	cfg    Config
	db     *sql.DB
	embed  EmbeddingProvider // may be nil → BM25-only mode
	mu     sync.Mutex
	syncAt time.Time // last full sync
}

// New creates a Manager. embedProvider may be nil to disable vector search.
func New(db *sql.DB, embedProvider EmbeddingProvider, cfg Config) *Manager {
	return &Manager{
		cfg:   cfg,
		db:    db,
		embed: embedProvider,
	}
}

// ─── Sync / indexing ──────────────────────────────────────────────────────────

// Sync rescans the memory directory and updates the index incrementally.
// Only files that have changed (by hash) since the last index are re-processed.
func (m *Manager) Sync(ctx context.Context) error {
	dir := m.cfg.memoryDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("memory: mkdir %s: %w", dir, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("memory: readdir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if err := m.indexFile(ctx, path); err != nil {
			// Log but continue with remaining files.
			_ = err
		}
	}

	m.mu.Lock()
	m.syncAt = time.Now()
	m.mu.Unlock()
	return nil
}

// IndexFile indexes a single Markdown file by path, re-indexing it only when
// its content has changed. It is safe to call concurrently.
func (m *Manager) IndexFile(ctx context.Context, path string) error {
	return m.indexFile(ctx, path)
}

func (m *Manager) indexFile(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("memory: read %s: %w", path, err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	modAt := info.ModTime().Unix()

	// Check if file is already indexed with the same hash.
	var fileID int64
	var storedHash string
	err = m.db.QueryRowContext(ctx,
		`SELECT id, hash FROM memory_files WHERE path = ?`, path,
	).Scan(&fileID, &storedHash)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("memory: query file: %w", err)
	}

	if err == nil && storedHash == hash {
		return nil // unchanged
	}

	// Chunk the file before opening the transaction so that embedWithCache
	// (which writes to the DB) does not contend with the open write transaction
	// under SQLite's single-writer constraint.
	chunks := SplitIntoChunks(string(data), ChunkOptions{
		MaxTokens:     m.cfg.ChunkTokens,
		OverlapTokens: m.cfg.ChunkOverlapTokens,
	})

	// Fetch embeddings outside the transaction.
	var embeddings [][]float32
	if m.embed != nil && len(chunks) > 0 {
		texts := make([]string, len(chunks))
		for i, c := range chunks {
			texts[i] = c.Content
		}
		if embs, embErr := m.embedWithCache(ctx, texts); embErr == nil {
			embeddings = embs
		}
		// Non-fatal: fall back to BM25-only if embedding fails.
	}

	// Upsert file record and replace chunks in a single transaction.
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if fileID == 0 {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO memory_files(path, hash, modified_at) VALUES(?,?,?)`,
			path, hash, modAt,
		)
		if err != nil {
			return fmt.Errorf("memory: insert file: %w", err)
		}
		fileID, _ = res.LastInsertId()
	} else {
		if _, err := tx.ExecContext(ctx,
			`UPDATE memory_files SET hash=?, modified_at=?, indexed_at=unixepoch() WHERE id=?`,
			hash, modAt, fileID,
		); err != nil {
			return fmt.Errorf("memory: update file: %w", err)
		}
		// Delete old chunks (triggers will clean FTS5).
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM memory_chunks WHERE file_id=?`, fileID,
		); err != nil {
			return fmt.Errorf("memory: delete chunks: %w", err)
		}
	}

	// Insert chunks.
	for i, c := range chunks {
		var embBlob []byte
		if embeddings != nil && i < len(embeddings) {
			embBlob = encodeFloat32s(embeddings[i])
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO memory_chunks(file_id, chunk_idx, content, start_line, end_line, embedding)
			 VALUES(?,?,?,?,?,?)`,
			fileID, i, c.Content, c.StartLine, c.EndLine, embBlob,
		); err != nil {
			return fmt.Errorf("memory: insert chunk: %w", err)
		}
	}

	return tx.Commit()
}

// DeleteFile removes a file and all its chunks from the index.
func (m *Manager) DeleteFile(ctx context.Context, path string) error {
	_, err := m.db.ExecContext(ctx, `DELETE FROM memory_files WHERE path = ?`, path)
	return err
}

// ─── Search ───────────────────────────────────────────────────────────────────

// Search performs a hybrid BM25 + cosine-similarity search and returns the
// top-k results ordered by score descending.
func (m *Manager) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}
	// Auto-sync before first search if never synced.
	m.mu.Lock()
	needsSync := m.syncAt.IsZero()
	m.mu.Unlock()
	if needsSync {
		_ = m.Sync(ctx)
	}

	bm25Results, err := m.bm25Search(ctx, query, topK*4)
	if err != nil {
		return nil, err
	}
	if len(bm25Results) == 0 {
		return nil, nil
	}

	// Vector re-rank when embedding is available.
	if m.embed != nil {
		queryEmb, err := m.embedWithCache(ctx, []string{query})
		if err == nil && len(queryEmb) > 0 {
			return m.hybridRerank(ctx, query, queryEmb[0], bm25Results, topK), nil
		}
	}

	// BM25-only: return top-k directly.
	if len(bm25Results) > topK {
		bm25Results = bm25Results[:topK]
	}
	return bm25Results, nil
}

// ftsQuery converts a free-text query into an FTS5 MATCH expression.
//
// The FTS5 trigram tokenizer requires each search token to be ≥ 3 Unicode code
// points.  Two transformations are applied:
//
//  1. ASCII / mixed words are quoted as-is.
//  2. CJK runs are slid into overlapping trigrams (window of 3 runes).
//     Strings shorter than 3 runes cannot produce valid trigrams; callers
//     should route those queries through likeSearch instead.
//
// All produced tokens are joined with OR so any matching token returns a hit.
// Returns ("", true) when the query consists entirely of short CJK words that
// must be handled by a LIKE scan; the second bool signals this condition.
func ftsQuery(q string) (expr string, needsLike bool) {
	var tokens []string
	allShortCJK := true
	for _, word := range strings.Fields(q) {
		runes := []rune(word)
		if !isCJKWord(word) {
			allShortCJK = false
			tokens = append(tokens, `"`+strings.ReplaceAll(word, `"`, `""`)+`"`)
			continue
		}
		if len(runes) < 3 {
			// Too short for a trigram; keep allShortCJK = true for this word.
			continue
		}
		allShortCJK = false
		for i := 0; i <= len(runes)-3; i++ {
			tri := string(runes[i : i+3])
			tokens = append(tokens, `"`+strings.ReplaceAll(tri, `"`, `""`)+`"`)
		}
	}
	if allShortCJK {
		return "", true
	}
	if len(tokens) == 0 {
		return q, false
	}
	return strings.Join(tokens, " OR "), false
}

// isCJKWord reports whether s consists primarily of CJK code-points.
func isCJKWord(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF { // CJK Unified Ideographs
			return true
		}
		if r >= 0x3000 && r <= 0x303F { // CJK Symbols and Punctuation
			return true
		}
		if r >= 0xAC00 && r <= 0xD7AF { // Hangul
			return true
		}
		if r >= 0x3040 && r <= 0x30FF { // Hiragana / Katakana
			return true
		}
	}
	return false
}

func (m *Manager) bm25Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	ftsExpr, needsLike := ftsQuery(query)
	if needsLike {
		return m.likeSearch(ctx, query, limit)
	}
	// FTS5 rank() returns negative values where lower (more negative) = better.
	rows, err := m.db.QueryContext(ctx, `
		SELECT mc.id, mf.path, mc.start_line, mc.end_line, mc.content, -fts.rank
		FROM memory_chunks_fts fts
		JOIN memory_chunks mc ON mc.id = fts.rowid
		JOIN memory_files mf ON mf.id = mc.file_id
		WHERE memory_chunks_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, ftsExpr, limit)
	if err != nil {
		return nil, fmt.Errorf("memory: fts search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	var maxScore float32
	for rows.Next() {
		var r SearchResult
		var score float32
		if err := rows.Scan(&sql.NullInt64{}, &r.FilePath, &r.StartLine, &r.EndLine, &r.Content, &score); err != nil {
			continue
		}
		r.Score = score
		if score > maxScore {
			maxScore = score
		}
		results = append(results, r)
	}
	// Normalise BM25 scores to [0,1].
	if maxScore > 0 {
		for i := range results {
			results[i].Score /= maxScore
		}
	}
	return results, rows.Err()
}

// likeSearch is a fallback for queries that are too short for the trigram FTS5
// index (e.g. 2-character CJK words). It scans memory_chunks.content with LIKE
// and returns results with a uniform score of 1.0.
func (m *Manager) likeSearch(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	pattern := "%" + strings.ReplaceAll(query, "%", "\\%") + "%"
	rows, err := m.db.QueryContext(ctx, `
		SELECT mf.path, mc.start_line, mc.end_line, mc.content
		FROM memory_chunks mc
		JOIN memory_files mf ON mf.id = mc.file_id
		WHERE mc.content LIKE ? ESCAPE '\'
		LIMIT ?`, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("memory: like search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.FilePath, &r.StartLine, &r.EndLine, &r.Content); err != nil {
			continue
		}
		r.Score = 1.0
		results = append(results, r)
	}
	return results, rows.Err()
}

func (m *Manager) hybridRerank(
	_ context.Context,
	_ string,
	queryEmb []float32,
	bm25 []SearchResult,
	topK int,
) []SearchResult {
	alpha := m.cfg.hybridAlpha()

	out := make([]scored, 0, len(bm25))

	for _, r := range bm25 {
		// Fetch embedding for this chunk.
		var embBlob []byte
		err := m.db.QueryRow(`
			SELECT mc.embedding
			FROM memory_chunks mc
			JOIN memory_files mf ON mf.id = mc.file_id
			WHERE mf.path = ? AND mc.start_line = ?`, r.FilePath, r.StartLine,
		).Scan(&embBlob)

		var cosSim float32
		if err == nil && len(embBlob) > 0 {
			chunkEmb := decodeFloat32s(embBlob)
			if len(chunkEmb) == len(queryEmb) {
				cosSim = cosine(queryEmb, chunkEmb)
			}
		}

		final := float32(alpha)*cosSim + float32(1-alpha)*r.Score
		if m.cfg.TimeDecay {
			final *= timeDecayFactor(r.FilePath, m.db)
		}
		out = append(out, scored{r: r, final: final})
	}

	// Sort descending.
	sortScored(out)

	results := make([]SearchResult, 0, topK)
	for i, s := range out {
		if i >= topK {
			break
		}
		s.r.Score = s.final
		results = append(results, s.r)
	}
	return results
}

// ─── Embedding cache ──────────────────────────────────────────────────────────

func (m *Manager) embedWithCache(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embed == nil {
		return nil, fmt.Errorf("memory: no embed provider")
	}
	provider := m.embed.Name()
	model := m.embed.Model()

	results := make([][]float32, len(texts))
	var uncached []int // indices of texts that need embedding

	for i, t := range texts {
		h := textHash(t)
		var blob []byte
		err := m.db.QueryRowContext(ctx,
			`SELECT embedding FROM memory_embedding_cache WHERE text_hash=? AND provider=? AND model=?`,
			h, provider, model,
		).Scan(&blob)
		if err == nil && len(blob) > 0 {
			results[i] = decodeFloat32s(blob)
		} else {
			uncached = append(uncached, i)
		}
	}

	if len(uncached) == 0 {
		return results, nil
	}

	batch := make([]string, len(uncached))
	for j, idx := range uncached {
		batch[j] = texts[idx]
	}
	embs, err := m.embed.Embed(ctx, batch)
	if err != nil {
		return nil, err
	}

	for j, idx := range uncached {
		if j >= len(embs) {
			break
		}
		results[idx] = embs[j]
		blob := encodeFloat32s(embs[j])
		h := textHash(texts[idx])
		_, _ = m.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO memory_embedding_cache(text_hash, provider, model, embedding)
			 VALUES(?,?,?,?)`, h, provider, model, blob,
		)
	}
	return results, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func textHash(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

// encodeFloat32s serialises a float32 slice as little-endian bytes.
func encodeFloat32s(vs []float32) []byte {
	b := make([]byte, len(vs)*4)
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v))
	}
	return b
}

// decodeFloat32s deserialises little-endian bytes into a float32 slice.
func decodeFloat32s(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	vs := make([]float32, len(b)/4)
	for i := range vs {
		bits := binary.LittleEndian.Uint32(b[i*4:])
		vs[i] = math.Float32frombits(bits)
	}
	return vs
}

// cosine computes the cosine similarity between two equal-length float32 slices.
func cosine(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	v := dot / denom
	// Clamp to [-1, 1] before converting.
	if v > 1 {
		v = 1
	} else if v < -1 {
		v = -1
	}
	return float32(v)
}

// timeDecayFactor returns an exponential decay factor for a file based on its
// indexed_at timestamp.  Half-life = 30 days; recent files score near 1.0.
func timeDecayFactor(filePath string, db *sql.DB) float32 {
	var indexedAt int64
	err := db.QueryRow(`SELECT indexed_at FROM memory_files WHERE path = ?`, filePath).Scan(&indexedAt)
	if err != nil {
		return 1.0
	}
	age := time.Since(time.Unix(indexedAt, 0)).Hours() / 24 // days
	halfLife := 30.0
	return float32(math.Exp2(-age / halfLife))
}

type scored struct {
	r     SearchResult
	final float32
}

// sortScored sorts in-place by final score descending (highest first).
func sortScored(s []scored) {
	// Simple insertion sort (results are typically small, ≤ 40 items).
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j].final < key.final {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

// ReadLines returns the content of a file between startLine and endLine (1-based, inclusive).
func ReadLines(filePath string, startLine, endLine int) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	lo := startLine - 1
	hi := endLine
	if lo < 0 {
		lo = 0
	}
	if hi > len(lines) {
		hi = len(lines)
	}
	if lo >= hi {
		return "", nil
	}
	return strings.Join(lines[lo:hi], "\n"), nil
}
