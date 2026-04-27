# Layout Skill: Section — CTA (Call to Action)

页面末尾的转化区。三种强度递进的变体。

---

## 变体 A: 全宽色块 CTA (最强冲击)

```
┌─────────────────────────────────────────────────────────┐
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░              BIG CTA HEADLINE                     ░░░│
│░░        One line sub that seals the deal           ░░░│
│░░                  [Get Started Free]               ░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
```

```css
.cta-fullwidth {
  background: var(--accent);           /* OR: accent gradient */
  padding: 100px 24px;
  text-align: center;
}

.cta-headline {
  font-size: clamp(32px, 5vw, 56px);
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1.1;
  color: white;
  margin-bottom: 16px;
}

.cta-sub {
  font-size: 18px;
  color: rgba(255, 255, 255, 0.75);
  margin-bottom: 40px;
  max-width: 480px;
  margin-left: auto;
  margin-right: auto;
}

.cta-btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: white;
  color: var(--accent);
  padding: 14px 32px;
  border-radius: 10px;
  font-size: 16px;
  font-weight: 600;
  border: none;
  cursor: pointer;
  box-shadow: 0 4px 20px rgba(0,0,0,0.2);
}
```

---

## 变体 B: 卡片式 CTA (中等强度，常用)

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│   ┌─────────────────────────────────────────────────┐  │
│   │  CTA Headline                 [Primary CTA]     │  │
│   │  Sub description text         [Secondary]       │  │
│   └─────────────────────────────────────────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

```css
.cta-card-section {
  padding: 80px 24px;
}

.cta-card {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: 24px;
  padding: 56px 64px;
  max-width: 960px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 48px;
}

.cta-card-text { flex: 1; }

.cta-card-headline {
  font-size: clamp(24px, 3vw, 36px);
  font-weight: 700;
  letter-spacing: -0.02em;
  margin-bottom: 8px;
}

.cta-card-sub {
  font-size: 15px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.cta-card-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: flex-end;
  flex-shrink: 0;
}

@media (max-width: 640px) {
  .cta-card { flex-direction: column; align-items: flex-start; }
  .cta-card-actions { align-items: flex-start; }
}
```

---

## 变体 C: 极简文字 CTA (低干扰，品牌感)

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│           Ready to get started?                         │
│           [Start for free]  No credit card required     │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

```css
.cta-minimal {
  padding: 96px 24px;
  text-align: center;
  border-top: 1px solid var(--border-subtle);
}

.cta-minimal-headline {
  font-size: clamp(28px, 4vw, 44px);
  font-weight: 700;
  letter-spacing: -0.02em;
  margin-bottom: 32px;
}

.cta-minimal-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  flex-wrap: wrap;
}

.cta-reassurance {
  font-size: 13px;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 6px;
}
```

---

## 选择指引

| 场景 | 变体 |
|------|------|
| 高转化目标，品牌色强 | A 全宽色块 |
| 标准 SaaS 落地页 | B 卡片式 |
| 内容站、工具类，不想太推销 | C 极简 |

## Rules

- CTA section 必须是页面最后一个内容 section（footer 之前）
- 主 CTA 按钮只有 1 个
- 添加 reassurance copy："No credit card required" / "Free forever" / "Cancel anytime"
- 变体 A 按钮用白色（在彩色背景上反色）

## Forbidden

- CTA section 列出功能 list
- 两个主要 CTA 按钮颜色相同
- 没有 reassurance 降低用户焦虑
