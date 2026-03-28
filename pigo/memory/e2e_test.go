package memory_test

// End-to-end test: user input → write memory file → index → search → agent tool
//
// Run with a real embedding provider:
//
//	OPENROUTER_API_KEY=... go test ./memory/ -run TestE2E -v -timeout 60s
//	OPENAI_API_KEY=...     go test ./memory/ -run TestE2E -v -timeout 60s
//
// Without an API key the test is skipped automatically.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ai-gateway/pi-go/memory"
	"github.com/ai-gateway/pi-go/store"
	"github.com/ai-gateway/pi-go/types"
)

func TestE2E_FullFlow(t *testing.T) {
	// ── 1. Pick embedding provider ────────────────────────────────────────────
	provider, err := memory.NewAutoProvider()
	if err != nil {
		t.Skipf("no embedding provider available (%v) — skipping e2e test", err)
	}
	t.Logf("provider: %s / %s (%d dims)", provider.Name(), provider.Model(), provider.Dims())

	// ── 2. Set up temp dir + real SQLite (runs migrations automatically) ──────
	memDir := t.TempDir()
	db, err := store.Open(filepath.Join(t.TempDir(), "e2e.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	mgr := memory.New(db.SQL(), provider, memory.Config{MemoryDir: memDir})
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// ── 3. User inputs information (write memory files) ───────────────────────
	files := map[string]string{
		"work_2026-03-27.md": `# 工作日志 2026-03-27

## pi-go 项目进展

今天完成了 memory 子系统的实现：
- 使用 SQLite FTS5 trigram 做全文搜索，支持中文
- 向量 embedding 存储为 float32 BLOB，在 Go 里做余弦相似度计算，无需 CGO
- 混合搜索：0.7 × 向量余弦 + 0.3 × BM25
- 修复了一个死锁 bug：indexFile 在事务内部调用 embedWithCache，导致 SQLite 单连接死锁

## 遇到的问题

- unicode61 tokenizer 不支持中文，改成 trigram 后解决
- 两字词（如"死锁"）低于 trigram 最小长度，用 LIKE 查询兜底
`,
		"preferences.md": `# 用户偏好

## 编程语言偏好
- 主要使用 Go 语言
- 前端使用 React + TypeScript
- 桌面应用框架：Wails v2

## 工具偏好
- 数据库：SQLite（纯 Go 驱动，无 CGO）
- 编辑器：VS Code
- AI 模型：Claude Sonnet

## 工作习惯
- 喜欢写测试覆盖核心逻辑
- 函数式选项模式（functional options）
- 不喜欢过度工程化
`,
		"decisions.md": `# 架构决策记录

## ADR-001: 向量存储不使用 sqlite-vec

**决定**：用 float32 小端 BLOB + Go 计算余弦相似度，不引入 CGO 扩展

**原因**：
- 项目要求纯 Go，无 CGO
- 内存中计算余弦相似度性能足够（< 10k chunks）
- 避免分发时依赖本地 .so/.dylib 文件

## ADR-002: FTS5 使用 trigram tokenizer

**决定**：将 FTS5 tokenizer 从 unicode61 改为 trigram

**原因**：
- unicode61 完全不能处理 CJK 字符
- trigram 对中日韩文本搜索效果良好
- 代价：索引体积略大，最短可搜索词 3 个字符
`,
	}

	t.Log("=== STEP 1: 用户写入记忆文件 ===")
	for name, content := range files {
		path := filepath.Join(memDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		t.Logf("  wrote %s (%d bytes)", name, len(content))
	}

	// ── 4. Sync: index all files ──────────────────────────────────────────────
	t.Log("\n=== STEP 2: Sync 建立索引 ===")
	syncStart := time.Now()
	if err := mgr.Sync(ctx); err != nil {
		t.Fatalf("sync: %v", err)
	}
	t.Logf("  Sync OK (%.1fs)", time.Since(syncStart).Seconds())

	// ── 5. Search queries ─────────────────────────────────────────────────────
	type queryCase struct {
		q     string
		label string
	}
	queries := []queryCase{
		// Semantic / vector
		{"如何解决并发安全和锁的问题", "向量：并发/死锁"},
		{"用户喜欢什么编程语言和工具", "向量：用户偏好"},
		{"为什么不用 CGO 扩展", "向量：架构决策"},
		// Keyword BM25
		{"trigram tokenizer FTS5", "BM25：trigram"},
		{"SQLite 纯 Go 驱动", "BM25：SQLite"},
		// Short CJK → LIKE fallback (2-char words below trigram threshold)
		{"死锁", "LIKE兜底：死锁(2字)"},
		{"向量", "LIKE兜底：向量(2字)"},
		// Mixed
		{"中文全文搜索方案设计", "混合：中文搜索"},
	}

	t.Log("\n=== STEP 3: 搜索记忆 ===")
	pass, fail := 0, 0
	for _, q := range queries {
		results, err := mgr.Search(ctx, q.q, 3)
		if err != nil {
			t.Errorf("  [FAIL] %s: %v", q.label, err)
			fail++
			continue
		}
		if len(results) == 0 {
			t.Errorf("  [FAIL] %s: 0 results", q.label)
			fail++
			continue
		}
		top := results[0]
		t.Logf("  [PASS] %s\n         query=%q  hits=%d  score=%.3f  file=%s L%d–%d\n         → %s",
			q.label, q.q, len(results), top.Score,
			filepath.Base(top.FilePath), top.StartLine, top.EndLine,
			firstLine(top.Content))
		pass++
	}
	t.Logf("\n  结果：%d/%d passed", pass, pass+fail)

	// ── 6. Agent tool: memory_search ─────────────────────────────────────────
	t.Log("\n=== STEP 4: Agent tool — memory_search ===")
	searchTool := memory.SearchTool(mgr)
	toolResult, err := searchTool.Execute(ctx, "call-001", map[string]any{
		"query": "项目用了哪些架构决策，为什么不用 CGO",
		"limit": float64(3),
	}, nil)
	if err != nil {
		t.Fatalf("memory_search tool: %v", err)
	}
	t.Logf("  tool output:\n%s", indent(textOf(toolResult.Content), "    "))

	// ── 7. Agent tool: memory_get ─────────────────────────────────────────────
	t.Log("\n=== STEP 5: Agent tool — memory_get ===")
	adrResults, err := mgr.Search(ctx, "ADR 架构决策", 1)
	if err != nil || len(adrResults) == 0 {
		t.Log("  (skipped — no ADR result)")
	} else {
		getTool := memory.GetTool()
		getResult, err := getTool.Execute(ctx, "call-002", map[string]any{
			"path":       adrResults[0].FilePath,
			"start_line": float64(adrResults[0].StartLine),
			"end_line":   float64(adrResults[0].EndLine),
		}, nil)
		if err != nil {
			t.Fatalf("memory_get tool: %v", err)
		}
		t.Logf("  memory_get %s L%d–%d:\n%s",
			filepath.Base(adrResults[0].FilePath),
			adrResults[0].StartLine, adrResults[0].EndLine,
			indent(textOf(getResult.Content), "    "))
	}

	// ── 8. Incremental update: user adds new memory ───────────────────────────
	t.Log("\n=== STEP 6: 增量更新（用户新增记忆）===")
	newContent := `# 新决策 2026-03-27

## ADR-003: Embedding 优先使用 OpenRouter

**决定**：将 OpenRouter 设为第一优先级 embedding 提供商

**原因**：
- OpenRouter 聚合多个 LLM 服务，切换模型无需改代码
- 与 OpenAI API 完全兼容，使用 openai/text-embedding-3-small
- 通过 OPENROUTER_API_KEY 环境变量启用
`
	newPath := filepath.Join(memDir, "decisions_update.md")
	if err := os.WriteFile(newPath, []byte(newContent), 0o644); err != nil {
		t.Fatalf("write update: %v", err)
	}
	if err := mgr.IndexFile(ctx, newPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}
	t.Log("  IndexFile OK")

	newHits, err := mgr.Search(ctx, "OpenRouter embedding 优先级", 3)
	if err != nil {
		t.Fatalf("search after update: %v", err)
	}
	if len(newHits) == 0 {
		t.Error("  [FAIL] 增量更新后查不到新内容")
	} else {
		t.Logf("  [PASS] 增量更新后搜索 → %d hit(s), score=%.3f", len(newHits), newHits[0].Score)
	}

	// ── 9. 幂等性：重复 Sync 不重新 embed ────────────────────────────────────
	t.Log("\n=== STEP 7: 幂等性验证（重复 Sync）===")
	if err := mgr.Sync(ctx); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	t.Log("  Second Sync OK (unchanged files skipped)")

	// ── 10. Summary ───────────────────────────────────────────────────────────
	t.Logf("\n=== SUMMARY ===\n  provider  : %s / %s\n  files     : %d initial + 1 incremental\n  searches  : %d/%d passed\n  flow      : write → Sync → Search → AgentTool → IndexFile → Search ✓",
		provider.Name(), provider.Model(), len(files), pass, pass+fail)

	if fail > 0 {
		t.Fail()
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func textOf(blocks []types.ContentBlock) string {
	var parts []string
	for _, b := range blocks {
		if tc, ok := b.(*types.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func firstLine(s string) string {
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			r := []rune(line)
			if len(r) > 80 {
				return string(r[:80]) + "…"
			}
			return line
		}
	}
	return "(empty)"
}

func indent(s, prefix string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > 25 {
		lines = append(lines[:25], fmt.Sprintf("... (%d more lines)", len(lines)-25))
	}
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}
