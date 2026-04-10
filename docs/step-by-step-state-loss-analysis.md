# WhipFlow Step-by-Step 状态丢失分析

## 症状

1. **Session 1 完成后，Continue 按钮点不了**
2. **点 Next Step 重新执行 Session 1，而不是 Session 2 — 状态丢失**

---

## Bug 1: Continue 按钮失效

### 调用链

```
ToolPanel.tsx:394  → onContinueSession(exec.id, idx, agentName)
ChatView.tsx:450   → OpenWhipflowSession(toolExecID, sessionIndex, agentName)
app.go:1811        → db.Messages().ListMessages(channelID="whipflow/"+toolExecID, userID=strconv.Itoa(sessionIndex))
```

### 根因: agentName 传错

`ToolPanel.tsx:730`:
```typescript
onContinueSession(exec.id, idx,
  whipflowArgs?.file?.split("/").pop()?.replace(".whip", "") ?? "")
```

`agentName` 是从 `.whip` 文件名推导的（如 `"workflow"`），而不是实际的 agent 名称（如 `"gpt4"`）。当使用 `source` 内联代码（不传 `file`）时，`agentName` 变成空字符串 `""`。

`OpenWhipflowSession` 用这个错误的 agentName 创建新 session：
```go
chID := "webchat/" + agentName  // "webchat/" 或 "webchat/workflow"
```

这导致新 session 找不到对应的 agent manager，或者创建在错误的 agent 下。

### 第二个条件：`has_history` 可能为 false

Continue 按钮只在 `step.has_history === true` 时显示（`ToolPanel.tsx:393`）。

`has_history` 来自 `WhipflowSessionStep.HasHistory`：
```go
HasHistory: p.Done && len(p.Messages) > 0,
```

只有 **NativeProvider**（Go 原生 agent）才会产生 Messages。如果 session 使用的是 **CliProvider**（外部 CLI 进程），`Messages` 始终为空，`has_history=false`，Continue 按钮不会出现。

---

## Bug 2: Next Step 重新执行 Session 0

### 调用链

```
ChatView.tsx:522  handleStepByStepNext()
  → 从 toolExecutions[].partialResult 收集 replayOutputs
  → send({ type: "run_tool", tool_args: { retry_from_session: 1, stop_after_session: 1, replay_outputs: [...] } })

handler.go:259    → 如果 replay_outputs 已存在，跳过 extractWhipflowReplayOutputs
agent.go:332      → ExecuteToolDirectly → whipflow_run.Execute
whipflow_run.go   → whipflow.Execute(program, WithRetryFromSession(1), WithReplaySessions(records), WithStopAfterSession(1))

interpreter.go:472  → 如果 sessionIndex < len(replaySessions)，replay；否则实际执行
```

### 根因链

#### ① Frontend replayOutputs 收集可能为空

`ChatView.tsx:534-536`:
```typescript
if (step && step.done && typeof step.index === "number" && step.index < next && step.output) {
  replayOutputs.push({ index: step.index, output: step.output });
}
```

条件 `step.output` 是 **truthy check**。但 `WhipflowSessionStep.Output` 的 JSON tag 是 `omitempty`：
```go
Output string `json:"output,omitempty"`
```

如果 `extractAgentOutput` 返回空字符串（比如 agent 只执行了 tool call 没有文本回复），则 `output` 字段在 JSON 中被省略，JS 端 `step.output` 为 `undefined`，被过滤掉。

**结果**：`replayOutputs` 为空数组，不会包含在 `tool_args` 里。

#### ② Backend fallback: extractWhipflowReplayOutputs 也找不到

当 frontend 没有发送 `replay_outputs` 时，`handler.go:260-265` 尝试从 message history 提取：

```go
if _, alreadyHasReplay := args["replay_outputs"]; !alreadyHasReplay {
    if replayOutputs := extractWhipflowReplayOutputs(sess.State().Messages); len(replayOutputs) > 0 {
```

`extractWhipflowReplayOutputs` 找最近一个 `whipflow_run` tool call 的 **ToolResultMessage**，然后在文本中找 `"JSON result (for replay_outputs on retry):"` marker 解析 `session_outputs`。

