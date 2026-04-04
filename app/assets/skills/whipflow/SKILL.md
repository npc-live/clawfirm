---
name: whipflow
version: 0.2.0
description: "WhipFlow DSL expert for writing .whip workflow scripts in clawfirm (AI Gateway). Use this skill whenever the user mentions whipflow, .whip files, wants to create an automated workflow, wants multiple AI agents to collaborate on a task, wants to orchestrate an AI pipeline or multi-step task, asks how to automate something with agents, or wants to write a workflow script. Also trigger when the user says things like 'write me a workflow', 'create a .whip file', 'how do I automate X with agents', 'make a pipeline for', or pastes a .whip file and asks for help."
---

# WhipFlow Skill

WhipFlow 是 clawfirm 内置的工作流 DSL，用于编排多步骤 AI agent 任务。文件扩展名 `.whip`，通过 clawfirm 桌面应用或 `whipflow_run` 工具执行。

## 架构：clawfirm 后端 + Canvas 前端

```
clawfirm（后端）                    Canvas（前端）
────────────────────────            ─────────────────────────────────
HTTP Gateway                        Chat Cell
  ws://127.0.0.1:{port}  ◄────────►   WebSocket 对话，用户发消息触发 whip
  /ws/{agent}/{sessionId}

~/.clawfirm/canvas/                 File Cell
  {name}.html  ◄── whip 写入          每 5 秒轮询，Preview tab 自动刷新
```

**Canvas 两种 Cell：**

| Cell | 用途 | 触发方式 |
|------|------|----------|
| **Chat Cell** | 对话 + 触发工作流 | 用户发消息 → agent 调用 `whipflow_run` |
| **File Cell** | 展示 workflow 输出 | whip 写 `~/.clawfirm/canvas/{name}.html` → 5s 自动刷新 |

**典型交互流程：**
1. 用户在 Chat Cell 发消息（或点击 canvas HTML 里的"一键复制"按钮后粘贴）
2. Agent 调用 `whipflow_run` 执行 .whip 文件
3. Whip 把结果渲染成 HTML 写入 `~/.clawfirm/canvas/{name}.html`
4. File Cell 自动刷新，展示最新结果

## 你的任务

根据用户描述，生成完整可运行的 `.whip` 文件。如果用户需要 Canvas 界面，同时生成对应的 HTML 输出（写入 canvas）。始终输出完整文件。

## 写入 Canvas 输出

Workflow 最后一步把结果渲染为 HTML 写入 canvas，供 File Cell 展示：

```whip
# ── 渲染并写入 Canvas ──
let page = session
  prompt: """
  把以下内容渲染成完整的 HTML 页面：
  {result}

  要求：
  - 完整 <!DOCTYPE html> 文档
  - 使用 Tailwind CSS class（已自动注入，无需引入）
  - 适合 620×520 的 iframe 展示
  - 包含"一键复制"按钮，方便用户触发下一个 workflow（见下方按钮模式）
  """

let _ = session
  prompt: "把以下内容写入文件 ~/.clawfirm/canvas/{canvas_name}.html：\n{page}"
```

## Canvas HTML 里的按钮触发 Whip

File Cell 是 `sandbox="allow-scripts"` 的 iframe，通过 `window.parent.postMessage` 触发 whip。
Canvas 会监听消息，自动通过 WebSocket 发给 agent 执行 `whipflow_run`。

**触发文件：**
```html
<button onclick="runWhip('~/.clawfirm/workflows/scan.whip')"
  class="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">
  开始扫描
</button>

<script>
function runWhip(file) {
  window.parent.postMessage({ type: 'run-whip', file }, '*');
}
</script>
```

**触发内联源码：**
```html
<button onclick="runWhipSource('session \"分析市场趋势\"')"
  class="px-4 py-2 bg-green-500 text-white rounded-lg">
  快速分析
</button>

<script>
function runWhipSource(source) {
  window.parent.postMessage({ type: 'run-whip', source }, '*');
}
</script>
```

postMessage 格式：`{ type: 'run-whip', file?: string, source?: string }`

## 语法速查

### 变量 & 输入
```
let x = "value"
const MAX = 10
ask topic: "请输入主题"      # 运行前弹出表单
```

### session（核心语句）
```
let result = session "你的问题"

let result = session: myagent
  prompt: "处理 {variable}"
```

### agent 定义
```
agent myagent:
  tools: ["bash", "read", "write", "web_search"]
  prompt: """
  专门化的系统提示词。
  """
```

> `provider` 和 `model` 可省略，默认使用 config.yml 中的 default agent。

### 控制流
```
if [result 表明成功]: print "ok" end

repeat 3 as i:
  let r = session "第 {i} 次"
end

loop max: 5:
  let r = session "继续"
  until [r 表明完成]
end

foreach item in items:
  let r = session "处理 {item}"
end

parallel:
  let a = session "任务 A"
  let b = session "任务 B"
end
```

### 管道 / 选择 / 错误处理
```
let results = items
  | map as item: session "处理 {item}" end
  | filter as r: [r 不为空] end

choice [根据类型]:
  "代码": let r = session: coder "分析"
  "文档": let r = session: writer "总结"
end

try:
  let r = session "可能失败的操作"
catch err:
  print "出错: {err}"
end
```

## 文件结构模板

```
# {filename}.whip
# 功能：说明这个 workflow 做什么
# 触发：用户在 Chat Cell 发送什么指令
# 输出：写入 ~/.clawfirm/canvas/{name}.html

[ask 语句（可选）]
[agent 定义（可选）]

# ── 核心步骤 ──
[workflow 步骤]

# ── 写入 Canvas ──
[渲染 HTML 并 write 文件]
```

## 最佳实践

**Canvas 联动**：有 UI 展示需求时，最后一步把结果写入 `~/.clawfirm/canvas/{name}.html`，配套建同名 File Cell。

**按钮交互**：canvas HTML 里用"一键复制"模式，用户点击复制指令后粘贴到 Chat Cell 触发下一步。

**多步流水线**：每个阶段（scan → evaluate → report）可以是独立的 .whip 文件，通过 canvas 按钮串联。

**并行加速**：互相独立的步骤用 `parallel`，特别适合多维度分析场景。

**状态持久化**：中间结果写 JSON 文件（`~/.clawfirm/canvas/data.json`），各步骤读写共享状态。

## 示例参考

详细示例见 `references/examples.md`
