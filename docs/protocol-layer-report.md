# Clawfirm 协议层全景报告

## 1. 支持的协议总览

共 **4 种原生协议实现**，映射到 **23 种 provider type**：

| 原生协议 | 实现文件 | 行数 | Endpoint | 使用的 provider type |
|---------|----------|------|----------|---------------------|
| **anthropic** | `provider/anthropic/provider.go` | 690 | `POST /v1/messages` | `anthropic`, `minimax` |
| **openai** | `provider/openai/provider.go` | 479 | `POST /v1/chat/completions` | `openai`, `deepseek`, `moonshot`, `volcengine`, `modelstudio`, `glm`/`zai`, `groq`, `openrouter`, `together`, `mistral`, `xai`, `nvidia`, `xiaomi`, `venice`, `huggingface`, `perplexity`, `litellm`, `sglang`, `vllm` |
| **gemini** | `provider/gemini/provider.go` | 394 | `POST /v1beta/models/{model}:streamGenerateContent` | `gemini` |
| **ollama** | `provider/ollama/provider.go` | 249 | `POST /api/chat` | `ollama` |

## 2. 前端如何设置协议

在桌面端 **Settings → Providers** 页面（`ProvidersPane.tsx`）：

```
每个 Provider 有 4 个可配字段：
┌──────────────────────────────────────────┐
│  protocol  ← 下拉选择（23 种 type 之一）    │
│  platform  ← 可选（zenmux, openrouter 等） │
│  api_key   ← API 密钥                     │
│  base_url  ← 自动填充默认值，可手动覆盖      │
└──────────────────────────────────────────┘
```

**协议解析优先级**（`config.ResolvedProtocol()`）：

1. `protocol` 字段（最高优先级）
2. `type` 字段（旧版兼容）
3. 默认 `"anthropic"`（都没填时）

前端保存时同时写入 `protocol` 和 `type`（双写兼容）。

## 3. OpenAI 协议 vs Anthropic 协议：逐步对比

### 3.1 请求格式

| | OpenAI | Anthropic |
|--|--------|-----------|
| URL | `/v1/chat/completions` | `/v1/messages` |
| Auth | `Authorization: Bearer {key}` | `x-api-key: {key}`（sk-ant- 开头）或 `Authorization: Bearer {key}`（中转 key）|
| system prompt | 作为 `role: "system"` 的 message | 顶层 `system` 字段（支持 cache_control）|
| tool 定义 | `tools: [{type: "function", function: {name, description, parameters}}]` | `tools: [{name, description, input_schema}]` |
| tool 结果 | `role: "tool"`, `tool_call_id` | `role: "user"` + `type: "tool_result"` |
| thinking | 不支持 | `thinking: {type: "enabled", budget_tokens}` |
| prompt caching | 不支持 | `cache_control: {type: "ephemeral"}` 标记在 system/tools/messages 上 |

### 3.2 SSE 流格式

**OpenAI — 简单行协议：**

```
data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

- 无 `event:` 字段，只有 `data:` 行
- `[DONE]` 作为结束标记
- 文本、tool_calls、finish_reason 都在同一个 chunk 结构里

**Anthropic — 完整 SSE 协议（带 event type）：**

```
event: message_start
data: {"message":{"usage":{"input_tokens":25}}}

event: content_block_start
data: {"index":0,"content_block":{"type":"text"}}

event: content_block_delta
data: {"index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_stop
data: {"index":0}

event: message_delta
data: {"delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":15}}

event: message_stop
data: {}
```

- 每个 SSE 都有 `event:` 类型
- 内容按 content block 分组（text、tool_use、thinking 各自独立）
- `message_stop` 作为结束标记

### 3.3 流解析器差异

| | OpenAI | Anthropic |
|--|--------|-----------|
| 解析器 | 直接 `bufio.Scanner` 逐行 | `stream.ParseSSEStream()` → `SSEReader` |
| buffer 上限 | **1 MB**（显式设置） | ~~64 KB（bufio 默认值）~~ → **1 MB**（已修复） |
| 超时机制 | 无（依赖 ctx cancel） | 3 分钟 stall timeout |
| 流中断处理 | `gotFinish=false` 警告，仍 emit Done | emit Done + `StopReasonError` |

### 3.4 HTTP 客户端差异 ~~（关键 bug）~~ （已修复）

| | `New()` | `NewWithBaseURL()` | builder 调用 |
|--|---------|-------------------|-------------|
| **OpenAI** | `NewStreamingHTTPClient()` | `NewStreamingHTTPClient()` | `NewWithBaseURL` |
| **Anthropic** | `NewStreamingHTTPClient()` | `NewStreamingHTTPClient()` ✅ | `NewWithBaseURL` |
| **Gemini** | `NewStreamingHTTPClient()` | `NewStreamingHTTPClient()` ✅ | `NewWithBaseURL` |
| **Ollama** | `&http.Client{Timeout: 10m}` | `&http.Client{Timeout: 10m}` | `NewWithBaseURL` |

`NewStreamingHTTPClient()`（`provider/httpclient.go`）的作用：

- **强制 HTTP/1.1**（`NextProtos: []string{"http/1.1"}`, `TLSNextProto: empty map`）
- **禁用压缩**（`DisableCompression: true`）
- 配置 30s dial/keepalive timeout

现在所有走 TLS 的 provider 都统一使用 streaming client。Ollama 是本地 HTTP，不需要。

## 4. 已修复的 3 个问题

| # | 问题 | 修复 | 文件 |
|---|------|------|------|
| **1** | `anthropic.NewWithBaseURL` 用 plain `http.Client`，HTTP/2 + gzip 导致代理环境 stream 截断 | 改用 `provider.NewStreamingHTTPClient()` | `provider/anthropic/provider.go:42` |
| **2** | `gemini.NewWithBaseURL` 同样问题 | 改用 `provider.NewStreamingHTTPClient()` | `provider/gemini/provider.go:40` |
| **3** | SSE reader scanner buffer 默认 64KB，大型 tool call 会静默截断 | 显式设为 1MB，与 OpenAI provider 一致 | `stream/sse.go:34-35` |
