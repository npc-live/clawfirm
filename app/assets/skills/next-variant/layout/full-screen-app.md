# Layout Skill: Full-Screen App

Inspired by: Generative art playgrounds, algorithm visualizers, IDE interfaces, full-screen dashboards

## Structure

```
┌─────────────────────────────────────────────┐
│  HEADER (fixed, 60-80px)                    │
├─────────────────────────────────────────────┤
│                                              │
│  MAIN CONTENT (flex: 1, centered)           │
│  - Visualization canvas                      │
│  - Large display area                        │
│                                              │
├─────────────────────────────────────────────┤
│  CONTROLS PANEL (fixed, auto height)        │
│  - Parameter sliders                         │
│  - Settings grid                             │
├─────────────────────────────────────────────┤
│  ACTION BAR (fixed, 60px)                   │
│  - Tool buttons                              │
└─────────────────────────────────────────────┘
```

## CSS Implementation

```css
.app-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  padding: 20px;
  gap: 16px;
}

.app-header {
  flex-shrink: 0;
  padding: 16px 24px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.app-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  padding: 40px;
  position: relative;
  overflow: hidden;
}

.app-controls {
  flex-shrink: 0;
  padding: 20px 24px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
}

.app-actions {
  flex-shrink: 0;
  display: flex;
  gap: 12px;
  justify-content: center;
  padding: 0;
}

/* Parameter grid layout */
.controls-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1px;
  background: var(--border-default);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.control-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
  background: var(--bg-secondary);
}

.control-label {
  font-size: var(--text-sm);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--text-secondary);
  min-width: 80px;
}

.control-slider {
  flex: 1;
  height: 2px;
  background: var(--border-default);
  border-radius: 2px;
  position: relative;
  cursor: pointer;
}

.control-value {
  font-size: var(--text-base);
  color: var(--text-accent);
  min-width: 48px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

/* Canvas container */
.canvas-container {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Metadata overlay */
.metadata-overlay {
  position: absolute;
  bottom: 20px;
  right: 20px;
  display: flex;
  gap: 24px;
}
```

## Rules

- App shell: 100vh height, no scrolling on the container
- All sections: use `border-radius` from 6-12px
- Header: 60-80px height, fixed
- Main content: `flex: 1`, vertically and horizontally centered
- Controls panel: auto height based on content, fixed position
- Action bar: 60px height, fixed at bottom
- Gap between sections: 16-20px
- Container padding: 20px on all sides
- Parameter controls: use 1px grid lines to separate items
- Control grid: auto-fit columns with 280px minimum width

## Component Patterns

### Header with Status
```jsx
<div className="app-header">
  <div>
    <h1 style={{ fontSize: 'var(--text-xl)', textTransform: 'uppercase', letterSpacing: '0.1em' }}>
      APP_NAME
    </h1>
    <p style={{ fontSize: 'var(--text-xs)', color: 'var(--text-secondary)', marginTop: '2px' }}>
      Subtitle or description
    </p>
  </div>
  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: 'var(--text-xs)' }}>
    <div style={{ width: '8px', height: '8px', background: 'var(--accent)', borderRadius: '50%' }}></div>
    <span>LIVE</span>
  </div>
</div>
```

### Main Visualization Area
```jsx
<div className="app-main">
  <div className="canvas-container">
    {/* Your visualization/canvas here */}
  </div>

  <div className="metadata-overlay">
    <div>
      <div style={{ fontSize: 'var(--text-xs)', color: 'var(--text-secondary)' }}>LABEL</div>
      <div style={{ fontSize: 'var(--text-sm)', color: 'var(--text-accent)' }}>Value</div>
    </div>
  </div>
</div>
```

### Parameter Controls Grid
```jsx
<div className="app-controls">
  <div className="controls-grid">
    <div className="control-item">
      <span className="control-label">Parameter</span>
      <div className="control-slider">
        <div style={{ width: '12px', height: '12px', background: 'var(--accent)', borderRadius: '50%', position: 'absolute', left: '60%', top: '-5px' }}></div>
      </div>
      <span className="control-value">5.2</span>
    </div>
    {/* More controls... */}
  </div>
</div>
```

### Action Bar
```jsx
<div className="app-actions">
  <button style={{
    padding: '12px 24px',
    border: '1px solid var(--border-default)',
    borderRadius: 'var(--radius-sm)',
    background: 'transparent',
    color: 'var(--text-primary)',
    fontSize: 'var(--text-sm)',
    textTransform: 'uppercase',
    cursor: 'pointer'
  }}>
    ACTION
  </button>
</div>
```

## Forbidden

- Scrollable main content area (use fixed layout instead)
- Multiple main content sections (keep it focused)
- Sidebar navigation (this is full-screen, not dashboard)
- Overlapping sections (all sections are clearly separated)
- Responsive stacking on desktop (keep the full-screen layout)
- More than 4-5 sections vertically
