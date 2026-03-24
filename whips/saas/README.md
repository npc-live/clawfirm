# whips/saas

SaaS 软件出海全流程自动化。从产品定位、落地页文案、多渠道获客，到 Product Hunt 发布、用户反馈监控、增长周报，覆盖出海 0→1 的核心环节。

## 工作流文件

| 文件 | 职责 |
|------|------|
| `setup.whip` | 竞品分析、目标市场选择、GTM 策略生成 |
| `landing.whip` | 英文落地页文案生成（Hero / Features / Pricing / FAQ） |
| `acquire.whip` | 多渠道获客内容生成（Reddit / HN / Cold Email / SEO） |
| `launch.whip` | Product Hunt 发布全套素材 + 小时级执行计划 |
| `monitor.whip` | 每日监控：评论反馈、竞品动态、生成回复草稿 |
| `report.whip` | 周报：渠道效果、MRR 进展、下周行动计划 |

## 快速开始

### 第一步：写产品描述

```bash
cat > data/current-run.json << 'EOF'
{
  "run_id": "saas-001",
  "product_name": "Notevo",
  "product_desc": "AI-powered meeting notes that auto-send action items to your team",
  "product_url": "https://notevo.app",
  "product_category": "productivity",
  "target_market": "US",
  "target_persona": "engineering managers and startup founders",
  "pricing_model": "freemium",
  "monthly_price_usd": 19,
  "current_mrr_usd": 0,
  "launch_channels": ["producthunt", "reddit", "hacker-news", "seo-article"],
  "competitor_names": ["Otter.ai", "Fireflies.ai", "Notion AI"]
}
EOF
```

### 第二步：初始化（竞品分析 + GTM 策略）

```bash
whipflow run whips/saas/setup.whip
# → docs/competitor-analysis.md   竞品矩阵 + 差异化机会
# → docs/saas-strategy.md         完整 GTM 策略（自动校验循环）
# → data/saas-config.json         配置文件
```

### 第三步：生成落地页文案

```bash
whipflow run whips/saas/landing.whip
# → docs/landing-copy.md          完整英文落地页文案，含 A/B 版本
```

### 第四步：生成获客内容

```bash
# 生成所有渠道内容
cat > data/current-run.json << 'EOF'
{ "channel": "all" }
EOF
whipflow run whips/saas/acquire.whip
# → docs/content/reddit-posts.md
# → docs/content/hn-submission.md
# → docs/content/cold-email-sequence.md
# → docs/content/seo-article-[keyword].md

# 只生成 Reddit 内容
cat > data/current-run.json << 'EOF'
{ "channel": "reddit" }
EOF
whipflow run whips/saas/acquire.whip

# 只生成冷邮件
cat > data/current-run.json << 'EOF'
{ "channel": "cold-email" }
EOF
whipflow run whips/saas/acquire.whip
```

### 第五步：准备 Product Hunt 发布

```bash
# 生成所有 PH 发布素材（Tagline / Description / Maker Comment / 社群预热文案）
cat > data/current-run.json << 'EOF'
{ "launch_mode": "assets" }
EOF
whipflow run whips/saas/launch.whip
# → docs/ph-launch-kit.md    完整发布素材包 + 发布前核查清单
```

### 第六步：启动每日监控

```bash
# 每天执行一次（建议加入 cron）
whipflow run whips/saas/monitor.whip
# → docs/reply-drafts-[today].md   回复草稿（紧急反馈自动标记 URGENT）
# → data/saas-metrics.json          每日指标快照
```

### 第七步：查看周报

```bash
# 每周一执行
whipflow run whips/saas/report.whip
# → docs/weekly-report-[date].md   增长周报 + 下周行动计划
```

---

## current-run.json 字段说明

### setup.whip 所需字段

| 字段 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `run_id` | 是 | 唯一 ID | `"saas-001"` |
| `product_name` | 是 | 产品名称 | `"Notevo"` |
| `product_desc` | 是 | 一句话描述（英文） | `"AI meeting notes..."` |
| `product_url` | 是 | 产品网址 | `"https://notevo.app"` |
| `product_category` | 是 | 类别 | `"productivity"` / `"devtools"` / `"marketing"` |
| `target_market` | 是 | 目标市场 | `"US"` / `"Global"` / `"Europe"` |
| `target_persona` | 是 | 目标用户描述 | `"startup founders"` |
| `pricing_model` | 是 | 定价模式 | `"freemium"` / `"free-trial"` / `"paid-only"` |
| `monthly_price_usd` | 是 | 付费版月价 | `19` |
| `current_mrr_usd` | 是 | 当前 MRR | `0`（刚起步）|
| `launch_channels` | 是 | 渠道列表 | `["producthunt","reddit","seo-article"]` |
| `competitor_names` | 否 | 已知竞品 | `["Notion","Coda"]` |

### acquire.whip 所需字段

| 字段 | 必填 | 说明 |
|------|------|------|
| `channel` | 是 | `"reddit"` / `"hn"` / `"cold-email"` / `"seo-article"` / `"all"` |

### report.whip 所需字段（每周更新）

| 字段 | 说明 |
|------|------|
| `mrr_usd` | 本周 MRR（手动填写） |
| `new_signups` | 本周新注册数 |
| `active_users` | 本周活跃用户数 |

---

## 生成文件说明

```
docs/
├── competitor-analysis.md         竞品矩阵 + 差异化机会点
├── saas-strategy.md               GTM 策略（定位 + 冷启动 + 增长飞轮）
├── landing-copy.md                英文落地页完整文案（含 A/B 变体）
├── ph-launch-kit.md               PH 发布素材包 + 核查清单
├── content/
│   ├── reddit-posts.md            Reddit 帖子（3 条，不同角度）
│   ├── hn-submission.md           HN Show HN 提交内容
│   ├── cold-email-sequence.md     冷邮件三封序列
│   └── seo-article-[keyword].md  SEO 长文（1500-2500 词）
└── weekly-report-[date].md        增长周报

data/
├── saas-config.json               产品配置
├── saas-posts.json                所有发布记录（status: draft/published）
├── saas-leads.json                冷邮件线索列表
├── saas-metrics.json              每日指标快照
└── reply-drafts-[date].md         当日评论回复草稿
```

---

## 典型场景示例

### 场景 1：开发工具，面向独立开发者

```json
{
  "product_category": "devtools",
  "target_persona": "indie developers and freelancers",
  "launch_channels": ["hacker-news", "reddit", "seo-article"],
  "pricing_model": "free-trial"
}
```
重点渠道：HN Show HN > r/webdev / r/programming > SEO（"best X for developers"）

### 场景 2：营销工具，面向中小企业

```json
{
  "product_category": "marketing",
  "target_persona": "SMB marketing managers",
  "launch_channels": ["producthunt", "cold-email", "seo-article"],
  "pricing_model": "freemium"
}
```
重点渠道：PH 首页 > Cold Email（Apollo 精准找 CMO）> SEO（"[competitor] alternative"）

### 场景 3：效率工具，面向全球用户

```json
{
  "product_category": "productivity",
  "target_persona": "remote teams and knowledge workers",
  "launch_channels": ["producthunt", "reddit", "hacker-news", "seo-article"],
  "pricing_model": "freemium",
  "target_market": "Global"
}
```
重点渠道：PH + HN 组合发布 > r/productivity / r/remotework > SEO 长尾
