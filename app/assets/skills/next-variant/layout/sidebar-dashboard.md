# Layout Skill: Sidebar Dashboard

Inspired by: Variant community dashboard interfaces, app-style navigation patterns

## Structure

```
┌─────────────────────────────────────────────────────────┐
│ ┌──────────┐  ┌───────────────────────────────────────┐ │
│ │          │  │  TOP BAR (48px)                       │ │
│ │          │  ├───────────────────────────────────────┤ │
│ │ SIDEBAR  │  │                                       │ │
│ │ (240px)  │  │  MAIN CONTENT AREA                   │ │
│ │          │  │  (scrollable independently)           │ │
│ │ nav items│  │                                       │ │
│ │          │  │                                       │ │
│ │          │  │                                       │ │
│ │  bottom  │  │                                       │ │
│ │  actions │  │                                       │ │
│ └──────────┘  └───────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## CSS Implementation

```css
.app-shell {
  display: grid;
  grid-template-columns: 240px 1fr;
  grid-template-rows: 1fr;
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border-default);
  padding: 16px 12px;
  overflow-y: auto;
}

.sidebar-logo {
  padding: 8px 12px 20px;
  font-weight: 600;
  font-size: 15px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 0.1s;
}

.nav-item:hover { background: var(--bg-hover); color: var(--text-primary); }
.nav-item.active { background: var(--bg-active); color: var(--text-primary); font-weight: 500; }

.main-content {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.top-bar {
  height: 48px;
  border-bottom: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  padding: 0 20px;
  gap: 12px;
  flex-shrink: 0;
}

.content-area {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}
```

## Rules

- Sidebar: exactly 240px wide, not collapsible in default state
- Top bar: exactly 48px tall
- Nav items: 6px border-radius, 8px vertical padding
- Active state uses background highlight, NOT just color change
- Sidebar bottom section: user avatar + settings, pinned to bottom
- Main content scrolls independently from sidebar
- Content area padding: 24px

## Forbidden

- Top navigation (horizontal nav bar layout)
- Tabs within the main content area for primary navigation
- Collapsible sidebar in default/desktop view
- Icons-only sidebar without labels
