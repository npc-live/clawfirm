# Layout Skill: Hero — Full Screen

背景占满整个视口（视频、图片、纯色块、动效 canvas）。文字浮于其上。强视觉冲击。

## 结构

```
┌──────────────────────────────────────────────────────────┐
│  nav (transparent, floating)        logo  ——  [CTA]     │
│                                                          │
│                                                          │
│                                                          │
│                 CENTERED HEADLINE                        │
│              OR BOTTOM-LEFT HEADLINE                     │
│                                                          │
│                   [Primary CTA]                          │
│                                                          │
│                                                          │
│                                                          │
│                   ↓ scroll indicator                     │
└──────────────────────────────────────────────────────────┘
  ░░░░░ FULL BACKGROUND: video / image / gradient ░░░░░
```

## CSS

```css
.hero-fullscreen {
  position: relative;
  width: 100vw;
  height: 100vh;
  min-height: 600px;
  overflow: hidden;
  display: flex;
  align-items: center;      /* center variant */
  /* OR: align-items: flex-end; padding-bottom: 80px; for bottom variant */
  justify-content: center;
}

/* Background layer */
.hero-bg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  z-index: 0;
}

/* Overlay — always present for text legibility */
.hero-overlay {
  position: absolute;
  inset: 0;
  z-index: 1;
  /* Choose based on background type: */
  background: rgba(0, 0, 0, 0.45);                          /* dark photo */
  /* background: linear-gradient(to bottom, transparent 40%, rgba(0,0,0,0.7)); */ /* gradient from bottom */
  /* background: rgba(255, 255, 255, 0.15);  */              /* light wash on illustration */
}

/* Content */
.hero-content {
  position: relative;
  z-index: 2;
  text-align: center;
  padding: 0 24px;
  max-width: 800px;
}

/* Bottom-left variant */
.hero-content.bottom-left {
  position: absolute;
  bottom: 80px;
  left: 64px;
  text-align: left;
  max-width: 640px;
}

/* Floating nav */
.hero-nav {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24px 48px;
}

/* Scroll indicator */
.scroll-indicator {
  position: absolute;
  bottom: 32px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: rgba(255, 255, 255, 0.6);
  font-size: 11px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  animation: bounce 2s ease-in-out infinite;
}

@keyframes bounce {
  0%, 100% { transform: translateX(-50%) translateY(0); }
  50%       { transform: translateX(-50%) translateY(6px); }
}
```

## Overlay 选择规则

| 背景类型 | Overlay |
|--------|---------|
| 暗色图片/视频 | `rgba(0,0,0,0.35~0.55)` |
| 亮色图片 | `rgba(0,0,0,0.5~0.65)` |
| 纯渐变色 | 不需要 overlay |
| 动效 canvas | `rgba(0,0,0,0.2)` 即可 |

**白色文字的最低对比度 overlay: `rgba(0,0,0,0.35)`**

## Rules

- 导航必须是 `position: absolute`（透明浮在顶部）
- 文字全白 `rgba(255,255,255,0.95)`
- 必须有 overlay 层（无论背景多暗）
- Scroll indicator 帮助用户知道有下文
- Headline `clamp(48px, 7vw, 88px)`
- 最多 1 个 CTA 按钮（全屏 hero 不需要两个选择）

## 变体

```
center-center:   align-items center + text-align center（经典）
bottom-left:     内容锚定左下角（电影海报感）
bottom-right:    内容锚定右下角（少见，有张力）
split-overlay:   左半暗 right 半亮，文字在左（高级感）
```

## Forbidden

- 没有 overlay 层直接把文字放在图片上
- 透明度 overlay 低于 0.3（文字不可读）
- 全屏 hero 内包含 feature 列表或价格
- 导航非透明（scroll 前 nav 不应该有背景色）
