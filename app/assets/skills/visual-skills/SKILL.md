---
name: visual-skills
description: "Use this skill whenever the user provides data (CSV, JSON, table, pasted numbers, or any structured dataset) and expects a visual output — even if they don't say 'chart' or 'visualize'. Triggers include: uploading a CSV/Excel and asking 'what does this look like', 'show me this data', 'display this', 'plot this', or any phrasing implying visual understanding. Also triggers when the user pastes tabular data inline and asks for analysis or presentation. The skill reads the data, identifies its semantic type (financial OHLCV, time series, categorical comparison, composition, correlation, etc.), routes to the correct visual template, and renders a production-quality interactive chart artifact. Do NOT trigger for pure text analysis, data transformation without visualization, or when the user explicitly asks for a table/report instead of a chart."
---
 
# visual-skills
 
## Why this skill exists
 
When a user provides data, the naive response is to guess a chart type and write ad-hoc code.
This produces inconsistent, low-quality visuals. Every chart looks different. Libraries change.
Code breaks.
 
This skill solves that by:
1. **Semantically understanding** the data first (not just column names or file extensions)
2. **Routing** to a pre-defined visual template for that data type
3. **Mapping** the user's data into the stable template
4. **Rendering** a consistent, production-quality artifact
 
The LLM's job is **understanding + mapping**. The template's job is **rendering**.
 
---
 
## Protocol: How to use this skill
 
### Step 1 — Read the data
 
Load and inspect the data before doing anything else.
 
```python
import pandas as pd
 
# For CSV
df = pd.read_csv('/mnt/user-data/uploads/file.csv', nrows=5)
print(df.dtypes)
print(df.head())
 
# For Excel
df = pd.read_excel('/mnt/user-data/uploads/file.xlsx', nrows=5)
 
# For inline/pasted data — parse it into a DataFrame first
```
 
**Goal**: understand column names, data types, shape, and value ranges.
 
### Step 2 — Identify semantic type
 
Do NOT rely on file extension or column count alone. Read the data semantically.
 
Ask yourself:
- Does it have **open, high, low, close** (or similar price columns)? → **OHLCV / Candlestick**
- Is there a **time column + one numeric metric**? → **Time Series**
- Is there a **time column + multiple numeric metrics** (revenue, cost, profit by year)? → **Multi-Series / Financial Bar**
- Does it represent **parts of a whole** (%, share, allocation)? → **Composition**
- Does it compare **categories without time** (country A vs B vs C)? → **Categorical Comparison**
- Is it a **matrix or cross-tab** (rows × columns of numeric values)? → **Heatmap / Correlation**
- Does it have **two numeric axes** suggesting relationship? → **Scatter**
- Is it **geographic** (lat/lon, country codes, region names)? → **Map** *(out of scope — flag to user)*
 
### Step 3 — Route to template
 
Use the Dispatch Table below. Find the matching row and follow its template section.
 
### Step 4 — Map data to template
 
Each template defines a **data contract** — a minimal object structure it expects.
Extract the user's data into that contract. Example:
 
```javascript
// Template expects: { dates: [], open: [], high: [], low: [], close: [] }
// User's columns are: ['date', 'Open', 'High', 'Low', 'Close', 'Volume']
 
const chartData = {
  dates: df['date'],
  open:  df['Open'],
  high:  df['High'],
  low:   df['Low'],
  close: df['Close']
}
```
 
### Step 5 — Render artifact
 
Inject the mapped data into the template code. Output the complete HTML as a ` ```html ` fenced code block **directly in your chat reply**.
⚠️ **禁止**写入文件或用 `open` 打开浏览器。前端会自动检测 ` ```html ` 代码块并渲染到 HTML Preview 面板。
 
---
 
## Dispatch Table
 
