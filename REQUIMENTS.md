# AI Gateway Desktop — 需求文档

> 面向非技术用户的 AI 消息网关桌面应用。双击打开，向导式配置，无需命令行。
>
> **技术栈**: Go + React + Wails
> **版本**: 0.3 · 2026-03-25

---

## M0 · Agent Core

> 目标：用 Go 实现与 pi-agent-core 等价的 AI Agent 核心库，作为整个系统的 AI 驱动层。

### 消息类型（types/）

- [ ] `TextContent` — 文本块，含可选 `textSignature`
- [ ] `ImageContent` — 图片块，支持 base64 data 或 URL，含 mimeType
- [ ] `AudioContent` — 音频块，支持 base64 data 或 URL，含 mimeType 和 duration（M0 定义类型，M2 实现转录）
- [ ] `ThinkingContent` — 推理过程块（Anthropic extended thinking），含 `redacted` 标志
- [ ] `ToolCall` — tool 调用块，含 id / name / arguments / thoughtSignature
- [ ] `UserMessage` — role=user，content 支持 Text / Image / Audio 混合数组
- [ ] `AssistantMessage` — role=assistant，含 api / provider / model / usage / stopReason / errorMessage
- [ ] `ToolResultMessage` — role=toolResult，含 toolCallId / toolName / content / details / isError
- [ ] `Usage` — input / output / cacheRead / cacheWrite / totalTokens / cost 各字段
- [ ] `StopReason` — stop / length / toolUse / error / aborted
- [ ] `AgentEvent` 联合类型：
  - [ ] `agent_start`
  - [ ] `agent_end`（含最终 messages）
  - [ ] `turn_start`
  - [ ] `turn_end`（含 message + toolResults）
  - [ ] `message_start`
  - [ ] `message_update`（含 AssistantMessageEvent delta）
  - [ ] `message_end`
  - [ ] `tool_execution_start`（含 toolCallId / toolName / args）
  - [ ] `tool_execution_update`（含 partialResult）
  - [ ] `tool_execution_end`（含 result / isError）
- [ ] `AssistantMessageEvent` 流式事件：text_start / text_delta / text_end / thinking_start / thinking_delta / thinking_end / toolcall_start / toolcall_delta / toolcall_end / done / error
- [ ] `Model` — id / name / api / provider / baseUrl / reasoning / input(text|image) / cost / contextWindow / maxTokens
- [ ] `StreamOptions` — temperature / maxTokens / signal / apiKey / transport / cacheRetention / sessionId / headers / metadata
- [ ] `ThinkingLevel` — off / minimal / low / medium / high / xhigh
- [ ] `ToolExecutionMode` — sequential / parallel

### LLM Provider 层（provider/）

- [ ] `LLMProvider` interface — `Stream(ctx, LLMRequest) (<-chan LLMEvent, error)`
- [ ] `ProviderRegistry` — 注册 / 查找 / 列出所有 provider
- [ ] **Anthropic provider**
  - [ ] Messages API SSE 实现
  - [ ] 支持 extended thinking（ThinkingContent 解析）
  - [ ] 支持图片输入（ImageContent → base64 blocks）
  - [ ] API Key 认证
  - [ ] OAuth PKCE 流程
  - [ ] 内置模型列表（claude-sonnet-4-6 / claude-opus-4-6 / claude-haiku-4-5）
- [ ] **OpenAI provider**
  - [ ] Chat Completions API SSE
  - [ ] Responses API SSE（可选）
  - [ ] 支持图片输入（vision）
  - [ ] API Key 认证
  - [ ] OpenAI Codex OAuth
  - [ ] 内置模型列表（gpt-5.4 等）
- [ ] **Gemini provider**
  - [ ] Google Generative AI SSE
  - [ ] 支持图片输入
  - [ ] API Key 认证
  - [ ] Google OAuth PKCE
  - [ ] 内置模型列表
