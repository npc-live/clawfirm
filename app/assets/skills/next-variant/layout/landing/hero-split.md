# Layout Skill: Hero — Split (50/50)

左文字 + 右视觉。给视觉资产（截图、3D、插图）充足的空间，文字有更多纵向延展。

## 结构

```
┌──────────────────────────────────────────────────────────┐
│  NAV  logo ————————————————————————— links   [CTA btn]  │
├──────────────────────────────────────────────────────────┤
│                         │                                │
│   EYEBROW               │   ┌──────────────────────┐   │
│                         │   │                      │   │
│   BIG                   │   │   VISUAL ASSET       │   │
│   HEADLINE              │   │   (screenshot /      │   │
│   HERE                  │   │    3D / illustration)│   │
│                         │   │                      │   │
│   Subheading text       │   │                      │   │
│   one or two lines      │   └──────────────────────┘   │
│                         │                                │
│   [Primary]  [Ghost]    │                                │
│                         │                                │
│   ── social proof ──    │                                │
│                         │                                │
└──────────────────────────────────────────────────────────┘
```

## CSS

```css
.hero-split {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 48px;
  align-items: center;
  min-height: 90vh;
  max-width: 1200px;
  margin: 0 auto;
  padding: 80px 48px;
}

.hero-split-text {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.hero-split-visual {
  position: relative;
}

.hero-split-visual img,
.hero-split-visual .mock {
  width: 100%;
  border-radius: 16px;
  border: 1px solid var(--border-default);
  box-shadow: 0 32px 80px rgba(0,0,0,0.18);
}

/* Floating badge on visual (optional) */
.visual-badge {
  position: absolute;
  bottom: -16px;
  left: -16px;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 12px;
  padding: 12px 16px;
  backdrop-filter: blur(12px);
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
  box-shadow: 0 8px 24px rgba(0,0,0,0.12);
}

@media (max-width: 768px) {
  .hero-split {
    grid-template-columns: 1fr;
    text-align: center;
    padding: 60px 24px;
  }
  .hero-split-visual { order: -1; }
}
```

## Rules

- 列宽 `1fr 1fr`，绝对对等 — 不要 `2fr 1fr` (那是另一种布局)
- 视觉资产有 `box-shadow` — 给截图/Mock 真实感
- 可选：在 visual 上叠加"浮动小卡片"（用户数、实时通知等），增加层次
- 文字列内间距用 `margin-bottom` 控制，不用 `gap`（避免 flex gap 过大）
- 移动端：视觉资产移至顶部，文字在下
- Headline: `clamp(32px, 4vw, 56px)`，左对齐

## 变体选项

```
mirror: 右文字 + 左视觉（改 grid 顺序或用 order）
angle:  视觉资产带 perspective transform: rotateY(-8deg) rotateX(4deg)
float:  视觉资产 animation: float 4s ease-in-out infinite (上下漂浮)
```

## Forbidden

- 视觉列用纯色占位（必须有真实 mock 内容）
- 两列都居中对齐（文字列左对齐）
- 视觉资产没有任何 shadow/border（会和背景融合）
