---
name: cover-gen
version: 1.0.0
description: |
  社交媒体封面生成 + 视频首帧插入。从视频成片提取关键帧，按各平台规格 AI 生成封面，
  最终将封面作为视频第一帧（0.1秒）合成到视频中。底层使用 media_gen (Gemini/OpenAI) + FFmpeg。
  触发词：封面、cover、生成封面、封面制作、首帧、cover gen、thumbnail、视频封面。
---

# 封面生成 + 视频首帧插入

## 功能

1. **提取关键帧**：从视频中截取参考帧用于风格指导
2. **平台适配**：按目标平台规格（尺寸、比例、文字规范）生成封面
3. **AI 生成**：通过 media_gen 工具（Gemini/OpenAI）生成封面图片
4. **首帧合成**：将封面作为视频第一帧（0.1秒），用 FFmpeg concat 合成

## 工作流概览

```
输入：视频成片 + 平台信息
  ↓
Step 1: 提取视频关键帧 + 技术参数 (ffprobe/ffmpeg)
  ↓
Step 2: 确定平台封面规格（尺寸、比例、文字规范）
  ↓
Step 3: AI 生成封面 (media_gen / cmd/media-gen)
  ↓
Step 4: 封面合成到视频第一帧 — 0.1s (ffmpeg concat)
  ↓
输出：带封面首帧的视频 + 独立封面图
```

---

## Step 1: 提取视频信息 + 关键帧

从原始视频中获取技术参数和关键帧，用于指导封面生成。

```bash
# 获取视频分辨率、帧率、时长
ffprobe -v error \
  -select_streams v:0 \
  -show_entries stream=width,height,r_frame_rate,codec_name \
  -show_entries format=duration \
  -of json \
  "$VIDEO_PATH"

# 截取关键帧（0s, 3s, 中间帧）作为参考
ffmpeg -y -i "$VIDEO_PATH" \
  -vf "select='eq(n\,0)+eq(n\,90)+eq(n\,$(echo "$TOTAL_FRAMES/2" | bc))',setpts=N/FRAME_RATE/TB" \
  -frames:v 3 -q:v 2 \
  "/tmp/cover_ref_frame_%d.jpg"
```

**输出：**
- `width`, `height`, `fps`, `duration` — 视频技术参数
- `cover_ref_frame_1.jpg` ~ `cover_ref_frame_3.jpg` — 关键帧参考图

---

## Step 2: 平台封面规格

根据目标发布平台确定封面规格：

| 平台 | 尺寸 | 比例 | media_gen size | 文字规范 |
|------|------|------|----------------|----------|
| 抖音 | 1080x1440 | 3:4 | `portrait` | ≤8字大标题, 高对比度, 人脸+30% CTR |
| 抖音(竖) | 1080x1920 | 9:16 | `portrait` | 同上 |
| 小红书 | 1080x1440 | 3:4 | `portrait` | ≤10字, 关键信息可视化 |
| B站 | 1920x1080 | 16:9 | `landscape` | ≤12字+描边, 可高文字密度, 【分类】标注 |
| LinkedIn | 1920x1080 | 16:9 | `landscape` | 英文标题, 专业风格 |
| Twitter | 1600x900 | 16:9 | `landscape` | 简洁大字, 品牌色 |
| YouTube | 1280x720 | 16:9 | `landscape` | ≤10字, 高饱和度 |

详见各平台 SKILL.md 的封面规范章节：
- `social-publish/douyin/SKILL.md` — 封面规范
- `social-publish/xiaohongshu/SKILL.md` — 封面设计要点
- `social-publish/bilibili/SKILL.md` — 封面规范

---

## Step 3: AI 生成封面

### 3a. 内置 media_gen 工具

`tool/builtin/media_gen.go` — 直接在 ClawFirm 内调用：

```json
{
  "prompt": "封面描述...",
  "size": "portrait",
  "output": "~/Desktop/cover_20260415.png"
}
```

支持 Gemini (默认) 和 OpenAI (dall-e-3) 两个后端：
- **Gemini**: `gemini-2.5-flash-preview-04-17`, aspect ratio 通过 prompt 提示
- **OpenAI**: `dall-e-3`, 精确尺寸映射 (portrait=1024x1792, landscape=1792x1024)

### 3b. CLI 插件 media-gen

`cmd/media-gen/main.go` — 独立 CLI 工具，支持参考图风格迁移：

```bash
CLAWD_TOOL_INPUT='{"prompt":"...", "output_path":"/tmp/cover.png", "size":"portrait", "reference_images":["/tmp/cover_ref_frame_1.jpg"]}' \
  ./bin/media-gen
```

**参考图 (reference_images) 用法：**
- 传入视频关键帧作为风格参考
- AI 模型会基于参考图风格生成封面
- 支持多张参考图

支持的尺寸：`portrait` (9:16), `landscape` (16:9), `landscape_4x3` (4:3), `square` (1:1)

### 3c. 封面 prompt 模板

