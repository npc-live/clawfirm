# Typography Skill: SF Pro (Apple System)

Inspired by: macOS HIG — San Francisco 字体，精准字距，系统级排版规范

## 设计哲学

SF Pro 是专为屏幕可读性优化的字体：小尺寸（< 20px）自动切换到 SF Pro Text（更宽松），大尺寸（≥ 20px）切换到 SF Pro Display（更紧凑）。这个行为通过 `-apple-system` 字体栈自动实现。

## Font Stack

```css
/* 正文 UI */
font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text",
             "Helvetica Neue", Arial, sans-serif;

/* Display / 大标题（手动指定时） */
font-family: -apple-system, BlinkMacSystemFont, "SF Pro Display",
             "Helvetica Neue", Arial, sans-serif;

/* 数字 / 代码 */
font-family: "SF Mono", ui-monospace, "Cascadia Code", monospace;
```

`-apple-system` 在 macOS/iOS 上自动映射到 SF Pro，Windows 上 fallback 到 Segoe UI（Helvetica Neue → Arial）。

## Apple HIG 字号体系

```css
:root {
  /* macOS 标准控件字号 */
  --text-large-title:   26px;   /* 大标题，少用 */
  --text-title-1:       22px;
  --text-title-2:       17px;   /* 窗口标题栏 */
  --text-title-3:       15px;
  --text-headline:      13px;   /* font-weight: 600 */
  --text-body:          13px;   /* macOS 主要正文字号 */
  --text-callout:       12px;
  --text-subhead:       11px;
  --text-footnote:      10px;
  --text-caption:        9px;

  /* Line heights */
  --leading-large-title: 32px;
  --leading-title-1:     26px;
  --leading-body:        18px;   /* 13px body → 18px lh */
  --leading-caption:     13px;

  /* Letter spacing（Apple 标准） */
  --tracking-large-title: 0.37px;
  --tracking-title-1:     0.35px;
  --tracking-title-2:     -0.43px;  /* 负值！大字号收紧 */
  --tracking-body:        -0.08px;
  --tracking-caption:      0.07px;
}
```

## Weight 使用

```
ultraLight (100) → 装饰性大数字
thin       (200) → 少用
light      (300) → 次要信息
regular    (400) → 正文、body
medium     (500) → 强调正文
semibold   (600) → headline、按钮、标签
bold       (700) → 标题
heavy      (800) → 大数字 metric
black      (900) → 极少用
```

macOS UI 控件主要用 **400（regular）和 600（semibold）**，其余 weight 极少见。

## Rules

- 正文字号：**13px**（macOS 标准），不是 14px 或 16px
- `line-height`：正文 **18px**（即 13/18），不是比例值
- 大于 20px 的标题用负 `letter-spacing`（-0.3px 到 -0.5px）
- 小于 13px 的文字用正 `letter-spacing`（+0.07px）
- 按钮文字：`font-size: 13px`，`font-weight: 600`
- 菜单/下拉项：`font-size: 13px`，`font-weight: 400`
- 窗口标题：`font-size: 13px`，`font-weight: 600`，居中
- 侧边栏 section header：`font-size: 11px`，`font-weight: 600`，大写，`letter-spacing: 0.06em`

## Component Examples

```jsx
// 窗口标题
<div style={{
  fontFamily: '-apple-system, BlinkMacSystemFont, sans-serif',
  fontSize: '13px',
  fontWeight: 600,
  letterSpacing: '-0.08px',
  color: 'rgba(0,0,0,0.85)',
  textAlign: 'center',
}}>
  Settings
</div>

// 侧边栏 section header
<div style={{
  fontFamily: '-apple-system, BlinkMacSystemFont, sans-serif',
  fontSize: '11px',
  fontWeight: 600,
  letterSpacing: '0.06em',
  textTransform: 'uppercase',
  color: 'rgba(0,0,0,0.35)',
  padding: '12px 16px 4px',
}}>
  Favorites
</div>

// 正文
<p style={{
  fontFamily: '-apple-system, BlinkMacSystemFont, sans-serif',
  fontSize: '13px',
  fontWeight: 400,
  lineHeight: '18px',
  letterSpacing: '-0.08px',
  color: 'rgba(0,0,0,0.85)',
}}>
  正文内容
</p>

// 大 metric 数字
<div style={{
  fontFamily: '-apple-system, BlinkMacSystemFont, sans-serif',
  fontSize: '48px',
  fontWeight: 200,
  letterSpacing: '-1px',
  fontVariantNumeric: 'tabular-nums',
  color: 'rgba(0,0,0,0.85)',
}}>
  12,847
</div>
```

## Forbidden

- `font-size: 16px` 用于 macOS app 正文（Web 标准，不是系统标准）
- `line-height: 1.5` 比例值（用绝对像素值 `18px` 保持和系统一致）
- 非系统字体（Inter、Roboto 等）
- `font-weight: 700` 用于正文（macOS 里 bold 只给标题）
- `letter-spacing: 0` 用于大号标题（必须加负值）
