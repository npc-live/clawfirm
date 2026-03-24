# ClawFirm

一人公司的终极武器 / The Ultimate Weapon for One-Person Companies

---

## 🦞 简介

**ClawFirm.dev：你的 AI 全能合伙人**

ClawFirm.dev 是专为新时代个体创业者打造的"一人公司"自动化引擎。我们不只是提供工具，而是通过 AI 深度嵌入商业全链路，提供三种核心盈利增长模式：

- **全栈软件出海：** 从深度用户调研、功能构建到自动化营销，AI 帮你像一支完整团队一样打造并分发软件产品，通过真实付费反馈快速迭代。
- **自动化套利交易：** 利用 AI 敏锐捕捉市场信号，通过算法实现高效率的交易获利，让资产在自动化流程中稳健增值。
- **自媒体矩阵分发：** 针对不同平台特征，AI 自动生成高转化内容并精准投喂流量，让你的个人品牌和产品实现全网裂变。

ClawFirm.dev 致力于消除"一人公司"的技术壁垒，让一个人也能拥有一座工厂的战斗力，完成从创意到现金流的完整商业闭环。

## 🦞 About

**ClawFirm.dev: Your All-in-One AI Partner**

ClawFirm.dev is an automation engine built for the new generation of solo entrepreneurs — powering the "one-person company." We don't just provide tools; we deeply embed AI across the entire business chain, offering three core growth models:

- **Full-Stack Software for Global Markets:** From deep user research and feature development to automated marketing, AI helps you build and distribute software products like a full team — iterating rapidly with real paid user feedback.
- **Automated Arbitrage Trading:** Leveraging AI to capture market signals with precision, executing efficient algorithmic trades to steadily grow your assets through automated workflows.
- **Personal Branding Building on Social Media:** Tailored to each platform's characteristics, AI auto-generates high-conversion content and precisely distributes traffic, enabling your personal brand and products to go viral across the web.

ClawFirm.dev is committed to eliminating the technical barriers of the "one-person company," giving a single individual the firepower of an entire factory — completing the full business loop from idea to cash flow.

## 🌍 社区 / Community

- **Discord:** https://discord.gg/JNXz2utFW8
- **WeChat 微信:** PpCiting

---

## 安装 / Installation

```bash
npm install -g clawfirm
```

## CLI 命令 / CLI Commands

```
clawfirm login                    登录 clawfirm.dev / Login to clawfirm.dev
clawfirm whoami                   显示当前会话 / Show current session
clawfirm logout                   退出登录 / Logout

clawfirm install [tool]           安装全部工具（或指定工具）/ Install all tools (or specific)
clawfirm uninstall [tool]         卸载工具 / Uninstall tool
clawfirm list                     列出已注册工具 / List registered tools

clawfirm new "<description>"      从自然语言描述生成新项目 / Generate new project from description
clawfirm <name> [args]            调用插件 / Invoke plugin
```

## 工具集 / Toolset

| 工具 / Tool | 说明 / Description |
|------|------|
| `openvault` | 加密本地密钥管理器 / Encrypted local secret manager |
| `skillctl` | 同步 AI skills 到编程工具 / Sync AI skills to dev tools |
| `whipflow` | 确定性 AI 工作流执行引擎 / Deterministic AI workflow engine |
| `agent-browser` | AI agent 浏览器自动化 / AI agent browser automation |

```bash
clawfirm install   # 安装全部工具并同步 skills 到 Claude Code / Install all tools & sync skills
```

---

## 快速开始 / Quick Start

```bash
clawfirm login
clawfirm install
```

---

## Whip 工作流 / Whip Workflows

每个业务模块在 `whips/` 下有独立子目录，包含 5 个标准文件：

Each business module has its own subdirectory under `whips/` with 5 standard files:

| 文件 / File | 职责 / Role | 运行时机 / When to Run |
|------|------|----------|
| `setup.whip` | 环境检查、API 验证、写入配置 / Env check, API validation, write config | 首次使用 / First-time setup |
| `scan.whip` | 拉取数据、识别信号/机会 / Fetch data, identify signals | 手动或由 monitor 触发 / Manual or triggered by monitor |
| `trade.whip` | 风控检查、执行核心动作 / Risk check, execute core action | 手动或由 monitor 触发 / Manual or triggered by monitor |
| `monitor.whip` | 持续轮询、状态管理、调度循环 / Continuous polling, state management | 长期后台运行 / Long-running background |
| `report.whip` | 统计指标、输出分析报告 / Stats, output analysis report | 随时查看 / Anytime |