```
为{平台}制作视频封面。

视频主题：{title}
核心卖点：{desc中的关键信息}

要求：
- 尺寸：{平台对应尺寸}
- 大字标题："{≤N字标题}"，高对比度，加描边/阴影
- 风格：{平台风格}
- 人物/产品突出展示
- 背景简洁不杂乱
- 关键数字/利益点可视化
```

---

## Step 4: 封面合成到视频第一帧 (0.1s)

生成的封面插入为视频的第一帧，持续 0.1 秒，然后紧接原始视频。

### 完整 ffmpeg 命令

```bash
#!/bin/bash
# 输入参数
COVER_IMAGE="$1"   # 封面图片路径
VIDEO_INPUT="$2"   # 原始视频路径
VIDEO_OUTPUT="$3"  # 输出视频路径

# Step 4a: 获取原视频的分辨率、帧率和 timescale
eval $(ffprobe -v error -select_streams v:0 \
  -show_entries stream=width,height,r_frame_rate,time_base \
  -of default=noprint_wrappers=1 "$VIDEO_INPUT")
FPS=$(echo "$r_frame_rate" | bc -l | xargs printf "%.0f")
TIMESCALE=$(echo "$time_base" | sed 's|1/||')

# Step 4b: 将封面图片转为 0.1 秒的视频片段
# 匹配原视频的分辨率、帧率、编码和 timescale，确保无缝拼接
# ⚠️ -video_track_timescale 必须匹配原视频，否则 concat -c copy 会导致画面加速
ffmpeg -y \
  -loop 1 -i "$COVER_IMAGE" \
  -t 0.1 \
  -vf "scale=${width}:${height}:force_original_aspect_ratio=decrease,pad=${width}:${height}:(ow-iw)/2:(oh-ih)/2:black" \
  -r "$FPS" \
  -c:v libx264 -preset fast -crf 18 \
  -pix_fmt yuv420p \
  -video_track_timescale "$TIMESCALE" \
  "/tmp/cover_clip.mp4"

# Step 4c: 统一原视频编码格式（确保 concat 兼容）
ffmpeg -y \
  -i "$VIDEO_INPUT" \
  -c:v libx264 -preset fast -crf 18 \
  -pix_fmt yuv420p \
  -r "$FPS" \
  -video_track_timescale "$TIMESCALE" \
  -c:a aac -b:a 128k \
  "/tmp/video_normalized.mp4"

# Step 4d: 使用 concat 协议拼接（封面片段 + 原视频）
cat > /tmp/concat_list.txt << EOF
file '/tmp/cover_clip.mp4'
file '/tmp/video_normalized.mp4'
EOF

ffmpeg -y \
  -f concat -safe 0 \
  -i /tmp/concat_list.txt \
  -c copy \
  "$VIDEO_OUTPUT"

# 清理临时文件
rm -f /tmp/cover_clip.mp4 /tmp/video_normalized.mp4 /tmp/concat_list.txt

echo "Done: $VIDEO_OUTPUT"
```

### 简化版（原视频已是 H.264/yuv420p 时）

如果原视频已经是标准 H.264 编码，可以跳过重编码：

```bash
# 封面 → 0.1s 视频（匹配原视频参数 + timescale）
ffmpeg -y -loop 1 -i "$COVER_IMAGE" \
  -t 0.1 \
  -vf "scale=${width}:${height}:force_original_aspect_ratio=decrease,pad=${width}:${height}:(ow-iw)/2:(oh-ih)/2:black" \
  -r "$FPS" -c:v libx264 -preset fast -crf 18 -pix_fmt yuv420p \
  -video_track_timescale "$TIMESCALE" \
  -an \
  "/tmp/cover_clip.mp4"

# 拼接
cat > /tmp/concat_list.txt << EOF
file '/tmp/cover_clip.mp4'
file '$VIDEO_INPUT'
EOF

ffmpeg -y -f concat -safe 0 -i /tmp/concat_list.txt -c copy "$VIDEO_OUTPUT"
```

### 带音频处理的版本

如果原视频有音频轨道，封面片段需要添加静音音轨才能使用 concat 协议：

```bash
# 封面 → 0.1s 视频 + 静音音轨（匹配 timescale）
ffmpeg -y -loop 1 -i "$COVER_IMAGE" \
  -f lavfi -i anullsrc=r=44100:cl=stereo \
  -t 0.1 \
  -vf "scale=${width}:${height}:force_original_aspect_ratio=decrease,pad=${width}:${height}:(ow-iw)/2:(oh-ih)/2:black" \
  -r "$FPS" -c:v libx264 -preset fast -crf 18 -pix_fmt yuv420p \
  -video_track_timescale "$TIMESCALE" \
  -c:a aac -b:a 128k \
  -shortest \
  "/tmp/cover_clip.mp4"
```

### 关键参数说明