- [ ] **Ollama provider**
  - [ ] `/api/chat` 本地调用（无需 auth）
  - [ ] 从 `/api/tags` 自动发现可用模型
  - [ ] 支持自定义 baseURL（默认 http://localhost:11434）

### 认证层（auth/）

- [ ] `AuthStorage` — 读写 `~/.ai-gateway/auth.json`
  - [ ] `GetAPIKey(provider)` / `SetAPIKey(provider, key)`
  - [ ] `GetOAuth(provider)` / `SetOAuth(provider, creds)`
  - [ ] `SetRuntimeAPIKey(provider, key)` — 临时覆盖，不落盘
- [ ] `AuthResolver` — 统一优先级解析 API Key
  - [ ] 优先级：runtime override > 环境变量 > Keychain > auth.json API Key > auth.json OAuth（自动 refresh）
  - [ ] 环境变量映射：`ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY` 等
- [ ] `Keychain` 集成
  - [ ] macOS：Keychain Services
  - [ ] Linux：Secret Service（freedesktop）
  - [ ] Windows：DPAPI
- [ ] OAuth 工具库
  - [ ] `pkce.go` — code_verifier / code_challenge 生成
  - [ ] `callback_server.go` — 本地 HTTP 回调（`:0` 随机端口，接收 redirect）
  - [ ] `OAuthProvider` interface — Login / RefreshToken / GetAPIKey / ModifyModels
  - [ ] Anthropic OAuth 实现
  - [ ] Google OAuth 实现
  - [ ] OpenAI Codex OAuth 实现

### Agent 主循环（agent/）

- [ ] `AgentLoopConfig`
  - [ ] model / tools / systemPrompt / toolExecution
  - [ ] `ConvertToLLM([]AgentMessage) ([]LLMMessage, error)`
  - [ ] `TransformContext([]AgentMessage) ([]AgentMessage, error)` — 上下文裁剪 / 注入
  - [ ] `GetAPIKey(provider) (string, error)`
  - [ ] `BeforeToolCall(ctx) (BeforeToolCallResult, error)` — 可 block tool 执行
  - [ ] `AfterToolCall(ctx) (AfterToolCallResult, error)` — 可替换 tool 结果
  - [ ] `GetSteeringMessages() ([]AgentMessage, error)` — 中途注入引导消息
  - [ ] `GetFollowUpMessages() ([]AgentMessage, error)` — agent 结束后继续
- [ ] `AgentLoop(ctx, prompts, agentCtx, config) <-chan AgentEvent`
  - [ ] 完整 turn 循环：LLM call → tool execution → 下一轮
  - [ ] stopReason=toolUse 时驱动 tool 执行
  - [ ] stopReason=stop 时检查 follow-up 消息
  - [ ] 所有生命周期事件正确 emit
- [ ] `AgentLoopContinue` — 从当前 context 继续（无需新 prompt，用于重试）
- [ ] **并行 tool 执行**
  - [ ] 同一 assistant message 的多个 tool call 并发执行
  - [ ] 结果按原始顺序回填
  - [ ] 单个 tool 失败不阻断其他 tool
- [ ] **顺序 tool 执行** — 逐个执行，适用于有副作用的 tool
- [ ] `Agent` struct（有状态封装）
  - [ ] `Prompt(ctx, input)` / `Prompt(ctx, messages)`
  - [ ] `Continue(ctx)` — 重试 / 续跑队列消息
  - [ ] `Abort()` — 取消当前 turn
  - [ ] `WaitForIdle()` — 等待 agent 空闲
  - [ ] `Steer(message)` — 中途注入引导消息（当前 turn 结束后生效）
  - [ ] `FollowUp(message)` — agent 结束后注入（无 tool call 且无 steering 时生效）
  - [ ] `SteeringMode` — all / one-at-a-time
  - [ ] `FollowUpMode` — all / one-at-a-time
  - [ ] `Subscribe(fn) unsubscribe` — 订阅 AgentEvent
  - [ ] `AppendMessage` / `ReplaceMessages` / `ClearMessages`
  - [ ] `SetSystemPrompt` / `SetModel` / `SetThinkingLevel` / `SetTools`
  - [ ] `State()` — 读取当前 AgentState（只读快照）