### 标准运行顺序 / Standard Run Order

```bash
# 1. 初始化（只需一次）/ Initialize (once only)
whipflow run whips/<module>/setup.whip

# 2. 启动监控（长期运行）/ Start monitor (long-running)
whipflow run whips/<module>/monitor.whip

# 3. 手动扫描 / 执行 / 看报告 / Manual scan / execute / view reports
whipflow run whips/<module>/scan.whip
whipflow run whips/<module>/trade.whip
whipflow run whips/<module>/report.whip
```

---

## 业务模块 / Business Modules

### polymarket — 天气预测市场交易 / Weather Prediction Market Trading

在 Polymarket 上交易天气温度合约。使用 Open-Meteo 集成预报模型计算胜率，与市场隐含概率比较，发现边缘时下单。

Trade weather temperature contracts on Polymarket. Uses Open-Meteo integrated forecast models to calculate win probability, compares with market-implied odds, and places orders when an edge is found.

```bash
# 初始化（填入 Polygon 钱包私钥）/ Initialize (enter Polygon wallet private key)
cat > data/current-run.json << 'EOF'
{
  "run_id": "poly-001",
  "wallet_address": "0xYourAddress",
  "budget_usd": 10,
  "min_edge": 0.05
}
EOF
whipflow run whips/polymarket/setup.whip

# 启动监控 / Start monitor
whipflow run whips/polymarket/monitor.whip
```

**策略参数（回测验证）/ Strategy Params (backtested):** 胜率/Win rate 57.26%, Sharpe 3.60, 最大回撤/Max drawdown 10.87%
**主要市场 / Primary Markets:** 上海/Shanghai ($183K liquidity) → 首尔/Seoul → 多伦多/Toronto → 纽约/New York

---

### hyperliquid — 新闻驱动期货交易 / News-Driven Futures Trading

监控加密货币新闻，用 Claude 评估信号强度，在 Hyperliquid 自动开多/空。

Monitors crypto news, uses Claude to assess signal strength, and automatically opens long/short positions on Hyperliquid.

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "hl-001",
  "wallet_address": "0xYourAddress",
  "budget_usd": 200,
  "max_position_usd": 50,
  "max_open_trades": 4,
  "min_signal_score": 60,
  "default_leverage": 3
}
EOF
whipflow run whips/hyperliquid/setup.whip
whipflow run whips/hyperliquid/monitor.whip
```

**策略参数 / Strategy Params:** 最多 4 个持仓/Max 4 positions, 3-5x 杠杆/leverage, 2h 最长持有/max hold, 5% 止损/stop-loss, 8% 止盈/take-profit

也可使用独立脚本直接运行 / Can also run standalone script:

```bash
HL_PRIVATE_KEY=0x... node scripts/hl-news-trader.js monitor
```

---

### social-media — 社交媒体内容自动化 / Social Media Content Automation

AI 生成内容，自动发布到小红书、微博、Bilibili、Twitter 等平台。

AI-generated content, auto-published to Xiaohongshu, Weibo, Bilibili, Twitter and more.

```bash
whipflow run whips/social-media/setup.whip

# 每日内容生成 + 发布 / Daily content generation + publishing
whipflow run whips/social-media/daily-content.whip
whipflow run whips/social-media/daily-publish.whip

