# Typography Skill: Inter Minimal

Inspired by: Variant community's own UI, Academic Supply, note-taking apps — the "invisible" type system

## Font Stack

```css
font-family: Inter, "Inter Fallback", system-ui, -apple-system, sans-serif;
```

## Type Scale

```css
:root {
  --text-xs:   11px;  /* labels, tags, captions */
  --text-sm:   13px;  /* secondary body, metadata */
  --text-base: 14px;  /* primary body text */
  --text-md:   15px;  /* slightly emphasized body */
  --text-lg:   18px;  /* subheadings */
  --text-xl:   22px;  /* section headings */
  --text-2xl:  28px;  /* page titles */
  --text-3xl:  36px;  /* hero headings */

  --leading-tight:  1.25;
  --leading-normal: 1.5;
  --leading-relaxed: 1.7;

  --tracking-tight: -0.02em;
  --tracking-normal: 0;
  --tracking-wide:  0.05em;
  --tracking-wider: 0.08em;
}
```

## Weight Usage

```
400 (regular)  → body text, labels, secondary content
500 (medium)   → nav items, table headers, emphasized body
600 (semibold) → headings, important metrics
700 (bold)     → hero headings ONLY, sparingly
```

## Rules

- NO font-weight 800, 900 — too heavy for Inter
- Heading tracking: `-0.02em` (slightly tight)
- Label/tag tracking: `+0.05em` to `+0.08em` (slightly loose)
- Body: `font-size: 14px`, `line-height: 1.5`, `font-weight: 400`
- Secondary text is always same font, reduced opacity — not a different font
- Uppercase labels: use letter-spacing `0.08em`, font-size never > 12px uppercase

## Component Examples

```jsx
// Page title
<h1 style={{ fontSize: '36px', fontWeight: 600, letterSpacing: '-0.02em', lineHeight: 1.2 }}>
  Dashboard
</h1>

// Body paragraph
<p style={{ fontSize: '14px', fontWeight: 400, lineHeight: 1.6, color: 'var(--text-secondary)' }}>
  Description text
</p>

// Small label (uppercase)
<span style={{ fontSize: '11px', fontWeight: 500, letterSpacing: '0.08em', textTransform: 'uppercase', color: 'var(--text-muted)' }}>
  Category
</span>
```

## Forbidden

- Serif fonts in any role
- Monospace fonts (unless code blocks)
- Font size below 11px
- Font weight above 700
- Line-height below 1.2 (even for headings)
