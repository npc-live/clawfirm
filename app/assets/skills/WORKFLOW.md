# 短视频制作工作流

## 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        video-skills Pipeline                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────┐                                        │
│  │ video-script-generator│  Step 1: 生成脚本+分镜                │
│  │  └─ scripts/generate.mjs                                      │
│  └──────────┬───────────┘                                        │
│             │                                                     │
│             ▼                                                     │
│  ┌──────────────────────┐                                        │
│  │   voice-clone-tts    │  Step 2: 生成配音                      │
│  │  ├─ scripts/minimax.mjs                                       │
│  │  └─ scripts/elevenlabs.mjs                                    │
│  └──────────┬───────────┘                                        │
│             │                                                     │
│     ┌───────┴───────┐                                            │
│     ▼               ▼                                            │
│  ┌────────────┐  ┌────────────────────┐                          │
│  │digital-    │  │scene-video-        │  Step 3: 生成视频片段    │
│  │avatar      │  │generator           │                          │
│  │(口播场景)  │  │(AI场景)            │                          │
│  │ ├─kling.mjs│  │ ├─kling-video.mjs  │                          │
│  │ └─jimeng   │  │ └─runway.mjs       │                          │
│  └─────┬──────┘  └─────────┬──────────┘                          │
│        │                   │                                      │
│        └─────────┬─────────┘                                      │
│                  ▼                                                │
│  ┌──────────────────────┐                                        │
│  │    video-stitcher    │  Step 4: 拼接成品                      │
│  │  └─ scripts/stitch.mjs                                        │
│  └──────────────────────┘                                        │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### 一键 Pipeline

```bash
# 完整流程
node scripts/pipeline.mjs --topic "AI写作工具" --duration 30s --backend kling

# 使用配置文件
node scripts/pipeline.mjs --config project.yaml
```

### 分步执行

```bash
# 1. 生成脚本
cd video-script-generator
node scripts/generate.mjs --topic "AI写作工具" --duration 30s --output script.yaml

# 2. 生成配音
cd ../voice-clone-tts
node scripts/minimax.mjs batch --input scenes.json --voice-id female-tianmei --output-dir ./audio

# 3. 生成视频片段
cd ../scene-video-generator
node scripts/kling-video.mjs generate --prompt "xxx" --duration 5

# 或数字人口播
cd ../digital-avatar
node scripts/kling.mjs generate --avatar-id xxx --text "大家好"

# 4. 拼接成品
cd ../video-stitcher
node scripts/stitch.mjs concat --dir ./clips -t fade -o final.mp4
```

---

## 环境配置

### 1. 安装依赖

```bash
# FFmpeg（拼接必需）
winget install FFmpeg    # Windows
brew install ffmpeg      # macOS

# Node.js 依赖
npm install yaml
```

### 2. 配置 API Keys

在 shell 或 `.env` 中设置：

```bash
# Kling 可灵（推荐）
export KLING_ACCESS_KEY="your_key"
export KLING_SECRET_KEY="your_secret"

# 即梦 Jimeng
export JIMENG_API_KEY="ak-xxx"

# MiniMax TTS（推荐）
export MINIMAX_API_KEY="your_key"
export MINIMAX_GROUP_ID="your_group"

# ElevenLabs（可选）
export ELEVENLABS_API_KEY="your_key"

# Runway（可选）
export RUNWAY_API_KEY="your_key"
```

---

## 各 Skill 详情

### video-script-generator

生成脚本和分镜结构。

```bash
# 列出模板
node scripts/generate.mjs templates

# 生成脚本
node scripts/generate.mjs --topic "xxx" --duration 30s --template pain-solution
```

输出示例：
```yaml
meta:
  title: "脚本标题"
  duration: "30s"
script:
  scenes:
    - id: 1
      type: hook
      narration: "你是不是也..."
      shot_description: "特写表情"
```

### voice-clone-tts

声纹克隆和语音合成。

