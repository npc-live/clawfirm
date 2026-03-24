# whips/creator

给定一段业务描述，自动生成完整的 whip 子目录。

## 工作原理

`create.whip` 读取 `data/current-run.json` 中的业务描述，经过需求分析和代码生成两个阶段，在 `whips/<name>/` 下输出可直接运行的 `.whip` 文件集合。

```
data/current-run.json   →   analyst 分析设计   →   writer 生成文件
                                                        ↓
                                              whips/<name>/
                                              ├── setup.whip
                                              ├── scan.whip
                                              ├── trade.whip   (或 publish / deploy)
                                              ├── monitor.whip
                                              └── report.whip
```

---

## 快速开始

### 第一步：写业务描述

```bash
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
```

### 第二步：运行

```bash
whipflow run whips/creator/create.whip
```

### 第三步：使用生成的 whips

```bash
# 初始化环境和 API 凭证
whipflow run whips/twitter-alpha/setup.whip

# 扫描信号
whipflow run whips/twitter-alpha/scan.whip

# 执行交易
whipflow run whips/twitter-alpha/trade.whip

# 启动持续监控（长期运行）
whipflow run whips/twitter-alpha/monitor.whip

# 查看报告
whipflow run whips/twitter-alpha/report.whip
```

---

## current-run.json 字段说明

| 字段 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `run_id` | 是 | 本次创建的唯一 ID | `"create-001"` |
| `name` | 是 | 子目录名，小写字母 + 连字符 | `"my-strategy"` |
| `description` | 是 | 自然语言描述业务逻辑 | `"监控 Reddit 帖子..."` |
| `files` | 否 | 要生成哪些文件，默认全部 | `["setup","scan","trade","monitor","report"]` |
| `apis` | 否 | 涉及的 API，帮助生成更准确的代码 | `["Binance API", "Twitter API v2"]` |
| `data_prefix` | 否 | 数据文件前缀，默认与 name 相同 | `"twitter-alpha"` |

---

## 更多示例

### 量化交易策略

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "create-002",
  "name": "funding-arb",
  "description": "监控 Binance 和 Hyperliquid 的资金费率差异，当差值超过 0.1% 时执行跨所套利",
  "apis": ["Binance USDT-M Futures", "Hyperliquid Info API"],
  "data_prefix": "funding-arb"
}
EOF
whipflow run whips/creator/create.whip
```

### 内容发布自动化

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "create-003",
  "name": "newsletter",
  "description": "每天抓取加密货币新闻，用 Claude 生成 200 字摘要，发布到 Substack 和 Twitter",
  "files": ["setup", "scan", "publish", "monitor", "report"],
  "apis": ["CryptoPanic API", "Substack API", "Twitter API v2"],
  "data_prefix": "newsletter"
}
EOF
whipflow run whips/creator/create.whip
```

### 域名捡漏

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "create-004",
  "name": "domain-sniper",
  "description": "扫描即将过期的 .io 和 .ai 域名，评分超过 80 分自动抢注",
  "apis": ["GoDaddy API", "Namecheap API", "DomainIQ API"],
  "data_prefix": "domain-sniper"
}
EOF
whipflow run whips/creator/create.whip
```

---

## 生成文件说明

| 文件 | 职责 | 运行方式 |
|------|------|----------|
| `setup.whip` | 环境检查、API 验证、写入配置文件 | 首次使用时运行一次 |
| `scan.whip` | 拉取数据源、识别信号、写入 signals.json | 手动或由 monitor 触发 |
| `trade.whip` | 风控检查、执行核心动作、记录结果 | 手动或由 monitor 触发 |
| `monitor.whip` | 持续轮询、管理活跃任务、调度 scan+trade | 长期后台运行 |
| `report.whip` | 统计指标、输出分析报告 | 随时查看绩效 |

生成的数据文件统一存放在 `data/` 目录：

```
data/
├── <name>-config.json    # 策略配置（无明文密钥）
├── <name>-signals.json   # 当前信号列表
├── <name>-trades.json    # 执行记录
├── <name>-reports.json   # 历史报告
└── stage-log.json        # 各阶段运行日志
```

---

## 密钥管理

生成的 whip 文件优先使用 `openvault` 存储密钥，回退到环境变量：

```bash
# 推荐：用 openvault 存储
openvault set clawfirm/<name>-api-key

# 回退：环境变量
export MY_API_KEY=xxx
whipflow run whips/<name>/setup.whip
```
