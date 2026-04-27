# Typography Skill: Big Display

Inspired by: Variant community HELLO I'M ELIM PAN specimen, bold landing pages, creative portfolio headers

## Font Stack

```css
/* Choose ONE of these based on desired personality */

/* Geometric/Modern */
font-family: "Neue Haas Grotesk", "Helvetica Neue", "Arial", sans-serif;

/* Grotesque/Bold */
font-family: "Space Grotesk", "DM Sans", system-ui, sans-serif;

/* Variable/Expressive */
font-family: "Cabinet Grotesk", "Satoshi", "Inter", sans-serif;
```

## Type Scale

```css
:root {
  --display-massive: clamp(80px, 15vw, 180px);  /* full-screen type */
  --display-hero:    clamp(56px, 9vw, 120px);
  --display-lg:      clamp(40px, 6vw, 80px);
  --body:            16px;
  --caption:         13px;

  --leading-display: 0.9;   /* tighter than 1 — letters almost touch */
  --leading-hero:    1.0;
  --leading-body:    1.5;
}
```

## Rules

- Hero text should be ENORMOUS — `clamp()` from 80px to 180px
- Line-height on display: 0.9 to 1.0 — aggressive tightness
- Tracking on display: `-0.04em` to `-0.06em` (very tight)
- Contrast: giant text + minimal small body text
- Text can overflow/bleed off screen — intentional
- Single focused message per screen — no competing headings
- Body text is secondary — maybe only 1-2 sentences near the display
- Color: often black/white high contrast, or bold single color

## Component Examples

```jsx
// Full-screen hero
<section style={{ position: 'relative', height: '100vh', overflow: 'hidden', background: '#000' }}>
  <h1 style={{
    fontFamily: '"Space Grotesk", system-ui, sans-serif',
    fontSize: 'clamp(80px, 14vw, 160px)',
    fontWeight: 800,
    letterSpacing: '-0.04em',
    lineHeight: 0.92,
    color: '#fff',
    position: 'absolute',
    bottom: '10%',
    left: '5%',
    right: '5%',
  }}>
    DESIGN<br/>WITHOUT<br/>LIMITS
  </h1>
</section>

// Oversized section number
<div style={{
  fontFamily: '"Space Grotesk", sans-serif',
  fontSize: 'clamp(120px, 20vw, 200px)',
  fontWeight: 800,
  lineHeight: 1,
  color: 'rgba(0,0,0,0.06)',
  position: 'absolute',
  right: '-0.05em',
  top: '-0.1em',
  userSelect: 'none',
}}>
  01
</div>
```

## Forbidden

- Multiple competing large headings on same screen
- Serif fonts (this is a grotesque/sans skill)
- Light font weights (< 600) on display text
- Justified or centered display text > 3 lines
- Body text font size > 18px (let the display be the hero)
