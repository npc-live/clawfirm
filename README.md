# clawfirm

AI 驱动的自动化工作流平台。组合交易机器人、内容发布、电商套利等业务模块，通过 whip 工作流引擎统一编排。

## 安装

```bash
npm install -g clawfirm
```

## CLI 命令

```
clawfirm login                    登录 clawfirm.dev
clawfirm whoami                   显示当前会话
clawfirm logout                   退出登录

clawfirm install [tool]           安装全部工具（或指定工具）
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

```bash
clawfirm install   # 安装全部工具并同步 skills 到 Claude Code
```

---

## 快速开始

```bash
clawfirm login
clawfirm install
```

---

## Whip 工作流

每个业务模块在 `whips/` 下有独立子目录，包含 5 个标准文件：

| 文件 | 职责 | 运行时机 |
|------|------|----------|
| `setup.whip` | 环境检查、API 验证、写入配置 | 首次使用时运行一次 |
| `scan.whip` | 拉取数据、识别信号/机会 | 手动或由 monitor 触发 |
| `trade.whip` | 风控检查、执行核心动作 | 手动或由 monitor 触发 |
| `monitor.whip` | 持续轮询、状态管理、调度循环 | 长期后台运行 |
| `report.whip` | 统计指标、输出分析报告 | 随时查看 |

### 标准运行顺序

```bash
# 1. 初始化（只需一次）
whipflow run whips/<module>/setup.whip

# 2. 启动监控（长期运行）
whipflow run whips/<module>/monitor.whip

# 3. 手动扫描 / 执行 / 看报告
whipflow run whips/<module>/scan.whip
whipflow run whips/<module>/trade.whip
whipflow run whips/<module>/report.whip
```

---

## 业务模块

### polymarket — 天气预测市场交易

在 Polymarket 上交易天气温度合约。使用 Open-Meteo 集成预报模型计算胜率，与市场隐含概率比较，发现边缘时下单。

```bash
# 初始化（填入 Polygon 钱包私钥）
cat > data/current-run.json << 'EOF'
{
  "run_id": "poly-001",
  "wallet_address": "0xYourAddress",
  "budget_usd": 10,
  "min_edge": 0.05
}
EOF
whipflow run whips/polymarket/setup.whip

# 启动监控
whipflow run whips/polymarket/monitor.whip
```

**策略参数（回测验证）：** 胜率 57.26%，Sharpe 3.60，最大回撤 10.87%
**主要市场：** 上海（$183K 流动性）→ 首尔 → 多伦多 → 纽约

---

### hyperliquid — 新闻驱动期货交易

监控加密货币新闻，用 Claude 评估信号强度，在 Hyperliquid 自动开多/空。

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

**策略参数：** 最多 4 个持仓，3-5x 杠杆，2 小时最长持有，5% 止损，8% 止盈

也可使用独立脚本直接运行：

```bash
HL_PRIVATE_KEY=0x... node scripts/hl-news-trader.js monitor
```

---

### social-media — 社交媒体内容自动化

AI 生成内容，自动发布到小红书、微博、Bilibili、Twitter 等平台。

```bash
whipflow run whips/social-media/setup.whip

# 每日内容生成 + 发布
whipflow run whips/social-media/daily-content.whip
whipflow run whips/social-media/daily-publish.whip

# 周报 & 互动
whipflow run whips/social-media/weekly-report.whip
whipflow run whips/social-media/comments.whip
```

---

### arbitrage — 电商跨平台套利

扫描闲鱼↔拼多多（国内）或 eBay↔Amazon（海外）价差，自动采购和上架。目标利润率 > 20%。

```bash
whipflow run whips/arbitrage/setup.whip
whipflow run whips/arbitrage/scan.whip    # 扫描价差
whipflow run whips/arbitrage/buy.whip     # 采购
whipflow run whips/arbitrage/list.whip    # 上架
whipflow run whips/arbitrage/report.whip  # ROI 报告
```

---

### domains — 域名捡漏

扫描即将过期的高价值域名，自动抢注，挂到 Sedo/Afternic 出售。

```bash
whipflow run whips/domains/setup.whip
whipflow run whips/domains/scan.whip      # 扫描过期域名
whipflow run whips/domains/snipe.whip     # 自动注册
whipflow run whips/domains/list.whip      # 挂牌出售
whipflow run whips/domains/report.whip    # 组合价值分析
```

---

### amazon-affiliate — 亚马逊联盟内容

关键词研究 → AI 写 SEO 文章 → 自动发布 → 排名监控。

```bash
whipflow run whips/amazon-affiliate/setup.whip
whipflow run whips/amazon-affiliate/research.whip  # 选品 + 关键词
whipflow run whips/amazon-affiliate/write.whip     # 生成文章
whipflow run whips/amazon-affiliate/publish.whip   # 发布
whipflow run whips/amazon-affiliate/seo-monitor.whip  # 排名追踪
```

---

## 创建新的 Whip 模块

`creator` 是一个 meta-whip，给定业务描述自动生成完整的 whip 子目录。

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
# 输出：whips/my-strategy/ 含 setup/scan/trade/monitor/report.whip
```

详见 [`whips/creator/README.md`](whips/creator/README.md)。

---

## 密钥管理

推荐使用 `openvault` 存储所有密钥，避免明文出现在环境变量或文件中：

```bash
# 存储密钥
openvault set clawfirm/polygon-private-key
openvault set clawfirm/hl-private-key
openvault set clawfirm/anthropic-api-key

# whip 文件内部自动读取，无需手动 export
```

回退方案（不推荐）：

```bash
export PRIVATE_KEY=0x...
export ANTHROPIC_API_KEY=sk-ant-...
```

---

## 数据文件

所有运行时数据统一存放在 `data/`，不提交到 git：

```
data/
├── current-run.json      # 当前工作流输入参数
├── stage-log.json        # 各阶段执行日志
├── config.json           # 策略配置（无明文密钥）
├── trades.json           # 交易历史
├── positions.json        # 当前持仓
└── reports.json          # 绩效报告
```

---

## 项目结构

```
clawfirm/
├── bin/cli.js            # CLI 入口
├── lib/                  # auth, login, dispatch, install, skills
├── scripts/              # 独立交易脚本（weather-trader, hl-news-trader）
├── skills/               # 可复用 AI skills（视频制作、内容发布）
├── whips/                # 工作流模块
│   ├── creator/          # Meta-whip：生成新 whip 目录
│   ├── polymarket/       # 天气预测市场交易
│   ├── hyperliquid/      # 新闻期货交易
│   ├── social-media/     # 社交媒体自动化
│   ├── arbitrage/        # 电商套利
│   ├── domains/          # 域名捡漏
│   └── amazon-affiliate/ # 联盟营销
├── data/                 # 运行时数据（git ignored）
└── clawfirm.config.js    # 工具注册表
```
