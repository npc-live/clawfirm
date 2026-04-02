# Clawfirm 部署指南

> 基于实际安装经验更新于 2026-04-02

## 前置条件

```bash
# macOS 安装 Go
brew install go

# 验证
go version  # 需要 Go 1.25.1+
```

## 第一步：克隆仓库

```bash
git clone https://github.com/npc-live/clawfirm.git
cd clawfirm
```

## 第二步：构建

> **注意**：`make all` 会尝试构建桌面应用（`cmd/desktop`），该目录当前未提交到仓库，会报错。请单独构建 CLI 工具。

```bash
# 构建 Makefile 中已注册的 CLI 工具
go build -o bin/clawfirm ./cmd/clawfirm
go build -o bin/gateway  ./cmd/gateway
go build -o bin/func     ./cmd/func
go build -o bin/wschat   ./cmd/wschat

# pi 未包含在 Makefile CMDS 中，需单独构建
go build -o bin/pi   ./cmd/pi
go build -o bin/whip ./cmd/whip
```

验证：
```bash
ls bin/
# 应看到: clawfirm  func  gateway  pi  whip  wschat
```

## 第三步：配置

```bash
mkdir -p ~/.clawfirm
```

**方式 A：直连 Anthropic API（需要自己的 sk-ant-... key）**

```yaml
# ~/.clawfirm/config.yml
providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    base_url: https://api.anthropic.com

default_provider: anthropic
default_model: claude-haiku-4-5-20251001

agents:
  - name: assistant
    provider: anthropic
    model: claude-sonnet-4-20250514
    system_prompt: "你是一个有用的AI助手。"
    tools: [read, write, edit, bash]
```

**方式 B：通过代理（如 Claude Code 内部代理、ZenMux 等）**

```yaml
# ~/.clawfirm/config.yml
providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    base_url: ${ANTHROPIC_BASE_URL}

default_provider: anthropic
default_model: anthropic/claude-haiku-4.5

agents:
  - name: assistant
    provider: anthropic
    model: anthropic/claude-sonnet-4.6
    system_prompt: "你是一个有用的AI助手。"
    tools: [read, write, edit, bash]
```

> **代理注意事项**：
> - 代理的模型名格式可能不同（如 `anthropic/claude-haiku-4.5` 而非 `claude-haiku-4-5-20251001`）
> - Claude Code 的内部代理 key 是会话级临时 token，会话结束后失效

## 第四步：验证安装

### 单轮对话

```bash
./bin/pi -p '你好'
```

> **注意**：使用英文单引号 `'`，不要用中文引号 `"`

### 运行 WhipFlow 工作流

仓库自带的 `examples/hello.whip` 可能有语法问题。以下是一个可用的测试文件：

```bash
cat > examples/hello.whip << 'WHIP'
agent ai:
  provider: assistant
  model: haiku

session: ai
  prompt: "你是哪个AI模型？用一句话介绍自己。"
WHIP
```

运行：
```bash
./bin/clawfirm run examples/hello.whip
```

### 安装 Claude Code Skill

```bash
./bin/clawfirm install-skills
# 输出: Installed: ~/.claude/skills/whipflow/SKILL.md
```

## 第五步：部署方式

### A. CLI 模式（本地使用）

```bash
# 单轮对话
./bin/pi -p '你好'

# 运行工作流
./bin/clawfirm run examples/hello.whip
./bin/clawfirm run -config ~/.clawfirm/config.yml my-workflow.whip
```

### B. Gateway 模式（服务器部署）

```bash
./bin/gateway -addr :9988
```

接入方式：

| 端点 | 说明 |
|------|------|
| `ws://host:9988/ws/{agentName}/{sessionID}` | 指定 Agent WebSocket |
| `ws://host:9988/ws/{sessionID}` | 默认 Agent WebSocket |
| `GET http://host:9988/health` | 健康检查 |

### C. 生产部署（systemd）

```bash
cat > /etc/systemd/system/clawfirm-gateway.service << 'EOF'
[Unit]
Description=Clawfirm AI Gateway
After=network.target

[Service]
Type=simple
User=clawfirm
WorkingDirectory=/opt/clawfirm
ExecStart=/opt/clawfirm/bin/gateway -addr :9988
Restart=always
RestartSec=5
Environment=ANTHROPIC_API_KEY=sk-ant-xxx

[Install]
WantedBy=multi-user.target
EOF

systemctl enable --now clawfirm-gateway
```

### D. 桌面应用

> **当前状态**：`cmd/desktop` 目录未提交到仓库，`make app` / `make install` 暂不可用。

## WhipFlow 语法速查

```
# 定义 Agent（冒号必须紧跟名称）
agent name:
  provider: assistant    # 引用 config.yml agents 中的名称
  model: haiku           # 短名：opus / sonnet / haiku

# Session（冒号后跟 agent 名）
session: agentName
  prompt: "你的提示词"

# 简单 session（无 agent 引用，用默认 provider）
session "你的提示词"

# 变量绑定
let x = "hello"

# 多步骤
session: researcher
  prompt: "研究 AI 趋势"

session: writer
  prompt: "写一篇总结"
```

> **踩坑记录**：
> - `print` 不是合法关键字，用 `return` 替代
> - `let result = session "..."` 不合法，`let` 只绑定字符串/数组
> - Agent 定义必须有冒号：`agent name:` 不是 `agent name`
> - Session 引用 agent 用冒号：`session: agentName` 不是 `session agentName "prompt"`
> - model 字段在 .whip 文件中用短名（`haiku`/`sonnet`/`opus`），validator 不接受完整模型 ID

## 数据位置

| 路径 | 用途 |
|------|------|
| `~/.clawfirm/config.yml` | 主配置 |
| `~/.clawfirm/data.db` | SQLite 主数据库（会话/消息/KV/cron） |
| `~/.clawfirm/memory/` | 语义记忆 Markdown 文件 |
| `~/.whipflow/state.db` | WhipFlow 工作流状态持久化 |
| `~/.clawfirm/skills/` | 已安装的 Skills |
| `~/.clawfirm/vault.db` | 加密密钥存储（bbolt） |

## 架构要点

- **零外部依赖**：纯 Go SQLite，不需要 PostgreSQL/Redis/Docker
- **单二进制部署**：`gateway` 一个文件搞定服务端
- **数据全在 SQLite**：WAL 模式，适合单机场景
- **渠道即插即用**：config 里开启 feishu/whatsapp 就自动连接
- **无状态网关**：Session 数据在 SQLite，gateway 可以重启不丢状态

## 可用 CLI 工具一览

| 工具 | 用途 |
|------|------|
| `clawfirm run <file.whip>` | 运行 WhipFlow 工作流 |
| `clawfirm vault init/set/get/list` | 加密密钥管理 |
| `clawfirm skill search/install/sync` | Skill 管理（registry 尚未上线） |
| `clawfirm install-skills` | 安装 Claude Code skill |
| `gateway` | HTTP/WebSocket 网关服务 |
| `pi -p "prompt"` | 单轮 CLI 对话 |
| `whip <file.whip>` | WhipFlow 独立运行器 |
| `func` | 技术指标计算 CLI |
| `wschat` | WebSocket 测试客户端 |
