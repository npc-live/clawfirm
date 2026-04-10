# WhipFlow Step-by-Step 重构方案

## 现状问题

当前 step-by-step 模式绕过了 LLM 对话循环，由前端直接发 `run_tool` WebSocket 消息：

```
前端按钮 → run_tool WS 消息 → handler.go ExecuteToolDirectly → whipflow 执行
         ↑                                                          |
         └── 前端手动拼 replay_outputs（从 React state 提取）──────────┘
```

**问题**：
- replay 依赖链脆弱（output omitempty、React state 丢失、retry_from_session 在 source 模式无效）
- 前端维护大量状态（stepByStep、pendingRunTools、stepByStepRef、replayOutputs 拼装逻辑）
- 5 个不同的 `run_tool` 发送函数，7 个回调 prop，3 个状态变量
- 状态恢复困难（关闭对话 / app 重启后状态全丢）

## 新设计

### 核心思想

> `whipflow_run` 第一次调用是一次普通工具调用（preview/validate）。
> 之后页面上的全部按钮都是**对主对话的一次消息命令**，让 LLM 根据上下文决定下一步。

```
用户点击按钮 → 发送用户消息 → LLM 看到对话历史（含之前所有 tool result）
                                   → LLM 自己调用 whipflow_run（带正确参数）
                                   → tool 执行 → 结果写入对话历史
```

**状态全部在对话历史中**，不需要前端维护 replay chain。

### 数据模型

DB 只需保存一个 **tool call 执行链**（list 形状的 tree）：

```sql
CREATE TABLE IF NOT EXISTS whipflow_chain (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  TEXT NOT NULL,     -- "webchat/agent-name"
    user_id     TEXT NOT NULL,     -- session ID
    call_id     TEXT NOT NULL,     -- tool_call_id
    parent_id   TEXT,              -- 上一个 tool_call_id（首次为 NULL）
    session_idx INTEGER DEFAULT -1,-- 执行的 whipflow session index（-1=preview）
    status      TEXT DEFAULT 'running', -- running | done | error
    created_at  INTEGER NOT NULL
);
CREATE INDEX idx_whipflow_chain_lookup ON whipflow_chain(channel_id, user_id, call_id);
```

**用途**：
- 给定任意 `call_id`，可以向上遍历 `parent_id` 找到整条执行链
- 前端重建 tool panel 时，查这个表就知道执行顺序和状态
- LLM 不需要这个表 — 它直接从对话历史中看到所有 tool result

### 交互流程

#### 1. 初始调用（Preview）

用户在对话中发送 whip 代码或文件路径 → LLM 调用 `whipflow_run(mode: "preview")` → 返回分析结果（session 列表、复杂度、ask 字段）。

前端从 tool result 中渲染 preview 卡片。

#### 2. Execute All（一次执行全部）

用户点击 "Execute" → 发送消息：
```
Execute this workflow.
```
LLM 看到上一个 preview 结果 → 调用 `whipflow_run(mode: "execute", source: "...")`。

#### 3. Step-by-Step: 第一步

用户点击 "Step" → 发送消息：
```
Execute session 0 only (step-by-step mode).
```
LLM 调用 `whipflow_run(mode: "execute", stop_after_session: 0)`。

#### 4. Next Step

用户点击 "Next Step" → 发送消息：
```
Continue to next session (session 1).
```
LLM 看到对话历史中 session 0 的 tool result（包含 `session_outputs`）→ 调用 `whipflow_run(mode: "execute", retry_from_session: 1, stop_after_session: 1, replay_outputs: [从上一个 tool result 中提取])`。

**关键**：LLM 从对话历史中的 tool result JSON 提取 replay_outputs，不需要前端拼装。

#### 5. Retry

用户点击 "Retry Session 2" → 发送消息：
```
Retry session 2 (keep sessions 0-1 results).
```
LLM 从对话历史提取 session 0-1 的 outputs → 调用 `whipflow_run(retry_from_session: 2, replay_outputs: [...])`。

#### 6. Continue（打开 session 对话）

用户点击 "Continue" → 发送消息：
```
Open session 1's conversation for follow-up.
```
LLM 调用新工具 `whipflow_open_session(call_id: "xxx", session_index: 1)` 或复用现有 `OpenWhipflowSession`。

---

## 前端改动

### 删除的状态和函数

