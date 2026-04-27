---
name: dev
version: 0.1.0
description: "Software development workflow dispatcher. Use when the user asks to write code, fix bugs, debug, run tests, review code, refactor, check PR readiness, improve coverage, do TDD, verify a feature, explore codebase, do QA, or run self-tests. Trigger phrases: 'help me code', 'fix this bug', 'debug', 'run tests', 'review my code', 'refactor', 'check PR', 'improve coverage', 'TDD', 'verify feature', 'explore codebase', 'QA', 'write tests', 'regression check', 'self-test', 'smart self-test', '帮我写代码', '修bug', '跑测试', '代码审查', '重构', '开始自测', '自测', '智能自测'."
---

# Dev Skill

根据用户的开发需求，自动选择并运行对应的 whipflow 工作流。

## 工作流路由表

根据用户意图匹配 **一个** 最合适的工作流：

| 意图 | 工作流 | 说明 |
|------|--------|------|
| 写代码 / 实现功能 / 改代码 | `dev-code.whip` | 架构师规划 + 开发者实现 + 验证 |
| 修 bug / 排错 / debug | `dev-debug.whip` | 定位根因 → 修复 |
| 跑测试 / 测试失败了 | `dev-test.whip` | 智能识别变更文件、跑测试、自动修复失败 |
| 写测试 / TDD | `dev-tdd.whip` | 先写测试(RED) → 实现代码(GREEN) → 循环 |
| 提高覆盖率 / 补测试 | `dev-coverage.whip` | 分析覆盖率缺口、自动补写测试 |
| 代码审查 / review | `dev-review.whip` | 审查未提交变更：正确性、安全、性能、风格 |
| 重构 | `dev-refactor.whip` | 分析影响 → 增量重构 → 验证不破坏 |
| PR 检查 / 合并前检查 | `dev-pr-check.whip` | 编译 + vet + 测试 + review 并行流水线 |
| QA / 全面质量检查 | `dev-qa.whip` | build → vet → test → coverage → 回归分析 |
| 回归测试 / 改动是否安全 | `dev-regression.whip` | 分析变更影响范围、运行受影响测试 |
| 验证功能 / 确认需求完成 | `dev-verify.whip` | 拆解验收点、逐条验证 |
| 了解代码 / 探索模块 | `dev-explore.whip` | 分析文件、类型、数据流、设计模式 |
| 自测 / 开始自测 / self-test | `dev-selftest.whip` | 启动 gateway + 运行 UI 自动化测试 + 日志分析 |
| 智能自测 / smart self-test | `dev-selftest-smart.whip` | 自适应环境 + 失败自修复 + AI 分析报告 |

## 执行规则

1. 分析用户消息，从路由表中选择 **最匹配** 的一个工作流
2. 如果用户意图模糊或跨多个类别，**先问用户**确认再执行
3. 立即调用 `whipflow_run`，使用 `source=` 内联模式（不要用 `file=`）：
   ```
   whipflow_run(mode="auto", source="let result = session \"执行任务描述\"")
   ```
4. 将用户的具体需求（要实现什么功能、修什么 bug、重构哪个模块等）通过 `user_inputs` 传入对应的 `ask` 变量
5. 当 `whipflow_run` 返回预览（`type="whipflow_preview"`）时，简短说明计划后**停止**，等用户确认

## 快捷触发示例

- "帮我实现 XXX 功能" → `dev-code.whip`
- "这个 bug 怎么回事" → `dev-debug.whip`
- "跑一下测试" → `dev-test.whip`
- "review 一下我的代码" → `dev-review.whip`
- "重构 store 包" → `dev-refactor.whip`
- "PR 能合吗" → `dev-pr-check.whip`
- "帮我用 TDD 写" → `dev-tdd.whip`
- "覆盖率太低了" → `dev-coverage.whip`
- "全面检查一下质量" → `dev-qa.whip`
- "这次改动会不会破坏别的" → `dev-regression.whip`
- "确认一下这个需求做完了没" → `dev-verify.whip`
- "帮我看看 agent 包怎么工作的" → `dev-explore.whip`
- "开始自测" / "自测" → `dev-selftest.whip`
- "智能测试" / "智能自测" → 直接执行：`whipflow_run(mode="execute", source="let r = session \\"运行 /Users/qing/projects/pi-go/scripts/selftest.sh 并报告结果\\"")`
