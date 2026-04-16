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
| **README** | `app/assets/workflows/saas/README.md` | [workflows/saas/README.md](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/README.md) |
| **setup** | `app/assets/workflows/saas/setup.whip` | [workflows/saas/setup.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/setup.whip) |
| **acquire** | `app/assets/workflows/saas/acquire.whip` | [workflows/saas/acquire.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/acquire.whip) |
| **landing** | `app/assets/workflows/saas/landing.whip` | [workflows/saas/landing.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/landing.whip) |
| **launch** | `app/assets/workflows/saas/launch.whip` | [workflows/saas/launch.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/launch.whip) |
| **monitor** | `app/assets/workflows/saas/monitor.whip` | [workflows/saas/monitor.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/monitor.whip) |
| **report** | `app/assets/workflows/saas/report.whip` | [workflows/saas/report.whip](https://github.com/npc-live/clawfirm/blob/main/app/assets/workflows/saas/report.whip) |

### 8.1 营销模块详解

营销链路：**setup(策略) → landing(转化页) → acquire(获客内容) → launch(PH引爆) → monitor(追踪) → report(复盘)**

#### setup.whip — 营销战略层

| 模块 | 说明 |
|------|------|
| 竞品分析 | 5-8 个竞品的获客渠道、定价、G2/PH 评分、用户差评 |
| SEO 关键词机会 | 竞品核心关键词 + 低竞争长尾词方向 |
| GTM 冷启动方案 | 0→100 用户各渠道具体打法 (PH/Reddit/HN/Cold Email/SEO) |
| 增长飞轮 | 100+ 用户后：免费→付费转化、referral/affiliate、内容 SEO 复利 |

#### landing.whip — 落地页文案 (转化)

| 模块 | 说明 |
|------|------|
| Hero Section | Headline + Subheadline + CTA button + 社会证明数字，含 A/B 版本 |
| Problem Section | 痛点共鸣 (用户语言描述 3 个痛点) |
| Features Section | outcome 导向功能描述 (非技术术语) |
| Social Proof | testimonial 模板 (3 条，含职位/公司规模) |
| Pricing Section | 含锚定价格策略，Free/Pro/Enterprise 梯度 |
| FAQ | 5 个最大购买阻力问题 |
| 自动审核循环 | validator agent 做转化率审核，不通过则自动修改 (max 2 轮) |

#### acquire.whip — 多渠道获客内容 (核心营销模块)

| 渠道 | 产出 | 输出文件 |
|------|------|----------|
| **Reddit** | 3 种帖子：教程软推 / Show Reddit / 问答参与 | `docs/content/reddit-posts.md` |
| **Hacker News** | Show HN 标题 + 首条评论 + 发布时间策略 | `docs/content/hn-submission.md` |
| **Cold Email** | 3 封序列邮件 (首次/跟进/最终) + Apollo/Hunter 找邮箱 | `docs/content/cold-email-sequence.md` |
| **SEO 文章** | 1500-2500 词英文长文 (对比测评/教程/竞品替代) | `docs/content/seo-article-[keyword].md` |

含内容合规 validator 审核：检查是否像真实用户、是否违反平台推广规则。

#### launch.whip — Product Hunt 发布 (引爆)

| 模块 | 说明 |
|------|------|
| PH 文案素材 | Tagline (3 备选) / Description (250-300 词) / Maker Comment / 回复模板 |
| 社群预热消息 | Twitter 3 条 (T-0/T+4h/T+12h)、LinkedIn 1 条、Slack/Discord 通知 |
| 发布核查清单 | D-7 → D-1 逐天 checklist (Hunter、截图、Demo Video、折扣码) |
| 小时级执行计划 | 发布当天 00:01 PST → 23:00 的每小时行动项 |

#### monitor.whip — 营销效果追踪

| 模块 | 说明 |
|------|------|
| 评论追踪 | Reddit/HN/PH 帖子新评论自动采集 + 情感分类 |
| 回复草稿 | 自动生成回复 (负面→问题→正面→功能请求 优先级) |
| 紧急标记 | 负面/bug 反馈自动 `URGENT` 标记 |
| 竞品监控 | 竞品新功能、定价变化、市场活动 |

#### report.whip — 增长周报 (复盘)

| 模块 | 说明 |
|------|------|
| 渠道效果分析 | 各渠道发布量/互动/ROI，效果最好/最差内容原因分析 |
| 用户反馈洞察 | Top 3 痛点、Top 3 功能请求、正面反馈核心原因 |
| 竞品动态 | 本周竞品重大变化对策略的影响 |
| 下周行动计划 | 红 (必须做) / 黄 (应该做) / 绿 (可以做) 优先级 |
| 30 天展望 | MRR 预测、最大风险/机会 |

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
