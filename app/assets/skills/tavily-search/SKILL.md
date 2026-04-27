---
name: tavily-search
version: 1.0.0
description: >-
  网页搜索与内容抓取工具。当用户需要搜索互联网信息、抓取网页内容、
  查找最新资讯、研究某个话题、或获取某个 URL 页面的详细内容时触发。
  底层使用 Tavily Search / Crawl API，通过 tavily-cli 执行。
  触发关键词：搜索、查一下、找一下、抓取、crawl、search、查资料、联网查。
author: pi-go
---

# tavily-search — 网页搜索与内容抓取 Skill

## 行为规则（必须遵守）

1. **必须使用 `tavily-search` 或 `tavily-crawl` CLI 命令执行，不要自己写 HTTP 请求代码。**
2. API Key 已通过环境变量 `TAVILY_API_KEY` 注入，执行前无需询问。
3. 用户给的是**关键词/问题** → 用 `tavily-search`；用户给的是**具体 URL** → 优先用 `tavily-crawl`，失败时自动 fallback（CLI 内部处理）。
4. 执行完毕后，**必须解读结果**，不要只把 JSON 丢给用户，要用自然语言总结关键信息。
5. 如果结果 `source` 字段为 `fallback_search` 或 `local_cache`，需告知用户"此结果来自降级/缓存"。
6. 如果结果 `status` 为 `error`，解读 `errors` 字段，给出排查建议。

---

## 执行流程

### Step 1 — 判断操作类型

| 用户输入 | 使用命令 |
|---------|---------|
| 关键词、问题、话题 | `tavily-search` |
| 以 http:// 或 https:// 开头的 URL | `tavily-crawl` |
| 先搜索再抓取 | 先 `tavily-search` 找到 URL，再 `tavily-crawl` 抓取正文 |

### Step 2 — 构造命令

**Search 命令：**
```bash
export TAVILY_API_KEY="$TAVILY_API_KEY"
tavily-search "<query>" [选项]
```

**Crawl 命令：**
```bash
export TAVILY_API_KEY="$TAVILY_API_KEY"
tavily-crawl "<url>" [选项]
```

### Step 3 — 执行并捕获输出

用 `bash` 工具执行命令，合并 stderr（状态日志）和 stdout（JSON 结果），timeout 设为 60s。

### Step 4 — 解读结果

- 提取 `answer`（AI 摘要）直接呈现
- 列出 `results` 中评分最高的条目（title + url + content 摘要）
- 如果是 crawl，提取 `results[0].content` 的关键段落
- 用中文总结核心信息

---

## 命令参数速查

### tavily-search

```
tavily-search <query>
  --search-depth basic|advanced   # 搜索深度（默认 basic，建议用 advanced）
  --max-results N                 # 结果数量 1-20（默认 5）
  --include-answer / --no-answer  # 是否返回 AI 摘要（默认开启）
  --raw-content / --no-raw-content # 是否返回原始内容（默认关闭）
  --images / --no-images          # 是否返回图片（默认关闭）
  --topic general|news            # 话题类型（默认 general）
```

### tavily-crawl

```
tavily-crawl <url>
  --max-depth N      # 爬取深度（默认 1）
  --max-breadth N    # 同层最大链接数（默认 20）
  --limit N          # 最大页面数（默认 50）
  --extract-depth basic|advanced  # 提取深度（默认 basic）
  --allow-external / --no-external # 是否爬取外部链接（默认关闭）
```

---

## 典型用法示例

### 搜索最新资讯
```bash
tavily-search "2025年 AI Agent 最新进展" --search-depth advanced --max-results 8 --topic news
```

### 深度研究某话题
```bash
tavily-search "Rust 异步编程最佳实践" --search-depth advanced --max-results 10 --raw-content
```

### 抓取某个页面全文
```bash
tavily-crawl "https://docs.python.org/3/library/asyncio.html" --max-depth 1 --limit 3
```

### 先搜索再精读（两步）
```bash
# Step 1: 找到最相关的 URL
tavily-search "PostgreSQL JSONB 索引优化" --max-results 3

# Step 2: 抓取评分最高的页面
tavily-crawl "<top_url_from_step1>" --max-depth 1 --extract-depth advanced
```

---

## Fallback 行为说明

CLI 内部已实现三层自动 fallback，无需手动处理：

| 情况 | 自动处理 |
|------|---------|
| Crawl API 失败（4xx/5xx/超时） | 自动降级到 Search，`source="fallback_search"` |
| API 不可用 / Key 无效 | 尝试本地缓存，`source="local_cache"` |
| 全部失败 | 输出 `status="error"` 的结构化 JSON，exit(1) |

输出中 `source` 字段含义：
- `"api"` — 实时 API 返回，数据最新
- `"fallback_search"` — Crawl 失败后降级的搜索结果，告知用户
- `"local_cache"` — 使用了本地缓存，告知用户缓存时间

---

## 安装检查

如果执行时报 `command not found`，需先安装：

```bash
cd ~/tavily-cli && pip install -e .
```

验证安装：
```bash
tavily-search --help
tavily-crawl --help
```

---

## 输出结构参考

```json
// Search 成功输出
{
  "status": "success",
  "source": "api",
  "tool": "search",
  "query": "...",
  "answer": "AI 生成的摘要...",
  "results": [
    { "title": "...", "url": "...", "content": "...", "score": 0.95 }
  ]
}

// Crawl 成功输出
{
  "status": "success",
  "source": "api",
  "tool": "crawl",
  "base_url": "...",
  "summary": { "total_pages": 5, "failed_pages": 0 },
  "results": [
    { "url": "...", "content": "页面正文..." }
  ]
}

// 错误输出
{
  "status": "error",
  "tool": "search|crawl",
  "fallback_chain": ["search_failed", "cache_miss"],
  "errors": { "primary": "API Key 无效（401）" }
}
```
