# ClawFirm

AI Gateway — 一个用 Go 构建的 AI 智能体平台，支持多 LLM 提供商、对话记忆、工作流编排，并提供 CLI、HTTP 网关和桌面应用三种使用方式。

AI Gateway — An AI agent platform built with Go, supporting multiple LLM providers, conversation memory, workflow orchestration, available as CLI, HTTP gateway, and desktop app.

---

## 业务模块 / Business Modules

ClawFirm.dev 是专为新时代个体创业者打造的"一人公司"自动化引擎。通过 AI 深度嵌入商业全链路，提供三种核心盈利增长模式：

ClawFirm.dev is an automation engine built for the new generation of solo entrepreneurs — powering the "one-person company." AI is deeply embedded across the entire business chain, offering three core growth models:

- **全栈软件出海 / Full-Stack Software for Global Markets** — AI 负责用户调研、写代码、审代码、自动化营销，目标月入 $1K+ 独立产品。From deep user research and feature development to automated marketing, AI helps you build and distribute software products like a full team.
- **自动化套利交易 / Automated Arbitrage Trading** — AI 捕捉市场信号，7x24 自动执行策略（链上/交易所价差、期货、预测市场）。Leveraging AI to capture market signals with precision, executing efficient algorithmic trades through automated workflows.
- **自媒体矩阵分发 / Social Media Matrix Distribution** — AI 生成内容并自动发布到小红书/X/B站/公众号等多平台，实现粉丝→流量→变现闭环。AI auto-generates high-conversion content and distributes across platforms, enabling your personal brand to go viral.

### Whip 工作流模块 / Whip Workflow Modules

每个业务模块在 `whips/` 下有独立子目录，包含 5 个标准文件：`setup.whip` · `scan.whip` · `trade.whip` · `monitor.whip` · `report.whip`

Each business module has its own subdirectory under `whips/` with 5 standard files: `setup.whip` · `scan.whip` · `trade.whip` · `monitor.whip` · `report.whip`

