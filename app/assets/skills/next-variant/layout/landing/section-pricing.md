# Layout Skill: Section — Pricing

定价方案区。转化率最高的 section，布局细节直接影响选择行为。

## 结构 (3 层级)

```
┌─────────────────────────────────────────────────────────┐
│              Pricing (centered title)                   │
│              [Monthly] [Annually ✦ Save 20%]            │
├──────────────┬──────────────────────┬──────────────────┤
│   STARTER    │   PRO  ← highlighted │   ENTERPRISE     │
│   Free       │   $29/mo             │   Custom         │
│              │                      │                  │
│  ✓ feature   │  ✓ everything        │  ✓ everything    │
│  ✓ feature   │    in Starter        │    in Pro        │
│  ✗ feature   │  ✓ feature           │  ✓ feature       │
│  ✗ feature   │  ✓ feature           │  ✓ SSO           │
│              │  ✓ feature           │  ✓ SLA           │
│  [Get Free]  │  [Start Free Trial]  │  [Contact Sales] │
└──────────────┴──────────────────────┴──────────────────┘
```

## CSS

```css
.pricing-section {
  padding: 96px 24px;
  text-align: center;
}

/* Toggle */
.billing-toggle {
  display: inline-flex;
  background: var(--bg-secondary);
  border: 1px solid var(--border-default);
  border-radius: 999px;
  padding: 4px;
  gap: 4px;
  margin-bottom: 48px;
}

.billing-option {
  padding: 6px 16px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-secondary);
}

.billing-option.active {
  background: var(--bg-card);
  color: var(--text-primary);
  box-shadow: 0 1px 4px rgba(0,0,0,0.08);
}

.savings-badge {
  background: #dcfce7;
  color: #15803d;
  font-size: 11px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
}

/* Cards grid */
.pricing-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  max-width: 960px;
  margin: 0 auto;
  align-items: start;
}

.pricing-card {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 20px;
  padding: 32px;
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* Highlighted (recommended) plan */
.pricing-card.highlighted {
  border-color: var(--accent);
  background: var(--bg-card);
  position: relative;
  transform: scale(1.02);            /* subtle lift */
  box-shadow: 0 0 0 1px var(--accent), 0 20px 60px rgba(0,0,0,0.12);
}

.popular-badge {
  position: absolute;
  top: -12px;
  left: 50%;
  transform: translateX(-50%);
  background: var(--accent);
  color: white;
  font-size: 11px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: 999px;
  white-space: nowrap;
  letter-spacing: 0.04em;
}

/* Price */
.plan-price {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.price-currency { font-size: 20px; font-weight: 600; color: var(--text-secondary); }
.price-amount   { font-size: 48px; font-weight: 700; letter-spacing: -0.04em; line-height: 1; }
.price-period   { font-size: 14px; color: var(--text-muted); }

/* Feature list */
.plan-features {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.plan-feature {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: var(--text-secondary);
}

.feature-check  { color: var(--accent);     font-size: 14px; }
.feature-cross  { color: var(--text-ghost); font-size: 14px; }
```

## Rules

- 中间推荐方案用 `transform: scale(1.02)` 轻微上浮 — 不要高度差异太大
- "Popular" badge 用绝对定位卡在顶边缘
- 价格数字 `font-size: 48px`，货币符号小一号
- ✗ 特性显示为灰色，不要隐藏（让用户了解差异）
- Billing toggle 在标题正下方，月/年切换影响价格展示
- CTA 按钮 3 个方案用不同 variant：ghost / primary / outline

## Forbidden

- 超过 4 个定价方案（超过就隐藏低频方案）
- 价格用 `$X,XXX` 整数（总显示 `/mo` 或 `/yr`）
- 没有推荐方案视觉标识（用户会不知道选哪个）
- 特性列表超过 8 条（超过折叠）
