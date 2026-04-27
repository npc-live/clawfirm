# Layout Skill: Vertical Feed

Inspired by: Mobile apps, e-commerce product pages, social media timelines, todo/task apps, messaging interfaces

## Structure

```
┌─────────────────────────┐
│  HEADER (sticky/fixed)  │
│  - Back button          │
│  - Title                │
│  - Action icons         │
├─────────────────────────┤
│                         │
│  CONTENT (scrollable)   │
│  - Hero/Image           │
│  - Sections             │
│  - Cards                │
│  - Lists                │
│                         │
│  (padding-bottom:       │
│   space for bottom bar) │
│                         │
├─────────────────────────┤
│  BOTTOM BAR (fixed)     │
│  - Primary actions      │
│  - Navigation           │
└─────────────────────────┘
```

## CSS Implementation

```css
/* Container: mobile-first, centered */
.app-container {
  max-width: 500px;
  margin: 0 auto;
  min-height: 100vh;
  background: var(--bg-card);
  display: flex;
  flex-direction: column;
}

/* Header: sticky or fixed */
.header-sticky {
  position: sticky;
  top: 0;
  z-index: 10;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border-default);
  padding: 16px 20px;
}

/* Alternative: fixed header (for overlays) */
.header-fixed {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  max-width: 500px;
  margin: 0 auto;
  z-index: 10;
  background: var(--bg-card);
}

/* Header content */
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.header-title {
  font-size: 15px;
  font-weight: 500;
  text-align: center;
  flex: 1;
}

/* Content: scrollable area */
.content-scrollable {
  flex: 1;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  padding-bottom: 100px;  /* space for bottom bar */
}

/* Content inner wrapper */
.content-inner {
  padding: 0 20px 24px;
}

/* Bottom bar: fixed */
.bottom-bar-fixed {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  max-width: 500px;
  margin: 0 auto;
  background: var(--bg-card);
  border-top: 1px solid var(--border-default);
  padding: 16px 20px;
  z-index: 10;
}

/* Bottom actions layout */
.bottom-actions {
  display: flex;
  gap: 12px;
}

.bottom-actions .btn {
  flex: 1;
}

/* Icon buttons (common in headers) */
.icon-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 1px solid var(--border-default);
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background 0.2s;
}

.icon-btn:hover {
  background: var(--bg-hover);
}
```

## Rules

- **Max-width: 500px** — mobile-first, centered on desktop
- **Header:** sticky or fixed, 16-20px padding
- **Content:** flex: 1, padding-bottom: 80-120px (for bottom bar)
- **Bottom bar:** fixed, z-index: 10, matches max-width
- **Scrolling:** smooth, -webkit-overflow-scrolling: touch
- **Spacing:** consistent 20px horizontal padding
- **Background:** usually white/light, not transparent

## Common Patterns

### Header Variations

```jsx
// Simple: Back + Title + Action
<div className="header-sticky">
  <div className="header-content">
    <button className="icon-btn">←</button>
    <h1 className="header-title">Page Title</h1>
    <button className="icon-btn">⋮</button>
  </div>
</div>

// Search: Back + Search Input
<div className="header-sticky">
  <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
    <button className="icon-btn">←</button>
    <input placeholder="Search..." style={{ flex: 1 }} />
  </div>
</div>

// Tabs: Title + Tab Navigation
<div className="header-sticky">
  <h1 className="header-title">Title</h1>
  <div className="tabs">
    <button className="tab active">Tab 1</button>
    <button className="tab">Tab 2</button>
  </div>
</div>
```

### Content Patterns

```jsx
// Hero Image + Details (e-commerce)
<div className="content-scrollable">
  <img src="product.jpg" style={{ width: '100%', aspectRatio: '1' }} />
  <div className="content-inner">
    <h1>Product Name</h1>
    <p>Description</p>
    {/* More content */}
  </div>
</div>

// Feed Items (social media)
<div className="content-scrollable">
  {items.map(item => (
    <div className="feed-item">
      {item.content}
    </div>
  ))}
</div>

// Task List (todo app)
<div className="content-scrollable">
  <div className="content-inner">
    <h2>Today</h2>
    <div className="task-list">
      {/* Tasks */}
    </div>
  </div>
</div>
```