| 删除项 | 原位置 | 原因 |
|--------|--------|------|
| `stepByStep` state | ChatView.tsx:69 | 不再需要前端跟踪 step-by-step 模式 |
| `stepByStepRef` | ChatView.tsx:70 | 同上 |
| `pendingRunTools` | ChatView.tsx:71 | 不再有 run_tool 调用 |
| `handleRunUntilSession()` | ChatView.tsx:462 | 改为发消息 |
| `handleRetryFromSession()` | ChatView.tsx:481 | 改为发消息 |
| `handleStepByStep()` | ChatView.tsx:501 | 改为发消息 |
| `handleStepByStepNext()` | ChatView.tsx:522 | 改为发消息 |
| `handleConfirmWhipflow()` | ChatView.tsx:560 | 改为发消息 |
| step-by-step banner UI | ChatView.tsx:862-888 | 不需要了 |
| `lastWhipflowArgs` state | ChatView.tsx:66 | 不需要缓存 args |
| replayOutputs 拼装逻辑 | ChatView.tsx:527-539 | LLM 自己从 tool result 提取 |

### 保留的

| 保留项 | 原因 |
|--------|------|
| `toolExecutions` state | 仍需渲染 tool panel |
| `tool_start`/`tool_update`/`tool_end` 处理 | 仍需实时渲染 tool 执行状态 |
| whip plan inline banner (Step/Execute 按钮) | 改为发消息而非 run_tool |

### 新增

`ChatView.tsx` 新增一个统一的按钮命令发送函数：

```typescript
function sendCommand(command: string) {
  if (wsStatus !== "open" || isStreaming) return;
  setMessages(prev => [...prev, { role: "user", content: command }]);
  setIsStreaming(true);
  send(JSON.stringify({ type: "message", content: command }));
}
```

所有按钮回调改为调用 `sendCommand`：

```typescript
// ToolPanel props 简化
interface Props {
  executions: ToolExecution[];
  onCommand: (command: string) => void;  // 统一命令入口
}
```

ToolPanel 内部按钮：
```typescript
// Next Step
<button onClick={() => onCommand("Execute next session (session 2).")}>Next Step</button>

// Retry
<button onClick={() => onCommand(`Retry session ${idx}, keep results of sessions 0-${idx-1}.`)}>Retry</button>

// Continue
<button onClick={() => onCommand(`Open session ${idx} conversation for follow-up.`)}>Continue</button>

// Execute All
<button onClick={() => onCommand("Execute this workflow now.")}>Execute</button>
```

### ToolPanel Props 简化

**Before (7 callbacks + 1 state)**:
```typescript
interface Props {
  executions: ToolExecution[];
  onRetryFromSession?: (sessionIndex: number, args: WhipflowArgs) => void;
  onConfirmPreview?: (source: string, userInputs?: Record<string, string>) => void;
  onEditPreview?: (source: string) => void;
  onRunUntilSession?: (stopAfter: number, source: string, userInputs?: Record<string, string>) => void;
  onContinueSession?: (toolExecID: string, sessionIndex: number, agentName: string) => void;
  onStepByStep?: (source: string, userInputs?: Record<string, string>, totalSessions?: number) => void;
  whipflowArgs?: WhipflowArgs;
}
```

**After (1 callback)**:
```typescript
interface Props {
  executions: ToolExecution[];
  onCommand: (command: string) => void;
  onEditPreview?: (source: string) => void;  // 纯 UI 操作，保留
}
```

---

## 后端改动

### 1. Agent System Prompt 补充

给 agent 的 system prompt 增加 whipflow 操作指引，让 LLM 知道如何响应用户的步骤命令：

```
When the user asks to execute a whipflow step-by-step:
- For "execute next session N": call whipflow_run with mode="execute",
  stop_after_session=N, and extract replay_outputs from previous tool results.
- For "retry session N": call whipflow_run with retry_from_session=N,
  extracting replay_outputs for sessions 0..N-1 from prior tool results.
- Always include replay_outputs from previous whipflow_run results in the conversation.
```

**位置**: `app/app.go` 的 `buildTools` 或 agent system prompt 构建处。也可以直接写在 `whipflow_run` 工具的 description 中。

### 2. whipflow_chain 表

新增 migration `store/migrations/NNN_whipflow_chain.sql`：

```sql
CREATE TABLE IF NOT EXISTS whipflow_chain (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    call_id     TEXT NOT NULL UNIQUE,
    parent_id   TEXT,
    session_idx INTEGER DEFAULT -1,
    status      TEXT DEFAULT 'running',
    created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_whipflow_chain_lookup
    ON whipflow_chain(channel_id, user_id);
```

