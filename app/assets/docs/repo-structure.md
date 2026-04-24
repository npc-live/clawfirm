# ClawFirm 代码仓库结构分析

GitHub: `https://github.com/npc-live/clawfirm`
Go Module: `github.com/ai-gateway/clawfirm`
本地路径: `/Users/olivia/Desktop/ClawFirm_OPC`

---

## 零、文件分类规则

### 分类原则

| 类型 | 判定标准 | 存放位置 | 提交方式 |
|------|----------|----------|----------|
| **纯代码** | Go、Web(HTML/JS/CSS)、YAML Runner 等非自然语言 | 仓库根目录对应模块 | 直接提交 GitHub，无需特殊处理 |
| **自然语言内容** | Skill 提示词(SKILL.md)、知识库(references/)、prompt 模板、工作流文档 | `app/assets/` | embed 打包进二进制 |
| **混合项目** | 只要包含自然语言文件(.md/prompt)就整体归入 | `app/assets/` | 以自然语言为主导 |
| **Skill 运行时** | 所有 Skill 需要被 clawfirm 运行时加载 | `~/.clawfirm/skills/` | 与 `app/assets/skills/` 保持同步 |
| **CDP 分发** | 自媒体浏览器自动化 shortcut | `app/assets/shortcuts/` | YAML 格式，browser/yaml_runner.go 消费 |

### 决策流程

```
新文件/目录
    │
    ├─ 纯 .go / .js / .html / .css / .yaml(runner 引擎代码) ?
    │   → 仓库根目录对应模块（直接提交 GitHub）
    │
    ├─ 含 SKILL.md / prompt / reference .md / 模板 ?
    │   → app/assets/skills/{skill-name}/
    │   → 同步到 ~/.clawfirm/skills/{skill-name}/
    │
    ├─ CDP 平台发布 .yaml ?
    │   → app/assets/shortcuts/{platform}.yaml
    │
    ├─ .whip 工作流定义 ?
    │   → app/assets/workflows/{domain}/
    │
    └─ 混合项目（代码 + 自然语言并存）?
        → 整体放 app/assets/（自然语言为主导）
```

### Skill 双路径同步

Skill 必须同时存在于两个位置：

| 位置 | 用途 | 说明 |
|------|------|------|
| `app/assets/skills/` | 源码仓库 | embed 打包进二进制，随代码版本管理 |
| `~/.clawfirm/skills/` | 运行时加载 | clawfirm 进程实际读取的路径 |

```bash
# 同步命令
rsync -av --delete app/assets/skills/ ~/.clawfirm/skills/
# 或
./bin/skillctl sync
```

---

## 一、核心引擎层 (Go Backend)