| Semantic Type | Key Signals | Template |
|---|---|---|
| **OHLCV / Candlestick** | Columns: open/high/low/close (any casing), price data with OHLC pattern | `→ TEMPLATE: Candlestick` |
| **Time Series (single)** | datetime col + 1 numeric metric, continuous over time | `→ TEMPLATE: TimeSeries` |
| **Time Series (multi)** | datetime col + 2–6 numeric metrics side by side | `→ TEMPLATE: MultiLine` |
| **Financial Bar (period comparison)** | Year/quarter labels + revenue/cost/profit style columns | `→ TEMPLATE: FinancialBar` |
| **Composition / Part-of-whole** | Rows sum to ~100%, or user asks "breakdown", "share", "allocation" | `→ TEMPLATE: Donut` |
| **Categorical Comparison** | String category col + 1–2 numeric metrics, no time | `→ TEMPLATE: Bar` |
| **Ranking** | Ordered list with scores/values, user asks "top N", "ranked" | `→ TEMPLATE: HorizontalBar` |
| **Correlation / Matrix** | Square-ish numeric matrix, heatmap language, correlation coefficients | `→ TEMPLATE: Heatmap` |
| **Scatter / Relationship** | Two independent numeric axes, no clear time dimension | `→ TEMPLATE: Scatter` |
| **Distribution** | Single numeric column, user asks "distribution", "spread", "histogram" | `→ TEMPLATE: Histogram` |
 
**If ambiguous between two types**: prefer the one with more semantic signal from the column names.
**If genuinely unclear**: ask the user one question before proceeding. Do not guess wildly.
 
---
 
## Templates
 
All templates use **ECharts** via CDN. Reasons:
- Single `<script>` tag, no bundler needed
- Handles all chart types in the dispatch table
- Consistent API, consistent output
- Works in HTML artifacts without React
 
**ECharts CDN**: `https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js`
 
Each template has:
- **Data Contract** — what shape of data to inject
- **Complete HTML** — copy, inject data, done
 
---
 
### TEMPLATE: Candlestick
 