### 3. whipflow_run 工具增强

`tool/builtin/whipflow_run.go` 改动：

- 执行开始时写入 `whipflow_chain` 记录（status=running）
- 执行结束时更新 status（done/error）
- tool result JSON 中显式包含 `call_id` 和 `parent_call_id`，方便 LLM 和前端关联

新增 schema 参数：
```go
"parent_call_id": map[string]any{
    "type":        "string",
    "description": "The tool_call_id of the previous whipflow_run step. Used to build the execution chain.",
},
```

### 4. 新 Wails 绑定：GetWhipflowChain

```go
func (a *App) GetWhipflowChain(channelID, userID string) ([]WhipflowChainEntry, error)
```

前端 mount 时调用，获取当前对话的 whipflow 执行链，用于：
- 重建 tool panel 中的执行顺序
- 知道哪个是最新的 step，下一步该执行哪个 session

### 5. 删除 run_tool WebSocket 消息类型（可选）

当前 `handler.go` 支持 `run_tool` 消息类型用于直接工具调用。重构后这个不再需要（至少 whipflow 不需要）。可以保留给其他用途，但 whipflow 不再使用。

### 6. 清理

| 删除项 | 位置 | 原因 |
|--------|------|------|
| `extractWhipflowReplayOutputs()` | handler.go:82-148 | 不再需要从 message history 提取 replay |
| `OpenWhipflowSession()` | app.go:1811 | 改为 LLM 调用方式 |
| KV ledger (`tool_execs:*`) | app.go | 被 whipflow_chain 表替代 |
| sidecar files (`~/.clawfirm/whipflow-steps/`) | whipflow_run.go | 前端从 tool result 获取步骤数据，不需要 sidecar |

---

## 文件改动清单

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `store/migrations/NNN_whipflow_chain.sql` | **新增** | whipflow_chain 表 |
| `store/whipflow_chain.go` | **新增** | CRUD 操作 |
| `tool/builtin/whipflow_run.go` | **修改** | 写入 chain 记录，增加 parent_call_id 参数，tool description 增加 step-by-step 指引 |
| `app/app.go` | **修改** | 新增 `GetWhipflowChain` 绑定，删除 KV ledger 相关代码 |
| `channel/webchat/handler.go` | **修改** | 删除 `extractWhipflowReplayOutputs`，可选删除 `run_tool` 处理 |
| `ChatView.tsx` | **大幅简化** | 删除 5 个 run_tool 函数、3 个状态变量、step-by-step banner；新增 `sendCommand` |
| `ToolPanel.tsx` | **简化** | Props 从 7 callbacks → 1 `onCommand`；按钮改为发命令字符串 |
| `App.ts` (Wails bindings) | **修改** | 新增 `GetWhipflowChain`，删除旧 `GetToolExecutions` |

---

## 迁移策略

1. **Phase 1**: 新增 `whipflow_chain` 表 + store 层 CRUD
2. **Phase 2**: `whipflow_run` 工具写入 chain，增强 tool description
3. **Phase 3**: 前端重构 — 删除 run_tool 路径，改为 `sendCommand`
4. **Phase 4**: 清理旧代码（KV ledger、sidecar、extractWhipflowReplayOutputs）

Phase 1-2 可以和旧前端并行工作（向后兼容）。Phase 3 是 breaking change，一次切换。

---

## 效果对比

| 维度 | Before | After |
|------|--------|-------|
| 前端状态变量 | 6 个（stepByStep, stepByStepRef, pendingRunTools, lastWhipflowArgs, toolExecutions, whipAskValues） | 1 个（toolExecutions，从 tool events 自动填充） |
| 前端回调 | 7 个 ToolPanel props | 1 个 `onCommand` |
| 前端 run_tool 函数 | 5 个 | 0 个 |
| 状态恢复 | 复杂（KV ledger + sidecar + message scan） | 简单（对话历史 + whipflow_chain 表） |
| replay 可靠性 | 脆弱（omitempty、React state、retry_from_session 无效） | 健壮（LLM 从 tool result JSON 提取，永远在对话历史中） |
| 关闭/重开对话 | 状态丢失 | 无影响（状态在对话历史中） |
| App 重启 | 需要 crash recovery | 无需特殊处理 |
