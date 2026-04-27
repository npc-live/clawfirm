# Layout Skill: Hero — Type Only

纯排版 hero。没有截图、没有插图，文字本身就是视觉。适合 agency、设计工作室、创意工具。

## 结构

```
┌──────────────────────────────────────────────────────────┐
│  nav (minimal, one line)                                 │
├──────────────────────────────────────────────────────────┤
│                                                          │
│                                                          │
│  MASSIVE                                                 │
│  HEADLINE TEXT                                           │
│  THAT BLEEDS ——————————————————                          │
│  OFF EDGE                                                │
│                                                          │
│              ← sometimes right-aligned                  │
│                                                          │
│  —————————————————  short sub  ————————  [CTA]          │
│                                                          │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

## CSS

```css
.hero-type {
  min-height: 85vh;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;       /* text anchors to bottom */
  padding: 0 48px 64px;
  overflow: hidden;
  position: relative;
}

.hero-type-headline {
  font-size: clamp(72px, 11vw, 140px);
  font-weight: 800;
  letter-spacing: -0.04em;
  line-height: 0.95;
  margin: 0;
  /* Allow text to bleed off right edge intentionally */
  white-space: nowrap;            /* single line bleeds off */
}

/* Multi-line variant */
.hero-type-headline.wrap {
  white-space: normal;
  max-width: 100%;
}

.hero-type-bottom-bar {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-top: 40px;
  padding-top: 20px;
  border-top: 1px solid var(--border-default);
  gap: 24px;
}

.hero-type-sub {
  font-size: 15px;
  color: var(--text-secondary);
  max-width: 360px;
  line-height: 1.5;
}

/* Oversized background character (decorative) */
.hero-type-bg-number {
  position: absolute;
  font-size: 40vw;
  font-weight: 900;
  line-height: 1;
  right: -5vw;
  top: 50%;
  transform: translateY(-50%);
  opacity: 0.04;
  pointer-events: none;
  user-select: none;
  color: currentColor;
}
```

## Rules

- Headline 锚定底部（`justify-content: flex-end`）— 感觉从页面长出来
- 允许文字故意溢出右边缘（`overflow: hidden` 裁掉）— 表达无限延伸
- Sub + CTA 放在底部分隔线下方，与 headline 保持层次对比
- 字号 `clamp(72px, 11vw, 140px)` — 在移动端也依然大
- 可以加透明度极低的超大背景字符（数字/字母）作为纹理
- 导航极简：logo + 最多 3 个链接

## 对齐变体

```
左对齐:  默认，headline 从左侧开始蔓延
右对齐:  text-align: right，headline 从右侧开始蔓延（少见，冲击力强）
交叉:    第一行左对齐，第二行右对齐（用 display:block + text-align 切换）
```

## Forbidden

- 任何产品截图或插图
- 超过 1 句的 subheading
- 居中对齐（type-only hero 依赖不对称张力）
- 行高 > 1.0（display 文字需要紧凑）
