# Layout Skill: Full Bleed

Inspired by: Variant community landing pages, portfolio sites, creative director showcases

## Structure

```
┌───────────────────────────────────────────────────────┐
│                                                       │
│              FULL WIDTH HERO IMAGE/VIDEO              │
│              (100vw, 80-100vh)                        │
│                                                       │
│         Centered text overlay                         │
│                                                       │
├───────────────────────────────────────────────────────┤
│  constrained content (max-width: 860px, centered)    │
├───────────────────────────────────────────────────────┤
│                                                       │
│              FULL WIDTH FEATURE SECTION               │
│                                                       │
├───────────────────────────────────────────────────────┤
│  constrained content                                  │
└───────────────────────────────────────────────────────┘
```

## CSS Implementation

```css
/* Full-bleed sections alternate with constrained content */
.section-full-bleed {
  width: 100vw;
  margin-left: calc(50% - 50vw);
  position: relative;
  overflow: hidden;
}

.section-constrained {
  max-width: 860px;
  margin: 0 auto;
  padding: 80px 24px;
}

/* Hero section */
.hero {
  height: 90vh;
  min-height: 600px;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  position: relative;
}

.hero-bg {
  position: absolute;
  inset: 0;
  object-fit: cover;
  width: 100%;
  height: 100%;
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
}

.hero-content {
  position: relative;
  z-index: 1;
}
```

## Rules

- Hero: minimum 80vh, full viewport width
- Alternating rhythm: full-bleed → constrained → full-bleed
- Content max-width: 860px (narrower than expected — forces focus)
- Section padding: 80-120px vertical
- Text overlays on images: always add overlay layer (0.35-0.55 opacity)
- Navigation: transparent on hero, solid after scroll
- No horizontal scrollbars — `overflow-x: hidden` on body

## Forbidden

- Boxed layouts on hero sections
- Consistent padding across all sections (variety is key)
- More than 1 full-bleed image stacked consecutively
