# 让 AI 应用自己找 Bug、自己修 Bug

> 我们在 [Clawfirm](https://clawfirm.com) 里实现了一个"自检"系统——应用可以模拟用户行为、分析自己的运行日志、然后让 AI agent 自动修复发现的问题。这是整个过程的记录。

---

## 一个让人抓狂的 Bug

几周前我们发现了一个偶现的 hang：用户发了消息，然后点了"停止"（关闭 WebSocket 连接），再重新连上来发新消息——没有任何响应。

堆栈没有 panic，日志看起来一切正常，但 agent 就是卡住了。

根因找到之后很简单：

```go
// 修复前：没有 deadline，客户端断开后 WriteMessage 会永久阻塞
writeMu.Lock()
conn.WriteMessage(websocket.TextMessage, b)
writeMu.Unlock()

// 修复后：5 秒 deadline，写失败就释放锁
conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
conn.WriteMessage(websocket.TextMessage, b)
conn.SetWriteDeadline(time.Time{})
```

三行代码。但找到这个 bug 花了不少时间——因为它只在特定的操作序列下才复现：**发消息 → 断开 → 重连 → 发消息**。

这让我们意识到：**需要一套能系统性复现这类"用户行为序列"的机制**，而不是靠肉眼盯日志。

---

## 三层检测

我们搭了三个工具，分别覆盖不同维度。

### 1. Scenario Runner：回放用户行为

`cmd/scenario-runner` 用 YAML 描述用户操作序列，通过 WebSocket 实际执行并断言结果。

```yaml
# scenarios/send_then_stop.yaml
name: send_then_stop
description: |
  复现 hang bug：发消息 → 断开连接（模拟用户点 Stop）→ 重连 → 发新消息
  预期：新消息在 30s 内得到响应。

steps:
  - send:
      type: message
      content: "列出当前目录下所有文件"

  - wait: 2000

  - disconnect: true   # 模拟用户点 Stop

  - wait: 500

  - connect:
      session_id: "s-stop-test"

  - send:
      type: message
      content: "你先做方案"

  - expect:
      event: done
      timeout: 35000

  - assert:
      got_done: true
      got_response: true
      done_within: 32000
```

关键在于 `disconnect` + `reconnect` 这个步骤组合——这正是那个 bug 的触发路径。有了这个场景，bug 100% 可复现，修复也 100% 可验证。

目前有四个场景：
- `basic_chat`：基础问答，验证整个链路通
- `send_then_stop`：复现 WebSocket hang bug
- `tool_then_stop`：工具执行过程中停止
- `rapid_fire`：快速连发消息，测并发稳定性

### 2. LogCheck：分析运行时日志

`cmd/logcheck` 读取 `app.log`，用正则匹配关键事件，找异常模式。

检测的问题类型：
- **turn_stuck**：agent turn 开始但超过 90s 没有 done
- **time_gap**：相邻两行日志间隔超过 10s（说明某个操作卡住了）
- **emit_blocked**：LLM 返回了 tool_use，但 tool_start 事件迟迟没有触发
- **queue_delay**：新消息收到，但 agent 很久之后才开始处理

```
$ go run ./cmd/logcheck --since 30m ~/.clawfirm/app.log
✘ [critical] 14:23:07  turn_stuck: turn 3 started but no done after 95s
⚠ [warning]  14:19:42  time_gap: 12.3s gap at handler.go:87
✔ No other anomalies
```

支持 `--format json` 输出，方便机器读取。

### 3. 静态分析：编译 + vet + 快速测试

```bash
go build ./...
go vet ./...
go test -count=1 -timeout 60s ./agent/... ./tool/... ./gateway/...
```

这层最简单，但也最致命——编译错误、类型错误、race condition 基本都在这里暴露。

---

## 把三层串起来：Bug Hunter

光有工具还不够，我们需要一个能"自动跑 → 自动分析 → 自动修"的流水线。

这就是 `bughunter.whip`，一个用 [WhipFlow](https://whipflow.dev) DSL 写的自动化 pipeline：

```
# 简化版流程
阶段 1：运行所有场景 → /tmp/bughunter/scenarios.json
阶段 2：检测运行日志 → /tmp/bughunter/log_findings.json
阶段 3：静态分析    → /tmp/bughunter/static.txt

loop until **所有 critical 问题解决** (max: 3):
  let fix = session: fix-agent
    prompt: """
    读取 findings 文件，修复 severity=critical 的问题。
    修复后运行 go build ./... 验证。
    输出 {"fixed": [...], "remaining": [...]}
    """
  验证（重新跑静态分析 + logcheck）

阶段 5：生成最终报告 → /tmp/bughunter/report.md
```

`fix-agent` 是一个配置了 `bash/read/write/edit` 工具的 Claude session，它拿到 findings 之后：
1. 读取涉及的源码文件
2. 理解上下文和调用链
3. 生成最小化 patch
4. 运行 `go build ./...` 验证不引入新问题

最多循环 3 轮，每轮结束后重新跑检测，直到 findings 清空或无法继续。

---

## 实际运行效果

我们第一次跑 bughunter 的时候，`send_then_stop` 场景稳定失败（35s 超时）。

logcheck 同时报告了 `emit_blocked`：agent 在 WebSocket 断开后卡在了 `WriteMessage`，导致事件队列全部堆积。

fix-agent 定位到 `channel/webchat/handler.go`，加上了 `SetWriteDeadline(5 * time.Second)`，重新编译通过，场景测试变绿。

整个过程大约 3 分钟，人工介入为零。

---

## 一些有意思的设计决策

**场景 YAML 描述的是"意图"，不是"实现"**

`disconnect: true` 这一步，runner 会直接关掉 WebSocket 连接，不管底层用的是 gorilla 还是 nhooyr。YAML 文件永远不需要改。

**logcheck 刻意做成无状态的**

每次调用都从头扫日志，支持 `--since` 过滤时间窗口。这样 bughunter 可以先跑一次拿 baseline，修复后缩窗口再跑，只看最近 5 分钟。

**fix-agent 的 prompt 里有明确的验证步骤**

不能只让 agent "修复代码"，必须要求它在修完之后运行 `go build ./...`。否则 agent 会生成语法正确但类型不匹配的代码，在下一轮才被发现。

**loop 的退出条件是自然语言**

WhipFlow 的 `loop until **...** (max: 3)` 语法，退出条件由同一个 agent 来判断（Discretion token），而不是硬编码的 boolean。这样 agent 可以在"技术上 findings 还有，但都是已知的低优先级 warning"时提前退出。

---

## 开发工作流里的其他 `.whip` 文件

bughunter 是其中最重的 pipeline。我们还有一套轻量的开发辅助 workflow：

| 文件 | 用途 |
|------|------|
| `dev-debug.whip` | 给一个具体 bug，两个 agent 协作定位+修复 |
| `dev-code.whip` | 功能开发：spec → 代码 → 测试 |
| `dev-review.whip` | 代码 review，生成 PR 描述 |
| `dev-qa.whip` | 功能验收，人工 + 自动测试结合 |
| `dev-tdd.whip` | TDD 流程：先写失败测试，再实现 |

这些 workflow 都在 `app/assets/workflows/dev/` 里，随应用打包分发，在 Clawfirm 里可以直接运行。

---

## 局限和接下来

**logcheck 有误报**

多轮 agent 对话中，每个 turn 都被独立判断是否"完成"，结果中间的 turn 被误报为 `turn_stuck`。需要加上对 `agent done after N turns` 日志的关联。

**场景覆盖率不够**

目前只有 4 个场景，覆盖的是我们已知的 bug 路径。流式响应中途断开、工具调用超时、多 session 并发这些还没有场景。

**fix-agent 处理不了架构问题**

能修三行代码的 bug，但如果问题出在整体设计（比如事件队列的背压机制缺失），agent 会在一个错误的方向上反复尝试。这种情况需要人工介入。

---

## 代码

所有代码都在 [github.com/ai-gateway/clawfirm](https://github.com/ai-gateway/clawfirm)：

- `cmd/scenario-runner/` — WebSocket 场景测试器
- `cmd/logcheck/` — app.log 异常检测
- `scenarios/` — YAML 测试场景
- `app/assets/workflows/dev/bughunter.whip` — 完整 pipeline
- `scripts/selftest.sh` — 独立测试环境启动脚本

---

*Clawfirm 是一个 AI Gateway 桌面应用，内置 agent 引擎、WhipFlow workflow runtime 和一套开发者工具。*
