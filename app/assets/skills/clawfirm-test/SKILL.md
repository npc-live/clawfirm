---
name: clawfirm-test
description: "Testing and automating the Clawfirm desktop app UI using the browser-agent CLI (NOT agent-browser). ⚠️ browser-agent ≠ agent-browser — they are different binaries: browser-agent talks to Clawfirm's WKWebView eval server; agent-browser uses Chrome/CDP for external websites. Use THIS skill (browser-agent) whenever the user wants to: test a Clawfirm feature, automate the Clawfirm UI, send messages in Clawfirm, test '打开新对话', run dual-instance tests (Primary drives Secondary), or control Clawfirm's WKWebView. Trigger on: 'test the app', '帮我测试', 'UI test', 'browser-agent', 'clawfirm-test', or any request involving Clawfirm app automation."
---

# Clawfirm UI Testing with browser-agent

> ⚠️ **`browser-agent` ≠ `agent-browser`**
> - `browser-agent` → Clawfirm WKWebView，通过 eval server（port 9310/9311）注入 JS，**本 skill 使用此工具**
> - `agent-browser` → Chrome/Chromium via CDP，用于外部网站自动化，**不适用于 Clawfirm**

## 环境检查

运行任何测试前，先确认环境：

```bash
# 1. 确认 binary 存在
which browser-agent || go install ./cmd/browser-agent/

# 2. 确认 Clawfirm app 正在运行（eval server 在 9310）
curl -s --noproxy localhost -X POST http://localhost:9310/api/eval \
  -H 'Content-Type: application/json' \
  -d '{"script":"document.title"}' | python3 -m json.tool
```

若 app 未运行，用 testserver 代替（指向任意 URL）：

```bash
# 在独立终端运行，会自动打开 agent-browser 并启动 mock eval server
go run ./cmd/browser-agent/testserver/ http://localhost:9988

# 或测试外部页面
go run ./cmd/browser-agent/testserver/ https://example.com
```

## 核心工作流

**始终先 snapshot 获取 refs，再操作。**

```bash
# Step 1: 查看当前界面交互元素
browser-agent snapshot -i

# Step 2: 用 ref 操作（refs 是 @e1 @e2 这样的格式）
browser-agent click @e3
browser-agent fill @e2 "hello world"
browser-agent press Enter

# Step 3: 操作后重新 snapshot 验证结果
browser-agent snapshot -i
```

> **Ref 生命周期**：每次 `snapshot` 会重置 refs。导航或页面变化后必须重新 snapshot。

## 命令速查

### Snapshot（元素分析）
```bash
browser-agent snapshot -i          # 只看交互元素（推荐）
browser-agent snapshot             # 完整 DOM 树
browser-agent snapshot -c          # 紧凑输出（只保留有 ref 的行）
browser-agent snapshot -d 3        # 限制深度为 3
browser-agent snapshot -s "#main"  # 只看某个 CSS selector 范围
```

### 交互（用 snapshot 拿到的 @ref）
```bash
browser-agent click @e1
browser-agent dblclick @e1
browser-agent fill @e2 "text"      # 清空后输入
browser-agent type @e2 "text"      # 不清空，直接追加
browser-agent press Enter          # 按键（支持 Control+a、Escape、Tab 等）
browser-agent hover @e1
browser-agent check @e1            # 勾选 checkbox
browser-agent uncheck @e1
browser-agent select @e1 "option"  # 下拉选择
browser-agent scroll down 500      # 滚动（up/down/left/right，默认 300px）
browser-agent scrollintoview @e1
```

### 获取信息
```bash
browser-agent get text @e1         # 元素文本
browser-agent get html @e1         # innerHTML
browser-agent get value @e1        # input 的值
browser-agent get attr @e1 href    # 属性值
browser-agent get title
browser-agent get url
```

### 状态检查
```bash
browser-agent is visible @e1
browser-agent is enabled @e1
browser-agent is checked @e1
```

