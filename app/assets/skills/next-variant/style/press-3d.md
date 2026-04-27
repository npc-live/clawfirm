# Style Skill: Press 3D

Inspired by: PostHog — 按钮有物理「按压感」，hover 触发颜色翻转，整体保持工整克制但细节充满个性

## 设计哲学

和 `skeuomorphic`（模拟材质/光照）不同，Press 3D 只在**一个轴**上做 3D：按钮底部的单层阴影（`0 2px 0`）模拟立体感，按下时 `translateY(2px)` + 阴影消失模拟物理按压。其余地方完全 flat，保持工程师审美。

Hover Invert（颜色翻转）是 PostHog 的第二个签名：元素 hover 时前景/背景互换，像印章盖上去的感觉。

## 核心 Token

```css
:root {
  --radius: 4px;           /* 极小圆角 — 工整不装饰 */
  --radius-md: 6px;
  --radius-lg: 8px;

  /* 3D 按压阴影（底部单层） */
  --shadow-press-orange:  0 2px 0 #B17816;
  --shadow-press-neutral: 0 2px 0 #8A8C82;
  --shadow-press-dark:    0 2px 0 #000000;

  /* 卡片阴影 — 同样克制 */
  --shadow-card: 0 1px 3px rgba(0,0,0,0.08), 0 1px 2px rgba(0,0,0,0.05);
  --shadow-card-hover: 0 4px 12px rgba(0,0,0,0.1), 0 2px 4px rgba(0,0,0,0.06);
}
```

## 按钮系统

### Primary (橙色 3D)

```css
.btn-primary {
  background: #EB9D2A;
  color: #000000;               /* 黑字，不是白字 */
  border: 1px solid #B17816;
  box-shadow: 0 2px 0 #B17816;
  border-radius: 4px;
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.01em;
  cursor: pointer;
  transition: background 0.1s, box-shadow 0.1s, transform 0.1s;
}
.btn-primary:hover {
  background: #D4891F;
}
.btn-primary:active {
  transform: translateY(2px);
  box-shadow: none;
}
```

### Secondary (奶油描边 3D)

```css
.btn-secondary {
  background: #FDFDF8;
  color: #4D4F46;
  border: 1px solid #BFC1B7;
  box-shadow: 0 2px 0 #BFC1B7;
  border-radius: 4px;
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 600;
}
.btn-secondary:hover {
  background: #F3F2EE;
}
.btn-secondary:active {
  transform: translateY(2px);
  box-shadow: none;
}
```

### Dark (深色 3D)

```css
.btn-dark {
  background: #1E1F23;
  color: #F3F2EE;
  border: 1px solid #000;
  box-shadow: 0 2px 0 #000;
  border-radius: 4px;
  padding: 8px 16px;
}
.btn-dark:active {
  transform: translateY(2px);
  box-shadow: none;
}
```

## Hover Invert 模式

PostHog 的导航 item、功能 chip、小标签在 hover 时**背景和前景颜色互换**。

```css
/* 例：导航链接 */
.nav-item {
  color: #4D4F46;
  background: transparent;
  border-radius: 4px;
  padding: 6px 10px;
  transition: background 0.1s, color 0.1s;
}
.nav-item:hover {
  background: #4D4F46;   /* 原来的文字色变背景 */
  color: #FDFDF8;        /* 原来的背景色变文字 */
}

/* 例：feature tag */
.feature-tag {
  background: #F3F2EE;
  color: #4D4F46;
  border: 1px solid #E0DFD9;
  border-radius: 4px;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.1s, color 0.1s, border-color 0.1s;
}
.feature-tag:hover {
  background: #4D4F46;
  color: #FDFDF8;
  border-color: #4D4F46;
}
```

## 卡片风格

```css
.card {
  background: #FFFFFF;
  border: 1px solid #E0DFD9;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
  transition: box-shadow 0.15s, transform 0.15s;
}
.card:hover {
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  transform: translateY(-2px);
}
```

## Section 分隔线

```css
/* PostHog 用极细的分隔线，不是大块 shadow */
.section-divider {
  border: none;
  border-top: 1px solid #E0DFD9;
  margin: 0;
}

/* 深色 section（CTA / Footer）的分隔 */
.section-dark {
  background: #1E1F23;
  border-top: 1px solid #2D2E34;
}
```

## 代码 block 风格

PostHog 页面上经常出现终端命令 block：

```css
.code-block {
  background: #1E1F23;
  border: 1px solid #2D2E34;
  border-radius: 6px;
  padding: 16px 20px;
  font-family: "IBM Plex Mono", ui-monospace, monospace;
  font-size: 13px;
  color: #F3F2EE;
  position: relative;
  overflow-x: auto;
}

/* 复制按钮（绝对定位右上角） */
.code-copy-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  background: #2D2E34;
  color: #A0A193;
  border: 1px solid #3A3B42;
  border-radius: 4px;
  padding: 4px 8px;
  font-size: 11px;
  cursor: pointer;
}
.code-copy-btn:hover {
  background: #3A3B42;
  color: #F3F2EE;
}
```

## 动画（保守但有趣）

PostHog 有少量 playful 动画，但不滥用：

```css
/* wiggle — 用于 icon 或 badge hover */
@keyframes wiggle {
  0%, 100% { transform: rotate(0deg); }
  25%       { transform: rotate(-6deg); }
  75%       { transform: rotate(6deg); }
}
.icon:hover { animation: wiggle 0.3s ease-in-out; }

/* 页面进入 — 轻微上浮 */
@keyframes fadeInUp {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}
.hero-content { animation: fadeInUp 0.4s ease-out; }
```

## Rules

- `border-radius` 最大 **8px**（PostHog 不用大圆角，工整优先）
- 所有按钮必须有 `0 2px 0 {darker}` 底部阴影
- 按下时 `translateY(2px)` + 阴影归零
- Hover invert 只用于**小型交互元素**（nav item、tag、chip），不用于大卡片
- 大卡片 hover 用 `translateY(-2px)` + 加深阴影（不翻转）
- 动画保守：duration ≤ 0.3s，只在有意义的地方用

## Forbidden

- `border-radius > 8px`（会变得太「圆润」）
- 渐变背景用于按钮（按钮用纯色）
- `box-shadow` 四个方向都扩散（只允许底部 `0 2px 0`）
- 按钮悬停时改变 `border-radius`
- 全页面都用 hover invert（会眼花）
