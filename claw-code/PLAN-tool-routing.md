# Tool Routing Plan: WhipFlow + TodoWrite 适配

## 现状分析

### System Prompt 构建流程

**文件:** `claw-code/rust/crates/runtime/src/prompt.rs`

`SystemPromptBuilder.build()` 按顺序拼接以下段落：

| # | 段落 | 函数 | 内容 |
|---|------|------|------|
| 1 | Intro | `get_simple_intro_section()` | "You are an interactive agent..." |
| 2 | Output Style | (可选) | 自定义输出风格 |
| 3 | System | `get_simple_system_section()` | 系统能力说明（6 条 bullet） |
| 4 | Doing Tasks | `get_simple_doing_tasks_section()` | 任务执行指导（6 条 bullet） |
| 5 | Actions | `get_actions_section()` | 操作安全性指导 |
| 6 | 分界线 | `SYSTEM_PROMPT_DYNAMIC_BOUNDARY` | 缓存标记 |
| 7 | Environment | `environment_section()` | model、cwd、date、platform |
| 8 | Project Context | `render_project_context()` | git status、instruction files |
| 9 | Instructions | `render_instruction_files()` | CLAUDE.md、.claw/instructions.md |
| 10 | Config | `render_config_section()` | settings.json 运行时配置 |
| 11 | Extra | `append_sections` | 外部追加的额外段落 |

**关键发现：当前 system prompt 中没有任何关于 TodoWrite、Skill、whipflow_run 工具的行为指导。**

### Tool 定义

**文件:** `claw-code/rust/crates/tools/src/lib.rs`

`mvp_tool_specs()` (line 384) 返回所有内置工具的 schema，发给 API 的 `tools` 字段：

| 工具 | 行号 | 描述 |
|------|------|------|
| `TodoWrite` | 529 | "Update the structured task list for the current session." |
| `Skill` | 557 | "Load a local skill definition and its instructions." |

描述都是一句话，**没有** "什么时候该用 / 不该用" 的行为指导。

### Skill 加载机制

- **启动时不注入** — skills 不在 system prompt 里
- **按需加载** — LLM 调用 `Skill` 工具时，`execute_skill()` (line 2939) 读取 SKILL.md 并返回全文
- **搜索路径:** `~/.clawfirm/skills/`, `~/.agents/skills/`, `~/.config/opencode/skills/`, `~/.codex/skills/`, `/home/bellman/.codex/skills/`
- **whipflow SKILL.md** 已有指导（line 20）：`不要使用 TodoWrite`

### 当前问题

1. LLM 不知道什么时候该用 `Skill` 工具去加载 whipflow skill
2. LLM 不知道什么时候该用 `TodoWrite` vs `whipflow_run`
3. 没有 "路由规则" 告诉 LLM：用户提到 `.whip` / `whipflow` / `harness` 时应该走 whipflow 分支

---

## 修改方案

### 修改点 1: `prompt.rs` — 新增 `get_tools_section()`

**位置:** `claw-code/rust/crates/runtime/src/prompt.rs`

在 `get_actions_section()` 之后、`SYSTEM_PROMPT_DYNAMIC_BOUNDARY` 之前插入新段落。

```rust
fn get_tools_section() -> String {
    let items = prepend_bullets(vec![
        "Use TodoWrite for tracking multi-step tasks, progress, and todo lists during normal work.".to_string(),
        "When the user mentions .whip files, whipflow, workflow DSL, harness, or wants to run/write/edit whipflow workflows, use the Skill tool to load the \"whipflow\" skill first, then follow its instructions. Do NOT use TodoWrite in this case — the workflow itself is the task tracker.".to_string(),
        "When the user provides a .whip file path or says \"whipflow run\", call whipflow_run directly without TodoWrite.".to_string(),
    ]);

    std::iter::once("# Tool routing".to_string())
        .chain(items)
        .collect::<Vec<_>>()
        .join("\n")
}
```

### 修改点 2: `prompt.rs` — 在 `build()` 中注入

**位置:** `SystemPromptBuilder::build()` (line 140)

```rust
// 现在：
sections.push(get_actions_section());
sections.push(SYSTEM_PROMPT_DYNAMIC_BOUNDARY.to_string());

// 改为：
sections.push(get_actions_section());
sections.push(get_tools_section());          // ← 新增
sections.push(SYSTEM_PROMPT_DYNAMIC_BOUNDARY.to_string());
```

### 修改点 3（可选）: 增强 TodoWrite 工具描述

**文件:** `claw-code/rust/crates/tools/src/lib.rs` line 530

```rust
// 现在：
description: "Update the structured task list for the current session.",

// 改为：
description: "Update the structured task list for the current session. Use this for general task tracking. Do not use when working with whipflow workflows.",
```

---

## 预期行为矩阵

| 用户输入 | 期望行为 | 工具调用链 |
|---------|---------|-----------|
| "帮我写个 .whip 文件" | 加载 whipflow skill → 生成 .whip | `Skill("whipflow")` → 生成代码 → `whipflow_run(mode="auto")` |
| "运行 scan.whip" | 直接执行 | `whipflow_run(file="scan.whip")` |
| "whipflow run media.whip" | 直接执行 | `whipflow_run(file="media.whip")` |
| "这个 harness 怎么写" | 加载 whipflow skill → 回答 | `Skill("whipflow")` → 回答语法问题 |
| "帮我做一个多步骤任务" | 用 TodoWrite 追踪 | `TodoWrite(todos=[...])` |
| "帮我重构这个模块" | 用 TodoWrite 追踪 | `TodoWrite(todos=[...])` |
| "写个工作流自动化发帖" | 加载 whipflow skill → 生成 .whip | `Skill("whipflow")` → 生成代码 |

---

## 测试计划

### Case 1: 用户提到 .whip 文件

```
输入: "帮我写一个 content-pipeline.whip"
期望: LLM 先调用 Skill("whipflow")，读取 SKILL.md，然后按规则生成 .whip 并调用 whipflow_run
不期望: LLM 调用 TodoWrite
```

### Case 2: 用户说 whipflow run

```
输入: "whipflow run ~/workflows/scan.whip"
期望: LLM 直接调用 whipflow_run(file="~/workflows/scan.whip")
不期望: LLM 调用 TodoWrite 或 Skill
```

### Case 3: 用户提到 harness

```
输入: "这个 harness 的语法怎么写"
期望: LLM 调用 Skill("whipflow")，然后回答语法问题
```

### Case 4: 普通任务

```
输入: "帮我重构 gateway 模块"
期望: LLM 使用 TodoWrite 追踪进度
不期望: LLM 调用 Skill("whipflow")
```

---

## 影响范围

| 文件 | 改动 | 风险 |
|------|------|------|
| `claw-code/rust/crates/runtime/src/prompt.rs` | 新增 `get_tools_section()` + 1 行调用 | 低：纯追加，不改现有逻辑 |
| `claw-code/rust/crates/tools/src/lib.rs` | TodoWrite description 增强（可选） | 极低：只改描述字符串 |
| `prompt.rs` 测试 | 更新 `renders_claude_code_style_sections` 测试 | 低：加一个 assert |

**不涉及的文件：** Go 代码、前端、SKILL.md、config.yml
