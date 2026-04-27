# Variant UI Generation — Skill Combinator

## How It Works

Every UI generation request uses **4 skills picked from 4 dimensions**. The result is:

```
UI = color_skill × layout_skill × typography_skill × style_skill
```

Skills constrain the LLM into a high-quality "cell" of the design space.  
Diversity comes from the combinatorial space (5×5×4×5 = **500 valid combinations**), not from LLM free-play.

---

## Skill Index

### Color (`skills/color/`)
| Skill | File | Mood |
|-------|------|------|
| `dark-moody` | `dark-moody.md` | Dark, transparent layers, blue accent |
| `gradient-vibrant` | `gradient-vibrant.md` | Colorful gradients, dramatic |
| `clean-minimal` | `clean-minimal.md` | Off-white, monochrome, silent |
| `warm-editorial` | `warm-editorial.md` | Cream, rust, earth tones |
| `neon-tech` | `neon-tech.md` | Near-black, neon glow, grid |
| `warm-cream-orange` | `warm-cream-orange.md` | 奶油白底 + 橙色 CTA，PostHog 风格 |
| `bright-blocks` | `bright-blocks.md` | 高饱和纯色块，Stripe/Notion 风格 |
| `apple-system` | `apple-system.md` | 系统语义色，蓝色 Accent，Light/Dark 双模式 |
| `terminal-violet` | `terminal-violet.md` | 近黑底 `#0b0b0b` + 暖壳白 + 紫 Accent，Lambda 风格 |

### Layout (`skills/layout/`)
| Skill | File | Shape |
|-------|------|-------|
| `masonry-gallery` | `masonry-gallery.md` | Pinterest grid, variable heights |
| `bento-box` | `bento-box.md` | Dashboard cells, mixed sizes |
| `full-bleed` | `full-bleed.md` | Edge-to-edge hero, alternating |
| `editorial-magazine` | `editorial-magazine.md` | Column text, pull quotes |
| `sidebar-dashboard` | `sidebar-dashboard.md` | App shell, 240px sidebar |

#### Landing Page 子维度 (`skills/layout/landing/`)

**Hero 变体** — 选一种作为 landing 的 layout skill：

| Skill | File | 适合场景 |
|-------|------|---------|
| `hero-centered` | `landing/hero-centered.md` | SaaS / 产品首页，截图在 fold 下方 |
| `hero-split` | `landing/hero-split.md` | 有强视觉资产（截图、3D）时 |
| `hero-type-only` | `landing/hero-type-only.md` | Agency / 创意工作室，字体即视觉 |
| `hero-full-screen` | `landing/hero-full-screen.md` | 视频/图片背景，强冲击力 |

**Section 组件** — 按需组合拼装 landing 页面：

| Skill | File | 用途 |
|-------|------|------|
| `section-features` | `landing/section-features.md` | 功能介绍（3列卡片 / 交替图文 / checklist）|
| `section-social-proof` | `landing/section-social-proof.md` | Logo bar / 评价卡片 / 数据指标条 |
| `section-pricing` | `landing/section-pricing.md` | 定价方案（3层级 + 推荐标识）|
| `section-cta` | `landing/section-cta.md` | 页尾转化区（全宽色块 / 卡片 / 极简）|

**Landing 完整页面组合公式：**
```
landing = hero_variant + [logo-bar] + features + [social-proof] + [pricing] + cta
```

### Typography (`skills/typography/`)
| Skill | File | Voice |
|-------|------|-------|
| `inter-minimal` | `inter-minimal.md` | System sans, invisible type |
| `display-serif` | `display-serif.md` | Playfair/Lora, editorial voice |
| `mono-tech` | `mono-tech.md` | Monospace everything, terminal |
| `big-display` | `big-display.md` | Massive grotesque, type as image |
| `humanist-sans` | `humanist-sans.md` | IBM Plex Sans，-0.015em tracking，人文温度 |
| `sf-pro` | `sf-pro.md` | -apple-system，13px 正文，macOS HIG 字号体系 |
| `swiss-grotesque` | `swiss-grotesque.md` | Suisse Intl / Helvetica Neue，国际主义，精密工程感 |