| 模块 / Module | 业务 / Description (CN) | Description (EN) | 类型 / Type |
|---|---|---|---|
| `polymarket` | 天气预测市场交易 | Weather prediction market trading on [Polymarket](https://polymarket.com/?r=Olivia) | 套利交易 / Trading |
| `hyperliquid` | 新闻驱动加密期货交易 | News-driven crypto futures trading on Hyperliquid | 套利交易 / Trading |
| `social-media` | 社交媒体内容自动生成与发布 | Auto-generate and publish content across social platforms | 自媒体 / Content |
| `arbitrage` | 电商跨平台套利 (闲鱼↔拼多多 / eBay↔Amazon) | Cross-platform e-commerce arbitrage (Xianyu↔Pinduoduo / eBay↔Amazon) | 电商 / E-Commerce |
| `domains` | 过期高价值域名抢注并转卖 | Snipe expiring high-value domains and resell on Sedo/Afternic | 数字资产 / Digital Assets |
| `amazon-affiliate` | AI 写 SEO 文章 → 亚马逊联盟佣金 | AI-written SEO articles for Amazon affiliate commissions | 联盟营销 / Affiliate |

```bash
# 标准运行顺序 / Standard run order
whipflow run whips/<module>/setup.whip    # 初始化（只需一次）/ Initialize (once)
whipflow run whips/<module>/monitor.whip  # 启动监控（长期运行）/ Start monitor (long-running)
whipflow run whips/<module>/scan.whip     # 手动扫描 / Manual scan
whipflow run whips/<module>/trade.whip    # 执行交易/动作 / Execute trade/action
whipflow run whips/<module>/report.whip   # 查看报告 / View report
```

---

## 目录结构 / Project Structure

```
clawfirm/
├── agent/              # 智能体核心 / Agent core: event loop, state machine, tool executor, context compression
│   ├── loop.go         #   主循环 / Main loop (stream → tool call → respond)
│   ├── agent.go        #   Agent 结构体 / Agent struct & functional options
│   ├── tool_executor.go#   工具调用分发 / Tool call dispatcher
│   ├── compressor.go   #   上下文压缩 / Context compression (LLM summary at 80% window)
│   ├── systemprompt.go #   系统提示词构建 / System prompt builder
│   ├── bootstrap.go    #   初始化引导 / Bootstrap
│   ├── context.go      #   上下文管理 / Context management
│   ├── state.go        #   状态机定义 / State machine definition
│   ├── queue.go        #   消息队列 / Message queue
│   └── temporal.go     #   时间相关工具 / Temporal utilities
│
├── provider/           # LLM 提供商适配层 / LLM provider adapters
│   ├── interface.go    #   Provider 接口定义 / Provider interface
│   ├── registry.go     #   全局 Provider 注册表 / Global provider registry
│   ├── anthropic/      #   Anthropic (Claude)
│   ├── openai/         #   OpenAI
│   ├── gemini/         #   Google Gemini
│   ├── ollama/         #   Ollama (local models)
│   └── zenmux/         #   Zenmux (aggregation proxy)
│
├── types/              # 核心类型定义 / Core type definitions
│   ├── message.go      #   Message, Role
│   ├── content.go      #   ContentBlock (text/image/tool_use/tool_result)
│   ├── event.go        #   AgentEvent (streaming events)
│   ├── model.go        #   Model metadata
│   └── stream.go       #   StreamOptions
│
├── memory/             # 语义记忆系统 / Semantic memory system
│   ├── manager.go      #   Memory Manager: 索引/搜索/混合重排 / index/search/hybrid rerank (0.7 cosine + 0.3 BM25)
│   ├── embed.go        #   Embedding 提供商接口 / Embedding provider interface (OpenAI/Gemini/Voyage/Mistral/Ollama)
│   ├── chunk.go        #   文本分块 / Text chunking (400 token, 80 overlap)
│   ├── tools.go        #   memory_search / memory_get 工具 / tools
│   └── summarizer.go   #   自动摘要 / Auto-summary (30 min timer → .md file)
│
├── store/              # SQLite 持久化层 / SQLite persistence (WAL mode, modernc.org/sqlite)
│   ├── db.go           #   数据库连接与迁移 / DB connection & migrations
│   ├── session.go      #   会话管理 / Session management
│   ├── message.go      #   消息存储 / Message storage
│   ├── kv.go           #   键值存储 / Key-value store
│   ├── vault.go        #   加密存储 / Encrypted storage
│   ├── cronjob.go      #   定时任务持久化 / Cron job persistence
│   └── migrations/     #   SQL 迁移文件 / SQL migration files (001-006)
│
├── stream/             # SSE 流式处理 / SSE streaming
│   ├── sse.go          #   SSE 解析器 / SSE parser
│   ├── event_stream.go #   EventStream 封装 / EventStream wrapper
│   └── retry.go        #   重试逻辑 / Retry logic
│
├── tool/               # 工具系统 / Tool system
│   ├── base.go         #   AgentTool 接口 / AgentTool interface + BaseToolImpl
│   ├── builtin/        #   内置工具 / Built-in tools
│   ├── registry.go     #   工具注册表 / Tool registry
│   └── registry_test.go
│
├── message/            # 消息处理工具 / Message utilities
│   ├── convert.go      #   格式转换 / Format conversion
│   ├── serialize.go    #   序列化 / Serialization
│   └── prune.go        #   消息裁剪 / Message pruning
│
├── skill/              # Skill 系统 / Skill system (Markdown-defined reusable prompt templates)
│   ├── skill.go        #   Skill 解析与加载 / Skill parsing & loading
│   └── format.go       #   格式化输出 / Formatted output
│
├── gateway/            # HTTP 网关服务 / HTTP gateway service
│   ├── server.go       #   HTTP 服务器 / HTTP server (default :9988)
│   ├── manager.go      #   Agent 管理器 / Agent manager
│   ├── session.go      #   会话路由 / Session routing
│   ├── agent_registry.go#  Agent 注册 / Agent registration
│   └── docparse.go     #   文档解析 / Document parsing
│
├── channel/            # 消息渠道适配 / Messaging channel adapters
│   ├── webchat/        #   WebSocket 聊天 / WebSocket chat (WebChat)
│   ├── feishu/         #   飞书机器人 / Feishu (Lark) bot
│   └── whatsapp/       #   WhatsApp (go-whatsmeow)
│
├── whipflow/           # WhipFlow 工作流引擎 / WhipFlow workflow engine (.whip DSL)
│   ├── whipflow.go     #   入口 / Entry point
│   ├── lexer/          #   词法分析器 / Lexer
│   ├── parser/         #   语法分析器 / Parser
│   ├── ast/            #   抽象语法树 / Abstract syntax tree
│   ├── token/          #   Token 定义 / Token definitions
│   ├── runtime/        #   运行时执行器 / Runtime executor
│   └── validator/      #   校验器 / Validator
│
├── auth/               # 认证与密钥管理 / Authentication & key management
│   ├── storage.go      #   密钥存储 / Key storage
│   ├── keychain.go     #   macOS Keychain 集成 / macOS Keychain integration
│   ├── resolver.go     #   Key 解析 / Key resolver
│   └── oauth/          #   OAuth PKCE 流程 / OAuth PKCE flow
│
├── config/             # 配置管理 / Configuration
│   ├── config.go       #   YAML 配置解析 / YAML config parser (~/.clawfirm/config.yml)
│   └── example.yml     #   示例配置 / Example config
│
├── cron/               # 定时任务调度 / Cron scheduler
│   ├── scheduler.go    #   Cron 调度器 / Cron scheduler
│   └── schedule.go     #   调度规则 / Schedule rules
│
├── funcs/              # 技术指标函数库 / Technical indicator function library
│   ├── env.go          #   计算环境 / Computation environment (OHLCV data)
│   ├── lexparser.go    #   公式解析器 / Formula parser
│   └── library.go      #   内置函数 / Built-in functions (MA, EMA, RSI, etc.)
│
├── app/                # 桌面应用后端 / Desktop app backend (Wails binding)
│   ├── app.go          #   App 结构体与 Wails 方法 / App struct & Wails methods
│   ├── config_writer.go#   配置写入 / Config writer
│   └── assets.go       #   嵌入资源 / Embedded assets
│
├── internal/
│   └── agentbuilder/   # 从配置构建 Agent 实例 / Build Agent instances from config
│
├── testutil/           # 测试辅助 / Test utilities
│   ├── fixtures.go
│   ├── mock_provider.go
│   └── mock_tool.go
│
├── cmd/                # 可执行入口 / Executable entry points
│   ├── pi/             #   CLI: 单轮对话 / Single-turn conversation (`pi -p "prompt"`)
│   ├── gateway/        #   HTTP 网关服务 / HTTP gateway (WebSocket + REST)
│   ├── desktop/        #   桌面应用 / Desktop app (Wails + React + Tailwind)
│   │   └── frontend/   #     React 前端 / React frontend (Vite + TypeScript)
│   ├── whip/           #   WhipFlow 运行器 / WhipFlow runner (`whip file.whip`)
│   ├── wschat/         #   WebSocket 测试客户端 / WebSocket test client
│   └── func/           #   技术指标计算 CLI / Technical indicator CLI
│
├── examples/           # 示例文件 / Example files
├── scripts/            # 辅助脚本 / Helper scripts
├── bin/                # 构建产物 / Build artifacts
├── Makefile            # 构建 / Build: make all / make app / make test
├── go.mod              # Go 1.25.1, module github.com/ai-gateway/clawfirm
└── go.work             # Go workspace
```

## 技术栈 / Tech Stack

| 层 / Layer | 技术 / Technology |
|---|---|
| 语言 / Language | Go 1.25.1 |
| 数据库 / Database | SQLite (modernc.org/sqlite, pure Go, WAL mode) |
| 桌面 / Desktop | Wails v2 + React + Vite + Tailwind CSS |
| 消息渠道 / Channels | WebSocket / Feishu (Lark) / WhatsApp |
| LLM | Anthropic · OpenAI · Gemini · Ollama · Zenmux |

## 快速开始 / Quick Start

```bash
# 构建所有 CLI 工具 + 桌面应用 / Build all CLI tools + desktop app
make all

# 仅构建 macOS .app / Build macOS .app only
make app

# 运行测试 / Run tests
make test

# CLI 单轮对话 / CLI single-turn conversation
go run ./cmd/pi -p "Hello"

# 启动网关 / Start gateway
go run ./cmd/gateway

# 运行 WhipFlow 工作流 / Run WhipFlow workflow
go run ./cmd/whip example.whip
```

## 社群入口 / Community

- **Discord:** [Join our Discord](https://discord.gg/JNXz2utFW8)
- **X (Twitter):** [@0xOliviaPp](https://x.com/0xOliviaPp)
- **WeChat 微信:** PpCiting

---

## 免责声明 / Disclaimer

**中文**

本项目（ClawFirm）及其所有代码、工作流、策略、脚本和文档仅供学习、研究和技术演示之用，**不构成任何投资建议、交易指导或财务咨询**。

1. **交易与金融风险**：本项目包含的加密货币交易、预测市场、期货合约等金融类代码和策略，可能导致部分或全部本金亏损。过往回测数据不代表未来表现。使用者应充分了解相关市场风险，并自行承担一切交易损失。
2. **一人公司与垫资风险**：本项目涉及的电商套利、域名抢注等业务模式可能需要使用者自行垫付资金。任何因资金投入产生的亏损、滞销或无法回本，均由使用者自行承担。
3. **无担保**：本项目按"原样"（AS IS）提供，不做任何明示或暗示的保证，包括但不限于适销性、特定用途适用性和盈利能力的保证。
4. **责任限制**：在任何情况下，本项目的作者和贡献者均不对因使用或无法使用本项目而产生的任何直接、间接、附带、特殊或后果性损害承担责任。

**使用本项目即表示您已阅读、理解并同意本免责声明的全部内容。如不同意，请勿使用本项目。**

**English**

This project (ClawFirm) and all its code, workflows, strategies, scripts, and documentation are provided solely for educational, research, and technical demonstration purposes. **Nothing in this project constitutes investment advice, trading guidance, or financial consulting.**

1. **Trading & Financial Risk**: The cryptocurrency trading, prediction market, and futures contract code and strategies included in this project may result in partial or total loss of capital. Past backtest results do not guarantee future performance. Users must fully understand the associated market risks and bear all trading losses themselves.
2. **Self-Funded Business Risk**: Business models in this project such as e-commerce arbitrage and domain sniping may require users to invest their own capital upfront. Any losses, unsold inventory, or inability to recoup funds are the sole responsibility of the user.
3. **No Warranty**: This project is provided "AS IS" without warranty of any kind, express or implied, including but not limited to warranties of merchantability, fitness for a particular purpose, and profitability.
4. **Limitation of Liability**: In no event shall the authors or contributors of this project be liable for any direct, indirect, incidental, special, or consequential damages arising from the use of or inability to use this project.

**By using this project, you acknowledge that you have read, understood, and agreed to this disclaimer in its entirety. If you do not agree, do not use this project.**

## License

MIT License
