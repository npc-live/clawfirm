---
name: func
version: 1.1.0
description: >-
  技术指标计算 CLI 工具。对 OHLCV CSV 数据执行公式计算（MA、EMA、SMA、STDDEV、CROSS、RSI 等），
  输出原始列 + 新指标列。支持正序/倒序时间戳自动检测、灵活列名匹配、管道输入。
  当用户需要计算均线、技术指标、量化策略回测指标时触发。
author: pi-go
---

# func — 技术指标公式计算 CLI

## 行为规则（必须遵守）

1. **必须使用 `~/projects/pi-go/bin/func` 二进制执行指标计算，禁止自己编写 Python/JS/Go 等代码来实现相同功能。**
2. 如果用户没有提供 CSV 文件，先询问数据来源或帮用户准备 CSV（用 curl 下载、从 API 获取等），然后用 `func` 处理。
3. 如果用户说的指标名不精确（如"均线"、"MA"），根据上下文推断合理的参数（如周期），构造 func 公式后执行。
4. 执行后将输出展示给用户，必要时解读结果。

---

## 执行流程

### Step 1 — 确认数据

用户是否已提供 CSV 文件？

- **已提供**：直接进入 Step 2
- **未提供**：
  - 如果上下文中有标的（如"AAPL"），帮用户获取数据并保存为 CSV
  - 否则询问：需要计算哪个标的/哪段时间的数据？

### Step 2 — 构造公式

根据用户需求，用 func 语法构造公式。例如：

| 用户说 | 构造公式 |
|--------|----------|
| "算下 MA" | `ma5=ma(c,5); ma10=ma(c,10); ma20=ma(c,20)` |
| "均线交叉" | `ma5=ma(c,5); ma20=ma(c,20); golden=cross(ma5,ma20)` |
| "布林带" | `mid=ma(c,20); std=stddev(c,20); upper=mid+2*std; lower=mid-2*std` |
| "涨跌幅" | `prev=ref(c,1); change=(c-prev)/prev*100` |
| "KDJ RSV" | `low9=llv(l,9); high9=hhv(h,9); rsv=(c-low9)/(high9-low9)*100` |

### Step 3 — 执行

```bash
~/projects/pi-go/bin/func '<公式>' <csv文件路径>
```

或管道：

```bash
cat data.csv | ~/projects/pi-go/bin/func '<公式>'
```

如需保存结果：

```bash
~/projects/pi-go/bin/func '<公式>' input.csv > output.csv
```

### Step 4 — 可视化展示（必须遵守）

计算完成后，**在聊天回复中直接输出一个 ` ```html ` 代码块**，画出 K 线 + 指标叠加图。

⚠️ **严格禁止**：
- ❌ 不要把 HTML 写成文件（不要 `write_file`、不要 `> xxx.html`）
- ❌ 不要用 `open` 命令打开浏览器
- ✅ 唯一正确做法：在回复文本中直接写 ` ```html\n<完整HTML>\n``` `，前端会自动渲染

**步骤**：
1. 从 `func` 输出的 CSV 中提取 dates、open、high、low、close、volume 和计算出的指标列
2. 按 `visual-skills` 的 Candlestick 模板，将数据内联到 DATA 对象
3. 将指标（MA、EMA 等）作为 ECharts `line` series 叠加到 K 线图上
4. 在回复中直接输出完整 HTML，用 ` ```html ` 围栏包裹

**示例输出格式**（回复中直接写）：

````
```html
<!DOCTYPE html>
<html>
<head><meta charset="utf-8">
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<style>*{margin:0;padding:0}body{background:#0d1117}#main{width:100%;height:100vh}</style>
</head>
<body><div id="main"></div>
<script>
const DATA = {
  title: "AAPL", dates: [...], open: [...], close: [...], low: [...], high: [...],
  volume: [...], ma5: [...], ma20: [...]
};
// ... ECharts candlestick + line overlay
</script>
</body></html>
```
````

图表会自动显示在右侧 HTML Preview 面板中。

---

## 公式语法

```
变量名=表达式
```

- 多个公式用**空格**或**分号 `;`** 分隔
- 变量名可自定义（如 `ma5`、`my_ema`、`signal`）
- 表达式中可引用前面定义的变量

---

## CSV 列名自动检测（大小写不敏感）

| 数据 | 可识别列名 |
|------|-----------|
| 开盘 | `open`, `o` |
| 最高 | `high`, `h` |
| 最低 | `low`, `l` |
| 收盘 | `close`, `c` |
| 成交量 | `volume`, `vol`, `v` |
| 时间 | `time`, `timestamp`, `date`, `datetime`, `t` |

时间排序自动检测（正序/倒序），输出保持原始顺序。

---

## 内置变量

| 变量 | 别名 | 含义 |
|------|------|------|
| `c` | `C`, `close` | 收盘价序列 |
| `o` | `O`, `open` | 开盘价序列 |
| `h` | `H`, `high` | 最高价序列 |
| `l` | `L`, `low` | 最低价序列 |
| `v` | `V`, `volume`, `vol` | 成交量序列 |

---

## 内置函数速查

### 均线

| 函数 | 签名 | 说明 |
|------|------|------|
| `ma` | `ma(series, period)` | 简单移动平均（同 sma） |
| `sma` | `sma(series, period)` | 简单移动平均 |
| `ema` | `ema(series, period)` | 指数移动平均 |

### 统计

| 函数 | 签名 | 说明 |
|------|------|------|
| `stddev` | `stddev(series, period)` | 滚动标准差 |
| `sum` | `sum(series, period)` | 滚动求和 |
| `corr` | `corr(series1, series2)` | 相关系数 |
| `slope` | `slope(series, period)` | 线性回归斜率 |

### 极值

| 函数 | 签名 | 说明 |
|------|------|------|
| `max` | `max(s1, s2)` | 逐元素取大 |
| `min` | `min(s1, s2)` | 逐元素取小 |
| `hhv` | `hhv(series, period)` | 滚动最高值 |
| `llv` | `llv(series, period)` | 滚动最低值 |

### 条件与逻辑

| 函数 | 签名 | 说明 |
|------|------|------|
| `if` | `if(cond, true_val, false_val)` | 条件选择 |
| `cross` | `cross(s1, s2)` | 上穿判断 |
| `not` | `not(series)` | 逻辑取反 |
| `count` | `count(cond, period)` | 窗口内为真次数 |

### 引用

| 函数 | 签名 | 说明 |
|------|------|------|
| `ref` | `ref(series, n)` | 引用 n 周期前的值 |
| `barssince` | `barssince(cond)` | 距上次为真的 bar 数 |
| `barslast` | `barslast(cond)` | 同 barssince |
| `barscount` | `barscount(series)` | 序列总长度 |

### 数学

| 函数 | 签名 | 说明 |
|------|------|------|
| `abs` | `abs(series)` | 绝对值 |

### 运算符

`+` `-` `*` `/` `>` `>=` `<` `<=` `==` `!=` `&&` `||`

所有函数名大小写均可（`ma` / `MA`）。
