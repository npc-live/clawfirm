# Typography Skill: Swiss Grotesque

Source: lambda.ai — Suisse Intl + Suisse Intl Mono，瑞士国际主义字体系统

## 设计哲学

瑞士国际主义（Swiss International Style）：字体作为纯粹的信息载体，没有情感装饰。Suisse Intl 是 Helvetica Neue 的现代重绘——更中性、更精准、更屏幕友好。大字号用极紧的字距，让排版有工程精密感。

## Font Stack

```css
/* 主字体 — Suisse Intl（收费）/ Helvetica Neue（系统）/ Arial */
font-family: "Suisse Intl", "Neue Haas Grotesk", "Helvetica Neue",
             Helvetica, Arial, sans-serif;

/* 等宽 — Suisse Intl Mono / 系统 mono */
font-family: "Suisse Intl Mono", "Courier New", ui-monospace, monospace;

/* 装饰性 Pixel 字体（Lambda 专属，可选）*/
font-family: "apkarchivr21", monospace;
```

> 没有 Suisse Intl 时：`"Helvetica Neue", Arial` 效果极接近。

## Type Scale（直接来自源码）

```css
:root {
  --text-2xs:  0.6875rem;   /* 11px */
  --text-xs:   0.75rem;     /* 12px */
  --text-sm:   0.875rem;    /* 14px */
  --text-base: 1rem;        /* 16px */
  --text-lg:   1.125rem;    /* 18px */
  --text-xl:   1.5rem;      /* 24px */
  --text-2xl:  2.15rem;     /* ~34px */
  --text-3xl:  3rem;        /* 48px */
  --text-4xl:  3.75rem;     /* 60px */
  --text-5xl:  4.5rem;      /* 72px */
  --text-6xl:  6rem;        /* 96px */
  --text-7xl:  7.315rem;    /* ~117px — hero 超大字 */

  --leading-none:    100%;
  --leading-tight:   110%;   /* 大标题用 */
  --leading-snug:    120%;
  --leading-normal:  130%;
  --leading-relaxed: 150%;
  --leading-loose:   160%;   /* 正文用 */

  --tracking-tighter: -0.02em;   /* hero 标题 */
  --tracking-tight:   -0.01em;   /* 普通标题 */
  --tracking-normal:   0;
  --tracking-wide:     0.01em;
  --tracking-wider:    0.02em;
  --tracking-widest:   0.05em;   /* 全大写 label */
}
```

## Weight 使用

```
300 (light)    → 数据大数字、超大 hero 场景
400 (regular)  → 正文、导航
600 (semibold) → 按钮、标签、强调
700 (bold)     → 标题
```

## Rules

- Hero 标题：`--text-6xl` ~ `--text-7xl`，`line-height: 110%`，`letter-spacing: -0.02em`
- 段落正文：`--text-base` (16px)，`line-height: 160%`，`letter-spacing: 0`
- 按钮/标签：`--text-sm` (14px)，`font-weight: 600`，`letter-spacing: 0.02em`
- Section eyebrow（小标签）：全大写，`--text-sm`，`letter-spacing: 0.05em`，`font-weight: 600`
- 等宽内容（命令行/代码）：切换到 Suisse Intl Mono 或 ui-monospace

## Component Examples

```jsx
// Hero headline
<h1 style={{
  fontFamily: '"Suisse Intl", "Helvetica Neue", Arial, sans-serif',
  fontSize: 'clamp(3rem, 8vw, 7.315rem)',
  fontWeight: 700,
  letterSpacing: '-0.02em',
  lineHeight: '110%',
  color: '#e7e6d9',
}}>
  The Superintelligence Cloud
</h1>

// Section eyebrow
<span style={{
  fontFamily: '"Suisse Intl", "Helvetica Neue", Arial, sans-serif',
  fontSize: '14px',
  fontWeight: 600,
  letterSpacing: '0.05em',
  textTransform: 'uppercase',
  color: '#95948c',
}}>
  Infrastructure
</span>

// Terminal / 命令行文字
<code style={{
  fontFamily: '"Suisse Intl Mono", "Courier New", monospace',
  fontSize: '14px',
  color: '#99f599',    /* 绿色终端字 */
  background: '#252525',
  padding: '4px 8px',
  borderRadius: '4px',
}}>
  lambda instance create --gpu h200
</code>

// 数据大数字（light weight）
<div style={{
  fontFamily: '"Suisse Intl", "Helvetica Neue", Arial, sans-serif',
  fontSize: 'clamp(3.75rem, 6vw, 6rem)',
  fontWeight: 300,
  letterSpacing: '-0.03em',
  lineHeight: '100%',
  fontVariantNumeric: 'tabular-nums',
}}>
  512
</div>
```

## Forbidden

- 衬线字体（serif 与瑞士风格绝对冲突）
- `font-weight: 800` 或 `900`（Suisse Intl 不提供，会 fallback 到 700）
- 正文 `letter-spacing` 负值（只有标题才收紧）
- `line-height < 110%`（即使最大字号也需要这个下限）
