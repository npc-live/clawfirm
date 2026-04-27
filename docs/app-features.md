# Clawfirm App 功能清单

> 细粒度功能文档，基于 `app/app.go` + 前端组件源码分析
> 更新于 2026-04-18

---

## 1. 对话功能 (Chats)

| 功能 | 说明 |
|------|------|
| 打开新对话 | 点击 Agent 后自动创建新 session（`sessionID: "s" + Date.now()`） |
| 打开历史会话 | 从 ChatSessions 列表中选择 |
| 发送消息 | 支持文本输入，Cmd+Enter 快捷发送 |
| 停止生成 | 调用 `AbortCurrentTurn` 中断 agent |
| 复制 Thread ID | 点击 session ID 复制到剪贴板 |
| 粘贴图片 | Ctrl+V 自动识别并附加图片文件 |
| 附件上传 | 支持 image/video/audio，单文件最大 20MB |
| 查看对话历史 | 调用 `GetHistory` 获取历史消息列表 |
| 恢复上次会话 | 启动时自动恢复上一次 session |

---

## 2. 消息输入与交互

| 功能 | 说明 |
|------|------|
| Skill 快捷调用 | 输入 `/` 触发技能选择器（模糊匹配） |
| `/new` | 在输入框输入 `/new` 开启新会话 |
| `/plan <描述>` | 根据描述生成 whipflow 工作流代码 |
| 自动补全 | 方向键上下选择，Enter/Tab 确认，Esc 关闭 |
| 粘贴截图片 | Clipboard paste 自动识别图片 |
| 多文件附件 | 一次选择多个文件，统一发送 |

---

## 3. 工具面板 (Tool Activity)

| 功能 | 说明 |
|------|------|
| 查看工具执行状态 | running / done / error / interrupted |
| 查看工具参数 | 显示 tool call ID、名称、传入参数 |
| 查看工具结果 | 显示 tool 返回值或错误信息 |
| 发送工具命令 | 通过 ToolPanel 按钮直接发送命令 |
| Whip Plan 编辑 | 从 assistant 消息提取 `whip` 代码块并渲染为可编辑 UI |
| Whip Plan Step | 分步调试工作流（逐 step 执行） |
| Whip Plan Execute | 直接执行工作流 |
| HTML 预览 | 从 assistant 消息提取 `html` 代码块并渲染到右侧面板 |

---

## 4. WhipFlow 工作流

| 功能 | 说明 |
|------|------|
| 生成工作流 | `/plan <需求描述>` 由 LLM 生成 whip 代码块 |
| 编辑工作流 | 在 ChatView 内联 banner 中修改 whip 代码 |
| 执行工作流 | Execute 按钮立即运行 |
| 分步调试 | Step 按钮逐 step 执行，便于排查 |
| Ask 语句表单 | 解析 `ask varName: "prompt text"` 自动生成输入框 |
| 确认输入 | 填完所有 ask 字段后点击 Confirm inputs |
| 条件语法 | `if` / `while` / `filter` 使用 `[...]` 包裹条件 |
| Loop Until 语法 | `loop until **cond** (max: N):` 使用 `**...**` 包裹条件 |
| 预览弹窗 | 自动从 assistant 消息提取 whip 代码并展示 |

---

## 5. Skills 管理

| 功能 | 说明 |
|------|------|
| 查看本地 Skills | `GetAllSkills` 获取所有已注册 skill |
| 搜索远程 Skills | 调用 `SearchRemoteSkills` 搜索 skillctl.dev |
| 安装远程 Skill | `InstallRemoteSkill` 下载并安装 |
| 查看 Skill 内容 | 点击展开查看 SKILL.md 全文 |
| Skill 快捷调用 | 输入 `/skillname` 或通过 picker 选择 |
| 查看 Skill 描述 | picker 列表显示 name + description |
| 按 Agent 过滤 | `GetAgentSkills` 只显示绑定到某 Agent 的 skills |

---

## 6. 定时任务 (Cron)

| 功能 | 说明 |
|------|------|
| 查看任务列表 | `ListCronJobs` 获取所有 Cron 任务 |
| 添加新任务 | `AddCronJob` 创建定时任务 |
| 编辑任务 | `UpdateCronJob` 修改现有任务 |
| 删除任务 | `DeleteCronJob` 移除任务 |
| 启用/禁用 | `ToggleCronJob` 开关任务 |
| 手动触发 | `TriggerCronJob` 立即执行一次 |
| 查看执行历史 | `GetCronJobHistory` 按任务查看 |
| 查看全部历史 | `GetCronJobHistoryAll` 查看所有任务历史 |

---

## 7. 记忆系统 (Memory)

| 功能 | 说明 |
|------|------|
| 查看记忆文件 | `ListMemoryFiles` 列出 memory 目录下所有文件 |
| 读取记忆内容 | `GetMemoryFileContent` 读取指定文件 |
| 编辑记忆 | `SaveMemoryFileContent` 保存修改 |
| 创建记忆文件 | `CreateMemoryFile` 新建文件 |
| 删除记忆文件 | `DeleteMemoryFile` 删除文件 |
| 语义搜索 | `SearchMemory` 使用 embedding 向量搜索 |
| 同步记忆 | `SyncMemory` 触发记忆同步 |
| 查看记忆目录 | `GetMemoryDir` 获取记忆文件夹路径 |

---

## 8. Agents / Channels 管理

