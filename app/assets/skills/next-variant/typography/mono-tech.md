# Typography Skill: Mono Tech

Inspired by: Variant community futuristic dashboards, Organic Intelligence Grid, terminal-aesthetic designs

## Font Stack

```css
/* Primary — all UI */
font-family: "JetBrains Mono", "Fira Code", "Cascadia Code", "Courier New", monospace;

/* Fallback for non-code display */
font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
```

## Type Scale

```css
:root {
  --text-xs:   10px;  /* metadata, IDs */
  --text-sm:   11px;  /* labels, captions */
  --text-base: 13px;  /* primary body in mono */
  --text-lg:   16px;  /* subheadings */
  --text-xl:   20px;  /* section headings */
  --text-hero: 32px;  /* hero text */

  --leading-code:   1.6;
  --leading-dense:  1.3;

  --tracking-label: 0.1em;   /* uppercase labels breathe more */
  --tracking-data:  0.05em;  /* numeric data */
}
```

## Rules

- Monospace for EVERYTHING — no font mixing except icons
- All text in uppercase OR lowercase — no mixed case headings
- Numbers: always tabular-nums (`font-variant-numeric: tabular-nums`)
- Data values: right-aligned in tables
- Labels: uppercase + letter-spacing `0.1em`
- Line height: 1.6 minimum — monospace needs more air
- No italics — use color or opacity for emphasis instead
- Typical max line length: 60-70ch

## Component Examples

```jsx
// Stat display
<div>
  <div style={{
    fontFamily: 'ui-monospace, monospace',
    fontSize: '10px',
    letterSpacing: '0.1em',
    textTransform: 'uppercase',
    color: 'var(--text-muted)',
  }}>
    REQUESTS/SEC
  </div>
  <div style={{
    fontFamily: 'ui-monospace, monospace',
    fontSize: '32px',
    fontVariantNumeric: 'tabular-nums',
    color: 'var(--neon-green)',
    lineHeight: 1.2,
  }}>
    1,247
  </div>
</div>

// Status badge
<span style={{
  fontFamily: 'ui-monospace, monospace',
  fontSize: '10px',
  letterSpacing: '0.08em',
  textTransform: 'uppercase',
  padding: '2px 8px',
  border: '1px solid currentColor',
}}>
  ACTIVE
</span>
```

## Forbidden

- Serif or humanist sans fonts
- Mixed case headings
- Italic text
- Font sizes above 32px (feels wrong in mono context)
- Justified text alignment
