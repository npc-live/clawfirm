---
name: video-stitcher
description: 视频片段拼接和后期处理。输入视频片段列表，输出完整视频。支持转场效果、背景音乐、字幕叠加。底层使用 FFmpeg 或 Remotion。触发词：拼接视频、合并视频、视频剪辑、video stitch、concatenate videos、add transitions。
---

# 视频拼接

## 功能

1. **拼接**：多个视频片段 → 完整视频
2. **转场**：添加片段间过渡效果
3. **音频**：背景音乐、音量调整
4. **字幕**：叠加字幕/文字
5. **导出**：多种格式和分辨率

## 工具选择

| 工具 | 适用场景 |
|------|----------|
| FFmpeg | 简单拼接、快速处理、命令行 |
| Remotion | 复杂动效、React 组件、品牌化 |

默认使用 FFmpeg。复杂需求用 Remotion。

## 输入参数

| 参数 | 必填 | 说明 |
|------|------|------|
| clips | ✓ | 视频片段列表（路径或URL）|
| output | ✓ | 输出路径 |
| transition | - | 转场类型 |
| transition_duration | - | 转场时长（默认0.5s）|
| bgm | - | 背景音乐路径 |
| bgm_volume | - | BGM音量（0-1，默认0.3）|
| subtitles | - | 字幕文件（SRT/ASS）|
| resolution | - | 输出分辨率 |
| fps | - | 帧率（默认30）|
| format | - | 输出格式（mp4/mov/webm）|

## 输入格式

```yaml
clips:
  - path: "scene_01.mp4"
    trim_start: 0        # 可选：裁剪起点
    trim_end: 5          # 可选：裁剪终点
    
  - path: "scene_02.mp4"
    transition: "fade"   # 到下一个片段的转场
    
  - path: "scene_03.mp4"

output: "final_video.mp4"

transition: "fade"           # 全局默认转场
transition_duration: 0.5

bgm: "background_music.mp3"
bgm_volume: 0.2
fade_bgm: true              # BGM 渐入渐出

subtitles: "captions.srt"

resolution: "1080x1920"     # 竖屏
fps: 30
format: "mp4"
```

## 转场类型

### FFmpeg 支持

| 转场 | 效果 | 命令 |
|------|------|------|
| fade | 淡入淡出 | `xfade=fade` |
| dissolve | 溶解 | `xfade=dissolve` |
| wipeleft | 左擦除 | `xfade=wipeleft` |
| wiperight | 右擦除 | `xfade=wiperight` |
| slideup | 上滑 | `xfade=slideup` |
| slidedown | 下滑 | `xfade=slidedown` |
| none | 无转场 | 硬切 |

### Remotion 额外支持

- 自定义 React 动画
- 品牌化转场
- 复杂遮罩效果

## FFmpeg 命令示例

### 简单拼接（无转场）

```bash
# 创建文件列表
cat > list.txt << EOF
file 'scene_01.mp4'
file 'scene_02.mp4'
file 'scene_03.mp4'
EOF

# 拼接
ffmpeg -f concat -safe 0 -i list.txt -c copy output.mp4
```

### 带转场拼接

```bash
ffmpeg -i scene_01.mp4 -i scene_02.mp4 \
  -filter_complex "[0:v][1:v]xfade=transition=fade:duration=0.5:offset=4.5[v]" \
  -map "[v]" output.mp4
```

### 添加背景音乐

```bash
ffmpeg -i video.mp4 -i bgm.mp3 \
  -filter_complex "[1:a]volume=0.3[bgm];[0:a][bgm]amix=inputs=2:duration=first[a]" \
  -map 0:v -map "[a]" output.mp4
```

### 添加字幕

```bash
ffmpeg -i video.mp4 -vf subtitles=captions.srt output.mp4
```

详见已有的 `ffmpeg-video-editor` skill。

## 工作流程

```
输入: clips[] + 配置
  ↓
验证所有片段可访问
  ↓
统一格式/分辨率（如需要）
  ↓
计算转场 offset
  ↓
生成 FFmpeg 命令 / Remotion 项目
  ↓
执行渲染
  ↓
输出: 完整视频
```

## 与上下游对接

**上游输入**：
- `scene-video-generator` 的视频片段
- `digital-avatar` 的口播视频
- Demo 视频片段

**Pipeline 集成示例**：

```yaml
# 从 video-script-generator 输出
scenes:
  - id: 1
    video: "hook.mp4"      # scene-video-generator 输出
    duration: 3s
    
  - id: 2
    video: "talking.mp4"   # digital-avatar 输出
    duration: 5s
    
  - id: 3
    video: "demo.mp4"      # demo skill 输出
    duration: 10s
    
  - id: 4
    video: "cta.mp4"       # scene-video-generator 输出
    duration: 3s

# 自动拼接
output: "final_promo.mp4"
transition: "fade"
bgm: "upbeat_music.mp3"
```

## 输出格式建议

| 平台 | 分辨率 | 比例 | 格式 |
|------|--------|------|------|
| 抖音/快手 | 1080x1920 | 9:16 | mp4 |
| 小红书 | 1080x1440 | 3:4 | mp4 |
| YouTube Shorts | 1080x1920 | 9:16 | mp4 |
| B站/YouTube | 1920x1080 | 16:9 | mp4 |
| 微信视频号 | 1080x1920 | 9:16 | mp4 |

## 注意事项

1. 输入片段分辨率不一致时，会自动缩放（可能影响质量）
2. 转场会减少总时长（每个转场减少 transition_duration）
3. BGM 时长不够会循环，太长会裁剪
4. 大量片段时建议分批处理
5. 4K 视频渲染较慢，考虑先用 1080p 预览
