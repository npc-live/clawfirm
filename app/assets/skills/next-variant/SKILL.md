---
name: next-variant
version: 1.0.0
description: 两个入口，一个出口。给定一个网站 URL 或一段内容描述，输出可用于生成 UI 的 skill 组合。
author: nextvariant
---

# Next Variant Style Engine

## 两个功能入口

---

## 功能一：网站风格反向提取（URL → Skills）

**触发词**：用户提供一个网站 URL，说「帮我分析这个网站的风格」「这个网站风格也不错」「参考这个网站」

### 执行流程

**Step 1 — 抓取原始数据**

同时执行以下操作：
1. 用 browser/fetch 获取页面 HTML
2. 从 HTML 中提取所有 `<link rel="stylesheet">` 的 href
3. 下载主 CSS 文件（优先选名称含 `main`、`global`、`theme` 的）
4. 在 CSS 中搜索 `:root {` 块，提取所有 `--` 开头的 custom properties

```
目标提取项：
- 背景色 (background, --bg-*, --color-background)
- 文字色 (color, --text-*, --color-text)
- Accent 色 (--color-brand, --color-primary, --accent)
- 字体 (font-family, --font-*)
- 字号体系 (--text-*, --font-size-*)
- 间距 (--space-*, --spacing-*)
- 圆角 (border-radius, --radius-*)
- 阴影 (box-shadow, --shadow-*)
- 特殊效果 (filter, backdrop-filter, animation)
```

**Step 2 — 视觉特征识别**

根据抓到的 token 值，判断以下维度：

```
色调判断：
  背景 L < 20%           → 深色系
  背景含暖色调 (rgb R≈G, 偏黄) → 暖色系  
  背景 L > 90% 且无彩     → 极简白
  accent 含渐变           → 渐变系

字体判断：
  font-family 含 monospace → 等宽风格
  font-family 含 serif      → 衬线风格
  letter-spacing 负值 + 大字号 → 展示型 grotesque
  -apple-system             → 系统 UI 风格

风格判断：
  border-radius: 0          → Brutalist
  backdrop-filter: blur      → Glassmorphism / Vibrancy
  box-shadow 含 translateY   → Press 3D
  特殊 CSS 变量名 (--color-terminal, --color-shell) → 找命名语义
  box-shadow 多彩 (#ff0, #0ff, #f0f) → Chromatic aberration
```

**Step 3 — 映射到现有 Skills**

对照下表找最接近的 skill 组合：

| 检测到的特征 | 推荐 color skill |
|------------|----------------|
| 深黑底 + 透明层叠 + 蓝色 accent | `dark-moody` |
| 近黑 + 暖白文字 + 紫色 accent | `terminal-violet` |
| 近黑 + 霓虹发光 + grid | `neon-tech` |
| 奶油白底 + 橙色 accent | `warm-cream-orange` |
| 纯白底 + 无彩 accent | `clean-minimal` |
| 米色/棕色系 + 大地色 accent | `warm-editorial` |
| 渐变色彩 + 深色底 | `gradient-vibrant` |
| 高饱和纯色块 | `bright-blocks` |
| rgba 语义色 + 系统蓝 `#007AFF` | `apple-system` |

| 检测到的特征 | 推荐 typography skill |
|------------|---------------------|
| `font-family` 含 monospace / Mono | `mono-tech` |
| `-apple-system` / `BlinkMacSystemFont` | `sf-pro` |
| `IBM Plex Sans` | `humanist-sans` |
| `Suisse Intl` / `Helvetica Neue` / letter-spacing 负值 | `swiss-grotesque` |
| `Playfair` / `Lora` / serif | `display-serif` |
| `clamp` 超大字号 + font-weight 800+ | `big-display` |
| `Inter` / `system-ui` + 14px 正文 | `inter-minimal` |

| 检测到的特征 | 推荐 style skill |
|------------|----------------|
| `backdrop-filter: blur` + rgba 透明 | `glassmorphism` |
| `backdrop-filter` + 系统 chrome | `macos-vibrancy` |
| `border-radius: 0` + 2px+ 粗边框 | `brutalist` |
| `box-shadow: 0 2px 0` + hover translateY | `press-3d` |
| 多彩 box-shadow RGB 分量 | `chromatic-terminal` |
| 无 shadow + 纯色填充 | `flat-clean` |
| 暖色纸质 + sepia filter | `retro-warm` |
| offset shadow `4px 4px 0` + 高饱和 | `playful` |
| 多层光照 gradient + inset shadow | `skeuomorphic` |

