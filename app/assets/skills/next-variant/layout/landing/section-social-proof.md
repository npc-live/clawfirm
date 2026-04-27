# Layout Skill: Section — Social Proof

Logo bar、评价、数据指标。建立信任的三种标准布局。

---

## 变体 A: Logo Bar (品牌背书)

```
┌─────────────────────────────────────────────────────────┐
│         Trusted by teams at                             │
│                                                         │
│   [Logo]  [Logo]  [Logo]  [Logo]  [Logo]  [Logo]        │
└─────────────────────────────────────────────────────────┘
```

```css
.logo-bar {
  padding: 48px 24px;
  border-top: 1px solid var(--border-subtle);
  border-bottom: 1px solid var(--border-subtle);
  text-align: center;
}

.logo-bar-label {
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-muted);
  margin-bottom: 32px;
}

.logo-bar-logos {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 48px;
  flex-wrap: wrap;
}

.logo-bar-logos img {
  height: 24px;
  opacity: 0.4;
  filter: grayscale(1);
  transition: opacity 0.2s;
}
.logo-bar-logos img:hover { opacity: 0.7; }
```

---

## 变体 B: 评价卡片 (3列或轮播)

```
┌─────────────────────────────────────────────────────────┐
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────┐ │
│  │ ★★★★★           │  │ ★★★★★           │  │ ★★★★★  │ │
│  │ "Quote text     │  │ "Quote text     │  │ "Quote  │ │
│  │  that is real   │  │  that spans     │  │  text"  │ │
│  │  and specific"  │  │  two lines"     │  │         │ │
│  │                 │  │                 │  │         │ │
│  │ ○ Name          │  │ ○ Name          │  │ ○ Name  │ │
│  │   Title, Co     │  │   Title, Co     │  │  Title  │ │
│  └─────────────────┘  └─────────────────┘  └─────────┘ │
└─────────────────────────────────────────────────────────┘
```

```css
.testimonials-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  max-width: 1100px;
  margin: 0 auto;
  padding: 0 24px;
}

.testimonial-card {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 16px;
  padding: 28px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.testimonial-stars {
  display: flex;
  gap: 2px;
  color: #f59e0b;
  font-size: 14px;
}

.testimonial-quote {
  font-size: 15px;
  line-height: 1.65;
  color: var(--text-primary);
  flex: 1;
}

.testimonial-author {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-top: 16px;
  border-top: 1px solid var(--border-subtle);
}

.testimonial-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--bg-secondary);
  flex-shrink: 0;
}

.testimonial-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.testimonial-role {
  font-size: 12px;
  color: var(--text-muted);
}
```

---

## 变体 C: 数据指标条 (Stats Bar)

```
┌─────────────────────────────────────────────────────────┐
│      10,000+        99.9%          4.9/5          2M+   │
│    Active Users    Uptime SLA     App Rating    Exports  │
└─────────────────────────────────────────────────────────┘
```

```css
.stats-bar {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1px;
  background: var(--border-default);      /* gap 作为分隔线 */
  border: 1px solid var(--border-default);
  border-radius: 16px;
  overflow: hidden;
  max-width: 900px;
  margin: 0 auto;
}

.stat-item {
  background: var(--bg-card);
  padding: 40px 32px;
  text-align: center;
}

.stat-value {
  font-size: clamp(28px, 4vw, 40px);
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.stat-label {
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.4;
}
```

---

## Rules

- Logo bar 始终放在 hero 紧接下方（第一个 section）
- 评价引号必须具体可信，不用 "This is amazing!" 这种泛泛而谈
- Stats bar 数字用 `font-variant-numeric: tabular-nums`
- 三种变体可以组合出现，但不要在同一 section 混用

## Forbidden

- Logo bar 中 logo 不灰化（会抢视觉）
- 评价卡片超过 6 张（超过用轮播）
- Stats 数字没有单位说明
