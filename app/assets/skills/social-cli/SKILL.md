---
name: social-cli
description: |
  Social media browser automation reference — full command & parameter docs
  for all supported platforms (小红书, 抖音, B站, X/Twitter, Boss直聘).
  Activate when the user asks about social media shortcuts, wants to
  automate social platforms, "怎么发小红书", "帮我点赞", "搜索抖音",
  "post to twitter", "bilibili comment", "Boss直聘候选人", etc.
---

# Social CLI — Browser Shortcut Reference

Pre-built YAML browser automations for social media platforms. Executed via the **Browser** panel or by asking the agent directly.

**Prerequisite**: Chrome must be running with `--remote-debugging-port=9222`. Use the Browser panel's "Launch Chrome" button or run manually.

**Login**: Each platform requires you to be logged in. On first use, log in manually in the launched Chrome. Sessions (cookies) are cached in `~/.clawfirm/sessions/`.

---

## 小红书 (Xiaohongshu) — `xhs.yaml`

Login URL: `https://www.xiaohongshu.com`
Session cookie: `web_session`

### `search` — 搜索笔记

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `keyword` | string | yes | 搜索关键词 |

Returns: `title`, `author`, `link` (笔记列表)

Example: `搜索小红书 AI法律`

---

### `hot` — 热门推荐

No parameters.

Returns: `title`, `author`, `link` (探索页热门笔记)

Example: `看看小红书热门`

---

### `like` — 点赞/取消点赞

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 笔记 URL |

Returns: `操作` (点赞成功/取消点赞), `状态`, `点赞数`

Example: `点赞这篇小红书 https://www.xiaohongshu.com/explore/xxx`

---

### `comment` — 发表评论

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 笔记 URL |
| `text` | string | yes | 评论内容 |

Returns: `状态`, `评论内容`, `输入确认`

Example: `评论小红书 https://www.xiaohongshu.com/explore/xxx 写得好！`

---

### `post_video` — 发布视频笔记

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `video` | string | yes | 本地视频文件路径 (绝对路径) |
| `title` | string | yes | 笔记标题 |
| `desc` | string | yes | 笔记描述/正文 |

Returns: `状态`, `标题`, `描述`, `跳转URL`

Note: 上传后需等待转码完成 (最长 5 分钟)。自动点击"发布"按钮。

Example: `发小红书视频 /tmp/demo.mp4 标题"AI演示" 描述"看看这个AI"`

---

### `post` — 发布图文笔记

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | yes | 笔记标题 |
| `content` | string | yes | 笔记正文 |
| `image` | string | yes | 本地图片路径 (绝对路径, 小红书图文必须有图) |

Returns: `状态`, `标题`, `正文`, `跳转URL`

Example: `发小红书图文 标题"周末探店" 内容"超好吃的..." 图片 /tmp/food.jpg`

---

## 抖音 (Douyin) — `douyin.yaml`

Login URL: `https://www.douyin.com`
Session cookie: `passport_csrf_token`

### `search` — 搜索视频

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `keyword` | string | yes | 搜索关键词 |

Returns: `link`, `text` (视频列表)

Example: `搜索抖音 猫咪日常`

---

### `like` — 点赞/取消点赞

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 视频 URL |

Returns: `操作` (点赞成功/取消点赞), `状态`

Note: 使用键盘快捷键 `z` 触发点赞。

Example: `点赞抖音 https://www.douyin.com/video/xxx`

---

### `comment` — 发表评论

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 视频 URL |
| `text` | string | yes | 评论内容 |

Returns: `状态`, `评论内容`, `输入确认`

Note: 抖音评论区使用 Draft.js 编辑器，通过 CDP `keyboard_insert` 逐字符输入，按 Enter 发送。

Example: `评论抖音 https://www.douyin.com/video/xxx 太搞笑了哈哈`

---

### `follow` — 关注作者

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 视频 URL (从视频页关注作者) |

Returns: `操作` (关注成功/已在关注中), `关注按钮消失`

Example: `关注这个抖音作者 https://www.douyin.com/video/xxx`

---

### `download` — 下载视频/图文

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 视频或图文 URL (支持分享短链 `v.douyin.com`) |

Returns: `类型` (video/gallery), `作者`, `描述`, `作品ID`, `下载状态`

Note: 自动识别视频或图文。视频通过 `video.currentSrc` 获取 CDN 地址并触发浏览器下载；图文逐张下载。文件保存到 Chrome 默认下载目录。

Example: `下载抖音视频 https://v.douyin.com/LhdEZKD2H0Q/`

---

### `post` — 发布视频

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `video` | string | yes | 本地视频文件路径 (绝对路径) |
| `title` | string | yes | 作品标题 |
| `desc` | string | yes | 作品描述 |

Returns: `状态`, `标题`, `描述`, `跳转URL`

Note: 通过抖音创作者平台上传。自动等待上传完成 (最长 5 分钟)，自动选择封面，跳过竖封面提示。

Example: `发布抖音视频 /tmp/vlog.mp4 "日常vlog" "今天去了..."`

---

## B站 (Bilibili) — `bilibili.yaml`

Login URL: `https://www.bilibili.com`
Session cookie: `SESSDATA`

### `search` — 搜索视频

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `keyword` | string | yes | 搜索关键词 |

Returns: `link`, `title`, `up` (视频列表)

Example: `搜索B站 编程教程`

