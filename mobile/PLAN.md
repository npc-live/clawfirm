# Mobile Remote — Tauri v2 手机 App + 桌面端 Remote Server

## 概述

手机原生 App（Tauri v2），扫 QR code 绑定桌面端 clawfirm，远程查看 Chat/Canvas、发消息控制 Agent。

**网络模式：**
- **LAN** — 同一 WiFi 直连，零配置
- **ngrok** — 跨网络公网隧道（Go 原生库，用户只需填 ngrok auth token）

## 架构

```
┌─────────────────────┐                                 ┌──────────────────────┐
│   Tauri v2 Mobile   │         HTTP/WebSocket          │  clawfirm Desktop    │
│                     │ ◄──────────────────────────────► │                      │
│  - React frontend   │                                 │  remote.Server       │
│  - barcode-scanner  │  LAN:   http://192.168.x.x:PORT │  0.0.0.0:随机端口     │
│  - tauri-http       │  ngrok: https://xxx.ngrok-free. │  token auth          │
│  - tauri-websocket  │         app                     │  ngrok-go (可选)      │
│                     │                                 │  QR code 生成         │
│  手机不需要装额外东西 │  token auth (QR 编码)            │                      │
└─────────────────────┘                                 └──────────────────────┘
```

---

## Part 1: 桌面端 — Remote Server

### 新建文件

#### 1.1 `channel/remote/remote.go` — 核心服务

```go
type Server struct {
    httpSrv          *http.Server
    token            string          // 16 字节 hex
    port             int             // 随机端口
    lanIP            string          // 192.168.x.x
    ngrokURL         string          // https://xxx.ngrok-free.app (可能为空)
    ngrokListener    net.Listener    // ngrok tunnel listener
    registry         *gateway.AgentRegistry
    db               *store.DB
    canvasDir        string
    connectedClients atomic.Int32
    channelStatusFn  func() []ChannelStatus
}

type RemoteStatus struct {
    Enabled    bool   `json:"enabled"`
    LanURL     string `json:"lanUrl"`      // http://192.168.x.x:PORT/remote/?token=xxx
    NgrokURL   string `json:"ngrokUrl"`    // https://xxx.ngrok-free.app/remote/?token=xxx (可选)
    QRCode     string `json:"qrCode"`      // data:image/png;base64,... (优先 ngrok URL)
    Token      string `json:"token"`
    Port       int    `json:"port"`
    LanIP      string `json:"lanIP"`
    NgrokOn    bool   `json:"ngrokOn"`     // ngrok 隧道是否开启
    Clients    int    `json:"clients"`     // 已连接客户端数
}
```

**核心功能：**
- Token 生成：`crypto/rand` 16 字节 → hex 编码（32 字符）
- Auth middleware：检查 `?token=` query param 或 `X-Remote-Token` header
- LAN IP 发现：遍历 `net.InterfaceAddrs()`，识别私有网段（`192.168/10/172.16`）
- ngrok 隧道：可选，用户提供 ngrok auth token 后启用
- QR code：复用 `skip2/go-qrcode`，编码完整 URL（优先 ngrok URL，否则 LAN IP）
- `Start(ctx)` / `Stop()` / `Status() RemoteStatus`

#### 1.2 `channel/remote/routes.go` — REST API

```
GET  /remote/                                          → 落地页（展示连接信息）
GET  /remote/api/status                                → 服务状态
GET  /remote/api/agents                                → agent 列表
GET  /remote/api/agents/{name}/sessions                → session 列表（含 subject、时间、token 用量）
GET  /remote/api/agents/{name}/sessions/{id}/history   → 聊天记录（types.Message 数组）
GET  /remote/api/canvas                                → canvas 文件名列表
GET  /remote/api/canvas/{name}                         → canvas HTML 内容
GET  /remote/api/channels                              → channel 状态（WhatsApp/Feishu 在线状态）
GET  /remote/ws/{agentName}/{sessionID}                → WebSocket（同 webchat 协议）
```

**路由实现：**
- 使用标准库 `net/http` + `http.ServeMux`（Go 1.22+ 支持路径参数）
- 所有 `/remote/api/*` 和 `/remote/ws/*` 走 auth middleware
- JSON 响应，Content-Type: application/json
- 聊天记录从 `store.DB.Messages().ListMessages()` 读取
- Agent 列表从 `gateway.AgentRegistry.Names()` + config 获取
- Canvas 从 `canvasDir` 读文件

#### 1.3 `channel/remote/websocket.go` — WebSocket 处理

