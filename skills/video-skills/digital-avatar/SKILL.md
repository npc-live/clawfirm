---
name: digital-avatar
description: 数字人/虚拟形象生成和口播视频制作。支持多个后端：可灵 Kling、即梦 Jimeng、HeyGen、D-ID、Synthesia。输入形象描述或真人照片，输出数字人资源ID或口播视频片段。触发词：数字人、虚拟人、AI主播、avatar、口播视频、talking head。
---

# 数字人生成

## 功能

1. **创建数字人形象**：从描述或照片生成数字人
2. **声纹克隆**（可选）：上传音频样本 → 克隆声纹
3. **生成口播视频**：数字人 + 台词/音频 → 口播视频片段

## 语音来源选项

| 方式 | 说明 | 推荐场景 |
|------|------|----------|
| 平台内置 TTS | 用平台预设声音 | 快速测试 |
| 上传音频 | 提供录好的音频文件 | 有现成配音 |
| 平台声纹克隆 | 上传样本 → 平台克隆 | **推荐，全链路统一** |
| 外部 TTS | 用 MiniMax 等生成后上传 | 平台不支持克隆时 |

**推荐：优先用平台自带的声纹克隆，保持后端一致性。**

## ⚠️ 重要：后端一致性原则

**同一个数字人项目必须全程使用同一个后端！**

- 可灵的 avatar_id 和即梦的不互通
- 即梦创建的形象，可灵用不了，反之亦然
- 选定后端后，从创建形象到生成口播都用同一个

## 支持的后端

| 后端 | 数字人 | 口播 | 声纹克隆 | 特点 |
|------|--------|------|----------|------|
| Kling 可灵 | ✓ | ✓ | ✓ | 质量高，国产首选 |
| Jimeng 即梦 | ✓ | ✓ | ✓ | 快，中文口型好，剪映生态 |
| HeyGen | ✓ | ✓ | ✓ | 模板丰富，出海/英文 |
| D-ID | ✓ | ✓ | - | 简单快速 |
| Synthesia | ✓ | ✓ | ✓ | 企业级，多语言 |

**推荐：国内项目优先用可灵或即梦，二选一后全程使用。**

## 工作流程

### 流程 A：创建数字人

```
输入: 形象描述 / 真人照片
  ↓
选择后端
  ↓
调用 API 生成
  ↓
输出: avatar_id + 预览图
```

### 流程 B：生成口播视频

```
输入: avatar_id + 台词文本/音频
  ↓
调用后端口播 API
  ↓
等待渲染
  ↓
输出: 视频文件 URL
```

## 输入参数

### 创建数字人

| 参数 | 必填 | 说明 |
|------|------|------|
| mode | ✓ | create |
| backend | - | kling / jimeng / heygen / d-id / synthesia |
| description | △ | 形象描述（二选一）|
| photo | △ | 真人照片路径（二选一）|
| style | - | realistic / cartoon / 3d |
| gender | - | male / female |

### 声纹克隆（可选）

| 参数 | 必填 | 说明 |
|------|------|------|
| mode | ✓ | voice_clone |
| backend | - | kling / jimeng（需支持）|
| audio_sample | ✓ | 音频样本（10s-3min）|
| name | - | 声纹名称 |

输出：`voice_id`，后续生成口播时使用。

### 生成口播视频

| 参数 | 必填 | 说明 |
|------|------|------|
| mode | ✓ | generate |
| backend | - | 同上 |
| avatar_id | ✓ | 数字人 ID |
| text | △ | 台词文本（三选一）|
| audio | △ | 音频文件路径（三选一）|
| voice_id | △ | 克隆声纹 ID + text（三选一）|
| emotion | - | neutral / happy / serious |
| speed | - | 语速 0.5-2.0（默认1.0）|

## 输出格式

### 创建数字人

```yaml
avatar:
  id: "avatar_abc123"
  backend: jimeng
  preview_url: "https://..."
  style: realistic
  created_at: "2024-01-01T00:00:00Z"
```

### 生成口播视频

```yaml
video:
  id: "video_xyz789"
  avatar_id: "avatar_abc123"
  url: "https://..."
  duration: 15.5
  status: completed
```

## 后端配置

在 `openclaw.json` 中配置（只需配置你选用的后端）：

### Kling 可灵（推荐）

```json
{
  "kling": {
    "access_key": "your_access_key",
    "secret_key": "your_secret_key"
  }
}
```

### Jimeng 即梦

```json
{
  "jimeng": {
    "api_key": "ak-xxxxxxxx"
  }
}
```

### HeyGen

```json
{
  "heygen": {
    "api_key": "xxx"
  }
}
```

详见 [references/backend-setup.md](references/backend-setup.md)

## 使用示例

### 从描述创建

```
用户：帮我创建一个数字人，25岁左右的职业女性，干练短发

执行：
1. mode=create, description="25岁职业女性，干练短发", style=realistic
2. 调用 Jimeng API
3. 返回 avatar_id
```

### 从照片创建

```
用户：用这张照片创建数字人 [附图]

执行：
1. mode=create, photo=<图片路径>
2. 调用 API 上传照片
3. 返回 avatar_id
```

### 生成口播

```
用户：用 avatar_abc123 说这段话："大家好，今天教大家..."

执行：
1. mode=generate, avatar_id="avatar_abc123", text="大家好..."
2. 调用口播 API
3. 等待渲染完成
4. 返回视频 URL
```

## 与上下游对接

**上游**：`video-script-generator` 输出的 narration 字段

**下游**：`scene-video-generator` / `video-stitcher` 消费口播视频

## 注意事项

1. 真人照片需获得授权
2. 商用需确认后端的版权协议
3. 口播视频渲染可能需要 1-5 分钟
4. 建议缓存 avatar_id 避免重复创建
