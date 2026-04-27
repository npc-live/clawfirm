# Style Skill: Playful

Inspired by: Variant community bright typography specimens, bold color block designs, app icons and mobile UI

## Visual Language

Playful = unapologetic color, friendly rounded shapes, bouncy spatial rhythm. Feels like a well-designed consumer app or creative tool landing page.

## Design Tokens

```css
:root {
  --bg: #fafafa;

  /* Bright palette — use 2-3 of these per design */
  --coral:    #ff6b6b;
  --yellow:   #ffd93d;
  --mint:     #6bcb77;
  --sky:      #4d96ff;
  --lavender: #c77dff;
  --peach:    #ff9a3c;

  --dark:     #1a1a2e;    /* near-black for text */
  --muted:    #555577;

  --radius-xs: 8px;
  --radius-sm: 12px;
  --radius-md: 20px;
  --radius-lg: 32px;
  --radius-pill: 999px;

  --shadow-playful: 4px 4px 0px currentColor;  /* flat offset shadow */
  --shadow-soft: 0 8px 32px rgba(0,0,0,0.1);
}
```

## Rules

- Pick 2-3 colors from the bright palette — use them consistently as block fills
- `border-radius` is generous: minimum 12px, often 20-32px on cards
- The "flat offset shadow" (`4px 4px 0px #000`) is the signature decorative effect
- Hover: elements move slightly — `transform: translate(-2px, -2px)` with shadow growing
- Icons: filled, rounded style (no outline icons)
- Headings: rounded sans-serif, weight 700-800
- Backgrounds: color blocks, NOT gradients — solid fills that tile or section
- White space is generous — elements don't crowd each other

## Component Examples

```jsx
// Playful feature card with offset shadow
<div style={{
  background: '#ffd93d',
  border: '2px solid #1a1a2e',
  borderRadius: '20px',
  padding: '28px',
  boxShadow: '4px 4px 0px #1a1a2e',
  transition: 'transform 0.1s, box-shadow 0.1s',
  cursor: 'pointer',
}}>
  <div style={{ fontSize: '32px', marginBottom: '12px' }}>✨</div>
  <h3 style={{ fontSize: '18px', fontWeight: 700, color: '#1a1a2e', lineHeight: 1.2 }}>
    Feature Name
  </h3>
  <p style={{ fontSize: '14px', color: '#555577', marginTop: '8px', lineHeight: 1.5 }}>
    Short description of this awesome thing.
  </p>
</div>

// Playful pill button
<button style={{
  background: '#ff6b6b',
  color: 'white',
  border: '2px solid #1a1a2e',
  borderRadius: '999px',
  padding: '12px 28px',
  fontSize: '15px',
  fontWeight: 700,
  boxShadow: '3px 3px 0px #1a1a2e',
  cursor: 'pointer',
  transition: 'transform 0.1s, box-shadow 0.1s',
}}>
  Get Started →
</button>

// Color block section header
<div style={{ background: '#4d96ff', padding: '64px 24px', borderRadius: '24px', textAlign: 'center' }}>
  <h2 style={{ color: 'white', fontSize: '36px', fontWeight: 800, letterSpacing: '-0.02em' }}>
    Section Title
  </h2>
</div>
```

## Forbidden

- Muted, desaturated color palette
- Small `border-radius` (< 8px)
- Dark/moody backgrounds as primary surface
- Thin font weights (< 500 on headings)
- Formal/corporate layout patterns
- More than 4 different brand colors at once
