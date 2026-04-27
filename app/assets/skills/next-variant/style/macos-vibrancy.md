# Style Skill: macOS Vibrancy

Inspired by: macOS Sonoma — 磨砂玻璃 Vibrancy、精准窗口阴影、Traffic Light 按钮、系统控件

## 设计哲学

macOS 的深度感来自三个来源：
1. **Vibrancy**：侧边栏/工具栏透过 `backdrop-filter` 模糊并吸收背后内容的颜色
2. **精准阴影**：窗口阴影不是一个模糊球，而是多层精确叠加
3. **材质层级**：每层 surface 只比下一层亮/暗一点，通过这种克制产生深度

## 窗口 Chrome

```css
/* macOS 窗口容器 */
.macos-window {
  border-radius: 12px;                /* Sonoma 窗口圆角 */
  overflow: hidden;
  box-shadow:
    0 0 0 0.5px rgba(0,0,0,0.18),    /* 窗口边框（极细描边）*/
    0 2px 6px rgba(0,0,0,0.12),      /* 近处阴影 */
    0 8px 24px rgba(0,0,0,0.12),     /* 中距阴影 */
    0 24px 64px rgba(0,0,0,0.16);    /* 远处扩散阴影 */
  background: #F5F5F5;
}

/* 深色模式 */
@media (prefers-color-scheme: dark) {
  .macos-window {
    box-shadow:
      0 0 0 0.5px rgba(255,255,255,0.08),
      0 2px 6px rgba(0,0,0,0.4),
      0 8px 24px rgba(0,0,0,0.4),
      0 24px 64px rgba(0,0,0,0.5);
    background: #1C1C1E;
  }
}
```

## Toolbar（标题栏）

```css
.macos-toolbar {
  height: 52px;                       /* 标准 toolbar 高度 */
  display: flex;
  align-items: center;
  padding: 0 16px;
  gap: 8px;

  /* Vibrancy */
  background: rgba(238, 238, 242, 0.85);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);

  /* 和 content 的分隔线 */
  border-bottom: 0.5px solid rgba(0,0,0,0.1);

  position: relative;
}

/* 窗口标题（居中绝对定位）*/
.macos-window-title {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.08px;
  color: rgba(0,0,0,0.85);
  white-space: nowrap;
}
```

## Traffic Light 按钮

```css
.traffic-lights {
  display: flex;
  gap: 8px;
  align-items: center;
}

.traffic-light {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  cursor: pointer;
  position: relative;
  transition: filter 0.1s;
}

.traffic-light-close   { background: #FF5F57; border: 0.5px solid rgba(0,0,0,0.12); }
.traffic-light-min     { background: #FFBD2E; border: 0.5px solid rgba(0,0,0,0.12); }
.traffic-light-max     { background: #28C840; border: 0.5px solid rgba(0,0,0,0.12); }

/* Hover: 显示图标（× − +）*/
.traffic-lights:hover .traffic-light::after {
  content: '';
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 8px;
  color: rgba(0,0,0,0.5);
}
.traffic-lights:hover .traffic-light-close::after  { content: '×'; }
.traffic-lights:hover .traffic-light-min::after    { content: '−'; }
.traffic-lights:hover .traffic-light-max::after    { content: '+'; }
```

## Sidebar（侧边栏 Vibrancy）

```css
.macos-sidebar {
  width: 220px;
  height: 100%;
  background: rgba(246, 246, 248, 0.85);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border-right: 0.5px solid rgba(0,0,0,0.1);
  padding-top: 8px;
  overflow-y: auto;
}

/* 侧边栏 item */
.sidebar-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px;
  border-radius: 6px;
  margin: 1px 8px;
  font-size: 13px;
  font-weight: 400;
  color: rgba(0,0,0,0.85);
  cursor: pointer;
  transition: background 0.1s;
}
.sidebar-item:hover    { background: rgba(0,0,0,0.05); }
.sidebar-item.selected {
  background: var(--accent, #007AFF);
  color: #FFFFFF;
}

/* Section Header */
.sidebar-section-header {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: rgba(0,0,0,0.35);
  padding: 12px 20px 4px;
}
```

## 系统按钮