### Tool 系统（tool/）

- [ ] `AgentTool[T]` 泛型接口 — name / description / schema / label / execute
- [ ] `ToolRegistry` — 注册 / 查找 / 列出工具
- [ ] `BeforeToolCallContext` — assistantMessage / toolCall / args / context
- [ ] `AfterToolCallContext` — 同上 + result / isError
- [ ] `BeforeToolCallResult` — `block bool` / `reason string`
- [ ] `AfterToolCallResult` — 可替换 content / details / isError
- [ ] `ToolResult[T]` — content []ContentBlock / details T
- [ ] 内置测试工具：echo / noop

### 消息工具（message/）

- [ ] `ConvertToLLM` — AgentMessage 数组转 LLM 兼容格式，过滤不支持类型
- [ ] `EstimateTokens` — 简单 token 估算（字符数 / 4）
- [ ] `PruneMessages` — 超过 contextWindow 时裁剪旧消息，保留 system + 最近 N 轮
- [ ] JSON 序列化 / 反序列化（用于 SQLite 持久化）

### 流式处理（stream/）

- [ ] SSE 解析器 — 兼容 `data:` / `event:` / `id:` 字段
- [ ] `EventStream[T, R]` — channel-based 流，支持 cancel
- [ ] 重试机制 — 指数退避，尊重 `Retry-After` / `x-stainless-retry-after` header
- [ ] 最大重试延迟上限（默认 60s，可配置）

### 测试工具（testutil/）

- [ ] `MockLLMProvider` — 可注入预设响应序列，模拟 streaming
- [ ] `MockTool` — 可配置 execute 返回值 / 延迟 / 错误
- [ ] `Fixtures` — 常用 UserMessage / AssistantMessage / ToolResultMessage 构造函数

### M0 验收标准

- [ ] 单轮文本对话：4 个 provider 均可流式输出
- [ ] 多轮 tool call：loop 正确驱动，ToolResult 正确回填
- [ ] 并行 tool：2 个 tool 同时执行，结果按顺序回填
- [ ] `BeforeToolCall` block：tool 不执行，loop 继续，emit error tool result
- [ ] `AfterToolCall` override：替换 content 生效
- [ ] Steering：中途 steer → 下一轮前注入
- [ ] FollowUp：agent 结束后注入 follow-up → 继续运行
- [ ] ImageContent：UserMessage 含图片，Anthropic / OpenAI 正确传递
- [ ] AudioContent：类型可序列化（实现在 M2）
- [ ] API Key 优先级：runtime > env > storage
- [ ] OAuth refresh：过期 token 自动 refresh 后继续请求
- [ ] 上下文裁剪：超过 token 限制时 prune，保留最近消息
- [ ] 单元测试覆盖：loop 主流程 / tool 执行 / auth resolver / prune 逻辑

---

## M1 · Gateway 核心 + Web Chat（纯 Go 后端）

> 目标：纯 Go 服务跑通，Web Chat 可用 curl / WebSocket 客户端测试，对话历史存 SQLite。
> 无 Wails、无 React，命令行启动。

### Gateway 路由层（gateway/）

- [ ] `Router` — 根据 channelID 路由入站消息到对应 Session
- [ ] `SessionManager`
  - [ ] 按 channelID + userID 查找或创建 Session
  - [ ] Session 空闲超时清理（默认 30 分钟）
  - [ ] 最大并发 Session 数限制（默认 100）
- [ ] `Session`
  - [ ] 绑定一个 Agent 实例
  - [ ] 串行处理来自同一用户的消息（queue）
  - [ ] 将 AgentEvent 通过回调转发给 channel
- [ ] `Server`
  - [ ] HTTP Server 监听（默认 port 9988，可配置）
  - [ ] `GET  /health` — 健康检查
  - [ ] `POST /webhook/:channelID` — 统一 webhook 入口
  - [ ] 请求去重（基于 message ID，防止重复投递）