### 等待
```bash
browser-agent wait 2000            # 等待 ms
browser-agent wait @e1             # 等待 ref 出现
browser-agent wait --text "成功"   # 等待文字出现（默认 30s 超时）
browser-agent wait --url "/chat"   # 等待 URL 包含字符串
```

### 语义定位（不用 ref）
```bash
browser-agent find text "发送" click
browser-agent find role button click
browser-agent find label "消息输入" fill "你好"
browser-agent find placeholder "输入消息" click
browser-agent find testid "send-btn" click
```

### JavaScript
```bash
browser-agent eval "document.title"
browser-agent eval -b "$(echo 'document.querySelectorAll("button").length' | base64)"
```

## Clawfirm 常见测试场景

### 发送消息
```bash
browser-agent snapshot -i
# 输入框 placeholder 完整文本（含省略号）：
# "Type a message… (/ for skills, /plan for workflows, ⌘↵ to send)"
# 用 ref 填入最稳定：
browser-agent fill @e6 "你好，请介绍一下自己"

# 发送：始终用 find text（Send 按钮会被 React 重建，ref 会失效）
browser-agent find text "Send" click

# 等待回复：先等 Stop 出现，再等 Send 回来
browser-agent wait --text "Stop" --timeout 15000
browser-agent wait --text "Send" --timeout 30000

# 读取 assistant 回复（CSS class 是 flex justify-start）
browser-agent eval "
(function(){
  var msgs = document.querySelectorAll('.flex.justify-start');
  return msgs.length ? msgs[msgs.length-1].textContent.trim().slice(0,200) : 'no reply';
})()"
```

### 停止生成
```bash
# 必须用 find text，Stop 按钮每次流式都被 React 重建，ref 会指向僵尸节点
browser-agent find text "Stop" click
```

### 打开新对话
```bash
# 方法 A: Dashboard 点 "+ New Chat"（多 agent 时弹下拉菜单）
browser-agent find text "+ New Chat" click
browser-agent snapshot -i   # 查看 dropdown 里的 agent 列表
browser-agent click @eN     # 点选目标 agent

# 方法 B: Chat 内输入 /new
browser-agent fill @eN "/new"
browser-agent find text "Send" click

# 验证新对话已就绪
browser-agent wait --text "Start a conversation" --timeout 10000

# 验证 session ID 变化（header 里的 Copy thread ID 按钮）
browser-agent eval "document.querySelector('header button[title=\"Copy thread ID\"]')?.textContent"
```

### 切换 Agent
```bash
browser-agent snapshot -i
# 找到 Agent 列表项
browser-agent click @e_agent
browser-agent wait --text "Start a conversation"
```

### 验证工具调用
```bash
# 等待工具执行完成
browser-agent wait --text "done"
# 检查工具面板是否有结果
browser-agent snapshot -s "#tool-panel"
```

## 写 Shell 脚本测试

可以把场景写成 shell 脚本，方便复用：

```bash
#!/bin/bash
set -e

echo "=== 测试：发送消息并等待回复 ==="

# 快照当前页面
browser-agent snapshot -i

# 填入消息
browser-agent fill @e2 "1+1等于几"
browser-agent press Enter

# 等待回复中包含数字
browser-agent wait --text "2" --timeout 30000

echo "✓ 回复正常"
browser-agent get url
```

## 输出格式说明

Snapshot 输出格式（与 agent-browser 一致）：

```
Page: Clawfirm
URL: http://localhost:9988

- heading "Clawfirm" [level=1, ref=e1]
- textbox "输入消息" [ref=e2]
- button "发送" [ref=e3]
- button "Stop" [disabled, ref=e4]
```

- `[ref=eN]` — 用于后续命令的引用
- `[disabled]` — 不可用状态
- `[checked=true/false]` — checkbox 状态
- `[expanded=true/false]` — 展开状态
- `: value` — input 的当前值

## 注意事项

