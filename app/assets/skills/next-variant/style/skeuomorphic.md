# Style Skill: Skeuomorphic

Inspired by: iOS 6 design, physical device mockups, calculator apps, music players, early 2010s UI design

## Visual Language

Skeuomorphic design mimics real-world materials and physics. Elements have depth, texture, and lighting that make them feel tangible and physical. Every button looks pressable, every knob looks turnable.

## Design Tokens

```css
:root {
  /* Multi-layer shadows for depth */
  --shadow-raised-sm: 0 1px 2px rgba(0,0,0,0.08), 0 2px 4px rgba(0,0,0,0.06);
  --shadow-raised: 0 2px 4px rgba(0,0,0,0.1), 0 8px 16px rgba(0,0,0,0.08);
  --shadow-raised-lg: 0 4px 8px rgba(0,0,0,0.12), 0 12px 24px rgba(0,0,0,0.1);
  --shadow-deep: 0 20px 40px rgba(0,0,0,0.15), 0 8px 16px rgba(0,0,0,0.1);

  /* Inset shadows for pressed/recessed elements */
  --shadow-inset-sm: inset 0 1px 2px rgba(0,0,0,0.08);
  --shadow-inset: inset 0 2px 4px rgba(0,0,0,0.1);
  --shadow-inset-deep: inset 0 4px 8px rgba(0,0,0,0.15);

  /* Highlights (for edges and reflections) */
  --highlight-top: inset 0 1px 2px rgba(255,255,255,0.8);
  --highlight-edge: inset 0 -1px 1px rgba(255,255,255,0.5);
  --highlight-subtle: inset 0 1px 1px rgba(255,255,255,0.3);

  /* Gradients - simulate lighting from above */
  --gradient-button: linear-gradient(135deg, #ffffff 0%, #f5f5f4 50%, #e5e7eb 100%);
  --gradient-button-pressed: linear-gradient(135deg, #e5e7eb 0%, #d1d5db 100%);
  --gradient-knob: linear-gradient(145deg, #ffffff 0%, #fafaf9 30%, #f5f5f4 70%, #e5e7eb 100%);
  --gradient-screen: linear-gradient(180deg, #2a2a2a 0%, #1a1a1a 50%, #0a0a0a 100%);
  --gradient-metal: linear-gradient(135deg, #e5e7eb 0%, #f5f5f4 25%, #e5e7eb 50%, #d1d5db 75%, #e5e7eb 100%);

  /* Borders - subtle but defines edges */
  --border-subtle: 1px solid rgba(0,0,0,0.08);
  --border-default: 1px solid rgba(0,0,0,0.12);
  --border-strong: 2px solid rgba(0,0,0,0.15);

  /* Border radius - smooth and physical */
  --radius-button: 50%;  /* circular buttons */
  --radius-knob: 50%;
  --radius-card: 16-24px;
  --radius-screen: 8-12px;
}
```

## Rules

- **Multiple shadows:** Use 2-3 shadow layers for depth (light, medium, dark)
- **Gradients everywhere:** Buttons, knobs, panels all use gradients to simulate lighting
- **Inset shadows:** Screens, text inputs, and pressed buttons use inset shadows
- **Top highlights:** Add `inset 0 1px 2px rgba(255,255,255,0.8)` to top edges
- **Circular elements:** Knobs and buttons are perfectly circular with centered indicators
- **Material consistency:** All elements in same material (plastic, metal, glass) share similar gradients
- **Hover states:** Slightly increase shadow depth, don't change color drastically
- **Active/pressed:** Invert shadow (inset), reduce highlights, darken gradient
- **Reflections:** Glass surfaces (screens) can have subtle gradient overlays
- **Textures:** Optionally add subtle noise or pattern backgrounds

## Component Examples

### Circular Button

```jsx
<button style={{
  width: '56px',
  height: '56px',
  border: '1px solid rgba(0,0,0,0.12)',
  borderRadius: '50%',
  background: 'linear-gradient(135deg, #ffffff 0%, #f5f5f4 50%, #e5e7eb 100%)',
  boxShadow: '0 2px 4px rgba(0,0,0,0.1), 0 8px 16px rgba(0,0,0,0.08), inset 0 1px 2px rgba(255,255,255,0.8)',
  cursor: 'pointer',
  transition: 'all 0.15s',
}}>
  ▶
</button>

/* Active state */
<button style={{
  background: 'linear-gradient(135deg, #e5e7eb 0%, #d1d5db 100%)',
  boxShadow: 'inset 0 2px 4px rgba(0,0,0,0.1), 0 1px 2px rgba(0,0,0,0.06)',
}}>
  ▶
</button>
```

### Volume Knob

