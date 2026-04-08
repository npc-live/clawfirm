---
name: whipflow
version: 0.4.0
description: "WhipFlow DSL expert for writing .whip workflow scripts in clawfirm (AI Gateway). Use this skill whenever the user mentions whipflow, .whip files, wants to create an automated workflow, wants multiple AI agents to collaborate on a task, wants to orchestrate an AI pipeline or multi-step task, asks how to automate something with agents, or wants to write a workflow script. Also trigger when the user says things like 'write me a workflow', 'create a .whip file', 'how do I automate X with agents', 'make a pipeline for', or pastes a .whip file and asks for help."
---

# WhipFlow Skill

WhipFlow 是 clawfirm 内置的工作流 DSL，用于编排多步骤 AI agent 任务。
文件扩展名 `.whip`，通过 clawfirm 桌面应用或 `whipflow_run` 工具执行。

## 你的任务

1. 根据用户描述，**先生成完整的 `.whip` 文件**。
2. 如果用户还需要**交互式 Canvas 界面**（可视化展示、按钮触发），再按「Canvas 集成」章节生成配套 HTML。
3. 始终输出完整文件，不要省略内容。

## 运行规则

- **不要使用 TodoWrite**：用户指定了 .whip 文件或 whipflow 正在运行时，workflow 本身就是任务追踪，直接调用 `whipflow_run`，不需要先记录任何 todo
- 用户已提供 `.whip` 文件路径时，直接调用 `whipflow_run(mode="auto", file=...)`，无需重新生成
- 生成 .whip 后立即调用 `whipflow_run(mode="auto", source=...)`，**不要输出裸代码块**让用户看
- 当 `whipflow_run` 返回预览（`type="whipflow_preview"`）时：UI 已显示预览卡片和 Run 按钮，简短说明计划后**停止**，等用户点击 Run
- `ask` 变量自动填充：如果上下文已有对应值（用户提到过、或当前消息包含），直接填入 `user_inputs` 参数，不要再问用户
- **不要**在 .whip 中指定 provider（不要写 `provider: "claude-code"` 之类的硬编码）
- 当用户要求"写个 whipflow"时，生成文件后**不要自动运行**

---

## 语法参考

### 变量与输入

```whip
let x = "value"             # 可变变量
const MAX = 10              # 常量
x = "new value"             # 重新赋值

ask topic: "请输入主题"      # 运行前弹出表单，收集用户输入
```

### session（核心语句）

每个 session 启动一个 agent 子任务，返回其输出。

```whip
# 最简：直接发 prompt
let result = session "分析这段数据并给出结论"

# 指定具名 agent
let result = session: myagent
  prompt: "处理 {variable}"

# 字符串插值：{varname} 把变量嵌入 prompt
let report = session "根据以下数据生成报告：\n{result}"
```

### agent 定义

```whip
agent myagent:
  tools: ["bash", "read", "write", "web_search"]
  prompt: """
  你是一个专业的数据分析师。
  收到任务后，先拆解步骤，再逐步执行。
  """
```

> `provider` 和 `model` 可省略，默认使用 config.yml 中配置的 default agent。
> 需要调用网络、文件、代码时在 `tools` 里声明。

### 控制流

```whip
# 条件（自然语言描述，AI 判断）
if [result 表明执行成功]:
  print "完成"
else if [result 包含错误]:
  print "需要重试"
else:
  print result
end

# 固定次数
repeat 3 as i:
  let r = session "第 {i} 轮尝试"
end

# 循环直到条件满足（until 条件放在 loop 头部，pre-check：条件为真则退出）
loop until **r 表明任务完成** (max: 5):
  let r = session "继续处理"
end

# while
while [还有待处理的数据]:
  let r = session "处理下一批"
end

# 遍历集合
foreach item in items:
  let r = session "处理：{item}"
end

# 并行执行（互相独立的任务用这个，节省时间）
parallel:
  let a = session "分析维度 A"
  let b = session "分析维度 B"
  let c = session "分析维度 C"
end
```

### 管道

