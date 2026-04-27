# Color Skill: Terminal Violet

Source: lambda.ai CSS — `--color-terminal-*` / `--color-shell-*` / `--color-ultraviolet-*`

## 设计哲学

Lambda 的命名本身就是设计语言：**Terminal**（终端黑）= 背景，**Shell**（壳层暖白）= 前景文字。不是纯黑配纯白——两者都带一丝暖意，在高对比中保持温度。Accent 用极高饱和度的紫罗兰 `#6236f4`，在黑色底上电光一闪。

## Design Tokens（直接来自源码）

```css
:root {
  /* Terminal 调色板（深色） */
  --color-terminal-475:      #252525;
  --color-terminal-500:      #0b0b0b;   /* 主背景 */
  --color-terminal-1000:     #000000;
  --color-terminal-475-faded: rgba(37, 37, 37, 0.6);

  /* Shell 调色板（浅色文字/元素） */
  --color-shell-500:         #e7e6d9;   /* 主文字色 — 暖奶油白 */
  --color-shell-600:         #b9b8ae;   /* 次要文字 */
  --color-shell-600-light:   rgba(185, 184, 174, 0.2);
  --color-shell-700:         #8b8a8a;
  --color-shell-800:         #5c5c57;

  /* Neutral 灰阶（完整 10 级）*/
  --color-neutral-0:         #ffffff;
  --color-neutral-100:       #e7e6d9;
  --color-neutral-200:       #cccbbf;
  --color-neutral-300:       #b0afa6;
  --color-neutral-400:       #95948c;
  --color-neutral-500:       #797872;
  --color-neutral-600:       #5e5d58;
  --color-neutral-700:       #42413e;
  --color-neutral-800:       #262625;
  --color-neutral-900:       #0b0b0b;
  --color-neutral-1000:      #000000;

  /* Ultraviolet / Brand */
  --color-ultraviolet-300:   #a186f8;
  --color-ultraviolet-400:   #815ef6;
  --color-ultraviolet-500:   #6236f4;   /* 主 Accent */
  --color-ultraviolet-600:   #4e2bc3;
  --color-ultraviolet-800:   #271662;
  --color-ultraviolet-900:   #140b31;
  --color-violet-500-light:  #a17ff7;   /* hover/link 态 */

  /* Special */
  --color-code:              #99f599;   /* 绿色终端代码字 */
  --color-red-400:           #ec3333;

  /* Semantic aliases */
  --bg:                      var(--color-terminal-500);     /* #0b0b0b */
  --bg-elevated:             var(--color-terminal-475);     /* #252525 */
  --text-primary:            var(--color-shell-500);        /* #e7e6d9 */
  --text-secondary:          var(--color-shell-600);        /* #b9b8ae */
  --text-muted:              var(--color-neutral-400);      /* #95948c */
  --accent:                  var(--color-ultraviolet-500);  /* #6236f4 */
  --accent-light:            var(--color-violet-500-light); /* #a17ff7 */
  --link:                    var(--color-violet-500-light);
  --link-hover:              var(--color-ultraviolet-500);
}
```

## 按钮颜色

```css
/* Primary — 亮色按钮（shell 背景 + terminal 文字，反色） */
.btn-primary {
  background: #e7e6d9;
  color: #0b0b0b;
  border: none;
}

/* Secondary — 品牌紫按钮 */
.btn-secondary {
  background: #6236f4;
  color: #e7e6d9;
  border: none;
}
.btn-secondary:hover { background: #a17ff7; }

/* Tertiary — 透明描边 */
.btn-tertiary {
  background: transparent;
  color: #e7e6d9;
  border: 1px solid #e7e6d9;
}
```

## Section 背景交替规则

```
默认 section:        #0b0b0b   （terminal）
卡片/elevated:       #252525   （terminal-475）
品牌 section:        #6236f4   （violet，少用）
浅色 section（罕见）: #e7e6d9  （shell，用于对比）
```

## Rules

- body 背景永远是 `#0b0b0b`，不是 `#000`（带一点暖意）
- 主文字 `#e7e6d9`（Shell），不是 `#fff`（带米黄）
- Accent 只有一个：紫色 `#6236f4`，hover 变亮版 `#a17ff7`
- 链接颜色 `#a17ff7`（亮紫），不用蓝色
- 代码/终端文字用绿色 `#99f599`
- 选中文字 highlight 用 `#6236f4`

## Forbidden

- 蓝色系 accent
- 纯黑 `#000000` 作为主背景
- 纯白 `#ffffff` 作为主文字（用 `#e7e6d9`）
- 冷灰色（所有灰度都带暖棕色调）
- 多种 accent 颜色