| 功能 | 说明 |
|------|------|
| 查看 Channels | `GetChannels` 获取所有 Agent/Channel |
| 保存 Channel 配置 | `SaveChannelConfig` 持久化 AgentConfig |
| 删除 Channel | `DeleteChannelConfig` 移除配置 |
| 测试 Channel 连接 | `TestChannelConnection` 验证连通性 |
| 查看 Chat Sessions | `GetChatSessions` 获取某 Agent 的 session 列表 |
| 重置 Session | `ResetSession` 清空某 session 的消息历史 |
| 列出所有 Sessions | `ListSessions` 获取详细 session 信息 |
| Agent 运行状态 | WebSocket 连接状态实时展示 |

---

## 9. Providers 配置

| 功能 | 说明 |
|------|------|
| 查看 Providers | `GetProviders` 列出所有 LLM 提供商 |
| 保存 API Key | `SaveAPIKey` 安全存储 provider key |
| 测试连接 | `TestProviderConnection` 验证 provider 连通性 |
| OAuth 登录 | `StartOAuthLogin` 启动 OAuth 流程 |
| 获取模型列表 | `GetModels` 按 provider 获取可用模型 |
| 配置模型参数 | Temperature、maxTokens 等（通过 SaveConfig） |

---

## 10. 飞书集成 (Feishu)

| 功能 | 说明 |
|------|------|
| 查看飞书配置 | `GetFeishuConfig` 获取 appID/appSecret |
| 保存飞书配置 | `SaveFeishuConfig` 写入配置 |
| Channel 状态 | 通过 `EmitChannelStatus` 实时更新连接状态 |

---

## 11. WhatsApp 集成

| 功能 | 说明 |
|------|------|
| 查看连接状态 | `GetWhatsAppStatus` 返回在线/离线/连接中 |
| 获取登录二维码 | `GetWhatsAppQR` 生成 QR 码用于扫码登录 |
| 登出 | `LogoutWhatsApp` 清除 WhatsApp 会话 |

---

## 12. Vault（安全存储）

| 功能 | 说明 |
|------|------|
| 查看 Vault 条目 | `GetVault` 列出所有 key-value 条目 |
| 添加/更新条目 | `SetVaultEntry` 创建或更新条目 |
| 删除条目 | `DeleteVaultEntry` 移除条目 |

---

## 13. Canvas（画布文件管理）

| 功能 | 说明 |
|------|------|
| 列出画布文件 | `ListCanvasFiles` 获取所有文件 |
| 读取文件内容 | `ReadCanvasFile` 读取指定文件 |
| 写入文件 | `WriteCanvasFile` 保存文件 |
| 独立画布面板 | 单独的 CanvasPane 界面 |

---

## 14. Browser（CDP 浏览器自动化）

| 功能 | 说明 |
|------|------|
| 测试 CDP 连接 | `BrowserTestCDP` 检测 Chrome DevTools 可用性 |
| 启动 Chrome | `BrowserLaunchChrome` 启动带 CDP 的 Chrome 实例 |
| 列出 Shortcuts | `BrowserListShortcuts` 查看所有已注册快捷方式 |
| 运行 Shortcut | `BrowserRunShortcut` 执行指定 shortcut |

---

## 15. Settings（设置）

| 功能 | 说明 |
|------|------|
| 首次运行检测 | `IsFirstRun` 判断是否需要引导设置 |
| 获取配置（结构化） | `GetConfig` 返回 Config 对象 |
| 保存配置 | `SaveConfig` 持久化配置 |
| 获取配置（原始） | `GetConfigRaw` 返回原始 YAML/JSON 字符串 |
| 保存配置（原始） | `SaveConfigRaw` 直接写入配置文件 |
| 打开日志目录 | `OpenLogsFolder` 调用系统文件管理器 |
| 查看版本号 | `GetVersion` 返回当前版本字符串 |
| Webhook URL | `GetWebhookBaseURL` 获取外网回调地址 |

---

## 16. 远程访问

| 功能 | 说明 |
|------|------|
| 启用远程访问 | `EnableRemote` 开启远程连接服务 |
| 禁用远程访问 | `DisableRemote` 关闭服务 |
| 启用 Ngrok | `EnableNgrok` 使用 Ngrok 隧道 |
| 禁用 Ngrok | `DisableNgrok` 关闭 Ngrok |
| 获取远程状态 | `GetRemoteStatus` 查看当前连接状态 |
| 查看 Channel 状态列表 | `channelStatusFunc` 实时上报所有 Channel 状态 |

---

## 17. 工具栏快捷操作（ChatView 内）

| 操作 | 描述 |
|------|------|
| ← 返回 | 返回 Dashboard |
| #sessionID | 点击复制当前 session ID |
| 绿色/黄色/红色圆点 | WebSocket 连接状态指示（open/connecting/closed） |
| 附件按钮 | 打开文件选择器（image/video/audio） |
| Stop 按钮 | 正在生成时显示，中断当前 turn |
| Send 按钮 | 发送消息（禁用状态：输入为空或 WS 未连接） |

---

## 18. 辅助功能

| 功能 | 说明 |
|------|------|
| 自动恢复会话 | 启动时读取 localStorage 恢复上次 session |
| 自动滚动 | 新消息自动滚动到底部 |
| Thinking 展示 | 支持 Anthropic extended thinking + DeepSeek/QwQ <think> 标签解析 |
| Markdown 渲染 | 消息内容支持 GFM（表格、代码块等） |
| 工具调用动画 | running 状态显示执行中动画 |
| 断线重连 | WS reconnect 后自动拉取 tool execution state |
