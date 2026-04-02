---
name: skill-index
description: |
  Master index of all clawfirm agent capabilities — skills, tools, and browser
  shortcuts. Activate when the user asks "what can you do", "what tools/skills
  do you have", "list capabilities", "help me find a skill", "功能列表",
  "你能做什么", "有哪些技能", or wants to discover what's available.
---

# Clawfirm Skill & Tool Index

You have access to **skills**, **built-in tools**, and **browser shortcuts**. Most advanced capabilities are packaged as skills that you load on demand.

> **Important**: To use a skill, invoke it by name with the `skill` tool: `/skill: <name>`

---

## Skills

### Browser Automation

| Skill | What it does |
|-------|--------------|
| **agent-browser** | Control Chrome via CDP — open URLs, snapshot interactive elements, click, fill forms, screenshot, scrape. Requires Chrome with `--remote-debugging-port=9222`. |
| **social-cli** | 社交平台自动化完整参考 — 小红书/抖音/B站/X/Boss直聘 全部命令及参数文档。 |

### Social Media Publishing

| Skill | What it does |
|-------|--------------|
| **twitter** | Twitter/X algorithm-optimized posting, hashtag strategy, reply weighting, best posting times, content rules. |
| **xiaohongshu** | 小红书 — 审核规避（禁用词+限流规则）、CES 算法适配、最佳发布时间。 |
| **douyin** | 抖音 — 话题标签策略、审核规避（机审+人审）、完播率优化、五级流量池。 |
| **bilibili** | B站 — 分区策略、一键三连引导、弹幕文化适配、稿件审核、创作者激励。 |
| **copywriting-base** | 病毒传播文案基础 — 中英双语写作规范、传播因子分析。被其他发布 skill 引用。 |

### Video Production

| Skill | What it does |
|-------|--------------|
| **remotion-video** | Remotion 视频工厂 — 分析仓库或对话上下文，自动生成 demo/promo 视频项目。 |
| **video-script-generator** | 视频脚本生成 — 剧情反转、痛点解决、前后对比等模板。 |
| **video-stitcher** | 视频拼接 — 多素材片段合成完整视频。 |
| **digital-avatar** | 数字人 — AI 虚拟形象视频生成。 |
| **scene-video-generator** | 场景视频 — 基于场景描述生成视频片段。 |
| **voice-clone-tts** | 语音克隆 TTS — 克隆声音并生成配音。 |

### DevOps

| Skill | What it does |
|-------|--------------|
| **clawfirm** | clawfirm CLI 使用指南 — install, login, new, run 等命令。 |

---

## Browser Shortcuts (CDP Automation)

Pre-built YAML browser automations in `~/.clawfirm/shortcuts/`. These control Chrome via CDP to automate social platform tasks.

**Prerequisite**: Chrome must be running with `--remote-debugging-port=9222`.

| File | Platform | Available Commands |
|------|----------|--------------------|
| `xhs.yaml` | 小红书 | `search` `hot` `like` `comment` `post` `post_video` |
| `douyin.yaml` | 抖音 | `search` `like` `comment` `follow` `post` `download` |
| `bilibili.yaml` | B站 | `search` `like` `comment` `reply` `follow` `post` |
| `x.yaml` | X/Twitter | `search` `like` `reply` `post` `retweet` |
| `zhipin.yaml` | Boss直聘 | `chat_stats` `candidates` |

To run a shortcut, use the **Browser** panel in the app UI, or ask the agent (e.g., "搜索小红书 AI法律", "下载这个抖音视频").

---

## Built-in Tools

Always available (when enabled in agent config):

| Tool | Description |
|------|-------------|
| `read` / `write` / `edit` | File operations |
| `bash` / `exec` / `process` | Shell commands and processes |
| `grep` / `find` / `ls` | Search and list files |
| `fetch` / `web_search` | HTTP requests and web search |
| `skill` | Load a skill by name |
| `sub_agent` | Spawn parallel sub-agents |
| `memory_search` / `memory_get` | Semantic memory retrieval |
| `whipflow_run` | 执行 .whip 工作流（file/source/user_inputs/retry_from_session） |
| `ask_user` | Ask user for input |

---

## Quick Examples

| User says | What happens |
|-----------|-------------|
| "搜索小红书 AI法律" | → `agent-browser` skill → `xhs.yaml search AI法律` |
| "下载抖音视频 https://v.douyin.com/xxx" | → `agent-browser` skill → `douyin.yaml download` |
| "写一条推特" | → `twitter` + `copywriting-base` skills |
| "生成产品演示视频脚本" | → `video-script-generator` skill |
| "发布视频到B站" | → `bilibili` skill + `agent-browser` for automation |
| "帮我在Boss直聘看候选人" | → `agent-browser` skill → `zhipin.yaml candidates` |
