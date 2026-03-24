# clawfirm

AI 驱动的自动化工作流平台。组合交易机器人、内容发布、电商套利等业务模块，通过 whip 工作流引擎统一编排。

---

## 安装

1. 前往 [clawfirm.dev](https://clawfirm.dev) 注册账号
2. 安装 CLI 并登录：

```bash
npm install -g clawfirm
clawfirm login
clawfirm install    # 安装全部工具并同步 skills 到 Claude Code
```

---

## CLI 命令

```
clawfirm login                    登录 clawfirm.dev
clawfirm whoami                   显示当前会话
clawfirm logout                   退出登录

clawfirm install [tool]           安装全部工具（或指定单个）
clawfirm uninstall [tool]         卸载工具
clawfirm list                     列出已注册工具

clawfirm new "<description>"      从自然语言描述生成新项目
clawfirm <name> [args]            调用插件
```

## 工具集

| 工具 | 说明 |
|------|------|
| `openvault` | 加密本地密钥管理器 |
| `skillctl` | 同步 AI skills 到编程工具 |
| `whipflow` | 确定性 AI 工作流执行引擎 |
| `agent-browser` | AI agent 浏览器自动化 |

---

## Whip 工作流

每个业务模块在 `whips/` 下有独立子目录，包含标准文件：

| 文件 | 职责 | 运行时机 |
|------|------|----------|
| `setup.whip` | 环境检查、API 验证、写入配置 | 首次使用时运行一次 |
| `scan.whip` | 拉取数据、识别信号/机会 | 手动或由 monitor 触发 |
| `trade.whip` | 风控检查、执行核心动作 | 手动或由 monitor 触发 |
| `monitor.whip` | 持续轮询、状态管理、调度循环 | 长期后台运行 |
| `report.whip` | 统计指标、输出分析报告 | 随时查看 |

**标准运行顺序：**

```bash
whipflow run whips/<module>/setup.whip     # 第一步：初始化（只需一次）
whipflow run whips/<module>/monitor.whip   # 第二步：启动监控（长期运行）

# 按需手动触发：
whipflow run whips/<module>/scan.whip
whipflow run whips/<module>/trade.whip
whipflow run whips/<module>/report.whip
```

所有工作流通过 `data/current-run.json` 接收输入参数，运行日志写入 `data/stage-log.json`。

---

## 业务模块

### polymarket — 天气预测市场交易

在 Polymarket 上交易天气温度合约。使用 Open-Meteo 集合预报（ECMWF/ICON，40-51 个成员）计算每个温度区间的模型概率，与市场隐含概率比较，edge 达标时用 Half-Kelly 公式下单。

**前置条件：** Polygon 钱包私钥、`ANTHROPIC_API_KEY`

```bash
# 1. 写入运行配置
cat > data/current-run.json << 'EOF'
{
  "run_id": "poly-001",
  "wallet_address": "0xYourPolygonAddress",
  "budget_usd": 10,
  "min_edge": 0.05,
  "max_positions": 3
}
EOF

# 2. 存储私钥（推荐 openvault）
openvault set clawfirm/polygon-private-key

# 3. 初始化
whipflow run whips/polymarket/setup.whip

# 4. 手动扫描一次，查看有没有机会
whipflow run whips/polymarket/scan.whip
# → 输出 data/opportunities.json，包含每个机会的 p_model / p_market / edge

# 5. 手动下单
PRIVATE_KEY=$(openvault get clawfirm/polygon-private-key) \
  whipflow run whips/polymarket/trade.whip

# 6. 启动自动监控（每小时触发 scan + trade）
whipflow run whips/polymarket/monitor.whip

# 7. 查看报告
whipflow run whips/polymarket/report.whip
```

**核心策略参数（回测验证，2024-03-21 ~ 2026-03-20）：**

| 参数 | 值 |
|------|----|
| 最小 edge | 5%（降水合约 15%，夏季 6-8 月 12%） |
| 跳过过度自信区间 | edge 30%–40% |
| 仓位公式 | Half-Kelly × 0.5，最大 10% 初始资金 |
| 最大同时持仓 | 3 |
| 日亏损熔断 | -5% 停止当日交易 |
| 回测胜率 | **57.26%**，Sharpe **3.60**，最大回撤 **10.87%** |

**优先市场：** 上海（$183K）→ 首尔 → 多伦多 → 纽约 → 伦敦 → 东京 → 新加坡 → 深圳

---

### hyperliquid — 新闻驱动期货交易

监控多个加密货币新闻源（CryptoPanic、CoinTelegraph RSS、The Block、Decrypt），用 Claude 对新闻打分（0-100），得分达标时在 Hyperliquid 自动开多/空。

**没有账号？** 通过返佣链接注册，双方均可获得手续费返还：
👉 **[https://app.hyperliquid.xyz/join/ARCANE](https://app.hyperliquid.xyz/join/ARCANE)**

**前置条件：** Hyperliquid 账号、钱包私钥、`ANTHROPIC_API_KEY`

```bash
# 1. 写入运行配置
cat > data/current-run.json << 'EOF'
{
  "run_id": "hl-001",
  "wallet_address": "0xYourHLAddress",
  "budget_usd": 200,
  "max_position_usd": 50,
  "max_open_trades": 4,
  "min_signal_score": 60,
  "stop_loss_pct": 0.04,
  "take_profit_pct": 0.08,
  "default_leverage": 3,
  "max_leverage": 5,
  "entry_window_minutes": 5,
  "poll_interval_seconds": 60
}
EOF

# 2. 初始化
openvault set clawfirm/hl-private-key
whipflow run whips/hyperliquid/setup.whip

# 3. 手动扫描一次看信号
whipflow run whips/hyperliquid/scan.whip
# → data/hl-signals.json，每个信号包含 coin / direction / score / reason

# 4. 手动执行一次下单
whipflow run whips/hyperliquid/trade.whip

# 5. 启动自动监控（60s 一轮：检查止损止盈 + 触发新一轮 scan+trade）
whipflow run whips/hyperliquid/monitor.whip

# 6. 查看绩效报告
whipflow run whips/hyperliquid/report.whip
```

**信号评分标准（Claude 评分）：**

| 分数 | 触发条件示例 |
|------|-------------|
| 90-100 | ETF 获批/拒绝、大型交易所被黑超 $50M、CEO 被捕 |
| 75-89 | Coinbase/Binance 上线、跨链桥被攻击、SEC 起诉 |
| 60-74 | 中小交易所上线、机构购买、美联储表态 |
| < 60 | 过滤掉，不交易 |

**策略参数：** 最多 4 个持仓，3-5x 杠杆，最长持仓 2 小时，止损 4%，止盈 8%

也可绕过 whip 直接用脚本：

```bash
HL_PRIVATE_KEY=0x... node scripts/hl-news-trader.js monitor
HL_PRIVATE_KEY=0x... node scripts/hl-news-trader.js scan
HL_PRIVATE_KEY=0x... node scripts/hl-news-trader.js positions   # 查看当前持仓
```

> **如果想用 Binance 代替 Hyperliquid：**
> - 国内用户（免翻墙）：[https://accounts.maxweb.red/register?ref=AIARCANE](https://accounts.maxweb.red/register?ref=AIARCANE)
> - 国际用户：[https://accounts.binance.com/en/register?ref=AIARCANE](https://accounts.binance.com/en/register?ref=AIARCANE)
>
> 已有 Binance 账号但非邀请码注册？可手动填入经纪商代码享受返佣：现货 `URWGV8AN`，合约 `t4eky4We`

---

### social-media — 社交媒体内容自动化

AI 生成内容，自动发布到小红书、微博、Bilibili、Twitter、Telegram。包含竞品分析、内容策略生成、每日发布、评论互动、周报等完整运营流程。

**前置条件：** 各平台 token、`social-cli` 已安装

```bash
# 1. 写入账号配置
cat > data/current-run.json << 'EOF'
{
  "run_id": "social-001",
  "niche": "AI 工具评测",
  "target_audience": "25-35 岁开发者和产品经理",
  "tone_style": "专业但不枯燥，有观点，举例子",
  "content_freq": "每天1篇",
  "growth_goal": "3个月涨粉10000",
  "xhs_account": "your_xhs_id",
  "twitter_account": "your_twitter_handle"
}
EOF

# 2. 初始化（自动做竞品分析 + 生成内容策略文档）
whipflow run whips/social-media/setup.whip
# → docs/competitive-analysis.md（竞品规律、蓝海方向）
# → docs/content-strategy.md（完整运营策略，自动校验循环，最多修 3 轮）

# 3. 每日运营
whipflow run whips/social-media/daily-content.whip   # 生成今日内容
whipflow run whips/social-media/daily-publish.whip   # 发布到各平台

# 4. 内容复用（把爆款改写成其他平台版本）
whipflow run whips/social-media/repurpose.whip

# 5. 互动管理
whipflow run whips/social-media/comments.whip        # 回复评论 + 社群互动

# 6. 数据复盘
whipflow run whips/social-media/analytics.whip       # 实时数据分析
whipflow run whips/social-media/weekly-report.whip   # 周报 + 下周计划
```

---

### arbitrage — 电商跨平台套利

扫描闲鱼↔拼多多（国内）或 eBay↔Amazon（海外）价差，自动采购和上架，目标毛利率 > 20%。

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "arb-001",
  "market": "cn",
  "category": "数码配件",
  "min_profit_pct": 0.20,
  "max_buy_price": 200,
  "daily_budget": 1000
}
EOF

whipflow run whips/arbitrage/setup.whip   # 分析品类利润模型
whipflow run whips/arbitrage/scan.whip    # 扫描当前价差机会
whipflow run whips/arbitrage/buy.whip     # 执行采购
whipflow run whips/arbitrage/list.whip    # 上架到目标平台
whipflow run whips/arbitrage/report.whip  # ROI 报告
```

---

### domains — 域名捡漏

扫描即将过期的高价值 .io / .ai 域名，自动评分，高分自动抢注，挂到 Sedo/Afternic 出售。

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "domain-001",
  "extensions": [".io", ".ai", ".com"],
  "min_score": 80,
  "max_reg_price": 50,
  "registrar": "namecheap"
}
EOF

whipflow run whips/domains/setup.whip    # 注册商账号配置
whipflow run whips/domains/scan.whip     # 扫描过期域名列表
whipflow run whips/domains/snipe.whip    # 自动抢注高分域名
whipflow run whips/domains/list.whip     # 挂牌出售
whipflow run whips/domains/report.whip   # 组合估值报告
```

---

### amazon-affiliate — 亚马逊联盟内容

关键词研究 → AI 写 SEO 文章 → 自动发布 → 排名监控。

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "aff-001",
  "niche": "home office equipment",
  "target_country": "US",
  "min_search_volume": 1000,
  "max_keyword_difficulty": 40
}
EOF

whipflow run whips/amazon-affiliate/setup.whip         # 选品 + 关键词研究
whipflow run whips/amazon-affiliate/research.whip      # 深度产品调研
whipflow run whips/amazon-affiliate/write.whip         # 生成 SEO 文章
whipflow run whips/amazon-affiliate/publish.whip       # 发布到博客/站点
whipflow run whips/amazon-affiliate/seo-monitor.whip   # 追踪关键词排名
```

---

## 创建新的 Whip 模块

`creator` 是一个 meta-whip，给定业务描述自动生成完整的 whip 子目录。

```bash
# 1. 描述你的业务
cat > data/current-run.json << 'EOF'
{
  "run_id": "create-001",
  "name": "twitter-alpha",
  "description": "监控 Twitter KOL 推文，识别 alpha 信号，自动在 Binance Spot 下单",
  "files": ["setup", "scan", "trade", "monitor", "report"],
  "apis": ["Twitter API v2", "Binance Spot REST API"],
  "data_prefix": "twitter-alpha"
}
EOF

# 2. 生成
whipflow run whips/creator/create.whip
# → whips/twitter-alpha/setup.whip
# → whips/twitter-alpha/scan.whip
# → whips/twitter-alpha/trade.whip
# → whips/twitter-alpha/monitor.whip
# → whips/twitter-alpha/report.whip

# 3. 用生成的模块
whipflow run whips/twitter-alpha/setup.whip
whipflow run whips/twitter-alpha/monitor.whip
```

详见 [`whips/creator/README.md`](whips/creator/README.md)。

---

## 密钥管理

推荐用 `openvault` 管理所有密钥，whip 文件自动检测并优先读取，不需要手动 export：

```bash
# 存储
openvault set clawfirm/polygon-private-key
openvault set clawfirm/hl-private-key
openvault set clawfirm/anthropic-api-key
openvault set clawfirm/xhs-token
openvault set clawfirm/twitter-token

# 读取（手动验证）
openvault get clawfirm/anthropic-api-key
```

回退方案（不推荐，密钥明文暴露在进程环境中）：

```bash
export PRIVATE_KEY=0x...
export ANTHROPIC_API_KEY=sk-ant-...
export HL_PRIVATE_KEY=0x...
```

---

## 数据文件

所有运行时数据存放在 `data/`，已加入 `.gitignore`：

```
data/
├── current-run.json         # 当前工作流输入参数
├── stage-log.json           # 各阶段执行日志（追加写入）
├── config.json              # 策略配置（无明文密钥）
├── api-creds.json           # 凭证引用（指向 openvault key 名）
├── opportunities.json       # polymarket 当前机会列表
├── trades.json              # 交易历史
├── positions.json           # 当前持仓
├── reports.json             # 绩效报告
├── hl-config.json           # hyperliquid 策略配置
├── hl-signals.json          # hyperliquid 当前信号
└── hl-trades.json           # hyperliquid 交易记录
```

`stage-log.json` 记录每个 whip 每个阶段的执行结果，格式：

```json
[
  { "run_id": "poly-001", "stage": "env-check", "index": 0, "status": "done", "ts": "2026-03-24T10:00:00Z" },
  { "run_id": "poly-001", "stage": "api-check",  "index": 1, "status": "done", "ts": "2026-03-24T10:00:05Z" }
]
```

每个 whip 最终输出 `WorkflowEvent` JSON-line，供 Tauri UI 或日志系统解析：

```json
{"ts":"2026-03-24T10:01:00Z","event":"stage_complete","stage":"scan","data":{"opportunities_found":2,"top_edge":0.12}}
```

---

## 项目结构

```
clawfirm/
├── bin/cli.js               # CLI 入口
├── lib/                     # auth, login, dispatch, install, skills
├── scripts/                 # 独立交易脚本
│   ├── weather-trader.js    # Polymarket 天气交易（独立版）
│   ├── weather-trader-v2.js
│   └── hl-news-trader.js    # Hyperliquid 新闻交易（独立版）
├── skills/                  # 可复用 AI skills
│   ├── remotion-video/      # 视频制作
│   ├── social-publish/      # 多平台内容发布
│   └── video-skills/        # 数字人视频生成
├── whips/                   # 工作流模块
│   ├── creator/             # Meta-whip：生成新 whip 目录
│   ├── polymarket/          # 天气预测市场交易
│   ├── hyperliquid/         # 新闻驱动期货交易
│   ├── social-media/        # 社交媒体自动化
│   ├── arbitrage/           # 电商套利
│   ├── domains/             # 域名捡漏
│   └── amazon-affiliate/    # 联盟营销
├── data/                    # 运行时数据（git ignored）
└── clawfirm.config.js       # 工具注册表
```