复制 `webchat/handler.go` 的 WebSocket 逻辑（~80 行），保持协议一致：

**Client → Server:**
```json
{"type": "message", "content": "你好"}
{"type": "run_tool", "tool_name": "...", "tool_id": "...", "tool_args": {...}}
```

**Server → Client:**
```json
{"type": "delta", "content": "..."}
{"type": "done", "stop_reason": "stop", "timestamp": 1234567890}
{"type": "tool_start", "tool_call_id": "...", "tool_name": "...", "tool_args": {...}}
{"type": "tool_update", "tool_call_id": "...", "partial_result": {...}}
{"type": "tool_end", "tool_call_id": "...", "tool_result": {...}}
{"type": "error", "content": "..."}
```

- 连接时 `connectedClients` +1，断开时 -1
- channelID 使用 `"remote/{agentName}"`，与 webchat 区分
- Token 验证在 WebSocket upgrade 前完成（从 query param 读取）

### 修改文件

#### 1.4 `channel/remote/tunnel.go` — ngrok 隧道

```go
// StartTunnel 启动 ngrok 隧道，返回公网 URL
func (s *Server) StartTunnel(ctx context.Context, authToken string) error {
    listener, err := ngrok.Listen(ctx,
        config.HTTPEndpoint(),
        ngrok.WithAuthtoken(authToken),
    )
    // listener.URL() → "https://xxxx.ngrok-free.app"
    s.ngrokListener = listener
    s.ngrokURL = listener.URL()
    // 在这个 listener 上 serve 同一个 handler
    go http.Serve(listener, s.handler())
    return nil
}

// StopTunnel 关闭 ngrok 隧道（LAN 服务不受影响）
func (s *Server) StopTunnel() error
```

- 使用 `golang.ngrok.com/ngrok/v2` Go 原生库
- ngrok listener 和 LAN listener **并行运行**，同一个 handler
- ngrok 是可选的：不填 token 就只有 LAN 模式
- 依赖：`go get golang.ngrok.com/ngrok/v2`

#### 1.5 `app/app.go` — 集成 Remote Server

**App struct 新增字段：**
```go
remoteSrv    *remote.Server
remoteCancel context.CancelFunc
```

**新增 Wails 绑定方法：**
```go
// EnableRemote 启动远程服务（LAN 模式），返回状态
func (a *App) EnableRemote() (remote.RemoteStatus, error)

// EnableNgrok 启动 ngrok 隧道（需要先 EnableRemote）
func (a *App) EnableNgrok(authToken string) (remote.RemoteStatus, error)

// DisableNgrok 关闭 ngrok 隧道（LAN 服务不受影响）
func (a *App) DisableNgrok() error

// DisableRemote 停止整个远程服务（含 ngrok）
func (a *App) DisableRemote() error

// GetRemoteStatus 获取当前状态（URL、QR、连接数）
func (a *App) GetRemoteStatus() remote.RemoteStatus
```

**EnableRemote 逻辑：**
1. 检查 gateway 是否已启动（需要 registry）
2. 创建 `remote.Server`，传入 registry、db、canvasDir、channelStatusFn
3. 启动 server（随机端口，绑定 `0.0.0.0`）
4. 返回 RemoteStatus（含 LAN QR code）

**EnableNgrok 逻辑：**
1. 检查 remote server 是否已启动
2. 调用 `remoteSrv.StartTunnel(ctx, authToken)`
3. 返回更新后的 RemoteStatus（QR code 切换为 ngrok URL）

**清理：**
- `stopGateway()` 中调用 `remoteSrv.Stop()`
- `OnShutdown` 中取消 remoteCancel

**channelStatusFn 回调：**
```go
func (a *App) channelStatusFunc() []remote.ChannelStatus {
    // 返回 WhatsApp、Feishu 的连接状态
    // 避免 remote 包直接依赖 whatsapp/feishu 包
}
```

#### 1.6 `cmd/desktop/frontend/src/components/Dashboard.tsx` — UI

在 Channels 面板（或新 tab）添加 **Remote Control** 卡片：

```
┌──────────────────────────────────────┐
│ 📱 Remote Control                     │
│                                      │
│ LAN:  [Enable] / [Disable]          │
│                                      │
│ ┌─────────┐  LAN:   192.168.1.5:... │
│ │ QR Code │  ngrok: (off)           │
│ │  image  │  Token: abc...          │
│ └─────────┘  Clients: 2             │
│                                      │
│ ── ngrok (跨网络访问) ──              │
│ Auth Token: [________________] [连接] │
│ Status: https://xxx.ngrok-free.app   │
│                                      │
└──────────────────────────────────────┘
```

