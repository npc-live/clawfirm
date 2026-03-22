---
name: scene-video-generator
description: 根据分镜描述生成视频片段。支持多个 AI 视频生成后端：即梦 Jimeng、Kling 可灵、Runway、Pika、Vidu。输入场景描述+可选的数字人口播，输出视频片段。触发词：AI视频、生成视频、分镜视频、scene video、text to video、图生视频。
---

# 分镜视频生成

## 功能

将分镜描述转换为 **AI 场景视频**（不含数字人）：
- 纯 AI 生成场景
- 图生视频（从参考图扩展）

## ⚠️ 与 digital-avatar 的分工

| 需求 | 用哪个 skill |
|------|-------------|
| 数字人口播视频 | → `digital-avatar` |
| 纯 AI 场景（无人/路人/产品展示等）| → `scene-video-generator`（本 skill）|

**数字人相关的视频生成请用 `digital-avatar` skill，确保后端一致性。**

## 支持的后端

| 后端 | 特点 | 时长限制 | 适用场景 |
|------|------|----------|----------|
| Jimeng 即梦 | 国内，快，中文理解好 | 5s | 国内平台 |
| Kling 可灵 | 国内，质量高 | 5-10s | 高质量需求 |
| Runway | 国际，Gen-3 质量顶级 | 5-10s | 高端制作 |
| Pika | 快速，风格化强 | 3-5s | 风格化内容 |
| Vidu | 国内，性价比高 | 4-8s | 一般需求 |

默认使用 Jimeng（如已配置）。

## 输入参数

| 参数 | 必填 | 说明 |
|------|------|------|
| backend | - | jimeng / kling / runway / pika / vidu |
| prompt | ✓ | 场景描述（英文效果更好）|
| duration | - | 目标时长（秒）|
| aspect_ratio | - | 16:9 / 9:16 / 1:1 |
| reference_image | - | 参考图（图生视频）|
| style | - | realistic / anime / cinematic |
| motion | - | low / medium / high |
| seed | - | 随机种子（复现用）|

## 输出格式

```yaml
video:
  id: "scene_001"
  backend: kling
  url: "https://..."
  duration: 5.0
  resolution: "1920x1080"
  aspect_ratio: "16:9"
  prompt: "原始 prompt"
  seed: 12345
  status: completed
```

## 工作流程

### 流程 A：纯 AI 生成

```
输入: prompt + 参数
  ↓
翻译/优化 prompt（如需要）
  ↓
调用后端 API
  ↓
等待渲染（30s-3min）
  ↓
输出: 视频 URL
```

### 流程 B：图生视频

```
输入: reference_image + prompt
  ↓
上传参考图
  ↓
调用 image-to-video API
  ↓
输出: 视频 URL
```

> 💡 **需要数字人口播？** 请使用 `digital-avatar` skill。

## Prompt 优化技巧

### 结构

```
[主体] + [动作] + [场景] + [镜头] + [风格]
```

### 示例

```
原始：女生在办公室看电脑

优化：A young professional woman sitting at a modern office desk, 
looking at computer screen with focused expression, 
soft natural lighting from window, 
medium shot, shallow depth of field, 
cinematic color grading
```

### 镜头术语

| 中文 | 英文 | 说明 |
|------|------|------|
| 特写 | close-up | 脸部/细节 |
| 中景 | medium shot | 半身 |
| 全景 | wide shot | 全身+环境 |
| 跟拍 | tracking shot | 跟随移动 |
| 推镜 | dolly in | 推进 |
| 拉镜 | dolly out | 拉远 |

详见 [references/prompt-guide.md](references/prompt-guide.md)

## 使用示例

### 生成场景

```
用户：生成一个5秒的视频，内容是：一个人在咖啡厅用笔记本工作

执行：
1. 优化 prompt
2. 调用 Kling API
3. 等待渲染
4. 返回视频 URL
```

### 批量生成分镜

```
用户：根据这个分镜列表生成视频 [附 YAML]

执行：
1. 解析分镜列表
2. 逐个生成（或并行）
3. 返回视频列表
```

## 与上下游对接

**上游输入**：
- `video-script-generator` 的 scenes[].shot_description（非口播场景）

**下游输出**：
- `video-stitcher` 消费视频片段

**并行 skill**：
- `digital-avatar` 负责口播场景视频
- 本 skill 负责 AI 场景视频
- 两者输出都汇入 `video-stitcher`

## 注意事项

1. 不同后端对 prompt 长度有限制（通常 500 字符内）
2. 复杂动作/多人场景效果不稳定
3. 建议先小批量测试再批量生成
4. 保存 seed 以便复现满意的结果
5. 渲染时间可能较长，使用异步处理
