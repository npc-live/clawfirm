# Layout Skill: Section — Features

Features section 的三种主流布局，按使用频率排序。

---

## 变体 A: 三列卡片格 (最常见)

```
┌─────────────────────────────────────────────────────────┐
│               Section Title (centered)                  │
│               Subtitle (centered, muted)                │
├──────────────┬──────────────┬──────────────────────────┤
│  icon        │  icon        │  icon                    │
│  Feature 1   │  Feature 2   │  Feature 3               │
│  desc text   │  desc text   │  desc text               │
├──────────────┼──────────────┼──────────────────────────┤
│  icon        │  icon        │  icon                    │
│  Feature 4   │  Feature 5   │  Feature 6               │
│  desc text   │  desc text   │  desc text               │
└──────────────┴──────────────┴──────────────────────────┘
```

```css
.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 2px;                          /* 2px gap = grid lines look like dividers */
  border: 1px solid var(--border-default);
  border-radius: 16px;
  overflow: hidden;
  max-width: 1000px;
  margin: 0 auto;
}

.feature-cell {
  padding: 36px 32px;
  background: var(--bg-card);
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: background 0.15s;
}

.feature-cell:hover { background: var(--bg-hover); }

.feature-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: var(--accent-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}

.feature-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.feature-desc {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
}
```

---

## 变体 B: 交替图文 (深度展示)

```
┌─────────────────────────────────────────────────────────┐
│  TEXT LEFT         │  VISUAL RIGHT                      │
│  Feature title     │  ┌──────────────────────────────┐ │
│  Long description  │  │   Product screenshot / demo  │ │
│  of this feature   │  │                              │ │
│  [Learn more →]    │  └──────────────────────────────┘ │
├────────────────────┴─────────────────────────────────── │
│  VISUAL LEFT   │  TEXT RIGHT                            │
│  ┌──────────┐  │  Feature title                         │
│  │          │  │  Long description                      │
│  └──────────┘  │  [Learn more →]                        │
└─────────────────────────────────────────────────────────┘
```

```css
.features-alternating {
  display: flex;
  flex-direction: column;
  gap: 120px;
  padding: 80px 48px;
  max-width: 1100px;
  margin: 0 auto;
}

.feature-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 80px;
  align-items: center;
}

.feature-row.reverse { direction: rtl; }
.feature-row.reverse > * { direction: ltr; }

.feature-row-visual img,
.feature-row-visual .mock {
  width: 100%;
  border-radius: 16px;
  border: 1px solid var(--border-default);
  box-shadow: 0 24px 64px rgba(0,0,0,0.12);
}

.feature-row-text {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.feature-row-label {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--accent);
}

.feature-row-title {
  font-size: clamp(24px, 3vw, 36px);
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.feature-row-desc {
  font-size: 16px;
  color: var(--text-secondary);
  line-height: 1.7;
}
```

---

## 变体 C: 左标题 + 右列表 (简洁信息密集)

```
┌─────────────────────────────────────────────────────────┐
│  BIG SECTION    │  ✓ Feature one — description          │
│  TITLE LEFT     │  ✓ Feature two — description          │
│                 │  ✓ Feature three — description        │
│  Short sub      │  ✓ Feature four — description         │
│                 │  ✓ Feature five — description         │
│  [CTA]          │  ✓ Feature six — description          │
└─────────────────────────────────────────────────────────┘
```

```css
.features-checklist {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 80px;
  align-items: start;
  padding: 80px 48px;
  max-width: 1000px;
  margin: 0 auto;
}

.features-checklist-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.checklist-item {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding: 16px 0;
  border-bottom: 1px solid var(--border-subtle);
  font-size: 15px;
}

.checklist-icon {
  color: var(--accent);
  font-size: 14px;
  flex-shrink: 0;
  margin-top: 1px;
}
```

---

## Rules

- Section 标题 padding-top: 96-120px（和上一节保持呼吸）
- 变体 A (卡片格): 适合 6-9 个 feature，用 `gap: 2px` 让格子感更强
- 变体 B (交替): 适合 2-4 个重要 feature，每个配截图
- 变体 C (checklist): 适合 6-12 个简短 feature，信息密集型产品

## Forbidden

- 超过 9 个卡片放在变体 A（超过就换变体）
- 变体 B 中视觉资产用纯色占位
- 在 section 内再嵌套 section 标题