- 调用 `EnableRemote()` / `DisableRemote()` / `GetRemoteStatus()`
- ngrok 区域：输入 auth token → 调用 `EnableNgrok(token)` → 显示公网 URL
- QR code 优先编码 ngrok URL（如已开启），否则 LAN URL
- 定时刷新连接数（每 5s 轮询 `GetRemoteStatus()`）

---

## Part 2: 手机端 — Tauri v2 App

### 项目结构

```
mobile/
├── package.json               # React + Vite + Tauri
├── vite.config.ts
├── tsconfig.json
├── index.html                 # 入口
├── src/
│   ├── main.tsx               # React 入口
│   ├── App.tsx                # 路由 + 主布局
│   ├── api.ts                 # HTTP/WebSocket 封装（用 tauri-plugin-http/websocket）
│   ├── store.ts               # 状态管理（连接状态、token、设备信息）
│   ├── pages/
│   │   ├── ScanPage.tsx       # 扫码绑定页
│   │   ├── ChatsPage.tsx      # Agent 列表 + Session 列表
│   │   ├── ChatView.tsx       # 聊天界面（流式显示）
│   │   ├── CanvasPage.tsx     # Canvas 文件列表
│   │   ├── CanvasView.tsx     # Canvas HTML 查看器
│   │   └── ChannelsPage.tsx   # Channel 状态
│   └── components/
│       ├── BottomNav.tsx      # 底部 tab 导航
│       ├── MessageBubble.tsx  # 消息气泡
│       ├── VoiceButton.tsx    # 语音输入按钮
│       └── DeviceCard.tsx     # 已绑定设备卡片
├── src-tauri/
│   ├── Cargo.toml             # Rust 依赖（tauri + plugins）
│   ├── tauri.conf.json        # Tauri 配置
│   ├── capabilities/
│   │   └── default.json       # 权限配置（camera、http、websocket）
│   ├── src/
│   │   ├── lib.rs             # Tauri 入口
│   │   └── commands.rs        # 自定义 Rust commands（如果需要）
│   └── gen/
│       ├── android/           # 生成的 Android 项目
│       └── apple/             # 生成的 iOS 项目
└── PLAN.md                    # 本文件
```

### 依赖

**Tauri 插件：**
| 插件 | 用途 |
|------|------|
| `tauri-plugin-barcode-scanner` | QR 扫码（官方，iOS/Android） |
| `tauri-plugin-http` | HTTP 请求（绕过 CORS，LAN/Tailscale 通信） |
| `tauri-plugin-websocket` | WebSocket 实时通信 |
| `tauri-plugin-store` | 本地持久化（已绑定设备信息） |

**前端：**
| 库 | 用途 |
|------|------|
| `react` + `react-dom` | UI 框架 |
| `react-router-dom` | 路由 |
| `@tauri-apps/api` | Tauri JS API |

### 页面流程

```
启动 App
  ├── 有已保存的设备？ ──→ 自动连接 ──→ 主界面
  └── 无设备 ──→ 扫码页
                    │
                    ▼
              扫 QR code
              解析 URL: http://<IP>:<PORT>/remote/?token=<TOKEN>
              保存 {ip, port, token, name} 到本地
                    │
                    ▼
              主界面（底部 tab 导航）
              ┌──────────┬──────────┬──────────┐
              │  Chats   │  Canvas  │ Channels │
              └──────────┴──────────┴──────────┘
```

### 核心页面

#### 2.1 ScanPage — 扫码绑定

```
┌─────────────────────┐
│                     │
│   ┌─────────────┐   │
│   │             │   │
│   │  Camera     │   │
│   │  Viewfinder │   │
│   │             │   │
│   └─────────────┘   │
│                     │
│  Scan QR code from  │
│  clawfirm desktop   │
│                     │
│  [ Manual Input ]   │  ← 手动输入 IP:PORT + Token
│                     │
│  ── Saved Devices ──│
│  📱 MacBook Pro     │  ← 已绑定设备列表
│     100.64.1.2:8080 │
│                     │
└─────────────────────┘
```

- 使用 `tauri-plugin-barcode-scanner` 扫码
- 解析 QR 内容：`http://<IP>:<PORT>/remote/?token=<TOKEN>`
- 调用 `GET /remote/api/status` 验证连接
- 保存设备信息到 `tauri-plugin-store`
- 支持保存多个设备（家里/公司不同机器）

