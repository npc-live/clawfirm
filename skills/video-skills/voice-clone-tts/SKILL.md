---
name: voice-clone-tts
description: 声纹克隆和语音合成。上传音频样本克隆声纹，用克隆声纹或预设声纹生成语音。支持多个后端：MiniMax、ElevenLabs、Fish Audio、Azure TTS、OpenAI TTS。支持情绪控制、语速调整、批量生成。触发词：语音合成、TTS、声纹克隆、voice clone、text to speech、配音、旁白。
---

# 声纹克隆 & 语音合成

## 何时使用此 Skill

| 场景 | 是否需要此 Skill |
|------|------------------|
| 数字人平台支持声纹克隆（可灵/即梦/HeyGen）| **不需要**，直接在 digital-avatar 里处理 |
| 数字人平台不支持声纹克隆 | 需要，生成音频后上传 |
| 纯音频输出（播客/有声书）| 需要 |
| 需要更精细的语音控制 | 可选，MiniMax/ElevenLabs 控制更细 |

**推荐：优先用数字人平台自带的声纹克隆，保持后端一致性。**

---

## 功能

1. **声纹克隆**：上传音频样本 → 生成声纹 ID
2. **语音合成**：文本 + 声纹 → 音频文件
3. **批量生成**：分镜列表 → 多个音频文件

## 支持的后端

| 后端 | 克隆 | 情绪 | 多语言 | 特点 |
|------|------|------|--------|------|
| MiniMax | ✓ | ✓ | 中/英 | 国内快，中文好 |
| ElevenLabs | ✓ | ✓ | 30+ | 质量顶级 |
| Fish Audio | ✓ | - | 中/英/日 | 开源，便宜 |
| Azure TTS | - | ✓ | 100+ | 稳定，多语言 |
| OpenAI TTS | - | - | 多语言 | 简单快速 |

默认使用 MiniMax（如已配置）。

## 工作流程

### 流程 A：声纹克隆

```
输入: 音频样本（10s-5min）
  ↓
上传到后端
  ↓
等待训练（即时-几分钟）
  ↓
输出: voice_id
```

### 流程 B：语音合成

```
输入: text + voice_id + 参数
  ↓
调用 TTS API
  ↓
输出: 音频文件
```

### 流程 C：批量生成

```
输入: scenes[] + voice_id
  ↓
逐条生成（或并行）
  ↓
输出: 音频文件列表
```

## 输入参数

### 声纹克隆

| 参数 | 必填 | 说明 |
|------|------|------|
| mode | ✓ | clone |
| backend | - | minimax / elevenlabs / fish |
| audio_sample | ✓ | 音频文件路径（10s-5min）|
| name | - | 声纹名称 |
| description | - | 声纹描述 |

### 语音合成

| 参数 | 必填 | 说明 |
|------|------|------|
| mode | ✓ | synthesize |
| backend | - | 同上 + azure / openai |
| text | ✓ | 要合成的文本 |
| voice_id | ✓ | 声纹 ID 或预设名 |
| output | - | 输出路径 |
| speed | - | 语速 0.5-2.0（默认1.0）|
| emotion | - | 情绪（见下表）|
| pitch | - | 音调调整 |
| format | - | mp3 / wav / ogg |

### 批量生成

| 参数 | 必填 | 说明 |
|------|------|------|
| mode | ✓ | batch |
| backend | - | 同上 |
| scenes | ✓ | 分镜列表（含台词）|
| voice_id | ✓ | 声纹 ID |
| output_dir | - | 输出目录 |

## 情绪控制

| 情绪 | 英文 | 适用场景 |
|------|------|----------|
| neutral | neutral | 默认/旁白 |
| happy | happy | 轻松/种草 |
| sad | sad | 共情/痛点 |
| angry | angry | 吐槽/愤怒 |
| excited | excited | 惊喜/CTA |
| serious | serious | 专业/严肃 |
| whisper | whisper | 悄悄话/ASMR |

## 输入格式（批量）

```yaml
mode: batch
voice_id: "voice_abc123"
backend: minimax

scenes:
  - id: 1
    text: "你是不是也遇到过这样的问题？"
    emotion: neutral
    speed: 1.0
    
  - id: 2
    text: "每次做 PPT 都要花好几个小时..."
    emotion: sad
    speed: 0.9
    
  - id: 3
    text: "现在只需要 30 秒！"
    emotion: excited
    speed: 1.1
    
  - id: 4
    text: "链接在评论区，快去试试吧"
    emotion: happy
    speed: 1.0

output_dir: "./audio_output/"
format: mp3
```

## 输出格式

### 声纹克隆

```yaml
voice:
  id: "voice_abc123"
  backend: minimax
  name: "我的声音"
  status: ready
  created_at: "2024-01-01T00:00:00Z"
```

### 语音合成

```yaml
audio:
  path: "./output/scene_01.mp3"
  duration: 3.5
  format: mp3
  sample_rate: 44100
```

### 批量生成

```yaml
audios:
  - id: 1
    path: "./audio_output/scene_01.mp3"
    duration: 2.8
    
  - id: 2
    path: "./audio_output/scene_02.mp3"
    duration: 4.2
    
  - id: 3
    path: "./audio_output/scene_03.mp3"
    duration: 2.1
    
  - id: 4
    path: "./audio_output/scene_04.mp3"
    duration: 3.0

total_duration: 12.1
```

## 后端配置

详见 [references/backend-setup.md](references/backend-setup.md)

## 音频样本要求

### 声纹克隆最佳实践

| 项目 | 要求 |
|------|------|
| 时长 | 30s-3min（最佳1-2min）|
| 格式 | mp3 / wav / m4a |
| 采样率 | ≥16kHz |
| 内容 | 清晰朗读，无背景噪音 |
| 情绪 | 自然平稳，不要太夸张 |

### 提高克隆质量

1. 安静环境录制
2. 保持语速稳定
3. 内容覆盖常用音节
4. 避免口水音/换气声
5. 使用高质量麦克风

## 使用示例

### 克隆声纹

```
用户：用这段音频克隆我的声音 [附音频]

执行：
1. mode=clone, audio_sample=<音频路径>
2. 上传到 MiniMax
3. 返回 voice_id
```

### 单条合成

```
用户：用 voice_abc123 合成这句话："大家好，欢迎来到我的频道"

执行：
1. mode=synthesize, voice_id="voice_abc123", text="..."
2. 调用 TTS API
3. 返回音频文件
```

### 批量合成分镜

```
用户：根据这个分镜列表生成配音 [附 YAML]

执行：
1. 解析 scenes 列表
2. 逐条调用 TTS
3. 返回音频列表 + 总时长
```

## 与上下游对接

**上游**：`video-script-generator` 的 scenes[].narration

**下游**：
- `digital-avatar`：音频输入生成口播
- `scene-video-generator`：音频轨道
- `video-stitcher`：音频合并

## Pipeline 集成

```yaml
# video-script-generator 输出
scenes:
  - id: 1
    narration: "你是不是也..."
    duration: 3s
    
# 自动提取 narration → voice-clone-tts
# 输出音频 → digital-avatar / video-stitcher
```

## 注意事项

1. 声纹克隆需遵守各平台使用条款
2. 不要用他人声音做误导性内容
3. 商用需确认版权协议
4. 批量生成建议控制并发，避免 rate limit
5. 保存 voice_id，避免重复克隆
