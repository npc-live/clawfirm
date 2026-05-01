# ClawFirm 业务模块 × 文件路径对照表

GitHub 根路径: `https://github.com/npc-live/clawfirm/blob/main/`
本地根路径: `/Users/olivia/Desktop/ClawFirm_OPC/`

---

## 1. 自媒体发布

> 多平台内容创作 + AI 文案 + CDP 自动发布 (抖音/小红书/B站/Twitter/LinkedIn/币安广场)

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **CDP 脚本** | `app/assets/shortcuts/douyin.yaml` | [shortcuts/douyin.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/douyin.yaml) |
| | `app/assets/shortcuts/xhs.yaml` | [shortcuts/xhs.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/xhs.yaml) |
| | `app/assets/shortcuts/bilibili.yaml` | [shortcuts/bilibili.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/bilibili.yaml) |
| | `app/assets/shortcuts/x.yaml` | [shortcuts/x.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/x.yaml) |
| | `app/assets/shortcuts/linkedin.yaml` | [shortcuts/linkedin.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/linkedin.yaml) |
| | `app/assets/shortcuts/binance-square.yaml` | [shortcuts/binance-square.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/binance-square.yaml) |
| **平台 Skill** | `app/assets/skills/social-publish/WORKFLOW.md` | [skills/social-publish/WORKFLOW.md](https://github.com/npc-live/clawfirm/blob/main/app/assets/skills/social-publish/WORKFLOW.md) |
| | `app/assets/skills/social-publish/copywriting-base/` | [skills/social-publish/copywriting-base/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/social-publish/copywriting-base) |
| | `app/assets/skills/social-publish/douyin/` | [skills/social-publish/douyin/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/social-publish/douyin) |
| | `app/assets/skills/social-publish/xiaohongshu/` | [skills/social-publish/xiaohongshu/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/social-publish/xiaohongshu) |
| | `app/assets/skills/social-publish/bilibili/` | [skills/social-publish/bilibili/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/social-publish/bilibili) |
| | `app/assets/skills/social-publish/twitter/` | [skills/social-publish/twitter/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/social-publish/twitter) |
| | `app/assets/skills/social-publish/binance-square/` | [skills/social-publish/binance-square/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/social-publish/binance-square) |
| **工作流** | `app/assets/workflows/social-media/setup.whip` | [workflows/social-media/setup.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/social-media/setup.whip) |
| | `app/assets/workflows/social-media/daily-content.whip` | [workflows/social-media/daily-content.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/social-media/daily-content.whip) |
| | `app/assets/workflows/social-media/daily-publish.whip` | [workflows/social-media/daily-publish.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/social-media/daily-publish.whip) |
| | `app/assets/workflows/social-media/repurpose.whip` | [workflows/social-media/repurpose.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/social-media/repurpose.whip) |
| | `app/assets/workflows/social-media/comments.whip` | [workflows/social-media/comments.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/social-media/comments.whip) |
| | `app/assets/workflows/social-media/analytics.whip` | [workflows/social-media/analytics.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/social-media/analytics.whip) |
| | `app/assets/workflows/social-media/weekly-report.whip` | [workflows/social-media/weekly-report.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/social-media/weekly-report.whip) |
| **底层引擎** | `browser/yaml_runner.go` | [browser/yaml_runner.go](https://github.com/npc-live/clawfirm/blob/main/browser/yaml_runner.go) |
| | `browser/executor.go` | [browser/executor.go](https://github.com/npc-live/clawfirm/blob/main/browser/executor.go) |
| | `browser/cdp.go` | [browser/cdp.go](https://github.com/npc-live/clawfirm/blob/main/browser/cdp.go) |
| | `tool/builtin/browser_shortcut.go` | [tool/builtin/browser_shortcut.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/browser_shortcut.go) |
| | `cmd/browser-shortcut/main.go` | [cmd/browser-shortcut/](https://github.com/npc-live/clawfirm/tree/main/cmd/browser-shortcut) |

---

## 2. 爆款选题

> 抖音爆款视频搜索、下载、AI 分析、脚本生成、口播文案

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **主工作流** | `media.whip` | [media.whip](https://github.com/npc-live/clawfirm/blob/main/media.whip) |
| **CDP (抖音搜索/下载)** | `app/assets/shortcuts/douyin.yaml` | (同上, search/download 命令) |
| **分析工具** | `tool/builtin/media_understand.go` | [tool/builtin/media_understand.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/media_understand.go) |
| | `cmd/media-understand/main.go` | [cmd/media-understand/](https://github.com/npc-live/clawfirm/tree/main/cmd/media-understand) |

工作流阶段: 搜索爆款(Phase 1) → 下载+AI分析(Phase 2) → 脚本生成(Phase 3) → 口播文案(Phase 4) → 声纹克隆+数字人视频(Phase 5)

---

## 3. 短视频合成

> 从脚本到成片：数字人、AI 场景、录屏、拼接、转场、BGM、字幕

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **制作流水线** | `app/assets/skills/video-skills/WORKFLOW.md` | [skills/video-skills/WORKFLOW.md](https://github.com/npc-live/clawfirm/blob/main/app/assets/skills/video-skills/WORKFLOW.md) |
| **Step 1: 脚本+分镜** | `app/assets/skills/video-skills/video-script-generator/` | [video-script-generator/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/video-skills/video-script-generator) |
| **Step 2a: 数字人** | `app/assets/skills/video-skills/digital-avatar/` | [digital-avatar/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/video-skills/digital-avatar) |
| **Step 2b: AI 场景** | `app/assets/skills/video-skills/scene-video-generator/` | [scene-video-generator/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/video-skills/scene-video-generator) |
| **声纹克隆+TTS** | `app/assets/skills/video-skills/voice-clone-tts/` | [voice-clone-tts/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/video-skills/voice-clone-tts) |
| **Step 3: 视频拼接** | `app/assets/skills/video-skills/video-stitcher/` | [video-stitcher/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/video-skills/video-stitcher) |
| **可灵 API** | `app/assets/skills/kling/` | [skills/kling/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/kling) |
| **Remotion 渲染** | `app/assets/skills/remotion-video/` | [skills/remotion-video/](https://github.com/npc-live/clawfirm/tree/main/app/assets/skills/remotion-video) |
| **爆款→视频全流程** | `media.whip` (Phase 5) | [media.whip](https://github.com/npc-live/clawfirm/blob/main/media.whip) |

---

## 4. 封面制作

> AI 生成各平台封面 + 封面插入视频首帧 (0.1s)

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **Step 4: 封面 Skill** | `app/assets/skills/video-skills/cover-gen/SKILL.md` | [cover-gen/SKILL.md](https://github.com/npc-live/clawfirm/blob/main/app/assets/skills/video-skills/cover-gen/SKILL.md) |
| **内置 media_gen** | `tool/builtin/media_gen.go` | [tool/builtin/media_gen.go](https://github.com/npc-live/clawfirm/blob/main/tool/builtin/media_gen.go) |
| **CLI media-gen** | `cmd/media-gen/main.go` | [cmd/media-gen/](https://github.com/npc-live/clawfirm/tree/main/cmd/media-gen) |
| **抖音封面规范** | `app/assets/skills/social-publish/douyin/SKILL.md` (L80-92) | (同上) |
| **小红书封面规范** | `app/assets/skills/social-publish/xiaohongshu/SKILL.md` (L58-70) | (同上) |
| **B站封面规范** | `app/assets/skills/social-publish/bilibili/SKILL.md` (L79-95) | (同上) |

---

## 5. Polymarket 天气预测交易

> 扫描天气预测市场 → Open-Meteo 集合预报概率 → edge 计算 → 自动交易
>
> 📌 还没有 Polymarket 账户？通过返佣链接注册 / Don't have a Polymarket account? Sign up with referral: [https://polymarket.com/?r=0xOlivia](https://polymarket.com/?r=0xOlivia)

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **setup** | `app/assets/workflows/polymarket/setup.whip` | [workflows/polymarket/setup.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/polymarket/setup.whip) |
| **scan** | `app/assets/workflows/polymarket/scan.whip` | [workflows/polymarket/scan.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/polymarket/scan.whip) |
| **trade** | `app/assets/workflows/polymarket/trade.whip` | [workflows/polymarket/trade.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/polymarket/trade.whip) |
| **monitor** | `app/assets/workflows/polymarket/monitor.whip` | [workflows/polymarket/monitor.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/polymarket/monitor.whip) |
| **report** | `app/assets/workflows/polymarket/report.whip` | [workflows/polymarket/report.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/polymarket/report.whip) |

---

## 6. 新闻交易 (Hyperliquid)

> 加密新闻采集 → AI 信号打分 → Hyperliquid 永续合约做多/做空

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **setup** | `app/assets/workflows/hyperliquid/setup.whip` | [workflows/hyperliquid/setup.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/hyperliquid/setup.whip) |
| **scan** | `app/assets/workflows/hyperliquid/scan.whip` | [workflows/hyperliquid/scan.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/hyperliquid/scan.whip) |
| **trade** | `app/assets/workflows/hyperliquid/trade.whip` | [workflows/hyperliquid/trade.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/hyperliquid/trade.whip) |
| **monitor** | `app/assets/workflows/hyperliquid/monitor.whip` | [workflows/hyperliquid/monitor.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/hyperliquid/monitor.whip) |
| **report** | `app/assets/workflows/hyperliquid/report.whip` | [workflows/hyperliquid/report.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/hyperliquid/report.whip) |

---

## 7. 量化交易 (Rockflow 港美股)

> MACD + ATR 技术指标 → Yahoo Finance 行情 → Rockflow 模拟盘港股/美股多空

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **主工作流** | `app/assets/workflows/rockflow.whip` | [workflows/rockflow.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/rockflow.whip) |
| **函数库 (指标计算)** | `funcs/library.go` | [funcs/library.go](https://github.com/npc-live/clawfirm/blob/main/funcs/library.go) |
| | `cmd/func/main.go` | [cmd/func/](https://github.com/npc-live/clawfirm/tree/main/cmd/func) |

标的: 港股 (00700.HK, 09988.HK 等 7 只) + 美股 (NVDA, TSLA, ARM 等 12 只)

---

## 8. 软件出海 (SaaS)

> 市场调研 → 竞品分析 → 收购/自建 → 落地页 → 发布 → 监控

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **setup** | `app/assets/workflows/saas/setup.whip` | [workflows/saas/setup.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/setup.whip) |
| **acquire** | `app/assets/workflows/saas/acquire.whip` | [workflows/saas/acquire.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/acquire.whip) |
| **landing** | `app/assets/workflows/saas/landing.whip` | [workflows/saas/landing.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/landing.whip) |
| **launch** | `app/assets/workflows/saas/launch.whip` | [workflows/saas/launch.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/launch.whip) |
| **monitor** | `app/assets/workflows/saas/monitor.whip` | [workflows/saas/monitor.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/monitor.whip) |
| **report** | `app/assets/workflows/saas/report.whip` | [workflows/saas/report.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/report.whip) |

---

## 9. 域名套利

> 扫描过期域名 → 价值评估 (反链/关键词/品牌) → 自动抢注

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **setup** | `app/assets/workflows/domains/setup.whip` | [workflows/domains/setup.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/domains/setup.whip) |
| **scan** | `app/assets/workflows/domains/scan.whip` | [workflows/domains/scan.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/domains/scan.whip) |
| **snipe** | `app/assets/workflows/domains/snipe.whip` | [workflows/domains/snipe.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/domains/snipe.whip) |
| **list** | `app/assets/workflows/domains/list.whip` | [workflows/domains/list.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/domains/list.whip) |
| **report** | `app/assets/workflows/domains/report.whip` | [workflows/domains/report.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/domains/report.whip) |

---

## 10. 闲鱼套利 (电商跨平台套利)

> 闲鱼低价采购 → 拼多多/eBay/Amazon 卖出，跨平台价差套利

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **setup** | `app/assets/workflows/arbitrage/setup.whip` | [workflows/arbitrage/setup.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/arbitrage/setup.whip) |
| **scan** | `app/assets/workflows/arbitrage/scan.whip` | [workflows/arbitrage/scan.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/arbitrage/scan.whip) |
| **buy** | `app/assets/workflows/arbitrage/buy.whip` | [workflows/arbitrage/buy.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/arbitrage/buy.whip) |
| **list** | `app/assets/workflows/arbitrage/list.whip` | [workflows/arbitrage/list.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/arbitrage/list.whip) |
| **report** | `app/assets/workflows/arbitrage/report.whip` | [workflows/arbitrage/report.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/arbitrage/report.whip) |
| **运营物料** | `_ops/xianyu/` | (本地, 未入 git) |

两种模式: 国内 (闲鱼→拼多多) + 跨境 (eBay→Amazon)

---

## 11. 亚马逊返佣联盟

> SEO 选题 → 产品调研 → AI 写测评 → 发布 → SEO 监控

| 类型 | 本地路径 | GitHub |
|------|----------|--------|
| **setup** | `app/assets/workflows/amazon-affiliate/setup.whip` | [workflows/amazon-affiliate/setup.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/amazon-affiliate/setup.whip) |
| **research** | `app/assets/workflows/amazon-affiliate/research.whip` | [workflows/amazon-affiliate/research.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/amazon-affiliate/research.whip) |
| **write** | `app/assets/workflows/amazon-affiliate/write.whip` | [workflows/amazon-affiliate/write.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/amazon-affiliate/write.whip) |
| **publish** | `app/assets/workflows/amazon-affiliate/publish.whip` | [workflows/amazon-affiliate/publish.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/amazon-affiliate/publish.whip) |
| **seo-monitor** | `app/assets/workflows/amazon-affiliate/seo-monitor.whip` | [workflows/amazon-affiliate/seo-monitor.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/amazon-affiliate/seo-monitor.whip) |

---

## 附：共享基础设施

所有业务共用的底层模块：

| 模块 | 路径 | 被哪些业务使用 |
|------|------|---------------|
| CDP 浏览器引擎 | `browser/` | 自媒体发布、爆款选题、闲鱼套利 |
| WhipFlow 引擎 | `whipflow/` | 所有 .whip 业务 |
| Agent 核心 | `agent/` | 全部 |
| LLM Provider | `provider/` | 全部 |
| 密钥库 | `vault/` | 交易类 (Polymarket/Hyperliquid/Rockflow) |
| 定时调度 | `cron/` | 自媒体发布、域名扫描、套利监控 |
| 消息通道 | `channel/telegram/` | 全部 (Telegram 通知) |
| 媒体生成 | `tool/builtin/media_gen.go` + `cmd/media-gen/` | 封面制作、自媒体发布 |
| 媒体理解 | `tool/builtin/media_understand.go` + `cmd/media-understand/` | 爆款选题、短视频分析 |
| 桌面应用 | `mobile/` (Tauri + React) | 全部 (Chat/Canvas/Channels UI) |
| 持久化 | `store/` (SQLite) | 全部 (session/message/cronjob) |
| 配置 | `config/` | 全部 |

---

## 附：其他未归类

| 本地路径 | GitHub | 说明 |
|----------|--------|------|
| `app/assets/workflows/gaokao/` | [workflows/gaokao/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/gaokao) | 高考志愿填报 (setup/research/match/plan/report) |
| `app/assets/workflows/creator/create.whip` | [workflows/creator/](https://github.com/npc-live/clawfirm/tree/main/app/assets/workflows/creator) | Whip 元生成器 (根据业务描述自动生成工作流) |
| `app/assets/shortcuts/zhipin.yaml` | [shortcuts/zhipin.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/zhipin.yaml) | Boss直聘 CDP |
| `app/assets/shortcuts/channels.yaml` | [shortcuts/channels.yaml](https://github.com/npc-live/clawfirm/blob/main/app/assets/shortcuts/channels.yaml) | 频道管理 CDP |
