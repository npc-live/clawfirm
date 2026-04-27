# Style Skill: Flat Clean

Inspired by: Variant community Academic Supply, product cards, e-commerce UI with minimal chrome

## Visual Language

Flat clean = no depth cues except spacing and color. Every element reads at the same "elevation". Hierarchy via size and color, not shadow or layering.

## Design Tokens

```css
:root {
  --surface-0: #f8f9fa;   /* page background */
  --surface-1: #ffffff;   /* card / panel */
  --surface-2: #f3f4f6;   /* input, secondary surface */
  --surface-inverse: #111827;  /* dark surface */

  --border: #e5e7eb;
  --border-strong: #d1d5db;

  /* Accent — choose ONE */
  --accent-green: #059669;
  --accent-blue: #2563eb;
  --accent-purple: #7c3aed;

  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 16px;

  /* No shadows — flat only */
}
```

## Rules

- Zero `box-shadow` — depth via border and background only
- Hover: change `background` color, not add shadow
- Active/selected: solid background fill, not border change
- `border-radius`: 6-16px depending on component size
- Buttons: solid fill, flat — `background: var(--accent)`, white text
- Status colors: use background tints not colored borders
  - Success: `background: #f0fdf4`, `color: #059669`
  - Error: `background: #fef2f2`, `color: #dc2626`
  - Warning: `background: #fffbeb`, `color: #d97706`

## Component Examples

```jsx
// Flat product card
<div style={{
  background: '#fff',
  border: '1px solid #e5e7eb',
  borderRadius: '10px',
  overflow: 'hidden',
}}>
  <div style={{ background: '#f3f4f6', aspectRatio: '4/3' }} /> {/* image placeholder */}
  <div style={{ padding: '16px' }}>
    <div style={{ fontSize: '11px', color: '#6b7280', fontWeight: 500, marginBottom: '4px' }}>CATEGORY</div>
    <div style={{ fontSize: '15px', color: '#111827', fontWeight: 500 }}>Product Name</div>
    <div style={{ fontSize: '14px', color: '#059669', fontWeight: 600, marginTop: '8px' }}>$29.00</div>
  </div>
</div>

// Flat primary button
<button style={{
  background: '#2563eb',
  color: '#fff',
  border: 'none',
  borderRadius: '8px',
  padding: '10px 20px',
  fontSize: '14px',
  fontWeight: 500,
  cursor: 'pointer',
}}>
  Add to Cart
</button>

// Flat status badge
<span style={{
  background: '#f0fdf4',
  color: '#059669',
  padding: '2px 10px',
  borderRadius: '999px',
  fontSize: '12px',
  fontWeight: 500,
}}>
  In Stock
</span>
```

## Forbidden

- `box-shadow` of any kind
- Gradients on interactive elements
- `border-radius: 0` (flat clean is still friendly)
- Semi-transparent overlays
- Multiple accent colors
