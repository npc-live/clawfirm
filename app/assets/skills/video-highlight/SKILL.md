---
name: video-highlight
version: 0.2.0
description: "给视频生成高光片段或封面图。触发词：帮我做视频高光、给视频加文字、做封面、做缩略图、thumbnail、cover图、提取高光、视频加字幕。"
---

# Video Highlight Skill

## 工具
- `exec` — ffprobe / ffmpeg
- `media_understand` — 分析帧（AI视觉）
- `media_gen` — 生成封面图（model: gemini-3.1-flash-image-preview）
- `write` — 写脚本

---

## 封面生成流程

### 1. 提取帧
全程均匀抽 8 帧（不只取前10秒）：
```bash
mkdir -p /tmp/vhl_frames
duration=$(ffprobe -v quiet -show_entries format=duration -of csv=p=0 "VIDEO")
ffmpeg -y -i "VIDEO" -vf "fps=8/$(echo $duration | awk '{printf "%d", $1}')" \
  -vf "scale=960:-1" -frames:v 8 /tmp/vhl_frames/f%02d.jpg 2>/dev/null
```

### 2. 分析帧：判断类型 + 情绪
用 `media_understand` 分析所有帧，一次性回答：
```
分析这组视频帧，输出 JSON：
{
  "has_person": true/false,
  "person_desc": "外貌简述（若有人）",
  "emotion": "视频整体情绪：excited/surprised/serious/happy/shocked",
  "content_type": "tutorial/demo/story/news/achievement/other",
  "content_summary": "视频核心内容一句话",
  "best_frame": "最适合做封面的帧文件名",
  "hook_cn": "封面主标题（4-6字，强钩子，不是标题原文）",
  "sub_cn": "封面副标题（8-12字，补充悬念或结果）"
}
钩子写法参考（根据 content_type 选风格）：
- tutorial → 突出学完能获得什么成果，如「3天学会X」「这招让我少走5年弯路」
- demo → 突出震撼结果，如「AI替我做完了」「老板看完沉默了」
- achievement → 数字+结果，如「收益翻3倍」「0代码跑通了」
- story/news → 反转悬念，如「没想到结局是这样」「所有人都错了」
```

### 3. 生成封面背景（media_gen）

根据分析结果，按类型构建 prompt，调用 `media_gen(mode="generate", prompt=...)`：

**有人物（has_person: true）：**

用 `person_desc` 描述人物外貌，根据 `emotion` 和 `content_type` 决定表情和场景：

| content_type | emotion → 表情 | 场景风格 |
|---|---|---|
| tutorial | excited/happy → huge smile, proud | 展示成果，背景有代码/屏幕/成绩单 |
| demo | surprised/shocked → jaw-drop, eyes wide | 指着屏幕，背景显示惊人结果 |
| achievement | happy/excited → celebrating, fist pump | 奖杯/数字大屏/confetti |
| story/news | serious/shocked → intense stare | 戏剧性侧光，暗色背景 |

Prompt 模板：
```
Hyper-realistic YouTube thumbnail photo. {person_desc}, {expression based on emotion}.
{scene based on content_type}.
Dramatic cinematic lighting, sharp focus on face, shallow depth of field.
Shot on Sony A7R V, 85mm portrait lens. No text, no watermarks.
Aspect ratio 16:9, ultra high quality.
```

**无人物（has_person: false）：**

根据 `content_type` + `content_summary` 生成场景：

| content_type | 视觉方向 |
|---|---|
| tutorial | 发光的终端/代码屏幕，深色房间，蓝紫色氛围光 |
| demo | 产品/界面特写，霓虹科技感，戏剧性角度 |
| achievement | 数字/图表/奖杯特写，金色高光，胜利感 |
| news/story | 新闻感场景，强烈对比，电影感构图 |

Prompt 模板：
```
Hyper-realistic editorial photo. {scene matching content_summary and content_type}.
Cinematic composition, dramatic lighting, ultra sharp.
No text, no UI elements, no watermarks. Aspect ratio 16:9.
```

### 4. 叠加文字
```bash
/usr/bin/python3 $HOME/.skillctl/skills/video-highlight/scripts/compose_cover.py \
  --frame /tmp/vhl_cover_bg.png \
  --text1 "{hook_cn}" \
  --text2 "{sub_cn}" \
  --output "{视频同目录}/cover.png" \
  --style bold \
  --blur-bg 0
```

有人物用 `--style bold`；无人物且背景复杂用 `--style block`。

### 5. 清理
```bash
rm -rf /tmp/vhl_frames /tmp/vhl_cover_bg.png
```

---

## 高光片段流程（Step 6 FFmpeg）

用户要高光片段时执行：

**竖屏 9:16（抖音/小红书）：**
```bash
ffmpeg -y -ss START -t DURATION -i "VIDEO" \
  -vf "scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920,
       drawtext=fontfile=/System/Library/Fonts/PingFang.ttc:text='TEXT':
       fontsize=72:fontcolor=white:borderw=4:bordercolor=black:
       x=(w-text_w)/2:y=h*0.75" \
  -c:v libx264 -crf 18 -preset fast -c:a aac -b:a 128k "OUTPUT.mp4"
```

**横屏 16:9（B站/YouTube）：**
```bash
ffmpeg -y -ss START -t DURATION -i "VIDEO" \
  -vf "scale=1920:1080:force_original_aspect_ratio=increase,crop=1920:1080,
       drawtext=fontfile=/System/Library/Fonts/PingFang.ttc:text='TEXT':
       fontsize=64:fontcolor=white:borderw=4:bordercolor=black:
       x=(w-text_w)/2:y=h*0.82" \
  -c:v libx264 -crf 18 -preset fast -c:a aac -b:a 128k "OUTPUT.mp4"
```

Linux 字体路径换 `/usr/share/fonts/` 下的黑体；`text='TEXT'` 中 `:` 转义为 `\:`。
