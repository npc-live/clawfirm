package memory

import "strings"

// Chunk represents a text fragment extracted from a Markdown file.
type Chunk struct {
	Content   string
	StartLine int // 1-based inclusive
	EndLine   int // 1-based inclusive
}

// ChunkOptions controls the splitting behaviour.
type ChunkOptions struct {
	// MaxTokens is the target chunk size in estimated tokens (default 400).
	MaxTokens int
	// OverlapTokens is the number of tokens carried over from the previous chunk (default 80).
	OverlapTokens int
}

func (o ChunkOptions) maxTokens() int {
	if o.MaxTokens <= 0 {
		return 400
	}
	return o.MaxTokens
}

func (o ChunkOptions) overlapTokens() int {
	if o.OverlapTokens <= 0 {
		return 80
	}
	return o.OverlapTokens
}

// estimateTokens returns a rough token count for s using the rule-of-thumb
// that 1 token ≈ 4 characters of English text.
func estimateTokens(s string) int {
	n := len(s) / 4
	if n == 0 && len(s) > 0 {
		return 1
	}
	return n
}

// SplitIntoChunks splits text into overlapping chunks by token estimate.
// Splitting is done on paragraph (blank-line) boundaries first, then on
// sentence / line boundaries when a paragraph is too large.
func SplitIntoChunks(text string, opts ChunkOptions) []Chunk {
	maxTok := opts.maxTokens()
	overlapTok := opts.overlapTokens()

	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return nil
	}

	// Group lines into paragraphs (separated by blank lines).
	type para struct {
		lines     []string
		startLine int // 1-based
	}
	var paragraphs []para
	cur := para{startLine: 1}
	for i, line := range lines {
		lineNo := i + 1
		if strings.TrimSpace(line) == "" {
			if len(cur.lines) > 0 {
				paragraphs = append(paragraphs, cur)
				cur = para{startLine: lineNo + 1}
			} else {
				cur.startLine = lineNo + 1
			}
		} else {
			cur.lines = append(cur.lines, line)
		}
	}
	if len(cur.lines) > 0 {
		paragraphs = append(paragraphs, cur)
	}

	// Accumulate paragraphs into chunks.
	var chunks []Chunk
	var buf []string   // current chunk lines
	bufStart := 1      // 1-based start line of buf
	bufTokens := 0

	flush := func(endLine int) {
		if len(buf) == 0 {
			return
		}
		content := strings.Join(buf, "\n")
		chunks = append(chunks, Chunk{
			Content:   content,
			StartLine: bufStart,
			EndLine:   endLine,
		})
	}

	carryOverLines := func() ([]string, int) {
		// Keep the last N tokens worth of lines as overlap.
		if len(buf) == 0 {
			return nil, 0
		}
		carried := 0
		start := len(buf)
		for start > 0 && carried < overlapTok {
			start--
			carried += estimateTokens(buf[start])
		}
		overlap := make([]string, len(buf)-start)
		copy(overlap, buf[start:])
		return overlap, start
	}

	for _, p := range paragraphs {
		pText := strings.Join(p.lines, "\n")
		pTok := estimateTokens(pText)
		pEnd := p.startLine + len(p.lines) - 1

		if bufTokens+pTok <= maxTok {
			// Paragraph fits in current chunk.
			if len(buf) == 0 {
				bufStart = p.startLine
			}
			buf = append(buf, p.lines...)
			bufTokens += pTok
			continue
		}

		// Flush current buffer before starting new chunk.
		if len(buf) > 0 {
			flush(bufStart + len(buf) - 1)
			// Overlap: carry over last N tokens.
			overlapLines, splitAt := carryOverLines()
			if splitAt < len(buf) {
				newStart := bufStart + splitAt
				buf = overlapLines
				bufStart = newStart
				bufTokens = estimateTokens(strings.Join(buf, "\n"))
			} else {
				buf = nil
				bufTokens = 0
				bufStart = p.startLine
			}
		}

		// If the paragraph itself exceeds maxTokens, split it line by line.
		if pTok > maxTok {
			for i, line := range p.lines {
				lineNo := p.startLine + i
				lTok := estimateTokens(line)
				if bufTokens+lTok > maxTok && len(buf) > 0 {
					flush(lineNo - 1)
					overlapLines, splitAt := carryOverLines()
					if splitAt < len(buf) {
						newStart := bufStart + splitAt
						buf = overlapLines
						bufStart = newStart
						bufTokens = estimateTokens(strings.Join(buf, "\n"))
					} else {
						buf = nil
						bufTokens = 0
						bufStart = lineNo
					}
				}
				if len(buf) == 0 {
					bufStart = lineNo
				}
				buf = append(buf, line)
				bufTokens += lTok
			}
		} else {
			// Start fresh chunk with this paragraph.
			if len(buf) == 0 {
				bufStart = p.startLine
			}
			buf = append(buf, p.lines...)
			bufTokens += pTok
		}
		_ = pEnd
	}

	// Flush remaining content.
	if len(buf) > 0 {
		flush(bufStart + len(buf) - 1)
	}

	return chunks
}