# 周报 & 互动 / Weekly report & engagement
whipflow run whips/social-media/weekly-report.whip
whipflow run whips/social-media/comments.whip
```

---

### arbitrage — 电商跨平台套利 / Cross-Platform E-Commerce Arbitrage

扫描闲鱼↔拼多多（国内）或 eBay↔Amazon（海外）价差，自动采购和上架。目标利润率 > 20%。

Scans price gaps between Xianyu↔Pinduoduo (domestic) or eBay↔Amazon (international), auto-purchases and lists. Target margin > 20%.

```bash
whipflow run whips/arbitrage/setup.whip
whipflow run whips/arbitrage/scan.whip    # 扫描价差 / Scan price gaps
whipflow run whips/arbitrage/buy.whip     # 采购 / Purchase
whipflow run whips/arbitrage/list.whip    # 上架 / List for sale
whipflow run whips/arbitrage/report.whip  # ROI 报告 / ROI report
```

---

### domains — 域名捡漏 / Domain Sniping

扫描即将过期的高价值域名，自动抢注，挂到 Sedo/Afternic 出售。

Scans expiring high-value domains, auto-registers them, and lists on Sedo/Afternic for resale.

```bash
whipflow run whips/domains/setup.whip
whipflow run whips/domains/scan.whip      # 扫描过期域名 / Scan expiring domains
whipflow run whips/domains/snipe.whip     # 自动注册 / Auto-register
whipflow run whips/domains/list.whip      # 挂牌出售 / List for sale
whipflow run whips/domains/report.whip    # 组合价值分析 / Portfolio analysis
```

---

### amazon-affiliate — 亚马逊联盟内容 / Amazon Affiliate Content

关键词研究 → AI 写 SEO 文章 → 自动发布 → 排名监控。

Keyword research → AI-written SEO articles → auto-publish → rank monitoring.

```bash
whipflow run whips/amazon-affiliate/setup.whip
whipflow run whips/amazon-affiliate/research.whip  # 选品 + 关键词 / Product selection + keywords
whipflow run whips/amazon-affiliate/write.whip     # 生成文章 / Generate articles
whipflow run whips/amazon-affiliate/publish.whip   # 发布 / Publish
whipflow run whips/amazon-affiliate/seo-monitor.whip  # 排名追踪 / Rank tracking
```

---

## 创建新的 Whip 模块 / Create New Whip Modules

`creator` 是一个 meta-whip，给定业务描述自动生成完整的 whip 子目录。

`creator` is a meta-whip that auto-generates a complete whip subdirectory from a business description.

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "create-001",
  "name": "my-strategy",
  "description": "监控 Reddit 帖子，识别热门话题，自动发布相关内容到 Twitter",
  "apis": ["Reddit API", "Twitter API v2"]
}
EOF

whipflow run whips/creator/create.whip
# 输出 / Output: whips/my-strategy/ 含 setup/scan/trade/monitor/report.whip
```

详见 / See [`whips/creator/README.md`](whips/creator/README.md)。

---

## 密钥管理 / Secret Management

推荐使用 `openvault` 存储所有密钥，避免明文出现在环境变量或文件中：

Recommended: use `openvault` to store all secrets, avoiding plaintext in env vars or files:

```bash
# 存储密钥 / Store secrets
openvault set clawfirm/polygon-private-key
openvault set clawfirm/hl-private-key
openvault set clawfirm/anthropic-api-key

# whip 文件内部自动读取，无需手动 export
# whip files read secrets automatically, no manual export needed
```

回退方案（不推荐）/ Fallback (not recommended):

```bash
export PRIVATE_KEY=0x...
export ANTHROPIC_API_KEY=sk-ant-...
```

---

## 数据文件 / Data Files

所有运行时数据统一存放在 `data/`，不提交到 git：

All runtime data is stored in `data/`, not committed to git:

```
data/
├── current-run.json      # 当前工作流输入参数 / Current workflow input params
├── stage-log.json        # 各阶段执行日志 / Stage execution logs
├── config.json           # 策略配置（无明文密钥）/ Strategy config (no plaintext secrets)
├── trades.json           # 交易历史 / Trade history
├── positions.json        # 当前持仓 / Current positions
└── reports.json          # 绩效报告 / Performance reports
```

---

## 项目结构 / Project Structure

```
clawfirm/
├── bin/cli.js            # CLI 入口 / CLI entry point
├── lib/                  # auth, login, dispatch, install, skills
├── scripts/              # 独立交易脚本 / Standalone trading scripts
├── skills/               # 可复用 AI skills / Reusable AI skills
├── whips/                # 工作流模块 / Workflow modules
│   ├── creator/          # Meta-whip：生成新 whip 目录 / Generate new whip dirs
│   ├── polymarket/       # 天气预测市场交易 / Weather prediction market
│   ├── hyperliquid/      # 新闻期货交易 / News-driven futures
│   ├── social-media/     # 社交媒体自动化 / Social media automation
│   ├── arbitrage/        # 电商套利 / E-commerce arbitrage
│   ├── domains/          # 域名捡漏 / Domain sniping
│   └── amazon-affiliate/ # 联盟营销 / Affiliate marketing
├── data/                 # 运行时数据（git ignored）/ Runtime data (git ignored)
└── clawfirm.config.js    # 工具注册表 / Tool registry
```

---

## ⚠️ 免责声明 / Disclaimer

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

## 📄 License

MIT License
