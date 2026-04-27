# Color Skill: Bright Blocks

Inspired by: Playful landing pages, non-profit websites, children's apps, brand-forward product pages with bold personality

## Design Tokens

```css
:root {
  /* Pastel/Bright block colors - choose 2-4 per design */
  --purple-light: #e9d5ff;   /* lavender */
  --pink-light: #fbcfe8;     /* rose */
  --blue-light: #bfdbfe;     /* sky */
  --lime-bright: #d9f99d;    /* lime */
  --yellow-light: #fef08a;   /* yellow */
  --orange-light: #fed7aa;   /* peach */
  --teal-light: #99f6e4;     /* teal */

  /* Text on bright backgrounds */
  --text-primary: #1a1a1a;
  --text-secondary: #374151;
  --text-muted: #6b7280;

  /* Optional dark block */
  --dark-block: #111827;
  --text-on-dark: #ffffff;

  /* Borders and accents */
  --border-bold: 2px solid #1a1a1a;
  --border-light: 1px solid rgba(0,0,0,0.1);
}
```

## Rules

- Use **solid color blocks**, NOT gradients
- Each section/block gets ONE flat background color
- Choose 2-4 colors maximum per page from the palette
- Colors should be **high saturation** but **light** (pastel-bright range)
- Text is dark (#1a1a1a) on light backgrounds — high contrast
- Blocks are full-width sections or large panels
- Borders (if used) are bold (2px) and black
- NO shadows on blocks — flat design only
- Color transitions are abrupt, NOT blended
- Optional: one dark block (#111827) for contrast

## Color Pairing Recommendations

### Energetic (Kids, Playful)
```css
--color-1: #fef08a;  /* yellow */
--color-2: #fbcfe8;  /* pink */
--color-3: #99f6e4;  /* teal */
```

### Fresh (Health, Nature)
```css
--color-1: #d9f99d;  /* lime */
--color-2: #bfdbfe;  /* blue */
--color-3: #e9d5ff;  /* lavender */
```

### Warm (Creative, Friendly)
```css
--color-1: #fed7aa;  /* peach */
--color-2: #fef08a;  /* yellow */
--color-3: #fbcfe8;  /* pink */
```

### Cool (Tech, Modern)
```css
--color-1: #bfdbfe;  /* blue */
--color-2: #e9d5ff;  /* lavender */
--color-3: #99f6e4;  /* teal */
```

## Component Examples

### Full-width Section Blocks

```jsx
// Hero section
<section style={{
  background: '#e9d5ff',  /* purple-light */
  padding: '80px 40px',
  minHeight: '70vh',
}}>
  <h1 style={{ color: '#1a1a1a', fontSize: '64px', fontWeight: 700 }}>
    Big bold heading
  </h1>
</section>

// Stats section
<section style={{
  background: '#d9f99d',  /* lime-bright */
  padding: '80px 40px',
}}>
  <div style={{ fontSize: '120px', fontWeight: 800, color: '#1a1a1a' }}>
    142
  </div>
</section>

// Dark accent block
<section style={{
  background: '#111827',
  padding: '80px 40px',
  color: '#ffffff',
}}>
  <h2>Stand out section</h2>
</section>
```

### Cards with Block Colors

```jsx
<div style={{
  display: 'grid',
  gridTemplateColumns: 'repeat(3, 1fr)',
  gap: '0',  /* no gap for block effect */
}}>
  <div style={{ background: '#fef08a', padding: '40px' }}>Card 1</div>
  <div style={{ background: '#fbcfe8', padding: '40px' }}>Card 2</div>
  <div style={{ background: '#99f6e4', padding: '40px' }}>Card 3</div>
</div>
```

### Buttons on Bright Blocks

```jsx
// Outline button (works on any bright block)
<button style={{
  background: 'transparent',
  border: '2px solid #1a1a1a',
  borderRadius: '999px',
  padding: '12px 28px',
  color: '#1a1a1a',
  fontWeight: 600,
  cursor: 'pointer',
}}>
  Click me
</button>

// Filled button
<button style={{
  background: '#1a1a1a',
  border: 'none',
  borderRadius: '999px',
  padding: '12px 28px',
  color: '#ffffff',
  fontWeight: 600,
  cursor: 'pointer',
}}>
  Get Started
</button>
```

## Usage Guidelines

### When to Use

Use bright-blocks for:
- Playful, friendly brands
- Non-profit and community organizations
- Children's products or education
- Creative agencies showing personality
- Landing pages that need to stand out
- Products targeting Gen Z / young audiences

### When NOT to Use

Avoid for:
- Enterprise/corporate sites (too playful)
- Luxury brands (too casual)
- Financial services (lacks seriousness)
- Healthcare (except pediatrics)
- Technical documentation (distracting)

## Typography Pairing

Best paired with:
- **big-display** (grotesque sans, bold) — RECOMMENDED
- **inter-minimal** (clean sans)

Avoid:
- display-serif (too elegant, clashes with playful colors)
- mono-tech (too technical)

## Style Pairing

Best paired with:
- **flat-clean** (no shadows, solid fills) — PERFECT
- **playful** (offset shadows, pill shapes) — WORKS WELL

Avoid:
- glassmorphism (blur conflicts with solid blocks)
- skeuomorphic (3D conflicts with flat blocks)
- brutalist (too harsh, though can work for edgy brands)

## Forbidden

- Gradients within blocks (keep them solid)
- Muted or desaturated colors (go bright!)
- White backgrounds (boring, defeats the purpose)
- More than 4 block colors per page (visual chaos)
- Subtle color transitions (be bold!)
- Drop shadows on blocks (flat only)
- Transparent or semi-transparent blocks
- Mixing bright blocks with dark-moody aesthetics

## Accessibility Notes

- All combinations in the palette meet WCAG AA for contrast with dark text (#1a1a1a)
- Test color blindness compatibility: avoid red-green only distinctions
- Provide sufficient spacing between color blocks for visual rest
- Use consistent text color (#1a1a1a) across all bright blocks for readability

## Examples in the Wild

- Stripe's gradient-free landing pages
- Notion's playful brand pages
- Kids' educational apps
- Non-profit fundraising pages
- Creative portfolio sites
- Community event pages