**问题**：在 step-by-step 模式下，Step 0 的 tool call ID 是 `"step-0-{timestamp}"`。Step 1 发起时，`sess.State().Messages` 包含 Step 0 的 AssistantMessage 和 ToolResultMessage。`extractWhipflowReplayOutputs` 应该能找到。

**但关键问题**：Step 0 执行时用的是 `stop_after_session=0`，所以 `whipflow.Execute` 只运行了 session 0 就停了。tool result 的 `session_outputs` 确实包含 session 0 的输出。**这条路径理论上应该工作**。

#### ③ 真正的 bug：replay_outputs 被发送但为空数组

回到 frontend，`handleStepByStepNext` 的逻辑：

```typescript
const replayOutputs: { index: number; output: string }[] = [];
// ... 收集逻辑 ...
send(JSON.stringify({
  tool_args: {
    retry_from_session: next,
    stop_after_session: next,
    ...(replayOutputs.length > 0 ? { replay_outputs: replayOutputs } : {}),
  },
}));
```

**当 `step.output` 为空字符串或 undefined 时**，`replayOutputs` 长度为 0，`replay_outputs` 不会包含在 args 里。

此时 handler.go fallback 到 `extractWhipflowReplayOutputs`。但如果这个也失败（可能 ToolResultMessage 的文本格式有变化，或 Details 字段没被正确序列化），则 **没有 replay_outputs 传给 whipflow 运行时**。

#### ④ retry_from_session 在 source 模式下完全无效

`whipflow.go:164-176` — `retryFromSession` 只在有 StateStore 时生效：

```go
if cfg.fileName != "" {                              // ← source 模式无 fileName！
    if prev, err := store.FindIncompleteRun(cfg.fileName); err == nil && prev != nil {
        if cfg.retryFromSession >= 0 {
            store.DeleteSessionsFrom(prev.ID, cfg.retryFromSession)
        }
        sessions, _ := store.GetCompletedSessions(prev.ID)
        interpOpts = append(interpOpts, runtime.WithReplaySessions(sessions))
    }
}
```

Step-by-step 模式使用 `source`（inline code），**没有 `fileName`**，所以：
- 不进入 state store 恢复逻辑
- `retry_from_session` 被设置了但完全没有效果
- 唯一能跳过 session 的是 `WithReplaySessions(records)` — 来自 `replay_outputs`

**结论**：`retry_from_session=1` + `replay_outputs` 为空 → `replaySessions` 长度为 0 → interpreter 对每个 session 都走实际执行路径 → session 0 被重新执行。

---

## 根因总结

| 编号 | Bug | 根因 | 位置 |
|------|-----|------|------|
| **1a** | Continue 失效 | `agentName` 从文件名推导，inline source 时为空 | `ToolPanel.tsx:730` |
| **1b** | Continue 不显示 | CliProvider session 没有 Messages，`has_history=false` | `whipflow_run.go:329` |
| **2a** | Next Step 重执行 | `output` 为空字符串时被 `omitempty` 省略，JS 端 truthy check 过滤掉 | `whipflow_run.go:96` + `ChatView.tsx:535` |
| **2b** | Fallback 也失败 | `retry_from_session` 没有独立于 `replay_outputs` 的 skip 机制 | `interpreter.go:472` |

## 核心设计缺陷

Step-by-step 模式的 replay 依赖链太脆弱：

```
Session 0 output → JSON omitempty → tool_update WS → partialResult accumulation
→ truthy check → replay_outputs array → retry_from_session + replaySessions → skip logic
```

任何一环断裂（output 为空、WS 消息丢失、状态未更新），整个 replay 链就失效，回退到从头执行。

## 修复方向

1. **Continue 按钮**：传入实际的 `agentName`（从 ChatView props 传入），不从文件名猜
2. **Output 空值**：把 `output` 的 JSON tag 去掉 `omitempty`，或者在 JS 端用 `step.done && step.index < next`（不检查 output 是否非空）
3. **replay_outputs 健壮性**：当 frontend 的 replay_outputs 为空时，确保 backend fallback 路径可靠工作
4. **根本方案**：让 `retry_from_session` 在 interpreter 层面直接跳过已完成的 session（不依赖 replay data），产出空 output 即可
