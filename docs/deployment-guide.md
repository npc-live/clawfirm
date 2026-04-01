# Clawfirm 部署指南

## 前置条件

```bash
# Go 1.25.1+
go version

# Node.js (桌面应用前端需要)
node -v

# Wails v2 (桌面应用需要)
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 第一步：配置

```bash
mkdir -p ~/.clawfirm
cp config/example.yml ~/.clawfirm/config.yml
```

编辑 `~/.clawfirm/config.yml`：

```yaml
providers:
  # 方案 A：ZenMux 聚合（一个 key 用所有模型）
  zenmux:
    api_key: ${ZENMUX_API_KEY}
    base_url: https://zenmux.ai/api/v1

  # 方案 B：直连各家
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
  openai:
    api_key: ${OPENAI_API_KEY}
  gemini:
    api_key: ${GEMINI_API_KEY}
  ollama:
    base_url: http://localhost:11434

default_provider: zenmux
default_model: anthropic/claude-haiku-4-5

# 网关 Agent 定义
agents:
  - name: assistant
    provider: anthropic
    model: claude-sonnet-4-20250514
    system_prompt: "你是一个有用的AI助手。"
    tools: [read, write, edit, bash]
    # skill_paths: ["~/.clawfirm/skills"]

# 飞书渠道（可选）
feishu:
  app_id: ${FEISHU_APP_ID}
  app_secret: ${FEISHU_APP_SECRET}

# WhatsApp 渠道（可选）
whatsapp:
  enabled: true

# 定时任务（可选）
cron_jobs:
  - name: daily-report
    schedule:
      kind: cron
      expr: "0 9 * * *"
      tz: Asia/Shanghai
    agent_name: assistant
    prompt: "生成今日报告"
    enabled: true
```

## 第二步：密钥管理（Vault）

```bash
# 初始化加密 vault（密钥存 macOS Keychain）
clawfirm vault init

# 存入 API keys
clawfirm vault set ANTHROPIC_API_KEY
clawfirm vault set ZENMUX_API_KEY

# 验证
clawfirm vault list
clawfirm vault get ANTHROPIC_API_KEY

# 注入环境变量运行任意命令
clawfirm vault run -- go run ./cmd/gateway
```

## 第三步：构建

```bash
# 克隆
git clone https://github.com/npc-live/clawfirm.git
cd clawfirm

# 构建所有 CLI 工具
make all
# 产出: bin/clawfirm, bin/func, bin/gateway, bin/wschat + 桌面应用

# 或分开构建
go build -o bin/clawfirm ./cmd/clawfirm     # 主 CLI
go build -o bin/gateway  ./cmd/gateway       # HTTP 网关
go build -o bin/pi       ./cmd/pi            # 单轮对话

# macOS .app 桌面应用
make app
# 产出: bin/clawfirm.app
# 安装: cp -r bin/clawfirm.app /Applications/
```

## 第四步：部署方式

### A. CLI 模式（本地使用）

```bash
# 单轮对话
./bin/pi -p "你好"

# 运行 WhipFlow 工作流
./bin/clawfirm run flows/hello.whip
./bin/clawfirm run -config ./config.yml flows/my-workflow.whip

# 安装 Claude Code skill
./bin/clawfirm install-skills

# Skill 管理
./bin/clawfirm skill search "code-review"
./bin/clawfirm skill install code-review
```

### B. Gateway 模式（服务器部署）

```bash
# 启动网关（默认 :9988）
./bin/gateway -addr :9988 -config ~/.clawfirm/config.yml

# 自定义 DB 路径
./bin/gateway -db /data/clawfirm.db
```

接入方式：

| 端点 | 说明 |
|------|------|
| `ws://host:9988/ws/{agentName}/{sessionID}` | 指定 Agent WebSocket |
| `ws://host:9988/ws/{sessionID}` | 默认 Agent WebSocket |
| `GET http://host:9988/health` | 健康检查 |

渠道说明：
- **飞书**：无需公网 webhook — SDK 主动建立 WebSocket 出站连接到飞书服务器
- **WhatsApp**：首次连接需扫 QR 码，凭证持久化到本地 JSON 文件

### C. 桌面应用模式

```bash
# 构建 + 安装
make install
# 或
make app && cp -r bin/clawfirm.app /Applications/

# 内嵌了 func 二进制（universal: arm64 + amd64）
```

### D. 生产部署（systemd）

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