### Bottom Bar Variations

```jsx
// CTA Buttons (e-commerce)
<div className="bottom-bar-fixed">
  <div className="bottom-actions">
    <button className="btn btn-outline">Add to Cart</button>
    <button className="btn btn-primary">Buy Now</button>
  </div>
</div>

// Single Action + Floating Button
<div className="bottom-bar-fixed">
  <button className="btn btn-primary btn-full">Continue</button>
</div>

// Tab Navigation (app)
<div className="bottom-bar-fixed">
  <nav className="tab-nav">
    <button className="tab-item active">Home</button>
    <button className="tab-item">Search</button>
    <button className="tab-item">Profile</button>
  </nav>
</div>

// Input Bar (chat)
<div className="bottom-bar-fixed">
  <input placeholder="Type a message..." style={{ flex: 1 }} />
  <button className="icon-btn">→</button>
</div>
```

## Responsive Behavior

```css
/* Desktop: centered with max-width */
@media (min-width: 768px) {
  .app-container {
    box-shadow: 0 0 0 1px var(--border-default);
  }
}

/* Mobile: full width */
@media (max-width: 500px) {
  .app-container {
    max-width: 100%;
  }

  .bottom-bar-fixed {
    max-width: 100%;
  }
}
```

## Use Cases

**Perfect for:**
- E-commerce product detail pages
- Todo/task applications
- Social media timelines
- Chat/messaging interfaces
- Article readers
- Form wizards
- Profile pages
- Settings screens

**NOT for:**
- Desktop-first dashboards (use sidebar-dashboard)
- Multi-column layouts (use bento-box)
- Landing pages (use hero-* layouts)
- Full-screen apps (use full-screen-app)

## Example: E-commerce Product Page

```jsx
<div className="app-container">
  {/* Header */}
  <div className="header-sticky">
    <div className="header-content">
      <button className="icon-btn">←</button>
      <h1 className="header-title">Product Name</h1>
      <button className="icon-btn">♡</button>
    </div>
  </div>

  {/* Content */}
  <div className="content-scrollable">
    {/* Product Image */}
    <img src="product.jpg" style={{ width: '100%', aspectRatio: '1' }} />

    {/* Product Details */}
    <div className="content-inner">
      <h2>Product Title</h2>
      <p className="price">$99.00</p>

      {/* Color Selector */}
      <div className="selector">
        <h3>Choose Color</h3>
        <div className="color-options">
          {/* Color circles */}
        </div>
      </div>

      {/* Size Selector */}
      <div className="selector">
        <h3>Choose Size</h3>
        <div className="size-options">
          {/* Size buttons */}
        </div>
      </div>

      {/* Description */}
      <div className="description">
        <p>Product description...</p>
      </div>
    </div>
  </div>

  {/* Bottom Actions */}
  <div className="bottom-bar-fixed">
    <div className="bottom-actions">
      <button className="btn btn-outline">Add to Cart</button>
      <button className="btn btn-primary">Buy Now</button>
    </div>
  </div>
</div>
```

## Forbidden

- Multi-column content (keep it single column)
- Horizontal scrolling (use vertical only)
- Overlapping header/content (use sticky, not absolute)
- No bottom padding on content (will be cut off by bottom bar)
- Desktop-first approach (mobile-first always)
- Complex nested scrolling (keep it simple)

## Tips

- Use `padding-bottom: 100px` on content to ensure bottom content isn't hidden
- Add `safe-area-inset` support for notched devices
- Keep bottom bar height under 80px for comfortable thumb reach
- Use skeleton loaders during content load
- Implement pull-to-refresh for feed-style content
- Consider adding a FAB (floating action button) for primary actions
