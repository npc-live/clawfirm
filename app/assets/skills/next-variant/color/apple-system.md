# Color Skill: Apple System

Inspired by: macOS Ventura/Sonoma — 系统级颜色语义，Light/Dark 双模式，蓝色 Accent

## 设计哲学

Apple 的颜色系统不是"选一个好看的颜色"，而是**语义化层级**：每一层 surface 有固定的亮度差，通过堆叠 surface 层产生深度，而不是靠阴影。Accent 颜色（蓝色）极克制，只用在可交互元素上。

## Design Tokens — Light Mode

```css
:root {
  /* Window / Surface 层级（越往上越白） */
  --bg-window:       #ECECEC;   /* 窗口后面的桌面/壁纸区 */
  --bg-app:          #F5F5F5;   /* App 主背景（content area）*/
  --bg-sidebar:      rgba(246, 246, 248, 0.85);  /* 侧边栏 vibrancy */
  --bg-toolbar:      rgba(238, 238, 242, 0.85);  /* toolbar vibrancy */
  --bg-card:         #FFFFFF;   /* 卡片 / 面板 */
  --bg-popover:      rgba(255, 255, 255, 0.92);  /* 弹出层 */
  --bg-input:        #FFFFFF;
  --bg-hover:        rgba(0, 0, 0, 0.05);
  --bg-selected:     rgba(0, 122, 255, 0.12);    /* 选中行 */

  /* Text */
  --text-primary:    rgba(0, 0, 0, 0.85);   /* 主要文字 */
  --text-secondary:  rgba(0, 0, 0, 0.55);   /* 次要文字 */
  --text-tertiary:   rgba(0, 0, 0, 0.35);   /* 占位符/禁用 */
  --text-on-accent:  #FFFFFF;

  /* Accent — 系统蓝 */
  --accent:          #007AFF;
  --accent-hover:    #0066D6;
  --accent-pressed:  #0055B3;
  --accent-light:    rgba(0, 122, 255, 0.12);

  /* Borders */
  --border-window:   rgba(0, 0, 0, 0.12);   /* 窗口边框 */
  --border-default:  rgba(0, 0, 0, 0.1);    /* 普通分隔线 */
  --border-subtle:   rgba(0, 0, 0, 0.06);   /* 极细分隔 */
  --border-input:    rgba(0, 0, 0, 0.2);    /* 输入框边框 */

  /* Semantic colors */
  --color-red:     #FF3B30;
  --color-green:   #34C759;
  --color-yellow:  #FFCC00;
  --color-orange:  #FF9500;
  --color-purple:  #AF52DE;
  --color-pink:    #FF2D55;
  --color-teal:    #5AC8FA;
}
```

## Design Tokens — Dark Mode

```css
@media (prefers-color-scheme: dark) {
  :root {
    --bg-window:   #1C1C1E;
    --bg-app:      #1C1C1E;
    --bg-sidebar:  rgba(40, 40, 42, 0.85);
    --bg-toolbar:  rgba(44, 44, 46, 0.85);
    --bg-card:     #2C2C2E;
    --bg-popover:  rgba(50, 50, 52, 0.95);
    --bg-input:    rgba(255, 255, 255, 0.05);
    --bg-hover:    rgba(255, 255, 255, 0.07);
    --bg-selected: rgba(0, 122, 255, 0.25);

    --text-primary:   rgba(255, 255, 255, 0.85);
    --text-secondary: rgba(255, 255, 255, 0.55);
    --text-tertiary:  rgba(255, 255, 255, 0.25);

    --border-window:  rgba(255, 255, 255, 0.08);
    --border-default: rgba(255, 255, 255, 0.08);
    --border-subtle:  rgba(255, 255, 255, 0.04);
    --border-input:   rgba(255, 255, 255, 0.18);

    --accent:       #0A84FF;   /* 深色模式蓝略亮 */
    --accent-hover: #2196FF;
  }
}
```

## Surface 层级规则

```
桌面壁纸
  └─ 窗口背景 (#F5F5F5 / #1C1C1E)
      ├─ 侧边栏 (vibrancy blur，略深)
      ├─ Toolbar (vibrancy blur，和侧边栏同色)
      └─ Content area
          ├─ 列表行（hover: rgba(0,0,0,0.05)）
          ├─ 选中行（rgba(0,122,255,0.12)）
          └─ 卡片/面板（#FFFFFF）
              └─ Popover/Sheet（白色 + 阴影）
```

## Rules

- Accent 颜色 `#007AFF` **只用于**：聚焦状态、选中、主要按钮、链接
- 文字颜色用 **rgba 而非 hex**（便于在任何背景上保持对比度）
- Sidebar 和 Toolbar 必须用 `rgba` + `backdrop-filter`（vibrancy 效果）
- 选中状态用蓝色 `rgba(0,122,255,0.12)` 填充行背景 + 全行高亮
- 语义颜色（红/绿/黄）只用于状态指示，不用于装饰

## Forbidden

- 纯黑/纯白文字（用 rgba 层）
- Accent 颜色用于非交互元素（标题、装饰色）
- 多种 Accent 颜色共存（系统只有一个 Accent）
- `background: #000` 或 `background: #fff` 用于背景（永远用语义 token）
