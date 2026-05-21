---
name: social-distribute
description: 一键多平台视频分发。编译脚本自动读取 content-rules、执行 CDP 填表，LLM 仅参与文案生成。Token 消耗约为原流程的 10%。
allowed-tools: Bash(skflow *, echo *, node *)
---

# social-distribute

一键将视频分发到多个社媒平台（填表不发布）。

## Usage

1. Set environment variable with distribution params:
```bash
export SOCIAL_DISTRIBUTE_INPUT='{"video":"/path/to/video.mp4","topic":"视频主题描述","platforms":["twitter","bilibili","youtube","tiktok","xiaohongshu","douyin"],"cover_h":"/path/to/cover_h.png","cover_v":"/path/to/cover_v.png"}'
```

2. Run the compiled script:
```bash
node scripts/skflow-run.mjs run social-distribute
```

3. Parse the JSON output:
   - If `yield`: read `yield.prompt` and `yield.data`, generate the requested copywriting JSON, then resume:
     ```bash
     node scripts/skflow-run.mjs resume <session> --answer='<copywriting JSON>'
     ```
   - If `done`: display `done.summary` — distribution complete
   - If `error`: report `error.message`

## Supported platforms

twitter, bilibili, youtube, tiktok, xiaohongshu, douyin, linkedin, wechat_channels, wechat_moments

## How it works

The compiled script (`.skflow/skills/social-distribute/script.compiled.js`) runs as a state machine:
- **Deterministic steps** (sh): CDP check, port detect, env read, batch-read content-rules, execute browser-shortcuts — all auto-executed, no LLM needed
- **Intelligent step** (ask): one single yield for LLM to generate copywriting for ALL platforms at once
- **Result**: done with per-platform success/failure summary
