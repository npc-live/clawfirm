# Layout Skill: Bento Box

Inspired by: Variant community dashboard designs, XHS Tech Assets, data visualization layouts

## Structure

```
┌────────────────────────────────────────┐
│  ┌──────────────┐  ┌──────┐  ┌──────┐ │
│  │              │  │      │  │      │ │
│  │  2×2 hero    │  │ 1×1  │  │ 1×1  │ │
│  │              │  │      │  │      │ │
│  └──────────────┘  └──────┘  └──────┘ │
│  ┌──────┐  ┌──────────────────────────┐│
│  │ 1×1  │  │      3×1 wide card       ││
│  └──────┘  └──────────────────────────┘│
│  ┌──────────┐  ┌──────┐  ┌──────────┐ │
│  │  2×1     │  │ 1×1  │  │  2×1     │ │
│  └──────────┘  └──────┘  └──────────┘ │
└────────────────────────────────────────┘
```

## CSS Implementation

```css
.bento-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-template-rows: auto;
  gap: 12px;
  padding: 24px;
}

.bento-card {
  border-radius: 16px;
  padding: 24px;
  overflow: hidden;
  position: relative;
}

.bento-card.span-2 { grid-column: span 2; }
.bento-card.span-3 { grid-column: span 3; }
.bento-card.row-2  { grid-row: span 2; }
```

## Rules

- Grid: 4 columns on desktop
- Gap: 12px between all cells
- Cards have `border-radius: 16px` — larger than standard cards
- Each bento cell has a distinct background color/tone — no uniform card color
- At least one cell spans 2+ columns (hero cell)
- Mix of content types: stats, charts, actions, media
- Internal padding: 24px

## Cell Type Examples

```jsx
// Stat card (1×1)
<div className="bento-card" style={{ background: 'var(--bg-card)', gridColumn: 'span 1' }}>
  <div style={{ fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>Total Views</div>
  <div style={{ fontSize: '36px', fontWeight: 600, color: 'var(--text-primary)', marginTop: '8px' }}>128K</div>
  <div style={{ fontSize: '12px', color: '#22c55e', marginTop: '4px' }}>↑ 12% this week</div>
</div>

// Feature card (2×2)
<div className="bento-card" style={{ background: 'var(--accent)', gridColumn: 'span 2', gridRow: 'span 2' }}>
  {/* Main feature content */}
</div>
```

## Forbidden

- Uniform card sizes (defeats the bento aesthetic)
- More than 4 columns on desktop
- Cards without rounded corners
- Empty cells (every cell must have content)
