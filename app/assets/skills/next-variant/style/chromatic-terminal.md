# Style Skill: Chromatic Terminal

Source: lambda.ai — RGB 色差特效 + 终端 UI 风格 + 深色精密工程感

## 设计哲学

Lambda 的界面语言来自**终端文化**：代码编辑器的绿色字体、CRT 显示器的扫描线、早期计算机的像素字体，加上现代感的 RGB 色差（Chromatic Aberration）。不是装饰性的"科技感"，而是**直接借用工程工具的 UI 语言**，因为用户本身就是工程师。

## 核心特效：RGB 色差（直接来自源码）

```css
/* Lambda 原始 token */
:root {
  --box-shadow-rgb:
    0 0.99px 0 0 #ff0,
    0.99px 0 0 0 #0ff,
    1.98px 0.99px 0 0 #0f0,
    0 -0.99px 0 0 #00f,
    -0.99px 0 0 0 #f0f,
    -1.98px 0 0 0 #f00;

  --text-shadow-rgb:
    0 0.99px 0 #ff0,
    0.99px 0 0 #0ff,
    1.98px 0.99px 0 #0f0,
    0 -0.99px 0 #00f,
    -0.99px 0 0 #f0f,
    -1.98px 0 0 #f00;
}

/* 用法：仅用于 hero 文字或特效元素，不用于正文 */
.rgb-text   { text-shadow:  var(--text-shadow-rgb); }
.rgb-border { box-shadow:   var(--box-shadow-rgb); }
```

> **RGB 色差** 模拟 CRT 显示器的像素偏移，让文字/元素边缘出现彩色虚影——6 个方向各一条细线（黄/青/绿/蓝/品红/红）。

## 终端 UI 组件

### Terminal Block（命令行窗口）

```jsx
<div style={{
  background: '#0b0b0b',
  border: '1px solid #262625',
  borderRadius: '8px',
  overflow: 'hidden',
  fontFamily: '"Suisse Intl Mono", "Courier New", monospace',
}}>
  {/* 窗口标题栏 */}
  <div style={{
    background: '#252525',
    borderBottom: '1px solid #262625',
    padding: '10px 16px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  }}>
    {/* 伪 traffic lights（但是暗色） */}
    {['#5c5c57','#5c5c57','#5c5c57'].map((c, i) => (
      <div key={i} style={{ width: 10, height: 10, borderRadius: '50%', background: c }} />
    ))}
    <span style={{ fontSize: '11px', color: '#5c5c57', marginLeft: '8px', fontFamily: 'var(--font-sans)' }}>
      // Lambda Agent Terminal //
    </span>
  </div>

  {/* 内容区 */}
  <div style={{ padding: '20px 24px', fontSize: '13px', lineHeight: '1.8' }}>
    <div style={{ color: '#95948c' }}>&gt; Session ID: <span style={{ color: '#e7e6d9' }}>abc-123</span></div>
    <div style={{ color: '#95948c' }}>&gt; [<span style={{ color: '#99f599' }}>✓</span>] Agent handshake initialized</div>
    <div style={{ color: '#95948c' }}>&gt; [<span style={{ color: '#99f599' }}>✓</span>] Status: <span style={{ color: '#99f599' }}>READY</span></div>
    <div style={{ display: 'flex', alignItems: 'center', gap: '4px', marginTop: '8px' }}>
      <span style={{ color: '#6236f4' }}>&gt;</span>
      <span style={{ color: '#e7e6d9' }}>_</span>
      {/* 光标闪烁 */}
    </div>
  </div>
</div>
```

### Section Label（01 / 02 / 03 序号）

```jsx
<div style={{
  display: 'flex',
  alignItems: 'center',
  gap: '16px',
  marginBottom: '12px',
}}>
  <span style={{
    fontFamily: '"Suisse Intl Mono", monospace',
    fontSize: '12px',
    color: '#95948c',
    letterSpacing: '0.05em',
  }}>
    01
  </span>
  <div style={{
    flex: 1,
    height: '1px',
    background: '#262625',
  }} />
</div>
```

### 卡片（技术参数展示）

```css
.spec-card {
  background: #252525;
  border: 1px solid #42413e;
  border-radius: 8px;
  padding: 32px;
  position: relative;
  overflow: hidden;
  transition: border-color 0.2s;
}

.spec-card:hover {
  border-color: #6236f4;
}

/* 左上角发光角 */
.spec-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, #6236f4 0%, transparent 60%);
  opacity: 0;
  transition: opacity 0.2s;
}
.spec-card:hover::before { opacity: 1; }
```

### 发光 CTA Section

```css
.cta-glow-section {
  background: radial-gradient(
    ellipse 80% 60% at 50% 100%,
    rgba(98, 54, 244, 0.2) 0%,
    transparent 70%
  ),
  #0b0b0b;
  text-align: center;
  padding: 120px 24px;
}
```

## 过渡动画（来自源码）

```css
:root {
  --transition-snappy: 0.1s cubic-bezier(0.6, 0, 0.4, 1);  /* 快速弹性 */
  --transition-smooth: 0.4s cubic-bezier(0.6, 0, 0.4, 1);  /* 流畅 */
}

/* 标准 hover 过渡 */
.interactive {
  transition:
    color var(--transition-snappy),
    background var(--transition-snappy),
    border-color var(--transition-snappy);
}
```

## 文字选中颜色

```css
::selection      { background: #6236f4; color: #e7e6d9; }
::-moz-selection { background: #6236f4; color: #e7e6d9; }
```

## Focus Ring

```css
:focus-visible {
  outline: 2px solid #6236f4;
  border-radius: 0;   /* 不圆角，工程感 */
}
```

## Rules

- RGB 色差特效：**只用于 hero 大标题**，不用于 body 文字（会影响可读性）
- 终端 block：用于产品功能展示，代码必须用绿色 `#99f599`
- 卡片 hover：边框变为 `#6236f4` + 顶边渐变光（不用 box-shadow 泛光）
- Section 序号：01/02/03，等宽字体，`#95948c` 暗色
- 分隔线：`1px solid #262625` 极细暗线
- `border-radius: 8px` 用于卡片，`border-radius: 4px` 用于标签/按钮
- CTA section 底部：紫色 `radial-gradient` 发光效果从底部扩散
- 光标闪烁动画：`animation: blink 1s step-end infinite`

## Forbidden

- RGB 色差用于正文（只有 hero 装饰）
- `box-shadow` 多层扩散（Lambda 不用这个，用 border-color 变化代替深度）
- 圆角 > 8px（工程感不需要大圆角）
- 亮色背景（这个 style 专为深色系设计）
- 装饰性图标/插图（用代码/数据/规格取代）
