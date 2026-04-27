# Color Skill: Dark Moody

Inspired by: Variant community dark-first gallery aesthetic, Organic Intelligence Grid, XHS Tech Assets

## Design Tokens

```css
:root {
  --bg-primary: rgb(34, 34, 34);
  --bg-secondary: rgb(30, 30, 28);
  --bg-card: rgba(255, 255, 255, 0.05);
  --bg-elevated: rgba(68, 68, 68, 0.65);

  --text-primary: rgb(240, 237, 229);
  --text-secondary: rgba(240, 237, 229, 0.65);
  --text-muted: rgba(255, 255, 255, 0.5);
  --text-ghost: rgba(255, 255, 255, 0.3);

  --border-default: rgba(255, 255, 255, 0.1);
  --border-subtle: rgba(255, 255, 255, 0.05);

  --accent: #2688f9;
  --accent-muted: rgba(38, 136, 249, 0.3);
}
```

## Rules

- Background is NEVER pure black — use `rgb(34,34,34)` or `rgb(30,30,28)` (warm dark gray)
- Text is NEVER pure white — use `rgb(240, 237, 229)` (warm off-white)
- Depth is created via transparency stacking, NOT shadows
- Cards use `rgba(255,255,255,0.05)` — barely visible white overlay
- Borders at `rgba(255,255,255,0.1)` — extremely subtle
- Interactive elements use `rgba(68,68,68,0.65)` for button backgrounds
- One blue accent only: `#2688f9` for calls-to-action and highlights
- No gradients on backgrounds — depth via alpha layers

## Forbidden

- `background: black` or `background: #000`
- `color: white` or `color: #fff`
- Box shadows for depth
- Multiple accent colors
- Saturated color splashes
