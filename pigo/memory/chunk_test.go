package memory

import (
	"strings"
	"testing"
)

// ─── estimateTokens ───────────────────────────────────────────────────────────

func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"a", 1},   // <4 chars but non-empty → 1
		{"abcd", 1},
		{"abcde", 1},
		{"abcdefgh", 2},
		{strings.Repeat("x", 100), 25},
	}
	for _, c := range cases {
		got := estimateTokens(c.in)
		if got != c.want {
			t.Errorf("estimateTokens(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// ─── SplitIntoChunks ─────────────────────────────────────────────────────────

func TestSplitIntoChunks_Empty(t *testing.T) {
	if got := SplitIntoChunks("", ChunkOptions{}); len(got) != 0 {
		t.Errorf("expected no chunks for empty text, got %d", len(got))
	}
}

func TestSplitIntoChunks_SingleLine(t *testing.T) {
	chunks := SplitIntoChunks("hello world", ChunkOptions{MaxTokens: 400})
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].StartLine != 1 || chunks[0].EndLine != 1 {
		t.Errorf("line range = %d–%d, want 1–1", chunks[0].StartLine, chunks[0].EndLine)
	}
	if chunks[0].Content != "hello world" {
		t.Errorf("content = %q, want %q", chunks[0].Content, "hello world")
	}
}

func TestSplitIntoChunks_MultiParagraph_AllFit(t *testing.T) {
	// Two small paragraphs that fit in one chunk.
	text := "Para one.\n\nPara two."
	chunks := SplitIntoChunks(text, ChunkOptions{MaxTokens: 400})
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if !strings.Contains(chunks[0].Content, "Para one") || !strings.Contains(chunks[0].Content, "Para two") {
		t.Errorf("combined content not found in single chunk: %q", chunks[0].Content)
	}
}

func TestSplitIntoChunks_ForceSplit_ParagraphBoundary(t *testing.T) {
	// Three paragraphs each ~3 tokens; maxTokens=5 → first two fit, third splits off.
	line := strings.Repeat("x", 12) // 3 tokens
	text := line + "\n\n" + line + "\n\n" + line
	chunks := SplitIntoChunks(text, ChunkOptions{MaxTokens: 5, OverlapTokens: 1})
	if len(chunks) < 2 {
		t.Fatalf("expected ≥2 chunks, got %d", len(chunks))
	}
	// All original text must appear in at least one chunk.
	all := ""
	for _, c := range chunks {
		all += c.Content
	}
	if !strings.Contains(all, line) {
		t.Error("original text not found in any chunk")
	}
}

func TestSplitIntoChunks_OversizeParagraph_LineSplit(t *testing.T) {
	// A single paragraph with 10 long lines — must be split line by line.
	longLine := strings.Repeat("w", 40) // 10 tokens each
	lines := make([]string, 10)
	for i := range 10 {
		lines[i] = longLine
	}
	text := strings.Join(lines, "\n")
	chunks := SplitIntoChunks(text, ChunkOptions{MaxTokens: 15, OverlapTokens: 3})
	if len(chunks) < 2 {
		t.Fatalf("oversize paragraph: expected ≥2 chunks, got %d: %+v", len(chunks), chunks)
	}
	for i, c := range chunks {
		if c.Content == "" {
			t.Errorf("chunk %d has empty content", i)
		}
		if c.StartLine < 1 {
			t.Errorf("chunk %d: StartLine %d < 1", i, c.StartLine)
		}
		if c.EndLine < c.StartLine {
			t.Errorf("chunk %d: EndLine %d < StartLine %d", i, c.EndLine, c.StartLine)
		}
	}
}

func TestSplitIntoChunks_LineNumbers_Sequential(t *testing.T) {
	// Verify that line numbers are monotonically non-decreasing.
	text := "a\nb\nc\n\nd\ne\nf\n\ng\nh"
	chunks := SplitIntoChunks(text, ChunkOptions{MaxTokens: 3, OverlapTokens: 1})
	prev := 0
	for i, c := range chunks {
		if c.StartLine <= prev && i > 0 {
			t.Errorf("chunk %d StartLine %d not > prev EndLine region", i, c.StartLine)
		}
		if c.EndLine < c.StartLine {
			t.Errorf("chunk %d: EndLine %d < StartLine %d", i, c.EndLine, c.StartLine)
		}
		prev = c.EndLine
	}
}

func TestSplitIntoChunks_DefaultOptions(t *testing.T) {
	// Zero-value options must not panic and must use defaults.
	text := strings.Repeat("word ", 50)
	chunks := SplitIntoChunks(text, ChunkOptions{})
	if len(chunks) == 0 {
		t.Error("expected at least one chunk with default options")
	}
}

func TestSplitIntoChunks_OnlyBlankLines(t *testing.T) {
	// All blank lines → no paragraphs → no chunks.
	chunks := SplitIntoChunks("\n\n\n", ChunkOptions{})
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for all-blank text, got %d", len(chunks))
	}
}

func TestSplitIntoChunks_TrailingNewline(t *testing.T) {
	// Trailing newline should not produce a phantom chunk.
	chunks := SplitIntoChunks("hello\n", ChunkOptions{MaxTokens: 400})
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
}