**Step 4 — 输出**

```
## 识别结果

网站：{URL}

### 提取到的关键 Token
- 背景色：{值}
- 主文字：{值}
- Accent：{值}
- 字体：{值}
- 特殊效果：{描述}

### 推荐 Skill 组合
color:      {skill}
typography: {skill}
style:      {skill}
layout:     {skill（根据页面结构判断）}

### 判断依据
- color → {skill}：因为 {具体 token 值和特征}
- typography → {skill}：因为 {字体/字号/字距特征}
- style → {skill}：因为 {具体效果/shadow/border特征}

### 是否需要新建 Skill？
{如果有强烈特征没有对应 skill，在此说明并创建新 skill 文件}
```

**Step 5（可选）— 如有显著特征未被覆盖，新建 skill 文件**

在 `skills/color/`、`skills/typography/`、`skills/style/` 下新建 `.md` 文件，格式参考现有任意 skill 文件，必须包含：
- Design Tokens（CSS 变量）
- Rules（正向约束）
- Component Examples（代码示例）
- Forbidden（禁止项）

---

## 功能二：内容匹配风格（Content → Skills）

**触发词**：用户描述他要做的产品/页面，说「我要做一个 XX 网站」「帮我找适合的风格」「这个内容适合什么风格」

### 执行流程

**Step 1 — 内容分析**

从用户输入中提取以下信息（没有的项不追问，用默认判断）：

```
产品类型：   SaaS / 电商 / 作品集 / 博客 / 工具 / 营销页 / App...
目标用户：   开发者 / 消费者 / 企业 / 创意从业者 / 学生...
品牌气质：   专业严肃 / 友好亲切 / 创意前卫 / 极简克制 / 活泼有趣...
内容密度：   信息密集（数据/功能多）/ 呼吸感（故事/情感为主）
竞品参考：   有没有提到参考网站或"像 XX 那样"
```

**Step 2 — 决策树**

```
产品类型
├── 开发者工具 / AI 基础设施 / GPU / 云计算
│   ├── 极简严肃风    → terminal-violet + swiss-grotesque + chromatic-terminal
│   └── 亲切工程师风  → warm-cream-orange + humanist-sans + press-3d
│
├── SaaS / 产品工具（非开发者向）
│   ├── 现代简洁      → clean-minimal + inter-minimal + flat-clean
│   ├── 暗色高端      → dark-moody + inter-minimal + glassmorphism
│   └── 活泼消费品    → bright-blocks + big-display + playful
│
├── 电商 / 产品展示
│   ├── 精品/手工艺   → warm-editorial + display-serif + retro-warm
│   ├── 现代零售      → clean-minimal + inter-minimal + flat-clean
│   └── 潮牌/创意     → dark-moody + big-display + brutalist
│
├── 作品集 / Agency
│   ├── 字体主导      → clean-minimal + big-display + brutalist
│   ├── 视觉主导      → gradient-vibrant + big-display + glassmorphism
│   └── 复古品味      → warm-editorial + display-serif + retro-warm
│
├── 博客 / 媒体 / 内容
│   ├── 严肃文学      → warm-editorial + display-serif + retro-warm
│   └── 现代数字媒体  → dark-moody + inter-minimal + flat-clean
│
├── 数据 / 仪表盘 / 内部工具
│   ├── 技术感强      → neon-tech + mono-tech + glassmorphism
│   └── 企业标准      → apple-system + sf-pro + macos-vibrancy
│
└── 游戏 / 娱乐 / 儿童
    └── → bright-blocks / gradient-vibrant + big-display + playful
```

**Step 3 — Layout 选择**

```
落地页（首次访问的营销页）→ 进入 landing/ 子维度：
  有强视觉资产（截图/3D）  → hero-split
  SaaS 标准               → hero-centered
  创意/agency             → hero-type-only
  背景视频/图片            → hero-full-screen

应用/工具界面  → sidebar-dashboard
画廊/作品展示  → masonry-gallery
数据仪表盘    → bento-box
文章/博客     → editorial-magazine
```

**Step 4 — 输出**

