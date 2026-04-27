# Layout Skill: Hero — Centered

经典居中 hero。文字和 CTA 垂直居中，视觉重量均匀，适合产品/SaaS 首页。

## 结构

```
┌──────────────────────────────────────────────────────────┐
│  NAV  logo ————————————————————————— links   [CTA btn]  │
├──────────────────────────────────────────────────────────┤
│                                                          │
│                    EYEBROW LABEL                         │
│                                                          │
│              BIG HEADLINE (2–3 lines)                    │
│              BIG HEADLINE                                │
│                                                          │
│         Subheading — one sentence, max 12 words          │
│                                                          │
│            [Primary CTA]   [Secondary CTA]               │
│                                                          │
│         social proof: "Trusted by 10,000+ teams"        │
│         ○ ○ ○ ○ ○  avatar strip                          │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │            PRODUCT SCREENSHOT / DEMO             │   │
│  │            (below the fold, peeking)             │   │
│  └──────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

## CSS

```css
.hero-centered {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 120px 24px 0;
  max-width: 720px;
  margin: 0 auto;
}

.hero-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 999px;
  border: 1px solid var(--border-default);
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 24px;
}

.hero-headline {
  font-size: clamp(36px, 5.5vw, 64px);
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1.1;
  margin-bottom: 20px;
}

.hero-sub {
  font-size: 18px;
  color: var(--text-secondary);
  line-height: 1.6;
  max-width: 520px;
  margin-bottom: 36px;
}

.hero-cta-row {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 48px;
}

.hero-social-proof {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--text-muted);
}

.avatar-strip {
  display: flex;
}
.avatar-strip img {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 2px solid var(--bg-primary);
  margin-left: -8px;
}
.avatar-strip img:first-child { margin-left: 0; }

.hero-product-shot {
  width: 100%;
  max-width: 960px;
  margin: 0 auto;
  border-radius: 12px 12px 0 0;
  border: 1px solid var(--border-default);
  border-bottom: none;
  overflow: hidden;
  box-shadow: 0 -4px 80px rgba(0,0,0,0.12);
}
```

## Rules

- Headline: `clamp(36px, 5.5vw, 64px)` — responsive, never fixed
- Max width of text block: 720px — forces tight reading column
- Subheading: max 12 words — ruthlessly edited
- CTA 按钮间距: 12px gap，primary 实色，secondary 描边或 ghost
- Product screenshot 从底部"冒出"，裁掉底边 — 暗示还有更多
- Social proof 紧跟 CTA 之后 — 降低决策摩擦
- Nav: 透明背景，scroll 后固定并加背景

## Forbidden

- 居中对齐的 body 段落（只有 hero 居中）
- 超过 2 个 CTA 按钮
- Hero 区域内出现 feature 列表
- 产品截图用 iframe 真实渲染（静态截图即可）