```whip
let results = items
  | map as item:
      session: worker "处理 {item}"
  end
  | filter as r:
      [r 质量合格且不为空]
  end
```

### 选择块（AI 动态路由）

```whip
choice [根据 {input} 的内容类型决定]:
  "是结构化数据":
    let r = session: analyst "统计分析"
  "是文本内容":
    let r = session: writer "提炼摘要"
  "其他":
    let r = session "通用处理"
end
```

### 错误处理

```whip
try:
  let r = session "调用外部 API"
catch err:
  print "出错: {err}"
  throw "无法继续"
finally:
  print "清理完毕"
end
```

### 其他语句

```whip
print variable              # 打印变量到输出
run "other.whip"            # 执行另一个 .whip 文件
throw "error message"       # 抛出错误，终止执行
```

---

## 文件模板

```whip
# {filename}.whip
# 功能：一句话说明
# 输入：需要哪些参数（ask 或外部文件）
# 输出：产出什么结果

# ── 收集输入（可选）──
ask param: "提示语"

# ── Agent 定义（可选）──
agent worker:
  tools: ["bash", "read", "write"]
  prompt: "角色描述"

# ── 步骤 ──
let step1 = session: worker
  prompt: "第一步：{param}"

let step2 = session: worker
  prompt: "第二步，基于以下结果：{step1}"

print step2
```

---

## Canvas 集成（按需）

**仅当用户需要可视化界面或点击按钮触发 workflow 时**，才生成 Canvas 配套代码。

### 架构

```
clawfirm（后端）                     Canvas（前端）
─────────────────────────            ──────────────────────────────
~/.clawfirm/canvas/{name}.html  ←──  File Cell：每 5s 轮询，自动刷新
  whip 把结果渲染成 HTML 写入 ↑

HTTP Gateway WebSocket          ←──  Chat Cell：对话触发 whipflow_run
```

- **File Cell**：绑定一个 HTML 文件，whip 写入后自动刷新展示结果
- **Chat Cell**：用户发消息 → agent 调用 `whipflow_run` 执行 .whip

### whip 写入 Canvas

在 workflow 最后一步把结果渲染成 HTML 写入 canvas：

```whip
# ── 渲染并写入 Canvas ──
let page = session
  prompt: """
  把以下内容渲染成完整 HTML 页面：
  {result}

  要求：
  - 完整 <!DOCTYPE html> 文档
  - 使用 Tailwind CSS class（自动注入，无需引入）
  - 适合 620×520 的 iframe 展示
  - 如需按钮触发下一步，使用 postMessage 模式（见下方）
  """

let _ = session
  prompt: "将以下 HTML 写入文件 ~/.clawfirm/canvas/{name}.html：\n{page}"
```

### Canvas HTML 按钮触发 Whip

File Cell 是 `sandbox="allow-scripts"` 的 iframe，通过 `window.parent.postMessage` 触发 whip，
Canvas 监听后自动通过 WebSocket 发给 agent 执行。

```html
<!-- 触发 .whip 文件 -->
<button onclick="runWhip('~/.clawfirm/workflows/scan.whip')"
  class="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">
  开始扫描
</button>

<!-- 触发内联 whip 代码 -->
<button onclick="runSource('session &quot;快速分析&quot;')"
  class="px-4 py-2 bg-green-500 text-white rounded-lg">
  快速分析
</button>

<script>
function runWhip(file)   { window.parent.postMessage({ type: 'run-whip', file }, '*'); }
function runSource(src)  { window.parent.postMessage({ type: 'run-whip', source: src }, '*'); }
</script>
```

postMessage 格式：`{ type: 'run-whip', file?: string, source?: string }`

### Canvas 最佳实践

- 多步流水线（scan → evaluate → report）每步是独立 .whip，通过 canvas 按钮串联
- 中间状态写 JSON 文件（如 `~/.clawfirm/canvas/state.json`），各步骤共享
- File Cell 配套 Chat Cell 放在旁边：Chat 触发执行，File 展示结果

---

## 详细示例

见 `references/examples.md`
