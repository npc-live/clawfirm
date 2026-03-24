# ClawFirm — 7*24 AI引擎

> 7×24 小时运行的量化交易机器人，在 Polymarket 天气预测市场中寻找并执行统计套利机会。

---

## 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        ClawFirm 引擎                             │
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐   │
│  │ 天气数据获取  │    │  市场扫描器  │    │    调度器        │   │
│  │ Open-Meteo   │    │  Polymarket  │    │  (每15分钟)      │   │
│  │ 集合预报API  │    │  Gamma API   │    │                  │   │
│  └──────┬───────┘    └──────┬───────┘    └────────┬─────────┘   │
│         │                  │                      │             │
│         ▼                  ▼                      │             │
│  ┌──────────────────────────────────┐             │             │
│  │         概率计算引擎             │◄────────────┘             │
│  │  P(bin) = count(members>T)/N     │                           │
│  └──────────────┬───────────────────┘                           │
│                 │                                                │
│                 ▼                                                │
│  ┌──────────────────────────────────┐                           │
│  │         Edge 计算器              │                           │
│  │  edge = P_model - P_market       │                           │
│  │  最小门槛: 5%（降水 15%）        │                           │
│  └──────────────┬───────────────────┘                           │
│                 │                                                │
│                 ▼                                                │
│  ┌──────────────────────────────────┐                           │
│  │         仓位计算器               │                           │
│  │  半Kelly = 0.5 × (edge/odds)     │                           │
│  └──────────────┬───────────────────┘                           │
│                 │                                                │
│                 ▼                                                │
│  ┌────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  风险管理  │  │  熔断机制   │  │      订单管理器          │  │
│  │ 5%日亏损  │  │  连亏5笔暂  │  │  EIP-712 签名            │  │
│  │ 限 3 持仓 │  │  停1小时    │  │  L2 HMAC 认证            │  │
│  └────────────┘  └─────────────┘  └─────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  组合追踪器 — 持仓管理 / PnL 计算 / 日报                 │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
         │                              │
         ▼                              ▼
   Open-Meteo API              Polymarket CLOB API
   (集合天气预报)               (链上结算，Polygon)
```

---

## 工作原理

ClawFirm 利用**数值天气预报模型**与**预测市场定价**之间的信息差套利：

1. **获取集合预报** — 每小时拉取 Open-Meteo ECMWF 51 成员集合预报，计算每个温度区间的真实概率
2. **扫描市场定价** — 扫描 Polymarket 上的天气合约，获取市场隐含概率
3. **计算信息优势** — `edge = P_模型 - P_市场`，仅在 edge > 5% 时下单
4. **半 Kelly 仓位** — 用凯利公式的一半控制仓位，限制最大单仓 10%
5. **多层风控** — 日亏损上限、持仓上限、熔断机制全面保护资金

**回测结果（2024-2026 上海）：胜率 55.96%，夏普 2.73，最大回撤 11.9%，两年总收益 +951%**

---

## 快速开始

### 前置条件
- Node.js >= 18
- Polygon 钱包（含 USDC.e 余额）
- Polymarket API 密钥

### 5 步启动

**第 1 步：克隆并安装依赖**
```bash
git clone https://github.com/your-org/clawfirm.git
cd clawfirm
npm install
```

**第 2 步：配置环境变量**
```bash
cp .env.example .env
# 编辑 .env，填入你的密钥（详见 docs/SETUP.md）
```

**第 3 步：先用 Paper Trading 模式验证**
```bash
DRY_RUN=true npm run trade:dry
```
看到 `[scheduler] all tasks scheduled` 和天气数据输出，说明配置正确。

**第 4 步：确认钱包有足够余额**
```bash
# 在 Polygon 上准备至少 15 USDC.e（10 用于交易 + 5 备用）
```

**第 5 步：启动实盘**
```bash
PRIVATE_KEY=0x你的私钥 \
POLY_API_KEY=你的API密钥 \
POLY_SECRET=你的Secret \
POLY_PASSPHRASE=你的Passphrase \
WEATHER_REGIONS=shanghai \
npm run trade
```

详细配置请参阅 → [docs/SETUP.md](./docs/SETUP.md)

---

## 文档索引

| 文档 | 说明 |
|------|------|
| [docs/SETUP.md](./docs/SETUP.md) | 完整安装与配置指南 |
| [docs/STRATEGY.md](./docs/STRATEGY.md) | 策略逻辑与回测结果 |
| [docs/RISK_DISCLAIMER.md](./docs/RISK_DISCLAIMER.md) | 风险声明（必读） |
| [docs/TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md) | 常见问题排查 |
| [docs/backtest-report.md](./docs/backtest-report.md) | 完整回测报告 |

---

## 截图

<!-- TODO: 添加以下截图 -->
<!-- ![引擎运行日志](./docs/screenshots/engine-running.png) -->
<!-- ![交易记录](./docs/screenshots/trades.png) -->
<!-- ![收益曲线](./docs/screenshots/pnl-curve.png) -->

---

## 免责声明

本系统仅供研究与学习用途。预测市场交易存在亏损风险，过往回测不代表未来表现。使用前请阅读 [风险声明](./docs/RISK_DISCLAIMER.md)。

---

## 自媒体运营平台

### 快速启动
```bash
# 首次：安装依赖
npm install
cd client && npm install

# 启动 Tauri 桌面应用（开发模式）
cd client && npm run tauri dev

# 或仅运行自动化（无 UI）
tsx src/social/scheduler.ts
```

### 单独运行某个模块
```bash
tsx src/social/content-creator.ts --run-id 1
tsx src/social/analytics-collector.ts --run-id 1
```

### 工作流文件说明
| 文件 | 用途 |
|---|---|
| whips/setup.whip | 首次配置（账号、策略） |
| whips/daily-content.whip | 每日内容创作 |
| whips/daily-publish.whip | 多平台发布 |
| whips/analytics.whip | 数据采集分析 |
| whips/comments.whip | 评论互动处理 |
| whips/weekly-report.whip | 周报 + 策略迭代 |
| whips/repurpose.whip | 微信文章改写 |
