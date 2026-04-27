# Color Skill: Neon Tech

Inspired by: Variant community sci-fi aesthetics, Organic Intelligence Grid, futuristic dashboard designs

## Design Tokens

```css
:root {
  --bg-primary: #050508;
  --bg-secondary: #0a0a12;
  --bg-grid: rgba(255, 255, 255, 0.03);
  --bg-card: rgba(10, 255, 180, 0.04);
  --bg-card-hover: rgba(10, 255, 180, 0.08);

  --neon-green: #0aff9d;
  --neon-blue: #00d4ff;
  --neon-purple: #bf5af2;
  --neon-orange: #ff6b35;

  --text-primary: rgba(255, 255, 255, 0.92);
  --text-secondary: rgba(255, 255, 255, 0.55);
  --text-accent: #0aff9d;

  --border-neon: rgba(10, 255, 157, 0.25);
  --border-default: rgba(255, 255, 255, 0.08);

  --glow-green: 0 0 20px rgba(10, 255, 157, 0.3), 0 0 60px rgba(10, 255, 157, 0.1);
  --glow-blue: 0 0 20px rgba(0, 212, 255, 0.3), 0 0 60px rgba(0, 212, 255, 0.1);
}
```

## Rules

- Background near-pure black: `#050508`
- Background grid pattern via CSS: `background-image: linear-gradient(rgba(255,255,255,0.03) 1px, transparent 1px)`
- One primary neon color per design — NEVER mix more than 2 neon colors
- Neon glow via `box-shadow` or `text-shadow` only on key elements
- Cards have neon-tinted borders on hover
- Text uses `rgba(255,255,255,0.92)` — near white but not pure
- Accent/highlight text uses neon color directly
- Monospace font for data values and technical content

## Component Example

```jsx
// Neon tech card
<div style={{
  background: 'rgba(10, 255, 180, 0.04)',
  border: '1px solid rgba(10, 255, 157, 0.25)',
  borderRadius: '6px',
  padding: '20px',
  transition: 'box-shadow 0.2s',
}}>
  <div style={{
    color: '#0aff9d',
    fontFamily: 'monospace',
    fontSize: '11px',
    letterSpacing: '0.1em',
    textTransform: 'uppercase',
  }}>
    SYSTEM_STATUS
  </div>
  <div style={{
    color: 'rgba(255,255,255,0.92)',
    fontSize: '28px',
    fontFamily: 'monospace',
    marginTop: '8px',
    textShadow: '0 0 20px rgba(10,255,157,0.5)',
  }}>
    ACTIVE
  </div>
</div>
```

## Forbidden

- Warm colors as accents (orange/red/yellow unless intentional)
- Rounded corners > 8px — keep it geometric
- Serif fonts
- Box shadows without glow effect (use neon glow instead)
- Pastel or desaturated neons
