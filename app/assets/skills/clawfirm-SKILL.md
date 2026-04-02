---
name: clawfirm
version: 1.0.0
description: |
  clawfirm 是内置的 AI Gateway CLI，集成了 whipflow（工作流引擎）、
  skillctl（技能管理）、openvault（密钥管理）和 browser（浏览器自动化）。
  Use this skill whenever the user mentions clawfirm, wants to run a .whip
  workflow, or asks about `clawfirm run`, `clawfirm skill`, `clawfirm vault`.

  clawfirm 是 Go 二进制，已内置于项目中，无需 npm 安装。
---

# clawfirm

clawfirm 是内置的 AI Gateway CLI，集成以下子系统：

| 子系统 | 说明 |
|--------|------|
| whipflow | AI 工作流引擎，执行 .whip 文件 |
| skillctl | 技能注册/搜索/安装/同步 |
| openvault | 加密本地密钥管理 |
| browser | 浏览器自动化（CDP，YAML shortcuts） |

## 工作流执行

使用内置工具 `whipflow_run` 执行 .whip 工作流，不需要命令行：

| 参数 | 说明 |
|------|------|
| `file` | .whip 文件路径 |
| `source` | 直接传入 WhipFlow 源码 |
| `user_inputs` | `ask` 变量的预填值，如 `{"track": "美食"}` |
| `retry_from_session` | 从第 N 个 session 重试（0-based） |

示例：
```json
{"file": "~/.clawfirm/workflows/media.whip", "user_inputs": {"track": "美食"}}
```

## 数据目录

clawfirm 数据存储在 `~/.clawfirm/`：

```
~/.clawfirm/
├── config.yml        # 全局配置
├── data.db           # SQLite 数据库
├── memory/           # 语义记忆
├── shortcuts/        # 浏览器 YAML shortcuts
├── sessions/         # 浏览器会话缓存
└── canvas/           # 工作流输出
```

## 内置浏览器 Shortcuts

社交平台自动化通过内置 browser shortcuts 实现（YAML 文件），不需要外部 social-cli：

| 文件 | 平台 | 命令 |
|------|------|------|
| `douyin.yaml` | 抖音 | search, like, comment, follow, post, download |
| `xhs.yaml` | 小红书 | search, hot, like, comment, post, post_video |
| `bilibili.yaml` | B站 | search, like, comment, reply, follow, post |
| `x.yaml` | X/Twitter | search, like, reply, post, retweet |
| `zhipin.yaml` | Boss直聘 | chat_stats, candidates |

前置条件：Chrome 需以 `--remote-debugging-port=9222` 启动。

## Common workflows

### 运行工作流

通过 `whipflow_run` 工具执行：
```json
{"file": "~/.clawfirm/workflows/media.whip"}
{"file": "whips/gaokao/run-all.whip", "user_inputs": {"province": "浙江"}}
```

## Troubleshooting

**"config: open ... no such file"** → 确认 `~/.clawfirm/config.yml` 存在。

**浏览器 shortcut 报错** → 确认 Chrome 已启动 remote debugging：
`/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222`

**工作流执行失败** → 检查 .whip 文件语法：确认 agent 定义在文件末尾、缩进用空格。