---

### `like` — 点赞视频

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 视频 URL |

Returns: `操作` (点赞成功/取消点赞), `当前状态`

Example: `点赞B站 https://www.bilibili.com/video/BVxxx`

---

### `follow` — 关注 UP 主

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 视频 URL (从视频页关注UP主) |

Returns: `操作` (关注/取消关注), `状态`

Example: `关注这个UP主 https://www.bilibili.com/video/BVxxx`

---

### `comment` — 发表评论

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 视频 URL |
| `text` | string | yes | 评论内容 |

Returns: `状态`, `评论内容`, `输入确认`

Note: B站评论区使用多层 Shadow DOM (`bili-comments → bili-comment-box → bili-comment-rich-textarea`)。通过 CDP `Input.insertText` 穿透 Shadow DOM 输入文字。

Example: `评论B站 https://www.bilibili.com/video/BVxxx 三连了！`

---

### `reply` — 回复评论

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 视频 URL |
| `text` | string | yes | 回复内容 |

Returns: `状态`, `回复内容`

Note: 自动点击第一条评论的"回复"按钮，在展开的回复框中输入并发送。同样使用 CDP `Input.insertText` 穿透 Shadow DOM。

Example: `回复B站评论 https://www.bilibili.com/video/BVxxx 说得对！`

---

### `post` — 投稿视频

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `video` | string | yes | 本地视频文件路径 (绝对路径) |
| `title` | string | yes | 稿件标题 |
| `desc` | string | yes | 稿件简介 |

Returns: `状态`, `标题`, `描述`, `页面URL`

Note: 通过 B站创作中心上传。自动关闭弹窗 (开启通知、二创计划等)，等待上传转码完成 (最长 5 分钟)，使用 Quill 富文本编辑器填写简介，点击"立即投稿"。

Example: `投稿B站 /tmp/tutorial.mp4 "Go教程第1集" "从零开始学Go"`

---

## X/Twitter — `x.yaml`

Login URL: `https://x.com`
Session cookie: `auth_token`

### `search` — 搜索推文

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `keyword` | string | yes | 搜索关键词 |

Returns: `text`, `user`, `link`, `time` (推文列表, 按热度排序)

Example: `搜索推特 Claude AI`

---

### `like` — 点赞/取消点赞

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 推文 URL |

Returns: `操作` (点赞成功/取消点赞), `状态`

Example: `点赞这条推特 https://x.com/user/status/xxx`

---

### `reply` — 回复推文

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 推文 URL |
| `text` | string | yes | 回复内容 |

Returns: `状态`, `回复内容`, `输入确认`

Example: `回复推特 https://x.com/user/status/xxx Great work!`

---

### `post` — 发推

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `text` | string | yes | 推文内容 |

Returns: `状态`, `内容`, `输入确认`

Example: `发一条推特 Just shipped a new feature!`

---

### `retweet` — 转推

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | 推文 URL |

Returns: `状态`, `URL`

Example: `转推 https://x.com/user/status/xxx`

---

## Boss直聘 (Zhipin) — `zhipin.yaml`

Login URL: `https://www.zhipin.com`
Session cookie: `wt2`

### `chat_stats` — 聊天统计

No parameters.

Returns: `总会话数`, `未读会话数`, `未读明细` (姓名, 应聘职位, 未读条数, 最新消息)

Note: 自动滚动加载所有聊天记录后统计。

Example: `Boss直聘聊天统计`

---

### `candidates` — 候选人列表

No parameters.

Returns: `姓名`, `应聘职位`, `最新消息` (所有候选人)

Note: 自动滚动加载所有聊天记录后提取。

Example: `看看Boss直聘有哪些候选人`

---

## Command Summary

| Platform | Command | Args | Description |
|----------|---------|------|-------------|
| 小红书 | `search` | `keyword` | 搜索笔记 |
| 小红书 | `hot` | — | 热门推荐 |
| 小红书 | `like` | `url` | 点赞/取消 |
| 小红书 | `comment` | `url`, `text` | 发表评论 |
| 小红书 | `post_video` | `video`, `title`, `desc` | 发布视频 |
| 小红书 | `post` | `title`, `content`, `image` | 发布图文 |
| 抖音 | `search` | `keyword` | 搜索视频 |
| 抖音 | `like` | `url` | 点赞/取消 |
| 抖音 | `comment` | `url`, `text` | 发表评论 |
| 抖音 | `follow` | `url` | 关注作者 |
| 抖音 | `download` | `url` | 下载视频/图文 |
| 抖音 | `post` | `video`, `title`, `desc` | 发布视频 |
| B站 | `search` | `keyword` | 搜索视频 |
| B站 | `like` | `url` | 点赞视频 |
| B站 | `follow` | `url` | 关注UP主 |
| B站 | `comment` | `url`, `text` | 发表评论 |
| B站 | `reply` | `url`, `text` | 回复评论 |
| B站 | `post` | `video`, `title`, `desc` | 投稿视频 |
| X | `search` | `keyword` | 搜索推文 |
| X | `like` | `url` | 点赞/取消 |
| X | `reply` | `url`, `text` | 回复推文 |
| X | `post` | `text` | 发推 |
| X | `retweet` | `url` | 转推 |
| Boss直聘 | `chat_stats` | — | 聊天统计 |
| Boss直聘 | `candidates` | — | 候选人列表 |