- **refs 会重置**：每次 `snapshot` 都从 `e1` 重新编号，不要跨 snapshot 复用 ref
- **页面跳转后必须重新 snapshot**：click 导致导航后，旧 refs 全部失效
- **流式状态按钮必须用 `find text`**：Stop 和 Send 在流式期间由 React 交替重建 DOM 节点，旧 ref 会指向已 detached 的"僵尸"元素（fiber 失效）。对这两个按钮永远用语义定位：
  ```bash
  browser-agent find text "Stop" click
  browser-agent find text "Send" click
  ```
  ref 只适合导航栏、标题等结构稳定、不会被 React 重建的元素。
- **eval server 超时 30s**：长时间等待操作前，先用 `wait` 命令等待页面稳定
- **proxy 干扰**：如果连接失败，检查 http_proxy 环境变量是否拦截了 localhost 请求

## 双实例测试环境（Primary → Secondary）

用第一个 Clawfirm（Primary，端口 9310）的 agent 驱动第二个 Clawfirm（Secondary，端口 9311），两者 workspace 完全隔离。

### 启动 Secondary 实例

```bash
# Secondary 使用独立数据目录和 eval server 端口
# 必须用二进制路径直接启动（open 不传递 env var）
# 注意：清除代理变量，防止 Secondary 内部 WebSocket 被代理拦截
CLAWFIRM_DATA_DIR=~/.clawfirm-test \
  CLAWFIRM_EVAL_PORT=9311 \
  CLAWFIRM_LOG_PATH=~/.clawfirm-test/app.log \
  http_proxy= https_proxy= HTTP_PROXY= HTTPS_PROXY= all_proxy= ALL_PROXY= \
  /Applications/clawfirm.app/Contents/MacOS/clawfirm &

# 等待 eval server 就绪
sleep 8
curl -s --noproxy localhost -X POST http://localhost:9311/api/eval \
  -H 'Content-Type: application/json' \
  -d '{"script":"document.title"}'
# 期待返回: {"result":"clawfirm"}
```

Secondary 的数据（DB、config、logs）全部在 `~/.clawfirm-test/`，不会和 Primary 的 `~/.clawfirm/` 冲突。

### Primary 读取 Secondary 的日志（debug）

Secondary 启动时传入 `CLAWFIRM_LOG_PATH`，Primary agent 可以直接读取：

```bash
# 查看 Secondary 最新日志（定位错误/卡顿）
cat ~/.clawfirm-test/app.log | tail -80

# 只看 ERROR 级别
grep -i "error\|panic\|fatal" ~/.clawfirm-test/app.log | tail -20

# 查看 agent 最近的 turn 流程
grep -E "agent:|tool:|provider:" ~/.clawfirm-test/app.log | tail -30
```

**典型 debug 流程**：
1. 向 Secondary 发送测试消息后，如果没有回复或行为异常
2. `cat ~/.clawfirm-test/app.log | tail -80` 读取最近日志
3. 分析错误原因（provider 超时、tool 执行失败、JS eval 错误等）
4. 根据日志调整测试策略或报告问题

`CLAWFIRM_LOG_PATH` 支持 `~/` 前缀展开，也可以指定任意绝对路径（如 `/tmp/secondary.log`）。

### Primary agent 中使用 browser-agent 控制 Secondary

```bash
# 指向 Secondary 的 eval server（端口 9311）
BROWSER_AGENT_PORT=9311 browser-agent snapshot -i
BROWSER_AGENT_PORT=9311 browser-agent find text "Send" click
```

或者在测试脚本中 export：

```bash
#!/bin/bash
export BROWSER_AGENT_PORT=9311

browser-agent snapshot -i
browser-agent fill @e2 "你好"
browser-agent find text "Send" click
browser-agent wait --text "Stop" --timeout 15000
browser-agent wait --text "Send" --timeout 30000
echo "✓ Secondary 回复正常"
```

### 确认 Secondary eval server 在线

```bash
curl -s --noproxy localhost -X POST http://localhost:9311/api/eval \
  -H 'Content-Type: application/json' \
  -d '{"script":"document.title"}' | python3 -m json.tool
```
