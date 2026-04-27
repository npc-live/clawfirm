# Color Skill: Warm Cream Orange

Inspired by: PostHog — developer tools that feel warm and approachable, not cold enterprise blue

## 设计哲学

拒绝 SaaS 蓝。背景用奶油白而非纯白，主色用橙而非蓝——在工程师审美中突围，传递「我们是真正懂你的开发者工具」的信号。

## Design Tokens

```css
:root {
  /* Backgrounds */
  --bg-body:      #FDFDF8;   /* 奶油白，NOT #ffffff */
  --bg-secondary: #F3F2EE;   /* 次级背景，section 分割 */
  --bg-card:      #FFFFFF;   /* 卡片用纯白，和奶油底形成层次 */
  --bg-dark:      #1E1F23;   /* 深色 section 用 */

  /* Text */
  --text-primary:   #4D4F46;   /* 不是黑，是暖灰 */
  --text-secondary: #747568;
  --text-muted:     #A0A193;
  --text-on-dark:   #F3F2EE;

  /* Orange accent system */
  --orange:         #EB9D2A;   /* 主 CTA 颜色 */
  --orange-hover:   #D4891F;
  --orange-shadow:  #B17816;   /* 3D 阴影颜色 */
  --orange-light:   #FEF3DC;   /* 浅橙背景，标签、高亮 */

  /* Secondary accent */
  --red-link:   #F54E00;   /* 链接、active 态 */
  --red-hover:  #D44200;

  /* Borders */
  --border-default: #BFC1B7;
  --border-subtle:  #E0DFD9;
  --border-strong:  #8A8C82;
}
```

## 背景层级规则

```
body:           #FDFDF8   ← 奶油底
section 交替:   #F3F2EE   ← 浅一级，形成节奏
card/panel:     #FFFFFF   ← 纯白，浮于奶油底上
dark section:   #1E1F23   ← 偶发深色区块（CTA、footer）
```

## CTA 按钮配色

```css
/* Primary — 橙色 3D 按钮 */
.btn-primary {
  background: #EB9D2A;
  color: #000000;            /* 黑字在橙底，不用白字 */
  border: 1px solid #B17816;
  box-shadow: 0 2px 0 #B17816;   /* 3D 感 */
}
.btn-primary:hover {
  background: #D4891F;
  box-shadow: 0 2px 0 #9A6A10;
}
.btn-primary:active {
  transform: translateY(2px);
  box-shadow: none;              /* 按下去时 shadow 消失 */
}

/* Secondary — 奶油底描边 */
.btn-secondary {
  background: #FDFDF8;
  color: #4D4F46;
  border: 1px solid #BFC1B7;
  box-shadow: 0 2px 0 #BFC1B7;
}
```

## Rules

- body 背景 **永远用 `#FDFDF8`**，不用 `#fff` 或 `#f9f9f9`
- 主文字颜色 **`#4D4F46`**（暖灰），不用 `#000` 或 `#333`
- 链接和 active 状态用 **`#F54E00`**（红橙），不用蓝色
- CTA 按钮用橙色 + **黑色文字**（不是白色）
- 暗色区块 `#1E1F23` 只用于 footer 或强调 CTA section
- Tag / badge 背景用 `--orange-light: #FEF3DC`

## Forbidden

- `background: white` 或 `background: #fff` 用在 body
- 蓝色系 accent（`#0066FF`、`#3B82F6` 等）
- 白色文字在橙色按钮上
- 纯黑文字 `#000000` 用于 body text