```jsx
<div style={{
  width: '120px',
  height: '120px',
  borderRadius: '50%',
  background: 'linear-gradient(145deg, #ffffff 0%, #fafaf9 30%, #f5f5f4 70%, #e5e7eb 100%)',
  boxShadow: `
    0 4px 8px rgba(0,0,0,0.1),
    0 12px 24px rgba(0,0,0,0.08),
    inset 0 -2px 4px rgba(0,0,0,0.05),
    inset 0 2px 4px rgba(255,255,255,0.8)
  `,
  position: 'relative',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
}}>
  {/* Indicator line */}
  <div style={{
    position: 'absolute',
    top: '15px',
    width: '3px',
    height: '30px',
    background: '#1a1a1a',
    borderRadius: '2px',
  }} />

  {/* Center cap */}
  <div style={{
    width: '40px',
    height: '40px',
    background: '#ffffff',
    borderRadius: '50%',
    boxShadow: '0 2px 4px rgba(0,0,0,0.08), inset 0 1px 2px rgba(0,0,0,0.03)',
  }} />
</div>
```

### Screen / Display

```jsx
<div style={{
  background: 'linear-gradient(180deg, #2a2a2a 0%, #1a1a1a 50%, #0a0a0a 100%)',
  border: '2px solid #000',
  borderRadius: '12px',
  padding: '32px 24px',
  boxShadow: `
    inset 0 4px 8px rgba(0,0,0,0.3),
    inset 0 1px 2px rgba(0,0,0,0.5),
    0 2px 4px rgba(0,0,0,0.1)
  `,
  position: 'relative',
}}>
  {/* Optional: glass reflection overlay */}
  <div style={{
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    height: '40%',
    background: 'linear-gradient(180deg, rgba(255,255,255,0.08) 0%, transparent 100%)',
    borderRadius: '10px 10px 0 0',
    pointerEvents: 'none',
  }} />

  {/* Screen content */}
  <div style={{
    color: '#ffffff',
    fontFamily: 'monospace',
    fontSize: '48px',
  }}>
    00:12:10
  </div>
</div>
```

### Device Case / Panel

```jsx
<div style={{
  background: '#ffffff',
  border: '1px solid rgba(0,0,0,0.08)',
  borderRadius: '24px',
  padding: '20px',
  boxShadow: `
    0 20px 40px rgba(0,0,0,0.12),
    0 8px 16px rgba(0,0,0,0.08),
    inset 0 1px 2px rgba(255,255,255,0.6)
  `,
}}>
  {/* Device content */}
</div>
```

### Toggle Switch

```jsx
{/* Track */}
<div style={{
  width: '60px',
  height: '32px',
  background: 'linear-gradient(180deg, #d1d5db 0%, #e5e7eb 100%)',
  borderRadius: '16px',
  boxShadow: 'inset 0 2px 4px rgba(0,0,0,0.15)',
  position: 'relative',
  cursor: 'pointer',
}}>
  {/* Thumb */}
  <div style={{
    width: '28px',
    height: '28px',
    background: 'linear-gradient(135deg, #ffffff 0%, #f5f5f4 100%)',
    borderRadius: '50%',
    position: 'absolute',
    top: '2px',
    left: '2px',
    boxShadow: '0 2px 4px rgba(0,0,0,0.2), inset 0 1px 1px rgba(255,255,255,0.8)',
    transition: 'left 0.2s',
  }} />
</div>
```

## Forbidden

- Completely flat buttons (must have gradient + shadow)
- Single-layer shadows (always use 2+ layers)
- No highlights on top edges
- Pure black or pure white backgrounds (use gradients)
- Sharp corners on physical elements (buttons, knobs need radius)
- Inconsistent light direction (always top-down or top-left)
- Neon glows (use soft shadows instead)
- Transparent backgrounds on "solid" elements

## Material Guidelines

### Plastic (white/light)
```css
background: linear-gradient(135deg, #ffffff 0%, #f5f5f4 50%, #e5e7eb 100%);
box-shadow:
  0 2px 4px rgba(0,0,0,0.1),
  inset 0 1px 2px rgba(255,255,255,0.8);
```

### Glass (screens)
```css
background: linear-gradient(180deg, #2a2a2a 0%, #1a1a1a 100%);
box-shadow: inset 0 4px 8px rgba(0,0,0,0.3);
border: 2px solid #000;
```

### Metal (buttons, trim)
```css
background: linear-gradient(135deg,
  #e5e7eb 0%,
  #f5f5f4 25%,
  #e5e7eb 50%,
  #d1d5db 75%,
  #e5e7eb 100%
);
box-shadow:
  0 1px 2px rgba(0,0,0,0.1),
  inset 0 1px 1px rgba(255,255,255,0.5);
```

## Hover & Active States

### Hover
- Increase shadow depth by 20-30%
- Optionally brighten gradient by 5%
- Subtle transform: `translateY(-1px)`

### Active / Pressed
- Invert shadows (use inset)
- Darken gradient
- Remove top highlight
- Subtle transform: `translateY(1px)`

```jsx
// Example with states
.button {
  box-shadow: 0 2px 4px rgba(0,0,0,0.1), inset 0 1px 2px rgba(255,255,255,0.8);
  transition: all 0.15s;
}

.button:hover {
  box-shadow: 0 4px 8px rgba(0,0,0,0.12), inset 0 1px 2px rgba(255,255,255,0.8);
  transform: translateY(-1px);
}

.button:active {
  box-shadow: inset 0 2px 4px rgba(0,0,0,0.15);
  transform: translateY(1px);
}
```