**Data Contract**
```javascript
const DATA = {
  title: "Stock Name or Ticker",        // string
  dates: ["2024-01-01", "2024-01-02"],  // string[]
  open:  [100, 102],                    // number[]
  close: [102, 101],                    // number[]
  low:   [99,  100],                    // number[]
  high:  [104, 103],                    // number[]
  volume: [1200000, 980000]             // number[] — optional, omit key if unavailable
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { background: #0d1117; font-family: -apple-system, sans-serif; }
    #main { width: 100%; height: 100vh; }
  </style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
const chart = echarts.init(document.getElementById('main'), 'dark')
 
const hasVolume = DATA.volume && DATA.volume.length > 0
 
const option = {
  backgroundColor: '#0d1117',
  animation: true,
  tooltip: {
    trigger: 'axis',
    axisPointer: { type: 'cross' },
    formatter: params => {
      const c = params.find(p => p.seriesType === 'candlestick')
      if (!c) return ''
      const [o, cl, l, h] = c.value
      return `<b>${c.name}</b><br/>
        Open: ${o}<br/>High: ${h}<br/>Low: ${l}<br/>Close: ${cl}`
    }
  },
  legend: { data: [DATA.title], top: 10, textStyle: { color: '#aaa' } },
  grid: hasVolume
    ? [{ left: 60, right: 20, top: 50, height: '60%' },
       { left: 60, right: 20, top: '75%', height: '15%' }]
    : [{ left: 60, right: 20, top: 50, bottom: 60 }],
  xAxis: [
    {
      type: 'category', data: DATA.dates, gridIndex: 0,
      axisLabel: { color: '#666', fontSize: 11 },
      axisLine: { lineStyle: { color: '#222' } }
    },
    ...(hasVolume ? [{
      type: 'category', data: DATA.dates, gridIndex: 1,
      axisLabel: { show: false }
    }] : [])
  ],
  yAxis: [
    {
      scale: true, gridIndex: 0,
      splitLine: { lineStyle: { color: '#1a1a2e' } },
      axisLabel: { color: '#666' }
    },
    ...(hasVolume ? [{
      scale: true, gridIndex: 1,
      splitNumber: 2,
      axisLabel: {
        color: '#666', fontSize: 10,
        formatter: v => v >= 1e6 ? (v/1e6).toFixed(1)+'M' : v
      },
      splitLine: { show: false }
    }] : [])
  ],
  dataZoom: [
    { type: 'inside', xAxisIndex: hasVolume ? [0,1] : [0], start: 60, end: 100 },
    { type: 'slider',  xAxisIndex: hasVolume ? [0,1] : [0], bottom: 10,
      height: 20, borderColor: '#333', fillerColor: 'rgba(100,100,255,0.1)',
      handleStyle: { color: '#555' }, textStyle: { color: '#666' } }
  ],
  series: [
    {
      name: DATA.title, type: 'candlestick',
      xAxisIndex: 0, yAxisIndex: 0,
      data: DATA.dates.map((_, i) => [DATA.open[i], DATA.close[i], DATA.low[i], DATA.high[i]]),
      itemStyle: {
        color: '#26a69a', color0: '#ef5350',
        borderColor: '#26a69a', borderColor0: '#ef5350'
      }
    },
    ...(hasVolume ? [{
      name: 'Volume', type: 'bar',
      xAxisIndex: 1, yAxisIndex: 1,
      data: DATA.volume,
      itemStyle: {
        color: params =>
          DATA.close[params.dataIndex] >= DATA.open[params.dataIndex]
            ? 'rgba(38,166,154,0.5)' : 'rgba(239,83,80,0.5)'
      }
    }] : [])
  ]
}
 
chart.setOption(option)
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
### TEMPLATE: TimeSeries
 
**Data Contract**
```javascript
const DATA = {
  title: "Metric Name",
  xLabel: "Date",          // x-axis label
  yLabel: "Value",         // y-axis label
  dates: ["2024-01", ...], // string[]
  values: [120, 135, ...], // number[]
  unit: "$"                // optional prefix/suffix hint
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { background: #0f172a; font-family: -apple-system, sans-serif; }
    #main { width: 100%; height: 100vh; }
  </style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
const chart = echarts.init(document.getElementById('main'), 'dark')
 
chart.setOption({
  backgroundColor: '#0f172a',
  title: { text: DATA.title, textStyle: { color: '#e2e8f0', fontSize: 16 }, left: 20, top: 20 },
  tooltip: { trigger: 'axis', formatter: p => `${p[0].name}<br/>${p[0].marker}${DATA.unit||''}${p[0].value}` },
  grid: { left: 70, right: 30, top: 70, bottom: 60 },
  xAxis: {
    type: 'category', data: DATA.dates,
    axisLabel: { color: '#64748b', rotate: DATA.dates.length > 20 ? 45 : 0 },
    axisLine: { lineStyle: { color: '#1e293b' } }
  },
  yAxis: {
    type: 'value', name: DATA.yLabel,
    nameTextStyle: { color: '#64748b' },
    axisLabel: { color: '#64748b', formatter: v => `${DATA.unit||''}${v}` },
    splitLine: { lineStyle: { color: '#1e293b' } }
  },
  dataZoom: [{ type: 'inside' }, { type: 'slider', bottom: 5, height: 18, borderColor: '#1e293b', fillerColor: 'rgba(99,102,241,0.15)' }],
  series: [{
    name: DATA.title, type: 'line', data: DATA.values,
    smooth: true, symbol: 'none',
    lineStyle: { color: '#6366f1', width: 2.5 },
    areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
      colorStops: [{ offset: 0, color: 'rgba(99,102,241,0.3)' }, { offset: 1, color: 'rgba(99,102,241,0)' }] } }
  }]
})
 
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
### TEMPLATE: MultiLine
 
