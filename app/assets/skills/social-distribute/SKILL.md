---
name: social-distribute
description: |
  一键多平台视频分发。读取各平台 content-rules，一次 LLM 调用生成全平台文案，
  再用 browser_shortcut 并行填表。支持：抖音、TikTok、B站、小红书、YouTube、LinkedIn、微信视频号。
  触发词：分发、发布、distribute、多平台发布、一键发布、social distribute。
---

# social-distribute — 多平台视频分发

## 输入参数

调用时需提供：

| 参数 | 必填 | 说明 |
|------|------|------|
| `video` | ✅ | 视频文件绝对路径 |
| `topic` | ✅ | 视频主题描述（用于生成文案） |
| `platforms` | ✅ | 目标平台列表，如 `["douyin","tiktok","bilibili","xiaohongshu","youtube","linkedin","channels"]` |
| `cover_h` | 可选 | 横版封面图路径（B站/YouTube/LinkedIn 用） |
| `cover_v` | 可选 | 竖版封面图路径（抖音/TikTok/小红书 用） |

---

## 平台 → Shortcut 映射

| 平台 | Shortcut 文件 | Command | Args |
|------|-------------|---------|------|
| 抖音 | `douyin-fill.yaml` | `post` | video, title, desc, cover, tags |
| TikTok | `tiktok-fill.yaml` | `post` | video, title, desc, tags, cover_h, cover_v |
| B站 | `bilibili-fill.yaml` | `post` | video, title, desc, cover, tags |
| 小红书 | `xhs-fill.yaml` | `post_video` | video, title, desc, cover |
| YouTube | `youtube-fill.yaml` | `post_video` | video, title, desc, visibility |
| LinkedIn | `linkedin-fill.yaml` | `post_video` | video, title, desc |
| 微信视频号 | `channels-fill.yaml` | `post` | video, title, desc, cover |

---

## 执行流程

### Step 1：读取 content-rules

对目标平台列表，读取对应的平台规范文件：

```
~/.clawfirm/bundled/skills/social-publish/{platform}/references/content-rules.md
```

平台目录名映射：
- `douyin` → `douyin`
- `tiktok` → `tiktok`
- `bilibili` → `bilibili`
- `xiaohongshu` → `xiaohongshu`
- `youtube` → `youtube`
- `channels` → 无 content-rules，跳过

### Step 2：一次生成全平台文案

读取所有 content-rules 后，**一次 LLM 调用**生成所有平台的文案，输出 JSON：

```json
{
  "douyin":      { "title": "...", "desc": "...", "tags": ["tag1","tag2","tag3"] },
  "tiktok":      { "title": "...", "desc": "...", "tags": ["tag1","tag2","tag3"] },
  "bilibili":    { "title": "...", "desc": "...", "tags": ["tag1","tag2","tag3"] },
  "xiaohongshu": { "title": "...", "desc": "..." },
  "youtube":     { "title": "...", "desc": "...", "visibility": "public" },
  "linkedin":    { "title": "...", "desc": "..." },
  "channels":    { "title": "...", "desc": "..." }
}
```

### Step 3：并行调用 browser_shortcut 填表

对每个目标平台，调用 `browser_shortcut` 工具：

**示例（抖音）：**
```json
{
  "file": "douyin-fill.yaml",
  "command": "post",
  "args": ["/path/to/video.mp4", "标题", "描述", "/path/to/cover_v.png", "tag1 tag2 tag3"]
}
```

**示例（TikTok）：**
```json
{
  "file": "tiktok-fill.yaml",
  "command": "post",
  "args": ["/path/to/video.mp4", "title", "desc", "tag1 tag2 tag3", "/path/to/cover_h.png", "/path/to/cover_v.png"]
}
```

**注意：**
- `cover` 参数传路径字符串；若该平台没有传封面，传空字符串 `""`
- tags 以空格分隔拼成单个字符串传入（与各 shortcut args 定义一致）
- YouTube visibility 默认传 `"public"`

### Step 4：汇总结果

各平台填表完成后，汇报每个平台的结果（成功/失败/跳过）。

---

## Whipflow 骨架

```whip
agent distributor:
  tools: [read, bash, browser_shortcut]
  prompt: "多平台视频分发专家。严格按 content-rules 生成文案，用 browser_shortcut 执行 CDP 填表。"

# Step 1+2: 读规则 + 生成文案
let copywriting = session: distributor
  prompt: """
  视频分发任务：
  - 视频：{video}
  - 主题：{topic}
  - 目标平台：{platforms}
  - 横版封面：{cover_h}
  - 竖版封面：{cover_v}

  1. 对每个平台，读取 ~/.clawfirm/bundled/skills/social-publish/{platform}/references/content-rules.md
  2. 根据各平台规范，一次生成所有平台文案，输出 JSON（含 title/desc/tags）
  """

# Step 3: 并行填表
parallel:
  let douyin_result = session: distributor
    prompt: "用 browser_shortcut 运行 douyin-fill.yaml post，参数：{video} {copywriting.douyin.title} {copywriting.douyin.desc} {cover_v} {copywriting.douyin.tags}"

  let tiktok_result = session: distributor
    prompt: "用 browser_shortcut 运行 tiktok-fill.yaml post，参数：{video} {copywriting.tiktok.title} {copywriting.tiktok.desc} {copywriting.tiktok.tags} {cover_h} {cover_v}"

  let bilibili_result = session: distributor
    prompt: "用 browser_shortcut 运行 bilibili-fill.yaml post，参数：{video} {copywriting.bilibili.title} {copywriting.bilibili.desc} {cover_h} {copywriting.bilibili.tags}"

  let xhs_result = session: distributor
    prompt: "用 browser_shortcut 运行 xhs-fill.yaml post_video，参数：{video} {copywriting.xiaohongshu.title} {copywriting.xiaohongshu.desc} {cover_v}"

  let youtube_result = session: distributor
    prompt: "用 browser_shortcut 运行 youtube-fill.yaml post_video，参数：{video} {copywriting.youtube.title} {copywriting.youtube.desc} public"

  let linkedin_result = session: distributor
    prompt: "用 browser_shortcut 运行 linkedin-fill.yaml post_video，参数：{video} {copywriting.linkedin.title} {copywriting.linkedin.desc}"

  let channels_result = session: distributor
    prompt: "用 browser_shortcut 运行 channels-fill.yaml post，参数：{video} {copywriting.channels.title} {copywriting.channels.desc} {cover_v}"

# Step 4: 汇总
session: distributor
  prompt: "汇总各平台结果：{douyin_result} {tiktok_result} {bilibili_result} {xhs_result} {youtube_result} {linkedin_result} {channels_result}"
```

---

## 快捷调用方式

直接告诉 agent：

```
分发这个视频到抖音、B站、小红书：
- 视频：~/Desktop/final_video.mp4
- 主题：AI 工具提升工作效率
- 横版封面：~/Desktop/cover_h.png
- 竖版封面：~/Desktop/cover_v.png
```

Agent 会自动加载本 skill，读取 content-rules，生成文案，并行填表。