### Web Chat Channel（channel/webchat/）

- [ ] WebSocket 连接管理（`GET /ws/:sessionID`）
- [ ] 消息协议：JSON
  - [ ] 上行：`{ type: "message", content: "...", images: [...] }`
  - [ ] 下行：`{ type: "delta"|"done"|"error", content: "...", timestamp: 0 }`
- [ ] 文本消息发送 / 接收
- [ ] 图片上传（base64 → ImageContent）
- [ ] 客户端 `ping/pong` 心跳（30s）

### 数据存储（store/）

- [ ] SQLite 初始化（`modernc.org/sqlite` 纯 Go，无 CGO）
- [ ] 数据库路径：`~/.pi-go/data.db`
- [ ] Migration 系统（嵌入 SQL，启动时自动执行）
- [ ] `MessageStore`
  - [ ] 保存 / 查询消息记录（role / content / channel / user / timestamp）
  - [ ] 按 channelID + userID 查询，支持分页（limit/offset）
- [ ] `KVStore`
  - [ ] 键值对存储（`key TEXT PRIMARY KEY, value JSON`）
  - [ ] `Get` / `Set` / `Delete`

### CLI 入口（cmd/gateway/）

- [ ] `main.go` — 读 config.yml，初始化 store / provider / gateway，启动 HTTP server
- [ ] 启动日志输出监听地址和已加载的 channel

### M1 验收标准

- [ ] `go run ./cmd/gateway` 启动成功，输出监听地址
- [ ] WebSocket 连接 `/ws/test`，发送文本，收到 AI 流式 delta 回复
- [ ] 发送 base64 图片，AI 能描述图片内容
- [ ] 对话记录写入 SQLite，重启后可查
- [ ] 并发 10 个 WebSocket 连接同时对话互不干扰
- [ ] 单元测试：SessionManager / MessageStore / WebChat handler

---

## M1.5 · Wails 桌面应用 + React 前端

> 目标：把 M1 后端包进桌面 App，双击可用。

### Wails 应用骨架

- [ ] `wails.json` — 应用名 / 版本 / 窗口尺寸（1200×800，最小 900×600）
- [ ] `main.go` — Wails 启动入口，绑定 App struct
- [ ] `wails_app.go` — 暴露给前端的全部 Go API
- [ ] Wails Dev 模式热更新（前后端均支持）
- [ ] 应用图标（macOS .icns / Windows .ico）
- [ ] 首次启动自动打开 Setup Wizard

### Go API（暴露给前端）

- [ ] **配置**：`GetConfig` / `SaveConfig` / `IsFirstRun`
- [ ] **AI Provider**：`GetProviders` / `SaveAPIKey` / `StartOAuthLogin` / `GetModels` / `TestProviderConnection`
- [ ] **Channel 管理**：`GetChannels` / `SaveChannelConfig` / `DeleteChannelConfig` / `TestChannelConnection`
- [ ] **对话**：`SendMessage` / `AbortCurrentTurn` / `GetHistory` / `GetSessions`
- [ ] **系统**：`GetVersion` / `OpenLogsFolder` / `GetWebhookBaseURL`

### 前端事件推送（Go → React）

- [ ] `agent:event` — AgentEvent delta
- [ ] `channel:status` — 渠道连接状态变更
- [ ] `message:new` — 新消息到达
- [ ] `oauth:callback` — OAuth 登录完成

### React 前端

- [ ] **Setup Wizard**（3步）：选 Provider → 输入 Key/OAuth → 完成展示 Webhook URL
- [ ] **Dashboard**：Channel 状态卡片、今日消息量、最近 5 条对话摘要
- [ ] **ChatView**：消息气泡、Markdown 渲染、流式输出、图片内联、停止按钮

### M1.5 验收标准

- [ ] 双击 .app 打开，窗口 2 秒内显示
- [ ] Setup Wizard 3 步完成 Web Chat 配置
- [ ] 通过内置 ChatView 发消息，AI 流式回复正常
- [ ] Wails dev 模式前后端热更新正常

