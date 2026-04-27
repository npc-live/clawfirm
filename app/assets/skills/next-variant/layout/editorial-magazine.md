# Layout Skill: Editorial Magazine

Inspired by: Variant community "The Dormant Phase" blog, typography specimens, HELLO I'M ELIM PAN

## Structure

```
┌────────────────────────────────────────────────────────┐
│  MASTHEAD (full width, typographic)                    │
├────────────────────────────────────────────────────────┤
│                                                        │
│  ┌──────────────────────┐   ┌──────────────────────┐  │
│  │                      │   │  HEADLINE             │  │
│  │     LEAD IMAGE       │   │  (large serif)        │  │
│  │     (square crop)    │   │                       │  │
│  │                      │   │  Standfirst text      │  │
│  └──────────────────────┘   │  that runs multiple   │  │
│                              │  lines                │  │
│                              └──────────────────────┘  │
├────────────────────────────────────────────────────────┤
│  Body text column (65-75ch width, centered)            │
│  with occasional full-width image breaks               │
├────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ related  │  │ related  │  │ related  │            │
│  └──────────┘  └──────────┘  └──────────┘            │
└────────────────────────────────────────────────────────┘
```

## CSS Implementation

```css
.masthead {
  border-bottom: 2px solid currentColor;
  padding: 16px 0;
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}

.article-hero {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 48px;
  padding: 48px 0;
}

.article-body {
  max-width: 65ch;
  margin: 0 auto;
  font-size: 18px;
  line-height: 1.75;
}

.article-body p + p {
  text-indent: 2em;
}

/* Pull quote */
.pull-quote {
  border-left: 3px solid currentColor;
  padding-left: 24px;
  margin: 40px 0;
  font-size: 22px;
  font-style: italic;
  line-height: 1.5;
}

/* Drop cap */
.article-body > p:first-child::first-letter {
  float: left;
  font-size: 4.5em;
  line-height: 0.8;
  margin: 0.1em 0.12em 0 0;
  font-weight: 700;
}
```

## Rules

- Body text column: exactly 65ch max-width (optimal reading width)
- First paragraph gets drop cap
- Pull quotes on key passages — break the text rhythm
- Image captions in small caps or italic
- Consistent vertical rhythm via `line-height: 1.75` on body
- Section dividers: thin horizontal rules or decorative ornaments (❧ ✦)
- Date and category labels in small caps with letter-spacing

## Forbidden

- Sans-serif body text (editorial requires serif)
- Full-width text blocks without breathing room
- Card-based article layouts
- Centered body text