```
## 风格推荐

根据你描述的「{产品简述}」，推荐以下 skill 组合：

color:      {skill}  — {一句话说明为什么}
layout:     {skill}  — {一句话说明为什么}
typography: {skill}  — {一句话说明为什么}
style:      {skill}  — {一句话说明为什么}

### 参考近似案例
{从 Combination Examples 中找最接近的，或说明这个组合的视觉感受}

### 备选方案
如果你想要更 {形容词} 的感觉，可以把 {skill A} 换成 {skill B}：
color: {alternative}  — {区别说明}
```

---

## Skill 索引（当前可用）

### Color
| Skill | Mood |
|-------|------|
| `dark-moody` | 深黑透明层叠 · 蓝色 accent |
| `gradient-vibrant` | 彩色渐变 · 戏剧性 |
| `clean-minimal` | 纯净白灰 · 无色彩 |
| `warm-editorial` | 奶油/锈橙 · 大地色 |
| `neon-tech` | 近黑底 · 霓虹发光 |
| `warm-cream-orange` | 奶油白 · 橙色 accent · PostHog 风 |
| `bright-blocks` | 高饱和纯色块 |
| `apple-system` | 系统语义色 · 蓝 accent · Light/Dark |
| `terminal-violet` | 近黑 `#0b0b0b` · 暖白 · 紫 accent · Lambda 风 |

### Layout
| Skill | Shape |
|-------|-------|
| `masonry-gallery` | Pinterest 瀑布流 |
| `bento-box` | 仪表盘格子 |
| `full-bleed` | 全出血 hero |
| `editorial-magazine` | 杂志专栏 |
| `sidebar-dashboard` | 240px 侧边栏 App Shell |
| `landing/hero-centered` | SaaS 居中 hero |
| `landing/hero-split` | 50/50 分栏 hero |
| `landing/hero-type-only` | 纯字体 hero |
| `landing/hero-full-screen` | 全视口背景 hero |
| `landing/section-features` | 功能介绍 section |
| `landing/section-social-proof` | 信任背书 section |
| `landing/section-pricing` | 定价方案 section |
| `landing/section-cta` | 页尾转化 section |

### Typography
| Skill | Voice |
|-------|-------|
| `inter-minimal` | 系统 sans · 隐形排版 |
| `display-serif` | Playfair/Lora · 编辑感 |
| `mono-tech` | 等宽全覆盖 · 终端感 |
| `big-display` | 巨型 grotesque · 字即图像 |
| `humanist-sans` | IBM Plex Sans · 人文温度 |
| `sf-pro` | -apple-system · macOS HIG |
| `swiss-grotesque` | Suisse Intl/Helvetica · 精密工程 |

### Style
| Skill | Texture |
|-------|---------|
| `glassmorphism` | 磨砂玻璃 · 渐变背景 |
| `brutalist` | B&W · 0圆角 · 粗边框 |
| `flat-clean` | 无阴影 · 纯色填充 |
| `retro-warm` | 纸质纹理 · 复古暖 |
| `playful` | 偏移阴影 · 胶囊 · 高饱和 |
| `press-3d` | 底部阴影按钮 · hover 翻转 · PostHog 风 |
| `macos-vibrancy` | vibrancy blur · 窗口 chrome |
| `skeuomorphic` | 多层光照 · 物理材质感 |
| `chromatic-terminal` | RGB 色差 · 终端 UI · Lambda 风 |

---

## 生成 Prompt 模板

确定 skill 组合后，用此模板生成 UI：

```
你是一名 UI 工程师，正在生成 React/HTML+CSS 组件。

严格执行以下 4 个 design skill 的所有约束。每个 skill 文件定义了 token、rules 和 forbidden——视为硬性规则。

## 激活的 Skills

### Color: {COLOR_SKILL 的完整文件内容}

### Layout: {LAYOUT_SKILL 的完整文件内容}

### Typography: {TYPOGRAPHY_SKILL 的完整文件内容}

### Style: {STYLE_SKILL 的完整文件内容}

## 任务
构建：{用户需求}

## 输出要求
- 使用 skill 中定义的 design token，不硬编码颜色值
- 遵守每个 skill 的所有 Rules
- 严格执行所有 Forbidden 项
- 输出单个自包含组件（inline style 或 <style> 块）
- 禁止 Tailwind（除非 skill 明确允许）
- 所有占位内容用真实感虚拟数据，禁止纯色灰色块
```