---

## M2 · 飞书 Channel + 完整 UI

> 目标：第一个真实外部渠道上线，UI 基本完善可演示。

### 飞书 Channel（channels/feishu/）

- [ ] 接入方式：飞书开放平台 Bot（自建应用）
- [ ] Webhook 事件验证（`X-Lark-Signature` HMAC-SHA256）
- [ ] 接收消息类型
  - [ ] 私聊文本消息
  - [ ] 群聊 @ 机器人文本消息
  - [ ] 私聊图片消息（下载图片 → ImageContent）
  - [ ] 私聊音频消息（下载音频 → AudioContent → 转文字后发给 AI）
- [ ] 发送消息类型
  - [ ] 纯文本
  - [ ] 富文本（Markdown 转飞书富文本格式）
  - [ ] 消息卡片（用于展示工具执行结果）
- [ ] access_token 管理（App ID + App Secret → 自动刷新，2 小时有效期）
- [ ] 支持多个飞书 Bot 实例（不同 App ID）
- [ ] Challenge 验证（首次配置 Webhook 时的 URL 验证请求）

### 音频转文字（M2 实现）

- [ ] 调用 AI Provider 的语音识别能力（OpenAI Whisper API / Gemini Audio）
- [ ] 转写结果作为 UserMessage 文本内容传给 Agent
- [ ] 转写失败时返回错误提示而非静默失败

### React 前端（M2 部分）

- [ ] **ChannelConfig 页面**
  - [ ] 飞书配置表单：App ID / App Secret / Verification Token
  - [ ] Webhook URL 一键复制（含图文说明：在飞书开放平台粘贴此 URL）
  - [ ] 连接测试按钮（发送测试请求验证凭据）
  - [ ] Web Chat 配置：主题色 / 标题 / 嵌入代码预览
- [ ] **AISettings 页面**
  - [ ] Provider 切换（当前已连接的 provider 列表）
  - [ ] 每个 Channel 独立绑定 Provider + Model
  - [ ] System Prompt 编辑器（多行文本，支持变量提示 `{{user_name}}`）
  - [ ] Temperature / MaxTokens 滑块
  - [ ] Thinking Level 选择（off / low / medium / high）
- [ ] **History 页面**
  - [ ] 按 Channel 筛选
  - [ ] 按用户 ID 筛选
  - [ ] 完整对话回放（含 tool call 详情可展开）
  - [ ] 导出为 JSON / Markdown
- [ ] **消息气泡增强**
  - [ ] 图片内联显示（点击放大）
  - [ ] 音频播放器（展示已转写文字 + 原始音频可播放）
  - [ ] Tool call 折叠展示（默认折叠，点击展开 args + result）
  - [ ] Thinking 内容折叠展示
  - [ ] 代码块语法高亮 + 复制按钮

### M2 验收标准

- [ ] 飞书私聊文本消息 → AI 回复正常
- [ ] 飞书群聊 @ Bot → AI 回复正常
- [ ] 飞书图片消息正确传递给 AI（vision 模型）
- [ ] 飞书语音消息转文字后发给 AI
- [ ] ChannelConfig 页面配置飞书，无需查看文档
- [ ] History 页面可查看完整对话记录

---

## M3 · WhatsApp + 打包发布

> 目标：MVP 完整，可以给真实用户使用。

### WhatsApp Channel（channels/whatsapp/）

- [ ] 接入方式：Meta WhatsApp Business Cloud API
- [ ] Webhook 验证（`X-Hub-Signature-256` HMAC-SHA256）
- [ ] Verify Token 验证（首次配置时的 GET 请求）
- [ ] 接收消息类型
  - [ ] 文本消息
  - [ ] 图片消息（下载 → ImageContent）
  - [ ] 音频消息（下载 → AudioContent → 转文字）
  - [ ] 文档消息（PDF 提取文字 → TextContent）
  - [ ] 位置消息（转为坐标文字）
