# Style Skill: Glassmorphism

Inspired by: Variant community transparent overlay aesthetics, layered card designs

## Core Effect

```css
.glass-card {
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 16px;
}

/* Stronger glass */
.glass-card-strong {
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(40px) saturate(200%);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

/* Dark glass (on light backgrounds) */
.glass-card-dark {
  background: rgba(0, 0, 0, 0.25);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(0, 0, 0, 0.1);
}
```

## Background Requirements

Glass REQUIRES a rich background to show through. Use one of:

```css
/* Option A: Gradient blob background */
.glass-bg {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 50%, #f093fb 100%);
  position: relative;
  overflow: hidden;
}

/* Option B: Blurred image background */
.glass-bg-image {
  background-image: url('...');
  background-size: cover;
  filter: blur(0); /* image itself not blurred, card has blur */
}

/* Option C: Colorful blobs */
.blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.6;
}
```

## Rules

- Glass ONLY works on colorful/rich backgrounds — never on flat white/black
- `backdrop-filter: blur()` value: 16-40px depending on depth desired
- Border: always `rgba(255,255,255,0.1-0.2)` — the "rim light"
- Inner glow on top edge: `box-shadow: inset 0 1px 0 rgba(255,255,255,0.15)`
- Cards stack at slightly different blur intensities for depth
- Text on glass: always white or very light

## Component Example

```jsx
<div style={{ position: 'relative', minHeight: '100vh', background: 'linear-gradient(135deg, #667eea, #764ba2)' }}>
  {/* Decorative blobs */}
  <div style={{ position: 'absolute', width: 400, height: 400, borderRadius: '50%', background: 'rgba(240, 147, 251, 0.4)', filter: 'blur(100px)', top: -100, right: -100 }} />

  {/* Glass card */}
  <div style={{
    background: 'rgba(255,255,255,0.1)',
    backdropFilter: 'blur(20px)',
    WebkitBackdropFilter: 'blur(20px)',
    border: '1px solid rgba(255,255,255,0.15)',
    borderRadius: '16px',
    padding: '32px',
    boxShadow: 'inset 0 1px 0 rgba(255,255,255,0.2), 0 20px 60px rgba(0,0,0,0.2)',
  }}>
    <h2 style={{ color: 'white', fontWeight: 600 }}>Glass Card</h2>
  </div>
</div>
```

## Forbidden

- Glass on flat/neutral backgrounds (no effect)
- `backdrop-filter: blur()` < 8px (too subtle to see)
- Sharp corners on glass elements (minimum 12px radius)
- Dark text on glass cards