### 1. Agent 引擎 — `agent/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `agent/agent.go` | [agent/agent.go](https://github.com/npc-live/clawfirm/blob/main/agent/agent.go) | Agent 主体定义 |
| `agent/loop.go` | [agent/loop.go](https://github.com/npc-live/clawfirm/blob/main/agent/loop.go) | Agent 对话循环 |
| `agent/bootstrap.go` | [agent/bootstrap.go](https://github.com/npc-live/clawfirm/blob/main/agent/bootstrap.go) | Agent 初始化引导 |
| `agent/systemprompt.go` | [agent/systemprompt.go](https://github.com/npc-live/clawfirm/blob/main/agent/systemprompt.go) | 系统提示词构建 |
| `agent/context.go` | [agent/context.go](https://github.com/npc-live/clawfirm/blob/main/agent/context.go) | 上下文管理 |
| `agent/compressor.go` | [agent/compressor.go](https://github.com/npc-live/clawfirm/blob/main/agent/compressor.go) | 上下文压缩 |
| `agent/microcompact.go` | [agent/microcompact.go](https://github.com/npc-live/clawfirm/blob/main/agent/microcompact.go) | 微压缩（轻量） |
| `agent/tool_executor.go` | [agent/tool_executor.go](https://github.com/npc-live/clawfirm/blob/main/agent/tool_executor.go) | 工具执行调度器 |
| `agent/state.go` | [agent/state.go](https://github.com/npc-live/clawfirm/blob/main/agent/state.go) | Agent 状态管理 |
| `agent/queue.go` | [agent/queue.go](https://github.com/npc-live/clawfirm/blob/main/agent/queue.go) | 消息队列 |
| `agent/temporal.go` | [agent/temporal.go](https://github.com/npc-live/clawfirm/blob/main/agent/temporal.go) | 时间相关处理 |

**功能**: 核心 AI Agent 运行时，负责对话循环、工具调用、上下文管理、系统提示词、消息压缩。

---

### 2. LLM Provider 适配 — `provider/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `provider/interface.go` | [provider/interface.go](https://github.com/npc-live/clawfirm/blob/main/provider/interface.go) | Provider 统一接口 |
| `provider/registry.go` | [provider/registry.go](https://github.com/npc-live/clawfirm/blob/main/provider/registry.go) | Provider 注册表 |
| `provider/classifier.go` | [provider/classifier.go](https://github.com/npc-live/clawfirm/blob/main/provider/classifier.go) | 模型分类器 |
| `provider/errors.go` | [provider/errors.go](https://github.com/npc-live/clawfirm/blob/main/provider/errors.go) | 错误处理 |
| `provider/anthropic/` | [provider/anthropic/](https://github.com/npc-live/clawfirm/tree/main/provider/anthropic) | Claude (Anthropic) |
| `provider/openai/` | [provider/openai/](https://github.com/npc-live/clawfirm/tree/main/provider/openai) | GPT (OpenAI) |
| `provider/gemini/` | [provider/gemini/](https://github.com/npc-live/clawfirm/tree/main/provider/gemini) | Gemini (Google) |
| `provider/ollama/` | [provider/ollama/](https://github.com/npc-live/clawfirm/tree/main/provider/ollama) | Ollama (本地模型) |
| `provider/zenmux/` | [provider/zenmux/](https://github.com/npc-live/clawfirm/tree/main/provider/zenmux) | ZenMux (多模型路由) |

**功能**: 多 LLM 后端适配层。统一接口支持 Anthropic Claude、OpenAI GPT、Google Gemini、Ollama 本地模型、ZenMux 路由。

---

### 3. 工具系统 — `tool/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `tool/base.go` | [tool/base.go](https://github.com/npc-live/clawfirm/blob/main/tool/base.go) | Tool 基础接口定义 |
| `tool/registry.go` | [tool/registry.go](https://github.com/npc-live/clawfirm/blob/main/tool/registry.go) | 工具注册表 |
| `tool/builtin/bash.go` | [tool/builtin/bash.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/bash.go) | Bash 命令执行 |
| `tool/builtin/read.go` | [tool/builtin/read.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/read.go) | 文件读取 |
| `tool/builtin/write.go` | [tool/builtin/write.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/write.go) | 文件写入 |
| `tool/builtin/edit.go` | [tool/builtin/edit.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/edit.go) | 文件编辑 |
| `tool/builtin/fs.go` | [tool/builtin/fs.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/fs.go) | 文件系统操作 |
| `tool/builtin/fetch.go` | [tool/builtin/fetch.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/fetch.go) | Web 抓取 |
| `tool/builtin/web_search.go` | [tool/builtin/web_search.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/web_search.go) | Web 搜索 |
| `tool/builtin/sub_agent.go` | [tool/builtin/sub_agent.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/sub_agent.go) | 子 Agent 生成 |
| `tool/builtin/ask_user.go` | [tool/builtin/ask_user.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/ask_user.go) | 用户交互 |
| `tool/builtin/media_gen.go` | [tool/builtin/media_gen.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/media_gen.go) | AI 图片生成 (封面) |
| `tool/builtin/media_understand.go` | [tool/builtin/media_understand.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/media_understand.go) | 多模态理解 (图片/视频) |
| `tool/builtin/browser_shortcut.go` | [tool/builtin/browser_shortcut.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/browser_shortcut.go) | 浏览器 CDP 快捷操作 |
| `tool/builtin/whipflow_run.go` | [tool/builtin/whipflow_run.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/whipflow_run.go) | Whip 工作流执行 |
| `tool/builtin/exec.go` | [tool/builtin/exec.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/exec.go) | 外部命令执行 |
| `tool/builtin/coding.go` | [tool/builtin/coding.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/coding.go) | 代码相关工具 |
| `tool/builtin/concurrency.go` | [tool/builtin/concurrency.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/concurrency.go) | 并发执行 |
| `tool/builtin/tool_search.go` | [tool/builtin/tool_search.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/tool_search.go) | 工具发现/搜索 |
| `tool/builtin/meta.go` | [tool/builtin/meta.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/meta.go) | 元信息工具 |
| `tool/builtin/echo.go` | [tool/builtin/echo.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/echo.go) | Echo 调试工具 |
| `tool/builtin/noop.go` | [tool/builtin/noop.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/noop.go) | 空操作 |
| `tool/builtin/patch.go` | [tool/builtin/patch.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/patch.go) | Patch 应用 |
| `tool/builtin/path.go` | [tool/builtin/path.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/path.go) | 路径处理 |
| `tool/builtin/time.go` | [tool/builtin/time.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/time.go) | 时间工具 |

**功能**: Agent 可调用的所有内置工具。包括文件操作、Bash、Web、媒体生成/理解、浏览器自动化、子 Agent、工作流执行等。

---

### 4. 浏览器自动化 (CDP) — `browser/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `browser/cdp.go` | [browser/cdp.go](https://github.com/npc-live/clawfirm/blob/main/browser/cdp.go) | Chrome DevTools Protocol 客户端 |
| `browser/yaml_runner.go` | [browser/yaml_runner.go](https://github.com/npc-live/clawfirm/blob/main/browser/yaml_runner.go) | YAML 步骤执行引擎 |
| `browser/executor.go` | [browser/executor.go](https://github.com/npc-live/clawfirm/blob/main/browser/executor.go) | 操作执行器 (click/fill/upload) |
| `browser/session.go` | [browser/session.go](https://github.com/npc-live/clawfirm/blob/main/browser/session.go) | 浏览器会话管理 |
| `browser/healer.go` | [browser/healer.go](https://github.com/npc-live/clawfirm/blob/main/browser/healer.go) | 自愈机制 (步骤失败恢复) |

**功能**: 通过 CDP 协议控制 Chrome 浏览器，执行 YAML 定义的自动化步骤（点击、填写、上传、截图等）。是社交媒体发布的底层引擎。

---

### 5. WhipFlow 工作流引擎 — `whipflow/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `whipflow/whipflow.go` | [whipflow/whipflow.go](https://github.com/npc-live/clawfirm/blob/main/whipflow/whipflow.go) | 入口 |
| `whipflow/lexer/lexer.go` | [whipflow/lexer/](https://github.com/npc-live/clawfirm/tree/main/whipflow/lexer) | 词法分析器 |
| `whipflow/token/token.go` | [whipflow/token/](https://github.com/npc-live/clawfirm/tree/main/whipflow/token) | Token 定义 |
| `whipflow/parser/parser.go` | [whipflow/parser/](https://github.com/npc-live/clawfirm/tree/main/whipflow/parser) | 语法分析器 |
| `whipflow/ast/ast.go` | [whipflow/ast/](https://github.com/npc-live/clawfirm/tree/main/whipflow/ast) | 抽象语法树 |
| `whipflow/validator/` | [whipflow/validator/](https://github.com/npc-live/clawfirm/tree/main/whipflow/validator) | 语法校验 |
| `whipflow/runtime/interpreter.go` | [whipflow/runtime/](https://github.com/npc-live/clawfirm/tree/main/whipflow/runtime) | 运行时解释器 |
| `whipflow/runtime/environment.go` | | 运行环境 |
| `whipflow/runtime/context.go` | | 运行上下文 |
| `whipflow/runtime/state_store.go` | | 状态持久化 |
| `whipflow/runtime/tools.go` | | 运行时工具注册 |
| `whipflow/runtime/provider.go` | | 运行时 Provider |
| `whipflow/complexity.go` | [whipflow/complexity.go](https://github.com/npc-live/clawfirm/blob/main/whipflow/complexity.go) | 复杂度评估 |

**功能**: `.whip` DSL 的完整编译器和运行时。Lexer → Parser → AST → Validator → Interpreter。支持 agent 定义、session、parallel、loop、if/throw 等语法。

---

### 6. 消息通道 — `channel/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `channel/telegram/channel.go` | [channel/telegram/](https://github.com/npc-live/clawfirm/tree/main/channel/telegram) | Telegram Bot |
| `channel/feishu/channel.go` | [channel/feishu/](https://github.com/npc-live/clawfirm/tree/main/channel/feishu) | 飞书 Bot |
| `channel/whatsapp/channel.go` | [channel/whatsapp/](https://github.com/npc-live/clawfirm/tree/main/channel/whatsapp) | WhatsApp Bot |
| `channel/webchat/handler.go` | [channel/webchat/](https://github.com/npc-live/clawfirm/tree/main/channel/webchat) | Web 聊天界面 |
| `channel/remote/` | [channel/remote/](https://github.com/npc-live/clawfirm/tree/main/channel/remote) | 远程隧道连接 (QR/WebSocket) |

**功能**: 多渠道消息接入。支持 Telegram、飞书、WhatsApp、Web 聊天、远程隧道。

---

### 7. 记忆系统 — `memory/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `memory/manager.go` | [memory/manager.go](https://github.com/npc-live/clawfirm/blob/main/memory/manager.go) | 记忆管理器 |
| `memory/chunk.go` | [memory/chunk.go](https://github.com/npc-live/clawfirm/blob/main/memory/chunk.go) | 文本分块 |
| `memory/embed.go` | [memory/embed.go](https://github.com/npc-live/clawfirm/blob/main/memory/embed.go) | 向量嵌入 |
| `memory/summarizer.go` | [memory/summarizer.go](https://github.com/npc-live/clawfirm/blob/main/memory/summarizer.go) | 摘要生成 |
| `memory/tools.go` | [memory/tools.go](https://github.com/npc-live/clawfirm/blob/main/memory/tools.go) | 记忆相关工具 |

**功能**: 长期记忆系统。文本分块、向量嵌入、摘要、检索。

---

### 8. 网关/多会话管理 — `gateway/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `gateway/server.go` | [gateway/server.go](https://github.com/npc-live/clawfirm/blob/main/gateway/server.go) | HTTP 服务器 |
| `gateway/manager.go` | [gateway/manager.go](https://github.com/npc-live/clawfirm/blob/main/gateway/manager.go) | 会话管理器 |
| `gateway/session.go` | [gateway/session.go](https://github.com/npc-live/clawfirm/blob/main/gateway/session.go) | 会话状态 |
| `gateway/runner.go` | [gateway/runner.go](https://github.com/npc-live/clawfirm/blob/main/gateway/runner.go) | Agent 运行器 |
| `gateway/agent_registry.go` | [gateway/agent_registry.go](https://github.com/npc-live/clawfirm/blob/main/gateway/agent_registry.go) | Agent 注册表 |
| `gateway/session_freshness.go` | | 会话保鲜 |
| `gateway/session_key.go` | | 会话密钥 |
| `gateway/docparse.go` | | 文档解析 |

**功能**: 多会话 HTTP 网关。管理多个并发 Agent 会话、消息路由、WebSocket 推送。

---

### 9. 密钥保险库 — `vault/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `vault/vault.go` | [vault/vault.go](https://github.com/npc-live/clawfirm/blob/main/vault/vault.go) | Vault 主接口 |
| `vault/input.go` | [vault/input.go](https://github.com/npc-live/clawfirm/blob/main/vault/input.go) | 密钥输入 |
| `vault/variant.go` | [vault/variant.go](https://github.com/npc-live/clawfirm/blob/main/vault/variant.go) | 变体 |
| `vault/crypto/crypto.go` | [vault/crypto/](https://github.com/npc-live/clawfirm/tree/main/vault/crypto) | 加密模块 |
| `vault/keychain/` | [vault/keychain/](https://github.com/npc-live/clawfirm/tree/main/vault/keychain) | macOS Keychain / 跨平台密钥链 |
| `vault/store/` | [vault/store/](https://github.com/npc-live/clawfirm/tree/main/vault/store) | BoltDB 密钥存储 |

**功能**: API Key 安全存储。加密 + macOS Keychain 集成 + BoltDB 持久化。

---

### 10. 其他核心模块

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `config/config.go` | [config/](https://github.com/npc-live/clawfirm/tree/main/config) | YAML 配置解析 (`config.yml`) |
| `types/` | [types/](https://github.com/npc-live/clawfirm/tree/main/types) | 公共类型定义 (Content, Message, Event, Model, Stream) |
| `message/` | [message/](https://github.com/npc-live/clawfirm/tree/main/message) | 消息序列化/裁剪 |
| `stream/` | [stream/](https://github.com/npc-live/clawfirm/tree/main/stream) | SSE 流式事件 |
| `store/` | [store/](https://github.com/npc-live/clawfirm/tree/main/store) | SQLite 持久化 (session/message/cronjob/kv/whipflow) |
| `cron/` | [cron/](https://github.com/npc-live/clawfirm/tree/main/cron) | 定时任务调度器 |
| `daemon/` | [daemon/](https://github.com/npc-live/clawfirm/tree/main/daemon) | 后台守护进程 |
| `auth/` | [auth/](https://github.com/npc-live/clawfirm/tree/main/auth) | OAuth + API Key 鉴权 |
| `skill/` | [skill/](https://github.com/npc-live/clawfirm/tree/main/skill) | Skill 解析/格式化 |
| `skillctl/` | [skillctl/](https://github.com/npc-live/clawfirm/tree/main/skillctl) | Skill 注册/同步控制器 |
| `funcs/` | [funcs/](https://github.com/npc-live/clawfirm/tree/main/funcs) | 内置函数库 (env/lexparser/library) |
| `internal/agentbuilder/` | [internal/agentbuilder/](https://github.com/npc-live/clawfirm/tree/main/internal/agentbuilder) | Agent 构建器 |

---

## 二、CLI 入口 — `cmd/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `cmd/clawfirm/main.go` | [cmd/clawfirm/](https://github.com/npc-live/clawfirm/tree/main/cmd/clawfirm) | **主 CLI** — 交互式 Agent + daemon + skill + vault 子命令 |
| `cmd/gateway/main.go` | [cmd/gateway/](https://github.com/npc-live/clawfirm/tree/main/cmd/gateway) | HTTP 网关服务 |
| `cmd/desktop/main.go` | [cmd/desktop/](https://github.com/npc-live/clawfirm/tree/main/cmd/desktop) | Wails 桌面应用入口 |
| `cmd/whip/main.go` | [cmd/whip/](https://github.com/npc-live/clawfirm/tree/main/cmd/whip) | Whip 工作流 CLI |
| `cmd/run-whip/main.go` | [cmd/run-whip/](https://github.com/npc-live/clawfirm/tree/main/cmd/run-whip) | Whip 工作流直接运行器 |
| `cmd/browser-shortcut/main.go` | [cmd/browser-shortcut/](https://github.com/npc-live/clawfirm/tree/main/cmd/browser-shortcut) | 浏览器 CDP 快捷操作独立 CLI |
| `cmd/media-gen/main.go` | [cmd/media-gen/](https://github.com/npc-live/clawfirm/tree/main/cmd/media-gen) | AI 图片生成插件 (支持 reference_images) |
| `cmd/media-understand/main.go` | [cmd/media-understand/](https://github.com/npc-live/clawfirm/tree/main/cmd/media-understand) | 多模态理解插件 |
| `cmd/func/main.go` | [cmd/func/](https://github.com/npc-live/clawfirm/tree/main/cmd/func) | 函数调用测试 |
| `cmd/wschat/main.go` | [cmd/wschat/](https://github.com/npc-live/clawfirm/tree/main/cmd/wschat) | WebSocket 聊天客户端 |

**编译产物** (`bin/`): `clawfirm`, `browser-shortcut`, `media-gen`, `media-understand`, `clawfirm.app`

---

## 三、资产层 — `app/assets/`

### 3.1 CDP 自动化脚本 — `app/assets/shortcuts/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `shortcuts/douyin.yaml` | [app/assets/shortcuts/douyin.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/douyin.yaml) | 抖音 — 视频发布/搜索/下载 + tag |
| `shortcuts/xhs.yaml` | [app/assets/shortcuts/xhs.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/xhs.yaml) | 小红书 — 视频发布 + tag |
| `shortcuts/xiaohongshu.yaml` | [app/assets/shortcuts/xiaohongshu.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/xiaohongshu.yaml) | 小红书 (别名) |
| `shortcuts/bilibili.yaml` | [app/assets/shortcuts/bilibili.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/bilibili.yaml) | B站 — 视频投稿 |
| `shortcuts/x.yaml` | [app/assets/shortcuts/x.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/x.yaml) | Twitter/X — 推文/Thread |
| `shortcuts/linkedin.yaml` | [app/assets/shortcuts/linkedin.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/linkedin.yaml) | LinkedIn — 视频/文本/媒体/点赞/评论 |
| `shortcuts/binance-square.yaml` | [app/assets/shortcuts/binance-square.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/binance-square.yaml) | 币安广场 — 短帖/长文/视频 |
| `shortcuts/channels.yaml` | [app/assets/shortcuts/channels.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/channels.yaml) | 频道管理 |
| `shortcuts/zhipin.yaml` | [app/assets/shortcuts/zhipin.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/zhipin.yaml) | Boss直聘 |

### 3.2 AI Skills (自然语言知识库) — `app/assets/skills/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| **social-publish/** | [skills/social-publish/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/social-publish) | 社交媒体发布技能集 |
| `social-publish/WORKFLOW.md` | | 发布工作流总览 |
| `social-publish/copywriting-base/` | | 通用文案基座 (标题公式/Hook/CTA) |
| `social-publish/douyin/` | | 抖音专属 (完播率/流量池/审核) |
| `social-publish/xiaohongshu/` | | 小红书专属 (CES/种草/图文) |
| `social-publish/bilibili/` | | B站专属 (三连/弹幕/分区) |
| `social-publish/twitter/` | | Twitter/X 专属 |
| `social-publish/binance-square/` | | 币安广场专属 (NFA/合规) |
| **video-skills/** | [skills/video-skills/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/video-skills) | 视频制作技能集 |
| `video-skills/WORKFLOW.md` | | 视频制作流水线总览 |
| `video-skills/video-script-generator/` | | Step 1: 脚本+分镜生成 |
| `video-skills/digital-avatar/` | | Step 2a: 数字人+声纹+口播 |
| `video-skills/scene-video-generator/` | | Step 2b: AI 场景视频 |
| `video-skills/voice-clone-tts/` | | 声纹克隆 + TTS |
| `video-skills/video-stitcher/` | | Step 3: 视频拼接+转场+BGM |
| `video-skills/cover-gen/` | | Step 4: 封面生成+首帧插入 |
| **其他 Skills** | | |
| `skills/agent-browser/` | [skills/agent-browser/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/agent-browser) | 浏览器 Agent 操作指南 |
| `skills/kling/` | [skills/kling/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/kling) | 可灵 AI 视频 API |
| `skills/remotion-video/` | [skills/remotion-video/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/remotion-video) | Remotion 视频渲染 |
| `skills/social-cli/` | [skills/social-cli/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/social-cli) | 社交平台 CLI 操作 |
| `skills/skill-index/` | [skills/skill-index/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/skill-index) | Skill 索引 |
| `skills/whipflow/` | [skills/whipflow/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/whipflow) | WhipFlow DSL 编写指南 |

### 3.3 Whip 工作流 — `app/assets/workflows/`

| 本地路径 | GitHub 路径 | 业务 |
|----------|-------------|------|
| **social-media/** | [workflows/social-media/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/social-media) | **自媒体运营** |
| `social-media/setup.whip` | | 初始化配置 |
| `social-media/daily-content.whip` | | 每日内容创作 (AI写文案+审核) |
| `social-media/daily-publish.whip` | | 每日多平台并行发布 |
| `social-media/repurpose.whip` | | 微信公众号→多平台改写 |
| `social-media/comments.whip` | | 评论管理 |
| `social-media/analytics.whip` | | 数据分析 |
| `social-media/weekly-report.whip` | | 周报生成 |
| **polymarket/** | [workflows/polymarket/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/polymarket) | **Polymarket 预测市场交易** |
| `polymarket/setup.whip` | | 环境+API+钱包 |
| `polymarket/scan.whip` | | 扫描市场信号 |
| `polymarket/trade.whip` | | 执行交易 |
| `polymarket/monitor.whip` | | 持仓监控 |
| `polymarket/report.whip` | | 绩效报告 |
| **hyperliquid/** | [workflows/hyperliquid/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/hyperliquid) | **Hyperliquid 永续合约交易** |
| `hyperliquid/setup~report.whip` | | 同上结构 |
| **arbitrage/** | [workflows/arbitrage/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/arbitrage) | **跨平台套利** |
| `arbitrage/setup~report.whip` | | setup/scan/buy/list/report |
| **amazon-affiliate/** | [workflows/amazon-affiliate/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/amazon-affiliate) | **亚马逊联盟营销** |
| `amazon-affiliate/*.whip` | | setup/research/write/publish/seo-monitor |
| **domains/** | [workflows/domains/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/domains) | **域名抢注** |
| `domains/*.whip` | | setup/scan/snipe/list/report |
| **saas/** | [workflows/saas/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/saas) | **SaaS 收购运营** |
| `saas/*.whip` | | setup/acquire/landing/launch/monitor/report |
| **gaokao/** | [workflows/gaokao/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/gaokao) | **高考志愿填报** |
| `gaokao/*.whip` | | setup/research/match/plan/report/run-all |
| **creator/** | [workflows/creator/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/creator) | **Whip 元生成器** |
| `creator/create.whip` | | 根据业务描述自动生成 whip 工作流 |

---

## 四、桌面/移动端 — `mobile/`

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `mobile/src-tauri/` | [mobile/src-tauri/](https://github.com/npc-live/clawfirm/tree/main/mobile/src-tauri) | Tauri (Rust) 后端 |
| `mobile/src/App.tsx` | [mobile/src/](https://github.com/npc-live/clawfirm/tree/main/mobile/src) | React 前端入口 |
| `mobile/src/pages/ChatsPage.tsx` | | 聊天列表页 |
| `mobile/src/pages/ChatView.tsx` | | 聊天详情页 |
| `mobile/src/pages/CanvasPage.tsx` | | Canvas 画板页 |
| `mobile/src/pages/CanvasView.tsx` | | Canvas 详情页 |
| `mobile/src/pages/ChannelsPage.tsx` | | 渠道管理页 |
| `mobile/src/pages/ScanPage.tsx` | | 扫码配对页 |
| `mobile/src/store.ts` | | 状态管理 |
| `mobile/src/api.ts` | | API 接口 |
| `mobile/vite.config.ts` | | Vite 构建配置 |

**功能**: Tauri + React 桌面/移动应用。提供 Chat、Canvas、Channels、Scan 等界面。

---

## 五、部署/运维

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `Dockerfile` | [Dockerfile](https://github.com/npc-live/clawfirm/blob/main/Dockerfile) | 主容器构建 |
| `deploy/Dockerfile` | [deploy/Dockerfile](https://github.com/npc-live/clawfirm/blob/main/deploy/Dockerfile) | Gateway 部署容器 |
| `deploy/clawfirm-gateway.service` | [deploy/](https://github.com/npc-live/clawfirm/tree/main/deploy) | systemd 服务文件 |
| `Makefile` | [Makefile](https://github.com/npc-live/clawfirm/blob/main/Makefile) | 构建命令 |
| `scripts/run-desktop.sh` | [scripts/](https://github.com/npc-live/clawfirm/tree/main/scripts) | 桌面应用启动脚本 |
| `wails.json` | [wails.json](https://github.com/npc-live/clawfirm/blob/main/wails.json) | Wails 桌面框架配置 |
| `go.work` | [go.work](https://github.com/npc-live/clawfirm/blob/main/go.work) | Go workspace |

---

## 六、其他

| 本地路径 | GitHub 路径 | 说明 |
|----------|-------------|------|
| `examples/` | [examples/](https://github.com/npc-live/clawfirm/tree/main/examples) | 示例 (.whip + quickstart) |
| `docs/` | [docs/](https://github.com/npc-live/clawfirm/tree/main/docs) | 开发文档 (部署指南/重构计划) |
| `media.whip` | [media.whip](https://github.com/npc-live/clawfirm/blob/main/media.whip) | 抖音爆款分析+创作完整工作流 |
| `skills-lock.json` | [skills-lock.json](https://github.com/npc-live/clawfirm/blob/main/skills-lock.json) | Skill 版本锁定 |
| `config/example.yml` | [config/example.yml](https://github.com/npc-live/clawfirm/blob/main/config/example.yml) | 配置文件示例 |
| `_ops/` | (本地, 未入 git) | 运营资料/分析文档 |

---

## 架构总图

```
┌─────────────────────────────────────────────────────────────────────┐
│                          用户界面层                                  │
│  cmd/clawfirm (CLI)  │  mobile/ (Tauri)  │  channel/ (Telegram等)   │
└──────────┬───────────┴────────┬──────────┴──────────┬───────────────┘
           │                    │                     │
┌──────────▼────────────────────▼─────────────────────▼───────────────┐
│                        gateway/ (HTTP 网关)                         │
│                     多会话管理 + WebSocket                           │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                         agent/ (核心引擎)                            │
│              对话循环 + 上下文 + 系统提示词 + 工具调度                  │
└────┬──────────┬──────────┬──────────┬──────────┬────────────────────┘
     │          │          │          │          │
┌────▼───┐ ┌───▼────┐ ┌───▼────┐ ┌───▼────┐ ┌───▼──────┐
│provider│ │ tool/  │ │memory/ │ │ skill/ │ │whipflow/ │
│多LLM   │ │内置工具│ │记忆系统│ │技能系统│ │工作流引擎│
└────────┘ └───┬────┘ └────────┘ └────────┘ └──────────┘
               │
    ┌──────────┼──────────┐
    │          │          │
┌───▼───┐ ┌───▼────┐ ┌───▼────────────┐
│browser│ │ vault/ │ │ app/assets/    │
│CDP引擎│ │密钥保险│ │shortcuts/skills│
│       │ │  库    │ │/workflows      │
└───────┘ └────────┘ └────────────────┘
```
