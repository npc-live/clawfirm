# Color Skill: Warm Editorial

Inspired by: Variant community "Rumah Hangat" design, The Dormant Phase blog, textile manufacturing cards

## Design Tokens

```css
:root {
  --bg-primary: #fdf6ee;
  --bg-secondary: #faf0e4;
  --bg-card: #fff8f0;
  --bg-accent-block: #f5e6d3;

  --text-primary: #2c1810;
  --text-secondary: #6b4c3b;
  --text-muted: #9c7b6e;
  --text-caption: #b89080;

  --accent-rust: #c4622d;
  --accent-amber: #d4840a;
  --accent-sage: #7a8c6e;
  --accent-clay: #a05c3a;

  --border-default: rgba(44, 24, 16, 0.12);
  --border-warm: rgba(196, 98, 45, 0.2);
}
```

## Rules

- All backgrounds are warm-toned — cream, tan, off-white with yellow/orange undertones
- Never use cool grays or blue-tinted whites
- Primary text is dark brown `#2c1810`, feels like aged ink
- Accent colors: rust orange, amber, sage green, clay — earth palette only
- Borders use warm semi-transparent darks
- Images use warm filters or sepia tones
- Typography leans serif for body, slab-serif for headings
- Generous margins — content breathes in warmth

## Component Example

```jsx
// Warm editorial article card
<article style={{
  background: '#fff8f0',
  borderLeft: '3px solid #c4622d',
  padding: '24px 28px',
  borderRadius: '4px',
}}>
  <span style={{ color: '#c4622d', fontSize: '11px', fontWeight: 600, letterSpacing: '0.08em', textTransform: 'uppercase' }}>
    Feature
  </span>
  <h2 style={{ color: '#2c1810', fontFamily: 'Georgia, serif', fontSize: '22px', marginTop: '8px' }}>
    Article Title
  </h2>
  <p style={{ color: '#6b4c3b', fontSize: '14px', lineHeight: '1.7' }}>
    Excerpt text here.
  </p>
</article>
```

## Forbidden

- Cool-toned colors (blue, purple, cool gray)
- Pure white or pure black
- Neon or saturated accent colors
- Dark mode variations
- Geometric/tech aesthetics
