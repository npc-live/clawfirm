# Style Skill: Retro Warm

Inspired by: Variant community "Rumah Hangat" design, vintage-toned portfolio pages, warm textile product cards

## Visual Language

Retro warm = analog textures, warm color temperatures, vintage print sensibility. Feels like a worn magazine or artisan product catalog.

## Design Tokens

```css
:root {
  --paper: #f5ede0;          /* base — aged paper */
  --paper-light: #fdf6ee;
  --paper-dark: #e8d9c8;

  --ink-primary: #2c1810;    /* dark brown, aged ink */
  --ink-secondary: #6b4c3b;
  --ink-muted: #9c7b6e;

  --rust: #c4622d;
  --amber: #d4840a;
  --sage: #7a8c6e;
  --dusty-rose: #c0827a;

  --texture-noise: url("data:image/svg+xml,..."); /* optional grain overlay */

  --border: rgba(44, 24, 16, 0.15);
}
```

## Texture Techniques

```css
/* Paper grain effect */
.paper-texture::after {
  content: '';
  position: absolute;
  inset: 0;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)' opacity='0.04'/%3E%3C/svg%3E");
  pointer-events: none;
  opacity: 0.5;
}

/* Sepia image treatment */
.retro-image {
  filter: sepia(0.3) contrast(0.9) brightness(1.05);
}

/* Worn border */
.torn-edge {
  border-bottom: 2px solid var(--rust);
  padding-bottom: 8px;
  position: relative;
}
```

## Rules

- Background: always warm-toned paper (`#f5ede0`, `#fdf6ee`)
- NEVER pure white or cool-toned white
- Type: serif or slab-serif for headings, warm body text
- Accents: rust, amber, sage, dusty rose — earth palette only
- No sharp geometric shapes — curves and organic forms
- Borders are warm and slightly translucent
- Images get sepia or warm filter
- Spacing is generous — magazine feel

## Component Examples

```jsx
// Retro product card
<div style={{
  background: '#fdf6ee',
  border: '1px solid rgba(44, 24, 16, 0.12)',
  borderRadius: '4px',
  overflow: 'hidden',
  fontFamily: 'Georgia, serif',
}}>
  <div style={{ background: '#e8d9c8', aspectRatio: '3/2', filter: 'sepia(0.2)' }} />
  <div style={{ padding: '20px' }}>
    <div style={{ fontSize: '10px', letterSpacing: '0.12em', textTransform: 'uppercase', color: '#c4622d', fontFamily: 'Inter, sans-serif' }}>
      Handmade
    </div>
    <h3 style={{ fontSize: '20px', fontWeight: 600, color: '#2c1810', marginTop: '6px', lineHeight: 1.3 }}>
      Artisan Product Name
    </h3>
    <p style={{ fontSize: '14px', color: '#6b4c3b', lineHeight: 1.6, marginTop: '8px' }}>
      Crafted with care.
    </p>
    <div style={{ fontSize: '16px', color: '#c4622d', fontWeight: 600, marginTop: '12px' }}>
      $48.00
    </div>
  </div>
</div>
```

## Forbidden

- Cool gray backgrounds
- Blue or purple accents
- Neon or saturated colors
- Geometric/tech aesthetics
- Heavy drop shadows (soft warm shadows only: `0 2px 8px rgba(44,24,16,0.1)`)
