# clawfirm

AI Gateway — 一个用 Go 构建的 AI 智能体平台，支持多 LLM 提供商、对话记忆、工作流编排，并提供 CLI、HTTP 网关和桌面应用三种使用方式。

## 目录结构

```
clawfirm/
├── agent/              # 智能体核心：事件循环、状态机、工具执行器、上下文压缩
│   ├── loop.go         #   主循环 (stream → tool call → respond)
│   ├── agent.go        #   Agent 结构体与 functional options
│   ├── tool_executor.go#   工具调用分发
│   ├── compressor.go   #   上下文压缩 (达到 80% 窗口时 LLM 摘要)
│   ├── systemprompt.go #   系统提示词构建
│   ├── bootstrap.go    #   初始化引导
│   ├── context.go      #   上下文管理
│   ├── state.go        #   状态机定义
│   ├── queue.go        #   消息队列
│   └── temporal.go     #   时间相关工具
│
├── provider/           # LLM 提供商适配层
│   ├── interface.go    #   Provider 接口定义
│   ├── registry.go     #   全局 Provider 注册表
│   ├── anthropic/      #   Anthropic (Claude) 适配
│   ├── openai/         #   OpenAI 适配
│   ├── gemini/         #   Google Gemini 适配
│   ├── ollama/         #   Ollama 本地模型适配
│   └── zenmux/         #   Zenmux 聚合代理适配
│
├── types/              # 核心类型定义
│   ├── message.go      #   Message, Role
│   ├── content.go      #   ContentBlock (text/image/tool_use/tool_result)
│   ├── event.go        #   AgentEvent (流式事件)
│   ├── model.go        #   Model 元数据
│   └── stream.go       #   StreamOptions
│
├── memory/             # 语义记忆系统
│   ├── manager.go      #   Memory Manager: 索引/搜索/混合重排 (0.7 余弦 + 0.3 BM25)
│   ├── embed.go        #   Embedding 提供商接口 (OpenAI/Gemini/Voyage/Mistral/Ollama)
│   ├── chunk.go        #   文本分块 (400 token, 80 overlap)
│   ├── tools.go        #   memory_search / memory_get 工具
│   └── summarizer.go   #   自动摘要 (30 分钟定时 → .md 文件)
│
├── store/              # SQLite 持久化层 (WAL 模式, modernc.org/sqlite)
│   ├── db.go           #   数据库连接与迁移
│   ├── session.go      #   会话管理
│   ├── message.go      #   消息存储
│   ├── kv.go           #   键值存储
│   ├── vault.go        #   加密存储
│   ├── cronjob.go      #   定时任务持久化
│   └── migrations/     #   SQL 迁移文件 (001-006)
│
├── stream/             # SSE 流式处理
│   ├── sse.go          #   SSE 解析器
│   ├── event_stream.go #   EventStream 封装
│   └── retry.go        #   重试逻辑
│
├── tool/               # 工具系统
│   ├── base.go         #   AgentTool 接口 + BaseToolImpl
│   ├── builtin/        #   内置工具
│   ├── registry.go     #   工具注册表
│   └── registry_test.go
│
├── message/            # 消息处理工具
│   ├── convert.go      #   格式转换
│   ├── serialize.go    #   序列化
│   └── prune.go        #   消息裁剪
│
├── skill/              # Skill 系统 (Markdown 定义的可复用提示模板)
│   ├── skill.go        #   Skill 解析与加载
│   └── format.go       #   格式化输出
│
├── gateway/            # HTTP 网关服务
│   ├── server.go       #   HTTP 服务器 (默认 :9988)
│   ├── manager.go      #   Agent 管理器
│   ├── session.go      #   会话路由
│   ├── agent_registry.go#  Agent 注册
│   └── docparse.go     #   文档解析
│
├── channel/            # 消息渠道适配
│   ├── webchat/        #   WebSocket 聊天 (WebChat)
│   ├── feishu/         #   飞书机器人
│   └── whatsapp/       #   WhatsApp (go-whatsmeow)
│
├── whipflow/           # WhipFlow 工作流引擎 (.whip DSL)
│   ├── whipflow.go     #   入口
│   ├── lexer/          #   词法分析器
│   ├── parser/         #   语法分析器
│   ├── ast/            #   抽象语法树
│   ├── token/          #   Token 定义
│   ├── runtime/        #   运行时执行器
│   └── validator/      #   校验器
│
├── auth/               # 认证与密钥管理
│   ├── storage.go      #   密钥存储
│   ├── keychain.go     #   macOS Keychain 集成
│   ├── resolver.go     #   Key 解析
│   └── oauth/          #   OAuth PKCE 流程
│
├── config/             # 配置管理
│   ├── config.go       #   YAML 配置解析 (~/.clawfirm/config.yml)
│   └── example.yml     #   示例配置
│
├── cron/               # 定时任务调度
│   ├── scheduler.go    #   Cron 调度器
│   └── schedule.go     #   调度规则
│
├── funcs/              # 技术指标函数库
│   ├── env.go          #   计算环境 (OHLCV 数据)
│   ├── lexparser.go    #   公式解析器
│   └── library.go      #   内置函数 (MA, EMA, RSI 等)
│
├── app/                # 桌面应用后端 (Wails binding)
│   ├── app.go          #   App 结构体与 Wails 方法
│   ├── config_writer.go#   配置写入
│   └── assets.go       #   嵌入资源
│
├── internal/
│   └── agentbuilder/   # 从配置构建 Agent 实例
│
├── testutil/           # 测试辅助
│   ├── fixtures.go
│   ├── mock_provider.go
│   └── mock_tool.go
│
├── cmd/                # 可执行入口
│   ├── pi/             #   CLI: 单轮对话 (`pi -p "prompt"`)
│   ├── gateway/        #   HTTP 网关服务 (WebSocket + REST)
│   ├── desktop/        #   桌面应用 (Wails + React + Tailwind)
│   │   └── frontend/   #     React 前端 (Vite + TypeScript)
│   ├── whip/           #   WhipFlow 运行器 (`whip file.whip`)
│   ├── wschat/         #   WebSocket 测试客户端
│   └── func/           #   技术指标计算 CLI
│
├── examples/           # 示例文件
├── scripts/            # 辅助脚本
├── bin/                # 构建产物
├── Makefile            # 构建: make all / make app / make test
├── go.mod              # Go 1.25.1, module github.com/ai-gateway/clawfirm
└── go.work             # Go workspace
```

## 技术栈

| 层 | 技术 |
|---|---|
| 语言 | Go 1.25.1 |
| 数据库 | SQLite (modernc.org/sqlite, 纯 Go, WAL 模式) |
| 桌面 | Wails v2 + React + Vite + Tailwind CSS |
| 消息渠道 | WebSocket / 飞书 / WhatsApp |
| LLM | Anthropic · OpenAI · Gemini · Ollama · Zenmux |

## 快速开始

```bash
# 构建所有 CLI 工具 + 桌面应用
make all

# 仅构建 macOS .app
make app

# 运行测试
make test

# CLI 单轮对话
go run ./cmd/pi -p "你好"

# 启动网关
go run ./cmd/gateway

# 运行 WhipFlow 工作流
go run ./cmd/whip example.whip
```