| 参数 | 说明 |
|------|------|
| `-loop 1` | 将静态图片循环为视频流 |
| `-t 0.1` | 持续 0.1 秒（约 3 帧 @30fps） |
| `scale=W:H` | 匹配原视频分辨率 |
| `pad=W:H` | 居中填充黑边（当比例不一致时） |
| `-r $FPS` | 匹配原视频帧率 |
| `-pix_fmt yuv420p` | 标准像素格式，确保 concat 兼容 |
| `-video_track_timescale` | **必须匹配原视频的 timescale，否则 concat copy 会导致画面加速** |
| `-crf 18` | 高质量编码 |
| `-c copy` | concat 阶段零损耗拷贝 |

---

## 与上下游对接

### 上游（视频来源）

- `video-stitcher` 的最终成片
- 用户直接提供的原始视频
- `digital-avatar` 生成的数字人视频

### 下游（发布渠道）

- `social-publish/{platform}` 平台 Skill 提供文案 + 封面 prompt
- `adapters/{platform}.yaml` CDP 自动发布

### Pipeline 集成

```
video-skills/video-stitcher → 视频成片
        ↓
social-publish/{platform}/SKILL.md → 文案 + 封面 prompt
        ↓
cover-gen (media_gen) → 封面图片
        ↓
cover-gen (ffmpeg concat) → 带封面首帧的视频
        ↓
adapters/{platform}.yaml (CDP) → 自动发布
```

### 与 daily-content.whip 对接

在 `daily-content.whip` 内容创作完成后、`daily-publish.whip` 发布之前：

```
daily-content.whip → 文案草稿 (status=approved)
        ↓
cover-gen (media_gen) → 各平台封面
        ↓
cover-gen (ffmpeg) → 视频 + 封面首帧
        ↓
daily-publish.whip → 发布
```

### 与 media.whip (抖音爆款) 对接

`media.whip` Phase 5h 视频汇总后新增：

```
Phase 5h: 视频汇总 → final video
        ↓
Phase 5i: 封面生成 (media_gen, 参考尾帧/关键帧)
        ↓
Phase 5j: 封面 → 视频首帧 (ffmpeg, 0.1s)
        ↓
Phase 6: 发布/总结
```

---

## 多平台批量封面 + 首帧示例

为同一个视频生成多平台封面并各自合成首帧：

```bash
#!/bin/bash
VIDEO="./final_video.mp4"
TITLE="AI帮你找人脉"

# 获取视频参数
eval $(ffprobe -v error -select_streams v:0 \
  -show_entries stream=width,height,r_frame_rate \
  -of default=noprint_wrappers=1 "$VIDEO")

# 截取参考帧
ffmpeg -y -i "$VIDEO" -vf "select=eq(n\,90)" -frames:v 1 -q:v 2 /tmp/ref_frame.jpg

# 抖音版封面（竖屏 3:4）
CLAWD_TOOL_INPUT='{"prompt":"为抖音制作视频封面。主题：'"$TITLE"'。大字标题，高对比度，人物突出，竖屏3:4比例","output_path":"/tmp/cover_douyin.png","size":"portrait","reference_images":["/tmp/ref_frame.jpg"]}' ./bin/media-gen

# B站版封面（横屏 16:9）
CLAWD_TOOL_INPUT='{"prompt":"为B站制作视频封面。主题：'"$TITLE"'。【教程】标签，大字标题+描边，横屏16:9比例","output_path":"/tmp/cover_bilibili.png","size":"landscape","reference_images":["/tmp/ref_frame.jpg"]}' ./bin/media-gen

# 为每个平台合成首帧
for PLATFORM in douyin bilibili; do
  COVER="/tmp/cover_${PLATFORM}.png"
  OUTPUT="./final_${PLATFORM}.mp4"

  case $PLATFORM in
    douyin)   W=1080; H=1920 ;;
    bilibili) W=1920; H=1080 ;;
  esac

  # 封面 → 0.1s 片段（含静音音轨）
  ffmpeg -y -loop 1 -i "$COVER" \
    -f lavfi -i anullsrc=r=44100:cl=stereo \
    -t 0.1 -vf "scale=${W}:${H}:force_original_aspect_ratio=decrease,pad=${W}:${H}:(ow-iw)/2:(oh-ih)/2:black" \
    -r 30 -c:v libx264 -preset fast -crf 18 -pix_fmt yuv420p \
    -c:a aac -b:a 128k -shortest \
    "/tmp/cover_clip_${PLATFORM}.mp4"

  # 拼接
  echo "file '/tmp/cover_clip_${PLATFORM}.mp4'" > /tmp/concat_${PLATFORM}.txt
  echo "file '$(realpath $VIDEO)'" >> /tmp/concat_${PLATFORM}.txt
  ffmpeg -y -f concat -safe 0 -i /tmp/concat_${PLATFORM}.txt -c copy "$OUTPUT"

  echo "Done: $OUTPUT"
done
```

---

## 注意事项

1. 封面图片和视频分辨率比例不一致时，会自动居中 + 黑边填充
2. 0.1 秒在 30fps 下约为 3 帧，60fps 下约为 6 帧
3. concat 协议要求所有片段编码参数一致（codec、分辨率、帧率、pix_fmt）
4. 如果原视频有音频轨道，封面片段必须包含静音音轨
5. 大分辨率视频（4K）建议先用 1080p 测试效果
