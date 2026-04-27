# Color Skill: Clean Minimal

Inspired by: Variant community note-taking apps, Academic Supply shop, typography-first designs

## Design Tokens

```css
:root {
  --bg-primary: #ffffff;
  --bg-secondary: #fafaf9;
  --bg-tertiary: #f5f5f4;
  --bg-card: #ffffff;
  --bg-hover: #f5f5f4;

  --text-primary: #1a1a1a;
  --text-secondary: #6b7280;
  --text-muted: #9ca3af;
  --text-placeholder: #d1d5db;

  --border-default: #e5e7eb;
  --border-subtle: #f3f4f6;
  --border-strong: #d1d5db;

  --accent: #111827;
  --accent-secondary: #374151;
}
```

## Rules

- Backgrounds use off-white, NEVER pure `#ffffff` on `<body>` — use `#fafaf9`
- Black text is `#1a1a1a`, not `#000000`
- Accent color is near-black `#111827` — no bright colors
- Borders are extremely light: `#e5e7eb`
- Shadows: `0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)`
- Card backgrounds pure white against `#fafaf9` body
- No color splashes — monochromatic grayscale system

## Component Example

```jsx
// Clean minimal card
<div style={{
  background: '#ffffff',
  border: '1px solid #e5e7eb',
  borderRadius: '8px',
  padding: '24px',
  boxShadow: '0 1px 3px rgba(0,0,0,0.06)',
}}>
  <h3 style={{ color: '#1a1a1a', fontWeight: 500, fontSize: '15px' }}>Title</h3>
  <p style={{ color: '#6b7280', fontSize: '13px', marginTop: '4px' }}>Description</p>
</div>
```

## Forbidden

- Bright accent colors (red, blue, green)
- Dark or colored backgrounds
- Heavy shadows
- Decorative borders or outlines
- Rounded corners > 12px