- [ ] 发送消息类型
  - [ ] 文本（含 Markdown 转纯文本，WhatsApp 不支持 Markdown）
  - [ ] 图片（URL 或 base64 上传）
  - [ ] 模板消息（仅用于主动发起对话，MVP 不实现）
- [ ] 用户标识：`wa_id`（手机号）
- [ ] 已读回执：消息投递后标记为 read
- [ ] 媒体文件管理
  - [ ] 下载媒体：`GET /{media-id}` 获取临时 URL 后下载
  - [ ] 上传媒体：`POST /{phone-number-id}/media`
  - [ ] 媒体缓存（避免重复下载，TTL 1 小时）
- [ ] Rate limiting 处理（429 响应自动退避重试）

### 打包与发布（wails build）

- [ ] **macOS**
  - [ ] `.app` bundle（arm64 + amd64 universal binary）
  - [ ] `.dmg` 安装包（拖拽安装）
  - [ ] 应用签名（Developer ID Application）
  - [ ] 公证（notarytool）
  - [ ] Info.plist：版本号 / Bundle ID / 权限声明
- [ ] **Windows**
  - [ ] `.exe` installer（NSIS）
  - [ ] 代码签名（可选，MVP 可跳过）
  - [ ] 安装到 `%LOCALAPPDATA%\AI Gateway`
- [ ] **Linux**
  - [ ] `.AppImage`（通用，无需安装）
  - [ ] `.deb` 包（Debian/Ubuntu）
- [ ] **自动更新**
  - [ ] Wails 内置 updater（检查 GitHub Releases）
  - [ ] 启动时静默检查，有更新时弹出提示
  - [ ] 用户可选择"现在更新"或"稍后"

### React 前端（M3 部分）

- [ ] **WhatsApp 配置表单**
  - [ ] Phone Number ID / Access Token / Verify Token 输入
  - [ ] Webhook URL 展示（含 Meta 开发者后台配置教程截图）
  - [ ] 连接测试
- [ ] **系统设置页面**
  - [ ] Webhook 监听端口修改
  - [ ] 日志级别
  - [ ] 数据目录
  - [ ] 检查更新按钮
  - [ ] 开机自启动开关（macOS LaunchAgent / Windows 注册表）
- [ ] **通知系统**
  - [ ] 系统原生通知（新消息，仅窗口未聚焦时）
  - [ ] 应用内 toast（操作成功 / 失败反馈）
- [ ] **空状态设计**
  - [ ] 无 Channel 时引导到 ChannelConfig
  - [ ] 无对话时展示使用说明
  - [ ] 连接错误时展示具体原因 + 修复建议

### M3 验收标准

- [ ] WhatsApp 文本消息 → AI 回复正常
- [ ] WhatsApp 图片消息正确传递给 AI
- [ ] WhatsApp 语音消息转文字后发给 AI
- [ ] macOS .dmg 安装后双击打开，< 2 秒显示界面
- [ ] Windows .exe 安装后正常运行
- [ ] 自动更新检测正常工作
- [ ] 首次使用完整流程 < 5 分钟（含配置 1 个 Channel）

---

## 非功能需求（全局）

| 指标 | 目标 |
|---|---|
| 应用启动时间 | < 2 秒 |
| 安装包大小（macOS） | < 30 MB |
| 空载内存占用 | < 80 MB |
| 首次配置完成时间 | < 5 分钟 |
| 无需安装 Go 运行时 | 必须 |
| 无需安装 Node.js | 必须 |
| 无需 Docker | 必须 |
| 无需命令行操作 | 必须 |
| 离线可用（Ollama） | 必须 |
| 数据本地存储 | 必须，不上传任何用户数据 |

---

## 明确不做（MVP 范围外）

- 插件 / 扩展系统
- 多用户 / 团队权限管理
- 云托管 / SaaS 版本
- Telegram / Discord / Slack / Signal
- TTS 语音合成输出
- 移动端（iOS / Android）
- 内嵌终端模拟器    
- WASM 插件沙箱