#### 2.2 ChatsPage — 聊天列表

```
┌─────────────────────┐
│ 🤖 Agents           │
│                     │
│ ┌─────────────────┐ │
│ │ assistant       │ │
│ │ claude-opus-4   │ │
│ │ 3 sessions      │ │
│ └─────────────────┘ │
│ ┌─────────────────┐ │
│ │ coder           │ │
│ │ claude-sonnet   │ │
│ │ 1 session       │ │
│ └─────────────────┘ │
│                     │
│ ── Sessions ──────  │
│ 📝 Debug login bug  │
│    2 min ago        │
│ 📝 Write unit tests │
│    1 hour ago       │
│                     │
└─────────────────────┘
```

- `GET /remote/api/agents` → 显示 agent 卡片
- 点击 agent → `GET /remote/api/agents/{name}/sessions` → session 列表
- 显示 subject、时间、token 用量

#### 2.3 ChatView — 聊天界面

```
┌─────────────────────┐
│ ← assistant / sess1 │
│─────────────────────│
│                     │
│ 👤 帮我写个函数     │
│                     │
│ 🤖 好的，这是一个   │
│    计算斐波那契...   │
│    ```python        │
│    def fib(n):      │
│        ...          │
│    ```              │
│                     │
│ 🔧 Using: executor │
│    ▶ Running...     │
│                     │
│─────────────────────│
│ ┌───────────────┐🎤 │
│ │ Type message  │   │
│ └───────────────┘   │
└─────────────────────┘
```

- 进入时 `GET /remote/api/agents/{name}/sessions/{id}/history` 加载历史
- 建立 WebSocket 连接 `ws://<IP>:<PORT>/remote/ws/{agentName}/{sessionID}?token=xxx`
- 使用 `tauri-plugin-websocket`（绕过 WebView 限制）
- 流式显示：收到 `delta` → 追加到当前消息
- tool_start/tool_update/tool_end → 显示工具调用状态
- 发送消息：`{"type": "message", "content": "..."}`
- 语音按钮：长按录音 → Web Speech API / 录音插件 → 转文字 → 发送

#### 2.4 CanvasPage — Canvas 列表 + 查看

```
┌─────────────────────┐     ┌─────────────────────┐
│ 🎨 Canvas           │     │ ← dashboard         │
│                     │     │─────────────────────│
│ ┌─────────────────┐ │     │                     │
│ │ 📄 dashboard    │ │ ──► │  ┌───────────────┐  │
│ │ 📄 chart        │ │     │  │  iframe 渲染   │  │
│ │ 📄 report       │ │     │  │  canvas HTML   │  │
│ └─────────────────┘ │     │  │  内容          │  │
│                     │     │  └───────────────┘  │
└─────────────────────┘     └─────────────────────┘
```

- `GET /remote/api/canvas` → 文件名列表
- `GET /remote/api/canvas/{name}` → HTML 内容
- iframe / WebView 渲染 HTML

#### 2.5 ChannelsPage — Channel 状态

```
┌─────────────────────┐
│ 📡 Channels         │
│                     │
│ WhatsApp   🟢 在线  │
│ Feishu     🔴 离线  │
│ WebChat    🟢 在线  │
│                     │
│ ── Device ────────  │
│ MacBook Pro         │
│ 100.64.1.2:8080    │
│ Connected ✓        │
│                     │
│ [ Disconnect ]      │
│ [ Switch Device ]   │
└─────────────────────┘
```

---

## Part 3: ngrok 跨网络支持

### 为什么选 ngrok-go

| 特性 | 说明 |
|------|------|
| **Go 原生库** | `go get golang.ngrok.com/ngrok/v2`，编译进 binary，不需要外部程序 |
| **手机零依赖** | 生成标准 HTTPS 公网 URL，手机直接访问 |
| **WebSocket 支持** | 完全支持，适合聊天流式传输 |
| **免费层** | 注册即用，个人开发够用 |
| **可选** | 不填 token 就只有 LAN 模式，ngrok 不是必须的 |

### 实现细节

**双 Listener 架构：**

```go
// LAN listener — 始终运行
lanListener, _ := net.Listen("tcp", "0.0.0.0:0")  // 随机端口
go http.Serve(lanListener, handler)

// ngrok listener — 可选，用户填了 token 才启动
ngrokListener, _ := ngrok.Listen(ctx,
    config.HTTPEndpoint(),
    ngrok.WithAuthtoken(userToken),
)
go http.Serve(ngrokListener, handler)  // 同一个 handler
```