```css
/* 主要按钮（蓝色填充）*/
.btn-system-primary {
  background: #007AFF;
  color: #FFFFFF;
  border: none;
  border-radius: 6px;
  padding: 5px 14px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: filter 0.1s;
}
.btn-system-primary:hover  { filter: brightness(1.08); }
.btn-system-primary:active { filter: brightness(0.92); }

/* 次要按钮（描边）*/
.btn-system-secondary {
  background: rgba(255,255,255,0.9);
  color: rgba(0,0,0,0.85);
  border: 0.5px solid rgba(0,0,0,0.2);
  border-radius: 6px;
  padding: 5px 14px;
  font-size: 13px;
  font-weight: 400;
  cursor: pointer;
  box-shadow: 0 1px 2px rgba(0,0,0,0.06);
  transition: background 0.1s;
}
.btn-system-secondary:hover  { background: rgba(255,255,255,1); }
.btn-system-secondary:active { background: rgba(240,240,240,1); }

/* Destructive 按钮（红色）*/
.btn-system-destructive {
  background: #FF3B30;
  color: #FFFFFF;
  border: none;
  border-radius: 6px;
  padding: 5px 14px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
```

## 输入框

```css
.macos-input {
  background: #FFFFFF;
  border: 0.5px solid rgba(0,0,0,0.2);
  border-radius: 6px;
  padding: 4px 8px;
  font-size: 13px;
  font-family: -apple-system, BlinkMacSystemFont, sans-serif;
  color: rgba(0,0,0,0.85);
  outline: none;
  transition: box-shadow 0.15s;
  box-shadow: 0 1px 2px rgba(0,0,0,0.06);
}

/* 聚焦态 — 蓝色光环 */
.macos-input:focus {
  border-color: #007AFF;
  box-shadow:
    0 0 0 3px rgba(0,122,255,0.25),
    0 1px 2px rgba(0,0,0,0.06);
}
```

## 列表 / Table Row

```css
.macos-list-row {
  display: flex;
  align-items: center;
  padding: 6px 16px;
  font-size: 13px;
  color: rgba(0,0,0,0.85);
  cursor: pointer;
  transition: background 0.05s;
}
.macos-list-row:hover    { background: rgba(0,0,0,0.04); }
.macos-list-row.selected {
  background: var(--accent, #007AFF);
  color: #FFFFFF;
}

/* 分隔线 */
.macos-list-row + .macos-list-row {
  border-top: 0.5px solid rgba(0,0,0,0.06);
}
```

## Popover / Sheet 阴影

```css
.macos-popover {
  background: rgba(255,255,255,0.92);
  backdrop-filter: blur(40px) saturate(180%);
  border: 0.5px solid rgba(0,0,0,0.15);
  border-radius: 10px;
  box-shadow:
    0 0 0 0.5px rgba(0,0,0,0.1),
    0 4px 12px rgba(0,0,0,0.12),
    0 16px 40px rgba(0,0,0,0.12);
  padding: 8px 0;
}

.macos-menu-item {
  padding: 5px 16px;
  font-size: 13px;
  cursor: pointer;
  border-radius: 4px;
  margin: 0 4px;
  transition: background 0.05s;
}
.macos-menu-item:hover {
  background: #007AFF;
  color: #FFFFFF;
}
```

## Rules

- 窗口圆角：**12px**（Sonoma 标准）
- 所有 vibrancy 元素必须用 `backdrop-filter: blur(20px) saturate(180%)`
- 分隔线：**0.5px**（1px 会太粗，macOS 用 hairline）
- 按钮 padding：`5px 14px`（不是 8px 或 12px）
- 所有 border 用 `0.5px` 或 `rgba` 而非 `1px solid #xxx`
- 窗口阴影分 4 层：hairline → 近 → 中 → 远扩散
- Focus ring：`box-shadow: 0 0 0 3px rgba(0,122,255,0.25)`（不用 outline）

## 窗口布局公式

```
┌─────────────────────────────────────────────────┐  ← border-radius: 12px
│  [● ● ●]    Window Title (centered)    [搜索]   │  ← Toolbar 52px, vibrancy
├──────────────┬──────────────────────────────────┤  ← 0.5px divider
│              │                                  │
│   SIDEBAR    │    CONTENT AREA                  │
│   220px      │                                  │
│   vibrancy   │    list rows / detail view       │
│              │                                  │
│              │                                  │
└──────────────┴──────────────────────────────────┘
```

## Forbidden

- `border-radius > 12px` 用于窗口（会变成 iOS 风格）
- `box-shadow: 0 4px 20px rgba(0,0,0,0.3)` 单层阴影（必须多层）
- `border: 1px solid #xxx`（用 `0.5px` 或 `rgba`）
- `outline: 2px solid blue` 作为 focus 态（用 `box-shadow` ring）
- 无 `backdrop-filter` 的侧边栏（失去 vibrancy 就失去 Mac 感）
- `border-radius: 999px` 用于按钮（macOS 按钮是 6px，不是胶囊形）
