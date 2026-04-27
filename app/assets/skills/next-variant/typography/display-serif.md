# Typography Skill: Display Serif

Inspired by: Variant community editorial designs, "The Dormant Phase", portfolio sites with strong typographic identity

## Font Stack

```css
/* Headings */
font-family: "Playfair Display", "Georgia", "Times New Roman", serif;

/* Body */
font-family: "Lora", Georgia, serif;

/* Accent/UI labels */
font-family: Inter, system-ui, sans-serif;
```

## Type Scale

```css
:root {
  --display-hero: clamp(48px, 8vw, 96px);  /* hero display */
  --display-xl:   clamp(36px, 5vw, 64px);  /* section hero */
  --heading-lg:   32px;
  --heading-md:   24px;
  --heading-sm:   20px;
  --body-lg:      18px;  /* article body */
  --body-base:    16px;
  --caption:      13px;

  --leading-display: 1.05;   /* very tight for display headings */
  --leading-heading: 1.25;
  --leading-body:    1.75;   /* generous for reading */
}
```

## Weight & Style Usage

```
Display headings: font-weight 700-900, italic for emphasis
Section headings: font-weight 600-700
Body text:        font-weight 400
Pull quotes:      font-weight 400, font-style italic
Captions:         Inter, font-size 12px, tracking +0.04em
```

## Rules

- Hero display text: `font-size: clamp()` responsive, tracking `-0.03em`
- Body text minimum 18px for comfortable reading in article context
- Serif body ONLY in article/reading contexts — not UI controls
- Mix serif headings + sans-serif labels freely (high contrast pairing)
- Drop caps on first paragraph of articles
- Italic for pull quotes and emphasis — NOT bold
- Generous line-height on body: 1.75

## Component Examples

```jsx
// Hero display heading
<h1 style={{
  fontFamily: '"Playfair Display", Georgia, serif',
  fontSize: 'clamp(48px, 8vw, 96px)',
  fontWeight: 700,
  fontStyle: 'italic',
  letterSpacing: '-0.03em',
  lineHeight: 1.05,
}}>
  The Story Begins
</h1>

// Article body
<p style={{
  fontFamily: '"Lora", Georgia, serif',
  fontSize: '18px',
  lineHeight: 1.75,
  fontWeight: 400,
}}>
  Body text reads beautifully at this size.
</p>

// UI label on serif page
<span style={{
  fontFamily: 'Inter, system-ui, sans-serif',
  fontSize: '11px',
  letterSpacing: '0.08em',
  textTransform: 'uppercase',
}}>
  Published
</span>
```

## Forbidden

- Sans-serif for article body text
- Serif for navigation or interactive UI elements
- Tight line-height on body text (min 1.6)
- Heavy bold weights on body text
