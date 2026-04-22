---
name: repo-classify
version: 1.0.0
description: |
  ClawFirm 代码库文件分类规则。当用户提到"上传代码库"、"提交到 GitHub"、
  "代码分类"、"文件归类"、"整理代码"、"push 到仓库"时自动激活。
  确保每个文件按类型放到正确的目录，并同步到对应的运行时位置。
triggers:
  - 上传代码库
  - 提交到GitHub
  - 代码分类
  - 文件归类
  - 整理代码
  - push到仓库
  - 代码组织
  - upload to repo
  - commit to github
---

# ClawFirm 代码库文件分类规则

## 核心原则

**一句话版本：** 代码直接提交，自然语言走 `app/assets`，Skill 双写，Shortcut 归 shortcuts。

---

## 分类决策树

```
文件是什么类型？
│
├─ 纯代码（Go / JS / TS / HTML / CSS / YAML配置 / Dockerfile ...）
│  → 直接提交到 GitHub 对应目录
│
├─ 纯自然语言（Skill MD / 提示词 / 文案模板 / 参考文档 ...）
│  → app/assets/ 对应子目录
│  → 如果是 Skill，还需同步到 ~/.clawfirm/skills/
│
├─ 混合项目（代码 + 自然语言文件）
│  → 只要包含自然语言文件 → 整体放 app/assets/
│
└─ 自媒体 CDP 分发 YAML
   → app/assets/shortcuts/
```

---

## 详细规则

### 1. 纯代码文件 → 直接提交 GitHub

| 文件类型 | 示例 | 目标位置 |
|----------|------|----------|
| Go 源码 | `*.go`, `go.mod`, `go.sum` | 项目对应 package 目录 |
| Web 前端 | `*.ts`, `*.tsx`, `*.vue`, `*.css` | `cmd/*/frontend/` |
| 配置文件 | `Dockerfile`, `Makefile`, `.yaml`(非Skill) | 项目根或对应目录 |
| Shell 脚本 | `*.sh` | 项目对应目录 |
| 二进制构建 | 编译产物 | `.gitignore` 排除 |

**处理方式：** `git add` → `git commit` → `git push`，无需额外操作。

### 2. 自然语言内容 → `app/assets/`

| 内容类型 | 目标路径 | 说明 |
|----------|----------|------|
| Skill 提示词 | `app/assets/skills/<name>/SKILL.md` | 平台文案规则、AI 能力定义 |
| Skill 参考文档 | `app/assets/skills/<name>/references/` | 算法指南、格式规格、审核规则 |
| 工作流文档 | `app/assets/skills/<name>/WORKFLOW.md` | 多 Skill 编排流程 |
| 文案模板 | `app/assets/skills/<name>/templates/` | Hook 公式、CTA 模板 |

### 3. Skill 文件 → 双写同步

Skill 文件必须同时存在于两个位置：

```
源码位置（Git 管理）:
  app/assets/skills/<skill-name>/SKILL.md
  app/assets/skills/<skill-name>/references/*.md

运行时位置（本地同步）:
  ~/.clawfirm/skills/<skill-name>/SKILL.md
  ~/.clawfirm/skills/<skill-name>/references/*.md
```

**同步命令：**
```bash
cp -r app/assets/skills/<skill-name> ~/.clawfirm/skills/
```

### 4. 自媒体 CDP Shortcut → `app/assets/shortcuts/`

| 文件 | 说明 |
|------|------|
| `app/assets/shortcuts/<platform>.yaml` | 完整发布流程（含点击发布） |
| `app/assets/shortcuts/<platform>-fill.yaml` | 仅填写不发布（用户手动确认） |

**已有 Shortcuts：**
- `xhs.yaml` — 小红书
- `douyin.yaml` / `douyin-fill.yaml` — 抖音
- `tiktok.yaml` / `tiktok-fill.yaml` — TikTok
- `bilibili.yaml` / `bilibili-fill.yaml` — B站
- `x.yaml` — Twitter/X
- `youtube.yaml` / `youtube-fill.yaml` — YouTube
- `zhipin.yaml` — Boss直聘

### 5. 敏感文件 → `.gitignore` 保护

以下内容**绝不提交**到 GitHub：

- `_ops/.env` — 环境变量 / API Key
- `_ops/finance/` — 财务数据
- `_ops/LegalEntity/` — 法律实体文件
- `_ops/PPT/` — 内部演示
- `_ops/demo-video/` — 演示视频
- `_ops/reports/` — 内部报告
- `.mcp.json` — MCP 配置（含 API Key）
- `bin/` — 编译产物

---

## 操作清单

每次提交代码库时，按此清单执行：

1. **分类** — 按上述决策树判断每个文件的归属
2. **放置** — 移动文件到正确目录
3. **同步 Skill** — 如有 Skill 变更，`cp -r` 到 `~/.clawfirm/skills/`
4. **检查 .gitignore** — 确认敏感文件未被追踪
5. **提交** — `git add` 具体文件（不用 `git add .`）→ `git commit` → `git push`
6. **更新索引** — 如新增 Skill/Shortcut，更新 `skill-index/SKILL.md`
