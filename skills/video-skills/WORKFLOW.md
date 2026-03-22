# Video Skills 编排工作流

## 概览

从产品描述到成片的完整视频制作流水线。

```
┌─────────────────────────────────────────────────────────────────┐
│                        输入：产品/主题描述                        │
└─────────────────────────────────┬───────────────────────────────┘
                                  ↓
┌─────────────────────────────────────────────────────────────────┐
│              Step 1: video-script-generator                      │
│              脚本 + 分镜生成                                      │
└─────────────────────────────────┬───────────────────────────────┘
                                  ↓
                    ┌─────────────┴─────────────┐
                    ↓                           ↓
            ┌───────────────┐           ┌───────────────┐
            │  口播场景      │           │  非口播场景    │
            │  (有数字人)    │           │  (纯AI/Demo)  │
            └───────┬───────┘           └───────┬───────┘
                    ↓                           ↓
┌───────────────────────────────┐   ┌───────────────────────────┐
│  Step 2a: digital-avatar      │   │  Step 2b: scene-video-gen │
│  数字人 + 声纹 + 口播          │   │  AI场景视频               │
│  ⚠️ 全程同一后端              │   │  (无数字人)               │
│  (可灵 / 即梦 / HeyGen)       │   │  (可灵/即梦/Runway/Pika)  │
└───────────────┬───────────────┘   └─────────────┬─────────────┘
                │                                 │
                │   ┌─────────────────────────┐   │
                │   │  Step 2c: demo-recorder │   │
                │   │  产品演示录屏           │   │
                │   │  (Screen Studio等)      │   │
                │   └────────────┬────────────┘   │
                │                │                │
                └────────────────┼────────────────┘
                                 ↓
┌─────────────────────────────────────────────────────────────────┐
│              Step 3: video-stitcher                              │
│              视频拼接 + 转场 + BGM + 字幕                         │
└─────────────────────────────────┬───────────────────────────────┘
                                  ↓
┌─────────────────────────────────────────────────────────────────┐
│                        输出：最终成片                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## Step 1: video-script-generator

**输入：**
- 产品/主题描述
- 目标受众
- 时长（15s/30s/60s）
- 平台（抖音/小红书/YouTube）
- 模板（痛点-解决/before-after/反转）

**输出：**
```yaml
scenes:
  - id: 1
    type: hook          # 场景类型
    duration: 3s
    narration: "..."    # 台词
    shot_description: "..." # 画面描述
    requires_avatar: true   # ← 关键：是否需要数字人
    demo_insert: false
```

**场景类型判断：**
| type | requires_avatar | 处理方式 |
|------|-----------------|----------|
| hook (口播开场) | true | → digital-avatar |
| pain (痛点展示) | false | → scene-video-gen |
| solution (口播) | true | → digital-avatar |
| demo (产品演示) | false | → demo-recorder |
| result (效果) | true/false | 视情况 |
| cta (号召) | true | → digital-avatar |

---

## Step 2a: digital-avatar

**处理 `requires_avatar: true` 的场景**

**⚠️ 重要：后端一致性原则**
- 选定一个平台后，全程使用
- 形象创建、声纹克隆、口播生成都用同一后端
- 不同平台的 avatar_id 不互通

**子流程：**
```
首次使用:
  1. 上传音频样本 → 声纹克隆 → voice_id
  2. 上传照片/描述 → 创建形象 → avatar_id

生成口播:
  avatar_id + voice_id + narration → 口播视频
```

**输入：**
- scenes[] 中 requires_avatar=true 的场景
- avatar_id（已创建的数字人）
- voice_id（已克隆的声纹）或 预设声音

**输出：**
- 每个场景的口播视频文件

---

## Step 2b: scene-video-generator

**处理 `requires_avatar: false` 且非 demo 的场景**

**输入：**
- scenes[].shot_description（作为 prompt）
- 时长
- 画面比例

**输出：**
- AI 生成的场景视频

**注意：**
- 不涉及数字人，后端可以独立选择
- 可以和 digital-avatar 用不同的平台

---

## Step 2c: demo-recorder（已有）

**处理 `demo_insert: true` 的场景**

**输入：**
- 产品界面/录屏指令
- 音频轨道（可选）

**输出：**
- 产品演示视频

---

## Step 3: video-stitcher

**拼接所有视频片段**

**输入：**
```yaml
clips:
  - scene_id: 1
    source: digital-avatar    # 来源
    path: "./videos/scene_01.mp4"
    transition: fade
    
  - scene_id: 2
    source: scene-video-gen
    path: "./videos/scene_02.mp4"
    transition: dissolve
    
  # ...

bgm: "./assets/music.mp3"
bgm_volume: 0.2

output:
  path: "./final/output.mp4"
  resolution: "1080x1920"
  fps: 30
```

**输出：**
- 最终成片

---

## 后端选择指南

### 数字人平台（需全程一致）

| 平台 | 数字人 | 声纹克隆 | 口播 | 适用 |
|------|--------|----------|------|------|
| 可灵 Kling | ✓ | ✓ | ✓ | 高质量 |
| 即梦 Jimeng | ✓ | ✓ | ✓ | 快速/中文 |
| HeyGen | ✓ | ✓ | ✓ | 出海/英文 |

### AI 视频平台（可独立选择）

| 平台 | 特点 |
|------|------|
| 可灵 Kling | 质量高 |
| 即梦 Jimeng | 快速 |
| Runway | 顶级质量 |
| Pika | 风格化 |

---

## 完整示例

**输入：**
```
产品：AI写作助手
受众：自媒体博主
时长：30秒
平台：抖音
```

**Step 1 输出（分镜）：**
```yaml
scenes:
  - id: 1, type: hook, requires_avatar: true
  - id: 2, type: pain, requires_avatar: false
  - id: 3, type: solution, requires_avatar: true
  - id: 4, type: demo, demo_insert: true
  - id: 5, type: result, requires_avatar: true
  - id: 6, type: cta, requires_avatar: true
```

**路由：**
- 场景 1,3,5,6 → digital-avatar（可灵）
- 场景 2 → scene-video-generator（可灵）
- 场景 4 → demo-recorder

**Step 3：**
- 6 个视频 → video-stitcher → 成片

---

## API 配置

```json
// openclaw.json > services
{
  "kling": {
    "access_key": "xxx",
    "secret_key": "xxx"
  },
  "jimeng": {
    "api_key": "xxx"
  },
  "minimax": {
    "api_key": "xxx",
    "group_id": "xxx"
  }
}
```

---

## 简化流程（推荐）

如果数字人平台支持声纹克隆，可以跳过 voice-clone-tts：

```
video-script-generator
        ↓
   digital-avatar（形象+克隆+口播一站式）
        +
   scene-video-generator（AI场景）
        +
   demo-recorder（产品演示）
        ↓
   video-stitcher
        ↓
      成片
```

**只需配置一个后端（可灵或即梦），全流程打通。**
