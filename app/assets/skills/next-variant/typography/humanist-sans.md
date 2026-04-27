# Typography Skill: Humanist Sans

Inspired by: PostHog — IBM Plex Sans 的人文感，比 Inter 更有温度，比 Roboto 更有个性

## 设计哲学

几何 sans（Inter、DM Sans）感觉精确冷静；人文 sans（IBM Plex Sans、Lato）在字母细节里藏着手写痕迹，更有亲和力。适合「严肃但不冷漠」的开发工具、文档类、数据产品。

## Font Stack

```css
/* 主字体 */
font-family: "IBM Plex Sans", "IBM Plex Sans Var", system-ui, -apple-system, sans-serif;

/* 代码 / 数据 */
font-family: "IBM Plex Mono", "Source Code Pro", ui-monospace, monospace;

/* Fallback（未加载时不崩） */
font-family: system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
```

## Type Scale

```css
:root {
  /* Display */
  --text-hero:    clamp(40px, 6vw, 72px);
  --text-display: clamp(32px, 4.5vw, 52px);

  /* Headings */
  --text-h1: 36px;
  --text-h2: 28px;
  --text-h3: 22px;
  --text-h4: 18px;

  /* Body */
  --text-lg:   18px;
  --text-base: 16px;
  --text-sm:   14px;
  --text-xs:   13px;

  /* Label / UI */
  --text-label: 12px;
  --text-code:  14px;

  /* Line heights */
  --leading-display: 1.1;
  --leading-heading: 1.3;
  --leading-body:    1.65;
  --leading-ui:      1.4;

  /* Letter spacing — 核心差异点 */
  --tracking-display: -0.025em;   /* 标题更紧 */
  --tracking-heading: -0.015em;   /* PostHog 全局用 -0.015em */
  --tracking-body:    -0.01em;    /* 正文也微微收紧 */
  --tracking-label:    0.02em;    /* 小标签略松 */
  --tracking-code:     0em;       /* 代码不动 */
}
```

## Weight 使用规则

```
400 (Regular)   → 正文、次级内容、input placeholder
500 (Medium)    → 导航、表格 header、强调 body
600 (SemiBold)  → h3、h4、按钮文字、badge
700 (Bold)      → h1、h2、hero headline
```

不用 800/900 — Plex Sans 的 Bold 已经足够重，再粗反而失去人文质感。

## Rules

- **全局 `letter-spacing: -0.015em`** — 这是 PostHog 的签名 token，让整个页面更紧凑
- Hero headline: `letter-spacing: -0.025em`，`line-height: 1.1`
- 正文: `font-size: 16px`，`line-height: 1.65`，`color: var(--text-primary)`
- 代码片段：IBM Plex Mono，`font-size: 13-14px`，背景 `#F3F2EE`，圆角 `4px`
- Small caps 标签：`font-size: 12px`，`font-weight: 600`，`letter-spacing: 0.04em`，`text-transform: uppercase`
- 链接颜色 `#F54E00`（配合 `warm-cream-orange`），不带下划线，hover 时加下划线

## Component Examples

```jsx
// Hero headline
<h1 style={{
  fontFamily: '"IBM Plex Sans", system-ui, sans-serif',
  fontSize: 'clamp(40px, 6vw, 72px)',
  fontWeight: 700,
  letterSpacing: '-0.025em',
  lineHeight: 1.1,
  color: '#1E1F23',
}}>
  Dev tools for product engineers
</h1>

// Body paragraph
<p style={{
  fontFamily: '"IBM Plex Sans", system-ui, sans-serif',
  fontSize: '16px',
  fontWeight: 400,
  letterSpacing: '-0.01em',
  lineHeight: 1.65,
  color: '#4D4F46',
}}>
  当内容在这里
</p>

// Inline code
<code style={{
  fontFamily: '"IBM Plex Mono", ui-monospace, monospace',
  fontSize: '13px',
  background: '#F3F2EE',
  border: '1px solid #E0DFD9',
  borderRadius: '4px',
  padding: '2px 6px',
  color: '#4D4F46',
}}>
  npx @posthog/wizard
</code>

// Section label
<span style={{
  fontFamily: '"IBM Plex Sans", system-ui, sans-serif',
  fontSize: '12px',
  fontWeight: 600,
  letterSpacing: '0.04em',
  textTransform: 'uppercase',
  color: '#A0A193',
}}>
  Pricing
</span>
```

## Forbidden

- 几何 sans（Inter、DM Sans、Geist）作为主字体 — 会失去人文温度
- `letter-spacing: 0` 或正值用于 heading（会变宽松）
- `font-weight: 800` 或 `900`
- 正文 `font-size < 15px`（Plex Sans 小字号可读性不如 Inter）
- Serif 字体（和整体风格冲突）
