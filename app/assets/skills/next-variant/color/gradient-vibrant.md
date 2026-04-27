# Color Skill: Gradient Vibrant

Inspired by: Variant community gradient-heavy designs, colorful landing pages, HELLO typography specimen

## Design Tokens

```css
:root {
  --bg-primary: #0a0a0f;
  --bg-secondary: #0f0f1a;

  --gradient-hero: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  --gradient-accent: linear-gradient(90deg, #f093fb 0%, #f5576c 100%);
  --gradient-warm: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
  --gradient-cool: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  --gradient-earth: linear-gradient(135deg, #f77062 0%, #fe5196 100%);

  --text-primary: #ffffff;
  --text-secondary: rgba(255, 255, 255, 0.8);
  --text-on-gradient: #ffffff;

  --border-glow: rgba(102, 126, 234, 0.4);
}
```

## Rules

- Hero sections MUST use a gradient — never flat backgrounds
- Gradient text via `background-clip: text; -webkit-text-fill-color: transparent`
- Cards can use subtle gradient borders: `border: 1px solid rgba(255,255,255,0.15)`
- Gradient overlays on images at 40-60% opacity
- Buttons use gradient backgrounds, white text
- Max 2 gradient directions per page — horizontal + diagonal only
- Dark base background so gradients pop

## Component Example

```jsx
// Gradient hero heading
<h1 style={{
  background: 'linear-gradient(135deg, #667eea 0%, #f093fb 100%)',
  WebkitBackgroundClip: 'text',
  WebkitTextFillColor: 'transparent',
  backgroundClip: 'text',
}}>
  Your heading here
</h1>

// Gradient CTA button
<button style={{
  background: 'linear-gradient(90deg, #667eea 0%, #764ba2 100%)',
  border: 'none',
  borderRadius: '8px',
  color: 'white',
  padding: '12px 24px',
}}>
  Get Started
</button>
```

## Forbidden

- Flat single-color backgrounds on hero sections
- Muted/desaturated gradients
- More than 3 different gradient palettes per page
- Gradient text on gradient backgrounds (unreadable)
