# Layout Skill: Masonry Gallery

Inspired by: Variant community's core grid layout — Pinterest-style, variable card heights, infinite scroll

## Structure

```
┌─────────────────────────────────────────────────────────┐
│ [narrow sidebar 52px] │    MASONRY GRID (3 columns)     │
│                       │  ┌──────┐ ┌──────────┐ ┌─────┐ │
│                       │  │ card │ │          │ │     │ │
│                       │  │      │ │  tall    │ │card │ │
│                       │  └──────┘ │  card    │ │     │ │
│                       │  ┌──────┐ │          │ └─────┘ │
│                       │  │ tall │ └──────────┘ ┌─────┐ │
│                       │  │      │ ┌──────────┐ │     │ │
│                       │  │ card │ │ wide card│ │     │ │
│                       │  └──────┘ └──────────┘ └─────┘ │
└─────────────────────────────────────────────────────────┘
```

## CSS Implementation

```css
.gallery-grid {
  columns: 3;
  column-gap: 12px;
  max-width: 1100px;
  margin: 0 auto;
  padding: 0 16px;
}

.gallery-card {
  break-inside: avoid;
  margin-bottom: 12px;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  position: relative;
}

/* Hover overlay */
.gallery-card:hover .card-actions {
  opacity: 1;
}

.card-actions {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 12px;
  background: linear-gradient(transparent, rgba(0,0,0,0.7));
  opacity: 0;
  transition: opacity 0.15s ease;
  display: flex;
  gap: 6px;
  justify-content: flex-end;
}
```

## Rules

- Columns: exactly 3 on desktop, 2 on tablet, 1 on mobile
- Column gap: 12px — tight, content-first
- Cards have NO fixed height — images/content determine height
- Sidebar is minimal: 52px wide, icon-only navigation
- Card hover reveals action buttons via gradient overlay from bottom
- Infinite scroll — no pagination UI visible
- Cards: `border-radius: 8px`, no box-shadow on default state

## Forbidden

- Fixed card heights (ruins the masonry effect)
- Pagination buttons
- Full-width sidebar navigation
- Header bars that take vertical space