两个 listener 并行运行，共享同一个 handler（路由、auth、WebSocket 全部复用）。

**QR Code 内容策略：**

| 场景 | QR 编码的 URL | 原因 |
|------|---------------|------|
| ngrok 已开启 | `https://xxx.ngrok-free.app/remote/?token=TOKEN` | 公网可达，跨网络 |
| 仅 LAN | `http://192.168.x.x:PORT/remote/?token=TOKEN` | 同 WiFi 可达 |

**手机端 App 处理：**
- 扫码后保存 `{url, token, name}` — URL 可能是 LAN 或 ngrok
- 同一设备可保存两种 URL（LAN + ngrok），自动选择可达的
- 连接失败时提示用户切换网络模式

### ngrok 免费层限制

- 每分钟 40 连接（够用）
- 随机域名（每次重启变化，付费可固定）
- 带 ngrok interstitial 页面（首次访问需点确认，API 调用不受影响）

---

## 实施顺序

### Phase 1: 桌面端 Remote Server（Go）
1. `channel/remote/remote.go` — Server struct、token、LAN IP 发现、QR
2. `channel/remote/routes.go` — REST API 路由
3. `channel/remote/websocket.go` — WebSocket handler
4. `channel/remote/tunnel.go` — ngrok-go 隧道（可选）
5. `app/app.go` — 集成 Enable/Disable/Ngrok/Status 方法
6. `cmd/desktop/frontend/.../Dashboard.tsx` — Remote Control 卡片 UI
7. **验证**：`go build ./...` + 启动桌面端 + curl 测试 API

### Phase 2: 手机端 Tauri App（React + Rust）
1. 初始化 Tauri v2 mobile 项目 (`npm create tauri-app`)
2. 配置插件：barcode-scanner、http、websocket、store
3. `ScanPage` — 扫码 + 设备管理
4. `api.ts` — HTTP/WebSocket 封装层
5. `ChatsPage` + `ChatView` — 聊天列表 + 流式聊天
6. `CanvasPage` + `CanvasView` — Canvas 浏览
7. `ChannelsPage` — Channel 状态
8. 语音输入（VoiceButton）
9. **验证**：iOS 模拟器 + Android 模拟器测试

### Phase 3: 打磨
1. 暗色主题 + 移动端适配
2. 断线重连逻辑
3. 多设备管理（保存多个桌面端）
4. 推送通知（可选，后续考虑）

---

## 关键设计决策

| 决策 | 选择 | 原因 |
|------|------|------|
| 手机端技术 | Tauri v2 Mobile | 原生扫码、绕过 CORS、可发布到商店 |
| 独立监听 vs 修改 gateway | 独立 `0.0.0.0` 监听器 | 不影响现有 localhost gateway 安全性 |
| 认证方式 | 随机 token（QR code 编码） | 简单有效，LAN 场景安全；ngrok 自带 HTTPS |
| 网络模式 | LAN + ngrok-go（可选） | LAN 零配置；ngrok 跨网络，手机零依赖 |
| ngrok 集成 | Go 原生库（`ngrok/v2`） | 编译进 binary，不需要外部程序 |
| 手机网络层 | Tauri HTTP/WebSocket 插件 | Rust 层通信，绕过 WebView CORS 限制 |
| 语音输入 | Web Speech API（优先）+ 录音插件（备选） | 浏览器原生方案，零后端依赖 |
| WebSocket 复用 | 复制 webchat 协议逻辑 | 代码量小（~80行），避免包间耦合 |
| 前端框架 | React（与桌面端一致） | 复用组件和开发经验 |

---

## 验证清单

- [ ] `go build ./...` 桌面端编译通过
- [ ] 桌面端 Enable Remote → 显示 LAN QR code
- [ ] curl 测试所有 REST API 端点
- [ ] 填入 ngrok token → Enable Ngrok → 显示公网 URL + QR code
- [ ] 手机 App 扫码（LAN）→ 解析 URL → 保存设备 → 连接成功
- [ ] 手机 App 扫码（ngrok）→ 跨网络连接成功
- [ ] 手机进入 Chats → 看到 agent 列表 → 进入 sessio
- [ ] 聊天界面发消息 → 流式回复正确显示
- [ ] 语音按钮 → 录音 → 文字 → 发送
- [ ] Canvas tab → 文件列表 → 查看 HTML 内容
- [ ] Disable Ngrok → ngrok 断开，LAN 仍可用
- [ ] Disable Remote → 全部断开
- [ ] 多设备：保存多台桌面端，切换连接
