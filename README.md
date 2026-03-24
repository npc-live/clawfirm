# WhipFlow

确定性 AI 工作流引擎 — [ClawFirm](https://clawfirm.dev) 的核心运行时

Deterministic AI Workflow Engine — the core runtime of [ClawFirm](https://clawfirm.dev)

---

## 简介 / About

WhipFlow 使用自定义的 `.whip` DSL 编排多 AI agent 协作工作流。它是 ClawFirm「一人公司自动化引擎」中实际执行任务的核心组件。

WhipFlow orchestrates multi-agent AI workflows using a custom `.whip` DSL. It is the core execution component of ClawFirm's "one-person company automation engine."

**核心能力 / Core Capabilities:**

- 多 agent 角色定义（不同模型、不同工具权限） / Multi-agent roles (different models, different tool permissions)
- 顺序 / 条件 / 循环执行 / Sequential, conditional, and loop execution
- 变量传递和上下文拼接 / Variable passing and context chaining
- 自动校验循环（validator agent loop）/ Auto-validation loops
- JSON-line 事件协议（对接 Tauri 桌面端）/ JSON-line event protocol (Tauri desktop integration)

---

## 安装 / Installation

```bash
# 全局安装 / Global install
npm install -g @harness.farm/whipflow

# 或通过 ClawFirm 安装全套工具 / Or install via ClawFirm
npm install -g clawfirm
clawfirm install
```

**本地开发 / Local development:**

```bash
bun install
bun run dev          # 开发模式 / Dev mode
bun run build        # 编译为单文件 / Compile to single binary
```

---

## CLI 命令 / CLI Commands

```bash
whipflow run <file.whip>          # 执行工作流 / Execute a workflow
whipflow validate <file.whip>     # 校验语法 / Validate syntax
whipflow compile <file.whip>      # 查看编译结果 / Show compiled form
whipflow install-skills           # 安装 skills 到 Claude Code / Install skills to Claude Code
whipflow install-skills --force   # 强制覆盖 / Force overwrite
whipflow help                     # 帮助 / Help
```

---

## .whip DSL 语法示例 / DSL Syntax Example

```whip
agent researcher:
  provider: "claude-code"
  model: opus
  tools: ["bash", "read", "write"]
  prompt: "你是一位市场研究专家。"

agent writer:
  provider: "claude-code"
  model: sonnet
  tools: ["bash", "write"]
  prompt: "将收到的内容写入指定文件。"

agent validator:
  provider: "claude-code"
  model: haiku
  tools: ["bash", "read"]
  prompt: "审核内容质量，输出 APPROVED 或 NEEDS_REVISION: [原因]。"

let research = session: researcher
  prompt: "分析目标市场的竞争格局..."

let check = session: validator
  prompt: "审核研究报告质量：{research}"

loop until **check 包含 APPROVED** (max: 2):
  research = session: researcher
    prompt: "根据审核意见修订：{check}"
  check = session: validator
    prompt: "重新审核：{research}"

session: writer
  prompt: "将最终报告写入 docs/report.md：{research}"
```

---

## ACP 模式 / ACP Mode

WhipFlow 可作为 MCP 兼容的工具服务器，让其他 agent（Cursor、Claude Code 等）通过 JSON-RPC 2.0 stdio 调用。

WhipFlow can act as an MCP-compatible tool server so other agents (Cursor, Claude Code, etc.) can call it over JSON-RPC 2.0 via stdio.

```bash
whipflow acp
```

在 `.cursor/mcp.json` 或任何 MCP 兼容主机中注册 / Register in MCP-compatible host:

```json
{
  "mcpServers": {
    "whipflow": {
      "command": "whipflow",
      "args": ["acp"]
    }
  }
}
```

**暴露的工具 / Exposed Tools:**

| 工具 / Tool | 说明 / Description |
|------|-------------|
| `whipflow_run_file` | 执行 `.whip` 工作流文件 / Execute a `.whip` workflow file |
| `whipflow_run_source` | 执行内联 `.whip` 源码 / Execute inline `.whip` source code |
| `whipflow_validate` | 校验 `.whip` 语法 / Validate `.whip` syntax without running |

---

## 配置 / Configuration

项目级配置文件 `.whipflow.json` / Project-level config:

```json
{
  "providers": {
    "mymodel": {
      "bin": "opencode",
      "args": ["run"],
      "promptMode": "arg"
    }
  },
  "defaultProvider": "claude",
  "conditionProvider": "claude",
  "toolsDir": "~/.whipflow/tools"
}
```

`defaultProvider` 为所有 session 设置默认 provider（agent 未指定时使用，默认 `claude-code`）。`conditionProvider` 仅用于条件判断（`discretion`）和选项选择（`choice`）的评估，未设置时回退到 `defaultProvider`。

`defaultProvider` sets the provider for all sessions when not specified on the agent (default: `claude-code`). `conditionProvider` overrides the provider for `discretion` and `choice` evaluation only; falls back to `defaultProvider`.

---

## 项目结构 / Project Structure

```
whipflow/
├── bin/whipflow.ts              CLI 入口 / CLI entry point
├── commands/                    Claude Code skill 命令 / Skill commands
│   ├── run.md                   /run 命令 / Run command
│   └── validate.md              /validate 命令 / Validate command
├── src/social/                  自媒体自动化引擎 / Social media automation engine
│   ├── pipeline-orchestrator.ts 12 阶段流水线编排 / 12-stage pipeline orchestrator
│   ├── scheduler.ts             定时调度器 / Scheduled task runner
│   ├── content-creator.ts       多平台内容生成 / Multi-platform content creator
│   ├── publish-orchestrator.ts  发布编排 / Publish orchestrator
│   ├── cdp-client.ts            Chrome DevTools Protocol 客户端 / CDP client
│   ├── cdp-session.ts           浏览器 Cookie 持久化 / Browser session persistence
│   ├── yaml-runner.ts           YAML 步骤执行器 / YAML step executor
│   ├── db.ts                    SQLite 数据层（7 张表）/ SQLite data layer (7 tables)
│   ├── cli.ts                   社媒 CLI 入口 / Social CLI entry
│   └── ...                      20+ 个功能模块 / 20+ functional modules
├── adapters/                    平台自动化适配器 / Platform automation adapters
│   └── x.yaml                   Twitter 完整适配器 / Twitter full adapter
├── client/                      Tauri 桌面应用「社媒运营助手」/ Tauri desktop app
│   ├── src-tauri/               Rust 后端 / Rust backend
│   └── src/                     前端（Vite + React）/ Frontend
├── flows/                       .whip 工作流文件 / Workflow files
├── skills/                      Claude Code skills 定义 / Skill definitions
│   ├── whipflow/SKILL.md        WhipFlow skill
│   ├── remotion-video/SKILL.md  视频制作 skill / Video production skill
│   └── WORKFLOW.md              视频流水线编排文档 / Video pipeline doc
├── macroflow/                   Web 端 monorepo / Web monorepo
│   ├── packages/web/            Next.js 前端 / Frontend
│   ├── packages/server/         Fastify 后端 / Backend
│   └── packages/shared/         共享模块（news + hyperliquid）/ Shared modules
├── demo-video/                  Remotion 演示视频项目 / Demo video project
├── data/                        运行时数据 / Runtime data
├── logs/                        运行日志 / Execution logs
├── research-output/             AI 生成的调研文档 / AI-generated research docs
└── docs/                        品牌 / 竞品 / 内容策略 / Brand & strategy docs
```

---

## 业务模块 / Business Modules

WhipFlow 为 ClawFirm 的七大业务模块提供工作流定义，每个模块遵循标准 5 文件结构：

WhipFlow provides workflow definitions for ClawFirm's seven business modules, each following a standard 5-file structure:

| 文件 / File | 职责 / Role | 运行时机 / When |
|---|---|---|
| `setup.whip` | 环境检查、API 验证、配置写入 / Env check, API validation, config | 首次 / First-time |
| `scan.whip` | 数据拉取、信号识别 / Data fetch, signal detection | 手动或定时 / Manual or scheduled |
| `trade.whip` | 风控检查、执行动作 / Risk check, execute action | 手动或定时 / Manual or scheduled |
| `monitor.whip` | 持续轮询、状态管理 / Continuous polling, state mgmt | 后台运行 / Background |
| `report.whip` | 统计指标、分析报告 / Stats, analysis report | 随时 / Anytime |

---

### 模块一览 / Module Overview

#### 1. saas — SaaS 软件出海 / SaaS Go-Global

从竞品分析到 Product Hunt 发布的全流程自动化。

Full pipeline from competitor analysis to Product Hunt launch.

```bash
whipflow run whips/saas/setup.whip      # 竞品分析 + GTM 策略 / Competitor analysis + GTM strategy
whipflow run whips/saas/landing.whip    # 英文落地页文案（A/B 版本）/ Landing page copy (A/B variants)
whipflow run whips/saas/acquire.whip    # 多渠道获客 / Multi-channel acquisition
whipflow run whips/saas/launch.whip     # Product Hunt 发布素材 + 小时级计划 / PH launch kit
whipflow run whips/saas/monitor.whip    # 每日反馈监控 / Daily feedback monitoring
whipflow run whips/saas/report.whip     # 增长周报 / Growth weekly report
```

**获客渠道 / Acquisition channels:** Reddit, Hacker News, Cold Email, SEO Article

#### 2. hyperliquid — 新闻驱动期货交易 / News-Driven Futures Trading

用 Claude 评估加密货币新闻信号，在 Hyperliquid 自动开多/空。

Uses Claude to assess crypto news signals, auto-opens long/short on Hyperliquid.

```bash
whipflow run whips/hyperliquid/setup.whip
whipflow run whips/hyperliquid/monitor.whip

# 或独立运行 / Or run standalone
HL_PRIVATE_KEY=0x... node scripts/hl-news-trader.js monitor
```

**策略参数 / Strategy params:** 最多 4 仓位 / Max 4 positions, 3-5x 杠杆 / leverage, 5% 止损 / SL, 8% 止盈 / TP

#### 3. polymarket — 天气预测市场交易 / Weather Prediction Market Trading

用 Open-Meteo 预报计算胜率，与 Polymarket 市场价比较，发现边缘时下单。

Calculates win probability via Open-Meteo forecasts, compares with market odds, bets when edge found.

```bash
whipflow run whips/polymarket/setup.whip
whipflow run whips/polymarket/monitor.whip
```

**回测指标 / Backtested:** 胜率 / Win rate 57.26%, Sharpe 3.60, 最大回撤 / Max drawdown 10.87%

#### 4. social-media — 社交媒体内容自动化 / Social Media Automation

AI 生成内容，自动发布到多个平台。

AI-generated content, auto-published to multiple platforms.

```bash
whipflow run whips/social-media/setup.whip
whipflow run whips/social-media/daily-content.whip    # 每日内容 / Daily content
whipflow run whips/social-media/daily-publish.whip    # 多平台发布 / Multi-platform publish
whipflow run whips/social-media/comments.whip         # 评论互动 / Comment management
whipflow run whips/social-media/weekly-report.whip    # 周报 / Weekly report
whipflow run whips/social-media/repurpose.whip        # 内容改写 / Content repurpose
```

**平台 / Platforms:** 小红书, 微博, B站, Twitter, Telegram

#### 5. arbitrage — 电商跨平台套利 / Cross-Platform E-Commerce Arbitrage

扫描价差，自动采购和上架。目标利润率 > 20%。

Scans price gaps, auto-purchases and lists. Target margin > 20%.

```bash
whipflow run whips/arbitrage/setup.whip
whipflow run whips/arbitrage/scan.whip
whipflow run whips/arbitrage/buy.whip
whipflow run whips/arbitrage/report.whip
```

**市场 / Markets:** 闲鱼↔拼多多（国内）/ eBay↔Amazon（海外）

#### 6. domains — 域名捡漏 / Domain Sniping

扫描过期域名，自动注册，挂牌出售。

Scans expiring domains, auto-registers, lists for resale.

```bash
whipflow run whips/domains/setup.whip
whipflow run whips/domains/scan.whip
whipflow run whips/domains/snipe.whip
whipflow run whips/domains/report.whip
```

#### 7. amazon-affiliate — 亚马逊联盟营销 / Amazon Affiliate Marketing

关键词研究 → AI 写 SEO 文章 → 自动发布 → 排名监控。

Keyword research → AI-written SEO articles → auto-publish → rank monitoring.

```bash
whipflow run whips/amazon-affiliate/setup.whip
whipflow run whips/amazon-affiliate/research.whip
whipflow run whips/amazon-affiliate/write.whip
whipflow run whips/amazon-affiliate/publish.whip
whipflow run whips/amazon-affiliate/seo-monitor.whip
```

---

## 创建新模块 / Create New Modules
### saas — SaaS 软件出海 / SaaS Global Launch

竞品分析 + GTM 策略 → 英文落地页文案 → 多渠道获客（Reddit / HN / Cold Email / SEO）→ Product Hunt 发布 → 增长周报。

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "saas-001",
  "product_name": "YourProduct",
  "product_desc": "One sentence description in English",
  "product_url": "https://yourproduct.com",
  "product_category": "productivity",
  "target_market": "US",
  "target_persona": "startup founders",
  "pricing_model": "freemium",
  "monthly_price_usd": 19,
  "current_mrr_usd": 0,
  "launch_channels": ["producthunt", "reddit", "hacker-news", "seo-article"]
}
EOF
whipflow run whips/saas/setup.whip     # 竞品分析 + GTM 策略
whipflow run whips/saas/landing.whip   # 英文落地页文案
whipflow run whips/saas/acquire.whip   # 获客内容
whipflow run whips/saas/launch.whip    # Product Hunt 发布套件
whipflow run whips/saas/monitor.whip   # 每日监控 + 回复草稿
whipflow run whips/saas/report.whip    # 增长周报
```

详见 [`whips/saas/README.md`](whips/saas/README.md)。

---

### gaokao — 高考志愿填报 / College Application Advisor

综合分数位次、地域偏好、家庭经济、职业目标，生成冲/稳/保志愿方案和可打印的填报清单。

```bash

whipflow run whips/gaokao/run-all.whip     
# 学生画像 + 填报策略框架  
# 院校调研 + 专业就业 + 城市成本
# 多维评分 + 冲稳保候选列表
# 最终志愿顺序 + 打印速查表
# 综合报告（家长可读版）
```

详见 [`whips/gaokao/README.md`](whips/gaokao/README.md)。

---

## 创建新的 Whip 模块 / Create New Whip Modules

`creator` 是 meta-whip，从自然语言描述自动生成完整的 whip 子目录。

`creator` is a meta-whip that auto-generates a complete whip subdirectory from a business description.

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "create-001",
  "name": "my-strategy",
  "description": "监控 Reddit，识别热门话题，自动发布到 Twitter",
  "apis": ["Reddit API", "Twitter API v2"]
}
EOF

whipflow run whips/creator/create.whip
# → whips/my-strategy/ (setup / scan / trade / monitor / report)
```

---

## 自媒体引擎 / Social Media Engine

`src/social/` 包含独立于 .whip 工作流的 TypeScript 自动化引擎，可直接运行或通过 Tauri 桌面应用调用。

`src/social/` contains a standalone TypeScript automation engine, runnable directly or via the Tauri desktop app.

**12 阶段流水线 / 12-Stage Pipeline:**

```
env-setup → account-config → competitive-analyst → strategy-planner
→ content-calendar → content-creator → content-validator
→ publish-orchestrator → analytics-collector → insights-engine
→ comment-manager → weekly-reporter
```

**浏览器自动化 / Browser Automation:**

通过 Chrome DevTools Protocol 直连浏览器，支持 YAML 适配器驱动的自动化操作。

Direct browser connection via Chrome DevTools Protocol, with YAML adapter-driven automation.

```bash
# 社媒 CLI / Social CLI
bun run src/social/cli.ts x search "claude ai"
bun run src/social/cli.ts xhs search "独立开发"
bun run src/social/cli.ts x post "Hello from WhipFlow!"
bun run src/social/cli.ts x like https://x.com/user/status/123

# 定时调度 / Scheduled runner
tsx src/social/scheduler.ts

# Tauri 桌面应用 / Tauri desktop app
cd client && npm run tauri dev
```

---

## Skills 体系 / Skills System

WhipFlow 通过 `skillctl` 将 AI skills 同步到 Claude Code，使其在任何会话中可用。

WhipFlow syncs AI skills to Claude Code via `skillctl`, making them available in any session.

```bash
whipflow install-skills
# 然后在 Claude Code 中使用 /whipflow
# Then use /whipflow in any Claude Code session
```

**已有 Skills / Available Skills:**

| Skill | 说明 / Description |
|---|---|
| `whipflow` | WhipFlow DSL 语法参考和使用指南 / DSL syntax reference and usage guide |
| `remotion-video` | Remotion 视频制作 / Video production with Remotion |
| `social-publish/*` | 各平台内容创作（小红书/B站/抖音/Twitter + 算法指南 + 爆文模板）/ Platform-specific content creation |
| `video-skills/*` | AI 视频全链路（脚本→数字人→场景→TTS→拼接）/ Full video pipeline |

---

## 技术栈 / Tech Stack

| 层 / Layer | 技术 / Technology |
|---|---|
| 运行时 / Runtime | Bun (primary), Node.js |
| 语言 / Language | TypeScript |
| 工作流 DSL | `.whip` (OpenProse) |
| AI 模型 / AI Models | Claude API (opus / sonnet / haiku) |
| 数据库 / Database | SQLite (better-sqlite3) |
| 浏览器自动化 / Browser | Chrome DevTools Protocol (WebSocket) |
| 桌面 / Desktop | Tauri 2.0 (Rust + WebView) |
| Web | Next.js + Fastify (macroflow/) |
| 链上 / On-chain | viem (EVM) |
| 视频 / Video | Remotion |
| 密钥 / Secrets | openvault |
| 部署 / Deploy | Fly.io, Vercel |

---

## 密钥管理 / Secret Management

推荐使用 `openvault` 管理所有密钥：

Recommended: use `openvault` for all secrets:

```bash
openvault set clawfirm/anthropic-api-key
openvault set clawfirm/hl-private-key
openvault set clawfirm/polygon-private-key
# whip 文件自动读取 / whip files read automatically
```

---

## 社区 / Community

- **Discord:** https://discord.gg/JNXz2utFW8
- **微信 / WeChat:** PpCiting
- **GitHub:** https://github.com/npc-live/clawfirm

---

## 免责声明 / Disclaimer

本项目及其所有代码、工作流、策略仅供学习和技术演示之用，**不构成任何投资建议或财务咨询**。交易类模块可能导致本金亏损，使用者自行承担一切风险。项目按"原样"提供，不做任何明示或暗示的保证。

This project and all its code, workflows, and strategies are provided solely for educational and technical demonstration purposes. **Nothing constitutes investment advice or financial consulting.** Trading modules may result in loss of capital. Users bear all risks. Provided "AS IS" without warranty of any kind.

---

## License

MIT