### Style (`skills/style/`)
| Skill | File | Texture |
|-------|------|---------|
| `glassmorphism` | `glassmorphism.md` | Blur, transparency, gradient bg |
| `brutalist` | `brutalist.md` | B&W, thick borders, no radius |
| `flat-clean` | `flat-clean.md` | No shadows, solid fills, friendly |
| `retro-warm` | `retro-warm.md` | Paper texture, sepia, artisan |
| `playful` | `playful.md` | Offset shadows, pill shapes, bold color |
| `press-3d` | `press-3d.md` | 底部单层阴影按钮 + hover 翻转，PostHog 风格 |
| `skeuomorphic` | `skeuomorphic.md` | 多层光照阴影，物理材质感，iOS6 风格 |
| `macos-vibrancy` | `macos-vibrancy.md` | 磨砂玻璃 vibrancy，窗口 chrome，Traffic Light |
| `chromatic-terminal` | `chromatic-terminal.md` | RGB 色差特效 + 终端 UI + 发光 CTA，Lambda 风格 |

---

## Generation Prompt Template

When generating UI, prepend this system prompt, filling in the 4 chosen skills:

```
You are a UI engineer generating React/HTML+CSS components.

Apply ALL of the following design skills exactly as specified. Each skill file defines tokens, rules, and forbidden patterns — treat them as hard constraints.

## Active Skills

### Color: {COLOR_SKILL}
{paste content of color skill file here}

### Layout: {LAYOUT_SKILL}
{paste content of layout skill file here}

### Typography: {TYPOGRAPHY_SKILL}
{paste content of typography skill file here}

### Style: {STYLE_SKILL}
{paste content of style skill file here}

## Task
Build: {USER_REQUEST}

## Requirements
- Use the design tokens defined in the active skills
- Follow every rule listed in each skill
- Respect every "Forbidden" section — these are hard stops
- Output a single self-contained component with inline styles or a <style> block
- No Tailwind classes unless the skill explicitly permits it
- No placeholder gray boxes — all content areas should have realistic dummy data
```

---

## Combination Examples

### Example A: Developer Tool Dashboard
```
color:      dark-moody
layout:     sidebar-dashboard
typography: mono-tech
style:      flat-clean
```

### Example B: Creative Portfolio
```
color:      gradient-vibrant
layout:     full-bleed
typography: big-display
style:      glassmorphism
```

### Example C: Artisan E-commerce
```
color:      warm-editorial
layout:     bento-box
typography: display-serif
style:      retro-warm
```

### Example D: Photography Gallery
```
color:      dark-moody
layout:     masonry-gallery
typography: inter-minimal
style:      flat-clean
```

### Example E: Sci-fi Dashboard
```
color:      neon-tech
layout:     bento-box
typography: mono-tech
style:      glassmorphism
```

### Example F: Kids / Consumer App
```
color:      clean-minimal
layout:     bento-box
typography: big-display
style:      playful
```

### Example G: Literary Blog
```
color:      warm-editorial
layout:     editorial-magazine
typography: display-serif
style:      retro-warm
```

### Example H: Brutalist Agency Site
```
color:      clean-minimal
layout:     full-bleed
typography: big-display
style:      brutalist
```

### Example I: Developer Tool Landing (PostHog 风格)
```
color:      warm-cream-orange
layout:     landing/hero-centered
typography: humanist-sans
style:      press-3d
```

### Example L: GPU Cloud / AI Infrastructure Landing (Lambda 风格)
```
color:      terminal-violet
layout:     landing/hero-centered
typography: swiss-grotesque
style:      chromatic-terminal
sections:   section-features(交替图文) + section-social-proof(stats) + section-cta(发光radial-gradient底)
```

### Example K: macOS App (原生系统感)
```
color:      apple-system
layout:     sidebar-dashboard
typography: sf-pro
style:      macos-vibrancy
```

### Example J: Open Source Product Page
```
color:      warm-cream-orange
layout:     landing/hero-split
typography: humanist-sans
style:      press-3d
sections:   section-features(交替图文) + section-pricing + section-cta(全宽深色)
```

---

## Style Swap (Style Dropper)

To change the look of an existing design without changing content:

```
Current combo: color=dark-moody, layout=masonry-gallery, type=inter-minimal, style=flat-clean
Target:        swap color=dark-moody → color=neon-tech

Prompt:
"Here is the existing component: {paste component}
Replace all color tokens and rules from [dark-moody] with [neon-tech].
Keep layout, typography, and style skills unchanged.
Output the restyled component."
```

No design token reverse-engineering needed — the skill file IS the source of truth.

---

## Validation Checklist

Before accepting a generated output, verify:

- [ ] Background color matches the color skill's `--bg-primary`
- [ ] No forbidden patterns from any active skill appear
- [ ] Typography uses the specified font family
- [ ] Layout structure matches the layout skill's diagram
- [ ] Radius / shadow rules match the style skill
