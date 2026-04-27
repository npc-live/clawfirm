# Style Skill: Brutalist

Inspired by: Variant community bold typography specimens, raw structural designs

## Visual Language

Brutalism in web design = raw structure, no decoration, typography as image, functional exposure of grid.

## Design Tokens

```css
:root {
  --bg: #ffffff;
  --fg: #000000;
  --accent: #ff0000;   /* ONE harsh accent, optional */

  --border: 2px solid #000000;
  --border-thick: 4px solid #000000;

  --radius: 0px;       /* NO rounded corners */
  --shadow: none;      /* NO shadows */
}
```

## Rules

- Background: `#ffffff` or `#000000` only — no off-whites, no grays
- Text: opposite of background — `#000000` on white, `#ffffff` on black
- Borders: `2-4px solid` — heavy, structural
- `border-radius: 0` everywhere — squares and rectangles only
- No gradients, no blur, no shadows
- Typography IS the decoration — size contrast is the only visual tool
- Layout uses CSS Grid with visible structure — columns feel like print grid
- One accent color maximum (red `#ff0000`, yellow `#ffff00`, or none)
- Hover states: invert colors (`filter: invert(1)`) or swap bg/fg

## Component Examples

```jsx
// Brutalist card
<div style={{
  border: '2px solid #000',
  padding: '24px',
  background: '#fff',
  position: 'relative',
}}>
  <div style={{ fontSize: '11px', fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase', borderBottom: '1px solid #000', paddingBottom: '8px', marginBottom: '16px' }}>
    PROJECT TYPE
  </div>
  <h2 style={{ fontSize: '32px', fontWeight: 900, lineHeight: 1.05, letterSpacing: '-0.02em' }}>
    PROJECT<br/>TITLE HERE
  </h2>
  <div style={{ position: 'absolute', top: '24px', right: '24px', fontSize: '72px', fontWeight: 900, opacity: 0.08, lineHeight: 1 }}>
    01
  </div>
</div>

// Brutalist button
<button style={{
  border: '2px solid #000',
  background: '#fff',
  color: '#000',
  padding: '12px 24px',
  fontSize: '13px',
  fontWeight: 700,
  letterSpacing: '0.08em',
  textTransform: 'uppercase',
  cursor: 'pointer',
  borderRadius: 0,
  transition: 'background 0.1s, color 0.1s',
}}>
  SUBMIT
</button>
```

## Forbidden

- Rounded corners (`border-radius > 0`)
- Gradients
- Shadows
- More than 2 colors (bg + fg + optional one accent)
- Decorative icons or illustrations
- Smooth animations (transitions max 0.1s, abrupt)