**Data Contract**
```javascript
const DATA = {
  title: "Revenue vs Cost vs Profit",
  dates: ["2020", "2021", "2022", "2023"],  // string[]
  series: [
    { name: "Revenue", values: [400, 520, 610, 780] },
    { name: "Cost",    values: [280, 340, 390, 450] },
    { name: "Profit",  values: [120, 180, 220, 330] }
  ],
  yLabel: "USD (millions)"
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>* { margin:0;padding:0;box-sizing:border-box; } body{background:#0f172a;} #main{width:100%;height:100vh;}</style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
const PALETTE = ['#6366f1','#22d3ee','#f59e0b','#10b981','#f43f5e','#a78bfa']
const chart = echarts.init(document.getElementById('main'), 'dark')
 
chart.setOption({
  backgroundColor: '#0f172a',
  title: { text: DATA.title, textStyle: { color: '#e2e8f0', fontSize: 16 }, left: 20, top: 20 },
  tooltip: { trigger: 'axis' },
  legend: { data: DATA.series.map(s => s.name), top: 55, textStyle: { color: '#94a3b8' } },
  grid: { left: 70, right: 30, top: 90, bottom: 60 },
  xAxis: { type: 'category', data: DATA.dates, axisLabel: { color: '#64748b' }, axisLine: { lineStyle: { color: '#1e293b' } } },
  yAxis: { type: 'value', name: DATA.yLabel, nameTextStyle: { color: '#64748b' }, axisLabel: { color: '#64748b' }, splitLine: { lineStyle: { color: '#1e293b' } } },
  dataZoom: [{ type: 'inside' }],
  series: DATA.series.map((s, i) => ({
    name: s.name, type: 'line', data: s.values,
    smooth: true, symbol: 'circle', symbolSize: 5,
    lineStyle: { color: PALETTE[i % PALETTE.length], width: 2 },
    itemStyle: { color: PALETTE[i % PALETTE.length] }
  }))
})
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
### TEMPLATE: FinancialBar
 
**Data Contract**
```javascript
const DATA = {
  title: "Annual Financial Summary",
  periods: ["FY2021", "FY2022", "FY2023", "FY2024"],  // string[]
  series: [
    { name: "Revenue", values: [400, 520, 610, 780], stack: null },
    { name: "OpEx",    values: [200, 250, 290, 330], stack: "cost" },
    { name: "CapEx",   values: [80,  90,  100, 120], stack: "cost" }
  ],
  yLabel: "USD (M)"
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>* { margin:0;padding:0;box-sizing:border-box; } body{background:#0f172a;} #main{width:100%;height:100vh;}</style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
const PALETTE = ['#6366f1','#22d3ee','#f59e0b','#10b981','#f43f5e','#a78bfa']
const chart = echarts.init(document.getElementById('main'), 'dark')
 
chart.setOption({
  backgroundColor: '#0f172a',
  title: { text: DATA.title, textStyle: { color: '#e2e8f0', fontSize: 16 }, left: 20, top: 20 },
  tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
  legend: { data: DATA.series.map(s => s.name), top: 55, textStyle: { color: '#94a3b8' } },
  grid: { left: 70, right: 30, top: 90, bottom: 50 },
  xAxis: { type: 'category', data: DATA.periods, axisLabel: { color: '#64748b' }, axisLine: { lineStyle: { color: '#1e293b' } } },
  yAxis: { type: 'value', name: DATA.yLabel, nameTextStyle: { color: '#64748b' }, axisLabel: { color: '#64748b' }, splitLine: { lineStyle: { color: '#1e293b' } } },
  series: DATA.series.map((s, i) => ({
    name: s.name, type: 'bar', data: s.values,
    stack: s.stack || undefined,
    barMaxWidth: 50,
    itemStyle: { color: PALETTE[i % PALETTE.length], borderRadius: s.stack ? 0 : [3,3,0,0] }
  }))
})
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
### TEMPLATE: Donut
 
**Data Contract**
```javascript
const DATA = {
  title: "Portfolio Allocation",
  centerLabel: "Total",
  items: [
    { name: "Equities",    value: 45 },
    { name: "Bonds",       value: 30 },
    { name: "Real Estate", value: 15 },
    { name: "Cash",        value: 10 }
  ]
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>* { margin:0;padding:0;box-sizing:border-box; } body{background:#0f172a;} #main{width:100%;height:100vh;}</style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
const chart = echarts.init(document.getElementById('main'), 'dark')
chart.setOption({
  backgroundColor: '#0f172a',
  title: { text: DATA.title, textStyle: { color: '#e2e8f0', fontSize: 16 }, left: 20, top: 20 },
  tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
  legend: { orient: 'vertical', right: 30, top: 'center', textStyle: { color: '#94a3b8' } },
  series: [{
    type: 'pie', radius: ['45%', '70%'], center: ['40%', '55%'],
    label: { formatter: '{b}\n{d}%', color: '#94a3b8' },
    emphasis: { label: { fontSize: 14, fontWeight: 'bold' } },
    data: DATA.items
  }]
})
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
### TEMPLATE: Bar
 
**Data Contract**
```javascript
const DATA = {
  title: "Sales by Region",
  yLabel: "Revenue ($M)",
  categories: ["North", "South", "East", "West", "Central"],
  series: [
    { name: "2023", values: [120, 95, 140, 88, 110] },
    { name: "2024", values: [145, 110, 165, 102, 130] }  // optional second series
  ]
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>* { margin:0;padding:0;box-sizing:border-box; } body{background:#0f172a;} #main{width:100%;height:100vh;}</style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
const PALETTE = ['#6366f1','#22d3ee','#f59e0b','#10b981']
const chart = echarts.init(document.getElementById('main'), 'dark')
chart.setOption({
  backgroundColor: '#0f172a',
  title: { text: DATA.title, textStyle: { color: '#e2e8f0', fontSize: 16 }, left: 20, top: 20 },
  tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
  legend: { data: DATA.series.map(s => s.name), top: 55, textStyle: { color: '#94a3b8' } },
  grid: { left: 70, right: 30, top: 90, bottom: 50 },
  xAxis: { type: 'category', data: DATA.categories, axisLabel: { color: '#64748b' }, axisLine: { lineStyle: { color: '#1e293b' } } },
  yAxis: { type: 'value', name: DATA.yLabel, nameTextStyle: { color: '#64748b' }, axisLabel: { color: '#64748b' }, splitLine: { lineStyle: { color: '#1e293b' } } },
  series: DATA.series.map((s, i) => ({
    name: s.name, type: 'bar', data: s.values, barMaxWidth: 40,
    itemStyle: { color: PALETTE[i % PALETTE.length], borderRadius: [3,3,0,0] }
  }))
})
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
### TEMPLATE: HorizontalBar
 
**Data Contract**
```javascript
const DATA = {
  title: "Top 10 Companies by Revenue",
  xLabel: "Revenue ($B)",
  // Sorted descending — top item appears at top of chart
  items: [
    { name: "Apple",   value: 394 },
    { name: "Samsung", value: 244 },
    { name: "Google",  value: 283 }
    // ...
  ]
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>* { margin:0;padding:0;box-sizing:border-box; } body{background:#0f172a;} #main{width:100%;height:100vh;}</style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
// Sort ascending for ECharts (it renders bottom-to-top)
const sorted = [...DATA.items].sort((a, b) => a.value - b.value)
const chart = echarts.init(document.getElementById('main'), 'dark')
chart.setOption({
  backgroundColor: '#0f172a',
  title: { text: DATA.title, textStyle: { color: '#e2e8f0', fontSize: 16 }, left: 20, top: 20 },
  tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
  grid: { left: 120, right: 50, top: 70, bottom: 50 },
  xAxis: { type: 'value', name: DATA.xLabel, nameTextStyle: { color: '#64748b' }, axisLabel: { color: '#64748b' }, splitLine: { lineStyle: { color: '#1e293b' } } },
  yAxis: { type: 'category', data: sorted.map(d => d.name), axisLabel: { color: '#94a3b8', fontSize: 12 }, axisLine: { lineStyle: { color: '#1e293b' } } },
  series: [{
    type: 'bar', data: sorted.map(d => d.value), barMaxWidth: 28,
    label: { show: true, position: 'right', color: '#64748b', fontSize: 11 },
    itemStyle: { color: { type: 'linear', x: 0, y: 0, x2: 1, y2: 0,
      colorStops: [{ offset: 0, color: '#6366f1' }, { offset: 1, color: '#22d3ee' }] },
      borderRadius: [0,3,3,0] }
  }]
})
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
### TEMPLATE: Heatmap
 
**Data Contract**
```javascript
const DATA = {
  title: "Correlation Matrix",
  rows: ["AAPL", "GOOGL", "MSFT", "AMZN"],   // string[]
  cols: ["AAPL", "GOOGL", "MSFT", "AMZN"],   // string[]
  // values[i] = row index, values[j] = col index, values[v] = numeric
  values: [
    [0, 0, 1.00], [0, 1, 0.72], [0, 2, 0.68], [0, 3, 0.55],
    [1, 0, 0.72], [1, 1, 1.00], [1, 2, 0.81], [1, 3, 0.63],
    // ...
  ],
  min: -1,   // color scale min
  max: 1     // color scale max
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>* { margin:0;padding:0;box-sizing:border-box; } body{background:#0f172a;} #main{width:100%;height:100vh;}</style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
const chart = echarts.init(document.getElementById('main'), 'dark')
chart.setOption({
  backgroundColor: '#0f172a',
  title: { text: DATA.title, textStyle: { color: '#e2e8f0', fontSize: 16 }, left: 20, top: 20 },
  tooltip: { position: 'top', formatter: p => `${DATA.rows[p.data[1]]} / ${DATA.cols[p.data[0]]}: <b>${p.data[2]}</b>` },
  grid: { left: 80, right: 80, top: 70, bottom: 80 },
  xAxis: { type: 'category', data: DATA.cols, splitArea: { show: true }, axisLabel: { color: '#64748b', rotate: 30 } },
  yAxis: { type: 'category', data: DATA.rows, splitArea: { show: true }, axisLabel: { color: '#64748b' } },
  visualMap: {
    min: DATA.min ?? 0, max: DATA.max ?? 1,
    calculable: true, orient: 'horizontal', bottom: 10, left: 'center',
    inRange: { color: ['#ef5350','#fff','#26a69a'] },
    textStyle: { color: '#64748b' }
  },
  series: [{
    type: 'heatmap', data: DATA.values,
    label: { show: true, fontSize: 10, color: '#1e293b', formatter: p => p.data[2].toFixed ? p.data[2].toFixed(2) : p.data[2] },
    emphasis: { itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.5)' } }
  }]
})
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
### TEMPLATE: Scatter
 
**Data Contract**
```javascript
const DATA = {
  title: "Price vs Volume",
  xLabel: "Volume",
  yLabel: "Price ($)",
  // Optional: groupBy for color-coded categories
  series: [
    { name: "Tech",    points: [[120, 45], [340, 89], [210, 67]] },
    { name: "Finance", points: [[80,  32], [190, 51], [440, 99]] }
  ]
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>* { margin:0;padding:0;box-sizing:border-box; } body{background:#0f172a;} #main{width:100%;height:100vh;}</style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
const PALETTE = ['#6366f1','#22d3ee','#f59e0b','#10b981','#f43f5e']
const chart = echarts.init(document.getElementById('main'), 'dark')
chart.setOption({
  backgroundColor: '#0f172a',
  title: { text: DATA.title, textStyle: { color: '#e2e8f0', fontSize: 16 }, left: 20, top: 20 },
  tooltip: { trigger: 'item', formatter: p => `${p.seriesName}<br/>${DATA.xLabel}: ${p.data[0]}<br/>${DATA.yLabel}: ${p.data[1]}` },
  legend: { data: DATA.series.map(s => s.name), top: 55, textStyle: { color: '#94a3b8' } },
  grid: { left: 70, right: 30, top: 90, bottom: 60 },
  xAxis: { type: 'value', name: DATA.xLabel, nameTextStyle: { color: '#64748b' }, axisLabel: { color: '#64748b' }, splitLine: { lineStyle: { color: '#1e293b' } } },
  yAxis: { type: 'value', name: DATA.yLabel, nameTextStyle: { color: '#64748b' }, axisLabel: { color: '#64748b' }, splitLine: { lineStyle: { color: '#1e293b' } } },
  series: DATA.series.map((s, i) => ({
    name: s.name, type: 'scatter', data: s.points, symbolSize: 8,
    itemStyle: { color: PALETTE[i % PALETTE.length], opacity: 0.8 }
  }))
})
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
### TEMPLATE: Histogram
 
**Data Contract**
```javascript
const DATA = {
  title: "Return Distribution",
  xLabel: "Return (%)",
  yLabel: "Frequency",
  bins: [-5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5],  // bin edges, length = bars+1
  counts: [2, 5, 12, 28, 45, 38, 22, 10, 4, 1]    // length = bins-1
}
```
 
**Template**
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>* { margin:0;padding:0;box-sizing:border-box; } body{background:#0f172a;} #main{width:100%;height:100vh;}</style>
</head>
<body>
<div id="main"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/echarts/5.4.3/echarts.min.js"></script>
<script>
// ── INJECT DATA HERE ──────────────────────────────────────────
const DATA = { /* ... */ }
// ─────────────────────────────────────────────────────────────
 
const labels = DATA.bins.slice(0,-1).map((b,i) => `${b} to ${DATA.bins[i+1]}`)
const chart = echarts.init(document.getElementById('main'), 'dark')
chart.setOption({
  backgroundColor: '#0f172a',
  title: { text: DATA.title, textStyle: { color: '#e2e8f0', fontSize: 16 }, left: 20, top: 20 },
  tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
  grid: { left: 70, right: 30, top: 70, bottom: 60 },
  xAxis: { type: 'category', data: labels, name: DATA.xLabel, nameTextStyle: { color: '#64748b' }, axisLabel: { color: '#64748b', rotate: 30, fontSize: 10 }, axisLine: { lineStyle: { color: '#1e293b' } } },
  yAxis: { type: 'value', name: DATA.yLabel, nameTextStyle: { color: '#64748b' }, axisLabel: { color: '#64748b' }, splitLine: { lineStyle: { color: '#1e293b' } } },
  series: [{
    type: 'bar', data: DATA.counts, barCategoryGap: '1%',
    itemStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
      colorStops: [{ offset: 0, color: '#6366f1' }, { offset: 1, color: 'rgba(99,102,241,0.3)' }] },
      borderRadius: [2,2,0,0] }
  }]
})
window.addEventListener('resize', () => chart.resize())
</script>
</body>
</html>
```
 
---
 
## Routing Edge Cases
 
### User provides raw numbers without context
Ask one clarifying question before routing:
> "I can see this looks like [time series / financial data / ...]. Should I show it as [X] or [Y]?"
 
Never render something ambiguous silently.
 
### Data has unexpected shape
- **Too many columns (7+)**: Identify the 3–5 most semantically meaningful ones. Mention which were dropped.
- **Mixed time and non-time columns**: Use time axis if present; ignore unrelated columns.
- **Missing values**: Fill forward or interpolate for time series. For bars, show as 0 and note the gap.
- **Negative values in bar charts**: Use diverging color (positive = indigo, negative = rose).
 
### Data is too large for an artifact
If the dataset has >10,000 rows:
1. Sample or aggregate before injecting (e.g., daily → weekly → monthly)
2. Tell the user: "Aggregated to [N] data points for performance"
 
### No file — user describes data verbally
Generate representative mock data that matches the description, then render with the appropriate template.
Make it clear in the artifact title: "(Sample Data — replace with your own)"
 
---
 
## Output Rules
 
1. **Always output HTML as a ` ```html ` fenced code block directly in your chat reply** — never write to a file, never use `open` command. The frontend auto-renders ` ```html ` blocks in a preview panel
2. **Always use dark theme** (`#0f172a` or `#0d1117` background) unless user specifies light
3. **Always include `window.addEventListener('resize', () => chart.resize())`**
4. **Always include dataZoom** for time series and candlestick charts
5. **Title in the chart** must match the data context — not a generic "Chart"
6. **Axis labels** must reflect real column names or inferred semantic names
7. **Do not add UI controls or dropdowns** unless the user asks for interactivity
 
---
 
## Anti-Patterns to Avoid
 
| ❌ Wrong | ✅ Right |
|---|---|
| Using Chart.js for candlesticks | ECharts only — consistent library |
| Generating different background colors each time | Always dark theme by default |
| Asking "what kind of chart do you want?" before reading the data | Read data first, route, render |
| Hardcoding sample data and not replacing it | Always inject the actual user data |
| Outputting only the `option` object | Always output a complete HTML file |
| Using a pie chart for time series | Follow the dispatch table strictly |