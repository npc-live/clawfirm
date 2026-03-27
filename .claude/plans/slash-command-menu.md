# Plan: 输入框 `/` 斜杠命令菜单

## 目标

在聊天输入框中输入 `/` 时，弹出命令菜单，显示当前 agent 可用的 skills，选中后将 skill 名称插入输入框作为指令前缀。

## 修改文件

```
新建:  components/SlashCommandMenu.tsx  — 弹出菜单组件
修改:  components/ChatView.tsx          — 集成菜单 + 加载 skills
```

## Step 1: 新建 SlashCommandMenu 组件

**文件**: `cmd/desktop/frontend/src/components/SlashCommandMenu.tsx`

Props:
- `skills: SkillInfo[]` — 可用 skill 列表
- `filter: string` — `/` 后输入的过滤文字
- `onSelect: (skill: SkillInfo) => void` — 选中回调
- `onClose: () => void` — 关闭菜单
- `visible: boolean`

功能:
- 根据 filter 模糊匹配 skill name/description
- 键盘导航：↑↓ 选择，Enter 确认，Esc 关闭
- 每项显示：🧩 图标 + skill name + description
- 从输入框底部向上弹出（absolute 定位）
- 暗色主题，与现有 UI 一致

## Step 2: ChatView 集成

**文件**: `cmd/desktop/frontend/src/components/ChatView.tsx`

### 新增状态
```typescript
const [skills, setSkills] = useState<SkillInfo[]>([]);
const [showSlashMenu, setShowSlashMenu] = useState(false);
```

### 加载 skills
在 useEffect 中调用 `GetAgentSkills(agentName)` 加载。

### 输入框 onChange 逻辑
- 当 input 以 `/` 开头时，`setShowSlashMenu(true)`，提取 `/` 后文字作为 filter
- 当 input 不以 `/` 开头时，`setShowSlashMenu(false)`

### onSelect 回调
选中 skill 后，将 input 替换为 `/skillName ` (带尾部空格)，关闭菜单，用户继续输入具体指令。

### handleSend 扩展
发送时如果 input 以 `/skillName` 开头，将 skill name 包含在消息中发给后端（后端已有 skill 处理能力）。

## 视觉效果

```
┌──────────────────────────────┐
│  🧩 web_search               │
│     Search the web           │
│  🧩 code_interpreter         │
│     Execute Python code      │
│  🧩 file_reader              │  ← 向上弹出菜单
│     Read file contents       │
├──────────────────────────────┤
│  /fi█                   Send │  ← 输入框，正在过滤
└──────────────────────────────┘
```