```bash
# 列出预设声音
node scripts/minimax.mjs voices

# 克隆声纹
node scripts/minimax.mjs clone --audio sample.mp3 --name "我的声音"

# 单条合成
node scripts/minimax.mjs tts --text "你好" --voice female-tianmei

# 批量合成
node scripts/minimax.mjs batch --input scenes.json --voice-id xxx --output-dir ./audio
```

### digital-avatar

数字人创建和口播视频生成。

```bash
# 从照片创建
node scripts/kling.mjs create --photo ./photo.jpg

# 克隆声纹（平台内）
node scripts/kling.mjs voice-clone --audio sample.mp3

# 生成口播
node scripts/kling.mjs generate --avatar-id xxx --text "大家好"
```

⚠️ **重要**：同一项目必须使用同一后端！

### scene-video-generator

AI 场景视频生成（非数字人）。

```bash
# 文生视频
node scripts/kling-video.mjs generate --prompt "A cat playing piano" --duration 5

# 图生视频
node scripts/kling-video.mjs generate --prompt "xxx" --image ref.jpg

# 查询状态
node scripts/kling-video.mjs status --task-id xxx
```

### video-stitcher

视频拼接和后期处理。

```bash
# 简单拼接
node scripts/stitch.mjs concat clip1.mp4 clip2.mp4 -o output.mp4

# 带转场
node scripts/stitch.mjs concat clip1.mp4 clip2.mp4 -t fade -d 0.5 -o output.mp4

# 添加BGM
node scripts/stitch.mjs add-bgm video.mp4 bgm.mp3 -v 0.3 -o output.mp4

# 添加字幕
node scripts/stitch.mjs add-subs video.mp4 captions.srt -o output.mp4

# 调整分辨率
node scripts/stitch.mjs resize video.mp4 -r 1080x1920 -o output.mp4

# 完整配置
node scripts/stitch.mjs concat --config project.yaml -o output.mp4
```

---

## 后端选择指南

| 场景 | 推荐后端 | 原因 |
|------|----------|------|
| 国内抖音/小红书 | Kling 或 Jimeng | 国内快，中文口型好 |
| 出海/英文内容 | HeyGen | 模板丰富，英文自然 |
| 最高质量场景 | Runway Gen-3 | 质量顶级 |
| 中文配音 | MiniMax | 效果好，有情绪控制 |
| 英文配音 | ElevenLabs | 质量顶级 |
| 预算有限 | Jimeng + MiniMax | 有免费额度 |

---

## 文件结构

```
video-skills/
├── WORKFLOW.md              # 本文档
├── scripts/
│   └── pipeline.mjs         # 一键 Pipeline
│
├── video-script-generator/
│   ├── SKILL.md
│   ├── scripts/
│   │   └── generate.mjs
│   └── templates/
│       ├── pain-solution.md
│       ├── before-after.md
│       └── plot-twist.md
│
├── voice-clone-tts/
│   ├── SKILL.md
│   ├── scripts/
│   │   ├── minimax.mjs
│   │   └── elevenlabs.mjs
│   └── references/
│       └── backend-setup.md
│
├── digital-avatar/
│   ├── SKILL.md
│   ├── scripts/
│   │   ├── kling.mjs
│   │   └── jimeng.mjs
│   └── references/
│       └── backend-setup.md
│
├── scene-video-generator/
│   ├── SKILL.md
│   ├── scripts/
│   │   ├── kling-video.mjs
│   │   └── runway.mjs
│   └── references/
│       ├── backend-setup.md
│       └── prompt-guide.md
│
└── video-stitcher/
    ├── SKILL.md
    ├── scripts/
    │   └── stitch.mjs
    └── references/
        └── ffmpeg-guide.md
```

---

## 常见问题

### Q: 数字人 API 返回错误？
A: 检查后端一致性，同一项目不要混用可灵和即梦。

### Q: 配音速度/情绪不对？
A: MiniMax 支持 `--speed` 和 `--emotion` 参数调整。

### Q: 拼接后画面模糊？
A: 确保输入视频分辨率一致，或使用 `resize` 统一处理。

### Q: 转场不生效？
A: 需要 FFmpeg 4.3+，检查版本 `ffmpeg -version`。
