---
name: kling
version: 1.1.0
description: "Kling AI API expert for video/image generation via Kuaishou's Kling API. Use this skill whenever the user mentions Kling AI, wants to generate videos or images using Kling API, asks about text-to-video, image-to-video, video extension, lip sync, custom voice cloning, virtual try-on, video effects, motion brush, camera control with Kling, needs help with JWT authentication for Kling, or wants to integrate Kling into an application. Also trigger when the user asks about 'kling-v1', 'kling-v2', 'kling-v2-6', 'kling-v3', klingai.com API, voice cloning, or custom voices."
---

# Kling AI API Skill

Kling AI 是快手（Kuaishou）开发的视频/图像生成 API，支持文生视频、图生视频、图像生成、唇形同步、虚拟换装等。

## 快速开始

### 1. 获取 API 密钥

1. 注册：https://app.klingai.com/global/dev
2. 进入「开发者控制台」→「API 密钥管理」
3. 创建密钥对：**AccessKey (AK)** 和 **SecretKey (SK)**
   - SK **只显示一次**，立即保存
4. 充值余额或订阅套餐（API 调用需要 Credits）

### 2. 生成 JWT Token（必须）

Kling API **不直接用 AK/SK**，必须每次请求前生成 JWT。

```python
import time, jwt

def get_kling_token(ak: str, sk: str) -> str:
    """生成有效期 30 分钟的 JWT Token"""
    payload = {
        "iss": ak,
        "exp": int(time.time()) + 1800,  # 30分钟后过期
        "nbf": int(time.time()) - 5,     # 提前5秒生效（防时钟偏差）
    }
    return jwt.encode(payload, sk, algorithm="HS256", headers={"alg": "HS256", "typ": "JWT"})
```

```js
// Node.js
const jwt = require("jsonwebtoken");
const token = jwt.sign(
  { iss: AK, exp: Math.floor(Date.now()/1000) + 1800, nbf: Math.floor(Date.now()/1000) - 5 },
  SK, { algorithm: "HS256" }
);
```

```bash
# 环境变量（推荐）
export KLING_AK=your_access_key
export KLING_SK=your_secret_key
```

> ⚠️ 常见错误：`401 code:1000` = 直接用了 AK/SK 而非 JWT。必须服务端生成，禁止在浏览器端生成。

### 3. 请求头格式

```
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

---

## API 基础信息

- **Base URL:** `https://api.klingai.com`
- **模式：** 全异步任务制 — POST 提交任务返回 `task_id`，GET 轮询状态直到 `succeed`

---

## 核心 API 端点

### 文生视频 (Text-to-Video)

```
POST /v1/videos/text2video
GET  /v1/videos/text2video/{task_id}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model_name` | string | 否 | 见「模型列表」，默认 `kling-v1-6` |
| `prompt` | string | 是 | 最多 2500 字符 |
| `negative_prompt` | string | 否 | 最多 2500 字符 |
| `cfg_scale` | float | 否 | 0.0–1.0，默认 0.5（v2-0 不支持） |
| `duration` | string | 否 | `"5"`（默认）或 `"10"`（v2-0 不支持） |
| `aspect_ratio` | string | 否 | `"16:9"`（默认）, `"9:16"`, `"1:1"`, `"4:3"`, `"3:4"`, `"3:2"`, `"2:3"`, `"21:9"` |
| `mode` | string | 否 | `"std"`（默认）或 `"pro"`（v2-0 不支持） |
| `camera_control` | object | 否 | 仅 V1 模型，见「相机控制」 |
| `callback_url` | string | 否 | 任务完成后的 Webhook 回调 URL |

**示例：**
```json
{
  "model_name": "kling-v1-6",
  "prompt": "一只金色的猫咪在草地上玩耍，阳光明媚",
  "negative_prompt": "模糊, 低质量, 变形",
  "cfg_scale": 0.5,
  "duration": "5",
  "aspect_ratio": "16:9",
  "mode": "std"
}
```

---

### 图生视频 (Image-to-Video)

```
POST /v1/videos/image2video
GET  /v1/videos/image2video/{task_id}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model_name` | string | 否 | 同上 |
| `image` | string | 是 | 图片 URL 或 Base64。最小 300px/边，最大 10MB |
| `image_tail` | string | 否 | 末帧图片（不能与 motion brush 同时使用） |
| `prompt` | string | 否 | 最多 2500 字符 |
| `negative_prompt` | string | 否 | 最多 2500 字符 |
| `cfg_scale` | float | 否 | 0.0–1.0 |
| `mode` | string | 否 | `"std"` 或 `"pro"` |
| `duration` | string | 否 | `"5"` 或 `"10"` |
| `static_mask` | string | 否 | 静止区域遮罩图 URL/Base64 |
| `dynamic_masks` | array | 否 | 运动笔刷路径，见「Motion Brush」 |
| `callback_url` | string | 否 | Webhook 回调 |

---

### 图像生成 (Text-to-Image)

```
POST /v1/images/generations
GET  /v1/images/generations?pageSize=500
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model_name` | string | 否 | `kling-v1`（默认）, `kling-v1-5` 等 |
| `prompt` | string | 是 | 最多 2500 字符 |
| `negative_prompt` | string | 否 | 最多 2500 字符 |
| `image_fidelity` | float | 否 | 图像保真度 0.0–1.0，默认 0.5 |
| `n` | int | 否 | 生成数量，默认 1 |
| `aspect_ratio` | string | 否 | 同视频比例选项 |
| `callback_url` | string | 否 | Webhook 回调 |

**轮询结果示例：**
```python
# 轮询直到 task_status == "succeed"
while True:
    resp = GET("/v1/images/generations", params={"pageSize": 500})
    task = next(t for t in resp["data"]["list"] if t["task_id"] == task_id)
    if task["task_status"] == "succeed":
        images = task["task_result"]["images"]  # [{"index": 0, "url": "..."}]
        break
    time.sleep(3)
```

---

### 视频延长 (Video Extension)

```
POST /v1/videos/video-extend
GET  /v1/videos/video-extend/{task_id}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `video_id` | string | 二选一 | Kling 生成的视频 ID（30天内） |
| `video_url` | string | 二选一 | 视频 URL |
| `prompt` | string | 否 | 延长方向描述，最多 2500 字符 |
| `cfg_scale` | float | 否 | 0.0–1.0 |
| `callback_url` | string | 否 | Webhook 回调 |

> 限制：仅支持 5s 和 10s 视频，且为 30 天内生成的 Kling 视频。

---

### 唇形同步 (Lip Sync)

```
POST /v1/videos/lip-sync
GET  /v1/videos/lip-sync/{task_id}
```

**文字驱动模式：**
```json
{
  "video_id": "xxx",
  "mode": "text2video",
  "text": "你好，这是一段测试文本。",
  "voice_id": "chengshu_jiejie",
  "voice_language": "zh",
  "speech_rate": 1.0
}
```

**音频驱动模式：**
```json
{
  "video_url": "https://example.com/video.mp4",
  "mode": "audio2video",
  "audio_type": "url",
  "audio_url": "https://example.com/speech.mp3"
}
```

> 完整参数见「自定义音色」章节中的参数表。

---

### 视频特效 (Video Effects)

```
POST /v1/videos/effects
GET  /v1/videos/effects/{task_id}
```

```json
{
  "model_name": "kling-v1-6",
  "effect_scene": "heart_gesture",
  "image": ["https://example.com/person1.jpg", "https://example.com/person2.jpg"],
  "duration": "5"
}
```

**`effect_scene` 可选值：**
- `"hug"` — 拥抱（需要 2 张含人脸图片）
- `"kiss"` — 亲吻（需要 2 张含人脸图片）
- `"heart_gesture"` — 爱心手势（需要 2 张含人脸图片）
- 单人特效（挤压/膨胀等，自动生成音效）

---

### 虚拟换装 (Virtual Try-On)

```
POST /v1/images/kolors-virtual-try-on
GET  /v1/images/kolors-virtual-try-on/{task_id}
```

```json
{
  "model_name": "kolors-virtual-try-on-v1-5",
  "human_image": "https://example.com/person.jpg",
  "cloth_image": "https://example.com/garment.jpg"
}
```

| 参数 | 类型 | 说明 |
|------|------|------|
| `model_name` | string | `"kolors-virtual-try-on-v1"` 或 `"kolors-virtual-try-on-v1-5"` |
| `human_image` | string | 人物图片 URL 或 Base64 |
| `cloth_image` | string | 服装图片 URL 或 Base64 |

---

## 模型列表

| 系列 | 模型名 | 特点 | 推荐场景 |
|------|--------|------|----------|
| V1 | `kling-v1` | 基础版，支持相机控制、motion brush | 低成本测试 |
| V1 | `kling-v1-5` | 增强版，支持相机控制、motion brush | 性价比首选 |
| V1 | `kling-v1-6` | **V1 推荐**，支持相机控制、motion brush | V1 默认推荐 |
| V2 | `kling-v2-0` | 不支持 cfg_scale/duration/mode/camera | 快速出图 |
| V2 | `kling-v2-1` | 均衡版 | 日常生产 |
| V2 | `kling-v2-master` | **V2 旗舰**，最高质量 | 商业交付 |
| V2 | `kling-v2-5-turbo` | 速度优先，低延迟 | 实时预览 |
| V2 | `kling-v2-6` | **原生音频**，支持自定义音色 | 带音频的视频 |
| V2 | `kling-v2-6-pro` | V2.6 Pro，更高质量原生音频 | 高质量音视频 |
| V3 | `kling-v3` | 最长 15s，AI Director，4K 图像 | 长视频/高分辨率 |

**模型能力对照：**
- 相机控制（Camera Control）：仅 V1 系列
- Motion Brush：仅 V1 系列
- 原生音频（Native Audio）：`kling-v2-6` 及以上
- 自定义音色（Custom Voice）：`kling-v2-6` 及以上
- 最长时长 15s：`kling-v3`
- `cfg_scale` / `mode=pro`：V1 和 V2.1+（不含 v2-0）

---

## 自定义音色 (Custom Voices) — V2.6+

> 需要 `kling-v2-6` 或更高版本。从音频/视频中克隆声音，生成可复用的 `voice_id`。

### 创建自定义音色

```
POST /v1/voices
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `voice_url` | string | 是 | 音频/视频 URL 或 Base64。支持 `.mp3`、`.wav`、`.mp4`、`.mov`。**时长 5–30 秒**，必须是干净的单人声音，无背景音乐/噪音 |

**响应：**
```json
{
  "code": 0,
  "data": {
    "voice_id": "custom_abc123xyz"
  }
}
```

**工作流程：**
1. 准备 5–30 秒干净录音（单人，无背景噪音）
2. POST 到 `/v1/voices`，获得 `voice_id`（持久保存，无需重复上传）
3. 在 lip-sync 或 v2.6 视频生成时通过 `voice_id` 引用

### 在唇形同步中使用自定义音色

```json
{
  "video_url": "https://example.com/video.mp4",
  "mode": "text2video",
  "text": "你好，欢迎使用 Kling AI。",
  "voice_id": "custom_abc123xyz",
  "voice_language": "zh",
  "speech_rate": 1.0,
  "voice_volume": 1.0
}
```

### 在 V2.6 视频生成中使用（多人对话）

在 prompt 中用 `<<<voice_1>>>` 和 `<<<voice_2>>>` 标签引用，最多同时使用 2 个音色：

```json
{
  "model_name": "kling-v2-6",
  "prompt": "<<<voice_1>>>你好，我是小明。<<<voice_2>>>你好，我是小红，很高兴认识你。",
  "voice_list": ["custom_abc123xyz", "commercial_lady_en_f-v1"]
}
```

### 内置 voice_id 列表

**中文语音：**
```
genshin_vindi2       # 默认
zhinen_xuesheng      # 知性学生
chengshu_jiejie      # 成熟姐姐
you_pingjing         # 悠平静
calm_story1          # 平静故事
tiexin_nanyou        # 贴心男友
laopopo_speech02     # 老婆婆
heainainai_speech02  # 和蔼奶奶
dongbeilaotie_speech02   # 东北老铁
chongqingxiaohuo_speech02 # 重庆小伙
chuanmeizi_speech02  # 川妹子
tianjinjiejie_speech02 # 天津姐姐
ai_taiwan_man2_speech02  # 台湾男声
```

**英文语音：**
```
reader_en_m-v1           # 男性朗读
commercial_lady_en_f-v1  # 商务女声（默认英文）
oversea_male1            # 海外男声
uk_boy1                  # 英国男孩
uk_man2                  # 英国男性
uk_oldman3               # 英国老人
ai_shatang               # 沙糖（双语）
ai_kaiya                 # 凯亚
```

**唇形同步完整参数（text2video 模式）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `video_id` | string | 二选一 | Kling 生成的视频 ID（30天内） |
| `video_url` | string | 二选一 | `.mp4`/`.mov`，≤100MB，2–10s，720p 或 1080p |
| `mode` | string | 是 | `"text2video"` 或 `"audio2video"` |
| `text` | string | text 模式 | 最多 120 字符 |
| `voice_id` | string | text 模式 | 内置或自定义 voice_id |
| `voice_language` | string | text 模式 | `"zh"` 或 `"en"` |
| `speech_rate` | float | 否 | 语速 0.8–2.0，默认 1.0 |
| `voice_volume` | float | 否 | 音量 0.1–2.0，默认 1.0 |
| `audio_type` | string | audio 模式 | `"file"` 或 `"url"` |
| `audio_url` | string | audio/url | 最小 2s，最大 60s，最大 5MB |
| `face_id` | string | 否 | 指定视频中人脸 ID |
| `start_ms` | int | 否 | 唇形同步开始时间（毫秒） |
| `end_ms` | int | 否 | 唇形同步结束时间（毫秒） |
| `callback_url` | string | 否 | Webhook 回调 |

---

## 相机控制 (Camera Control)

仅 V1 模型支持，在 text2video / image2video 请求中传入：

```json
{
  "camera_control": {
    "type": "simple",
    "config": {
      "horizontal": 0,   // 左右移动，范围 -10 到 10
      "vertical": 0,     // 上下移动，范围 -10 到 10
      "pan": 0,          // 左右旋转，范围 -10 到 10
      "tilt": 0,         // 上下俯仰，范围 -10 到 10
      "roll": 0,         // 顺/逆时针滚动，范围 -10 到 10
      "zoom": 5          // 缩放，范围 -10 到 10（正数=推近，负数=拉远）
    }
  }
}
```

---

## Motion Brush（运动笔刷）

在 image2video 请求中使用，仅 V1 模型支持。

**遮罩颜色对照（mask PNG 中必须精确匹配）：**

| 索引 | RGB 颜色 | 含义 |
|------|----------|------|
| 0 | (0, 0, 0) 黑色 | 静止区域 |
| 1 | (114, 229, 40) 绿色 | 运动路径 1 |
| 2 | (171, 105, 255) 紫色 | 运动路径 2 |
| 3 | (0, 170, 255) 青色 | 运动路径 3 |
| 4 | (240, 38, 173) 粉色 | 运动路径 4 |
| 5 | (255, 225, 29) 黄色 | 运动路径 5 |
| 6 | (255, 34, 0) 红色 | 运动路径 6 |

```json
{
  "static_mask": "https://example.com/static-mask.png",
  "dynamic_masks": [
    {
      "mask": "https://example.com/dynamic-mask.png",
      "trajectories": [
        {"x": 100, "y": 200},
        {"x": 150, "y": 250},
        {"x": 200, "y": 300}
      ]
    }
  ]
}
```

**规则：**
- 每张图最多 6 条动态遮罩
- 每条轨迹至少 2 个点，建议 10 个以上
- 坐标原点 (0,0) 在左上角，X 向右，Y 向下
- 遮罩图分辨率必须与输入图片一致
- `image_tail`（末帧）与 motion brush 不能同时使用

---

## 标准响应格式

### 任务提交响应（所有 POST）

```json
{
  "code": 0,
  "message": "string",
  "request_id": "string",
  "data": {
    "task_id": "string",
    "task_status": "submitted",
    "created_at": 1722769557708,
    "updated_at": 1722769557708
  }
}
```

### 任务状态

| 状态值 | 含义 |
|--------|------|
| `submitted` | 已提交排队 |
| `processing` | 生成中 |
| `succeed` | 成功完成 |
| `failed` | 生成失败 |

### 完成后视频响应

```json
{
  "code": 0,
  "message": "SUCCEED",
  "data": {
    "task_id": "CjMkWmdJhuIAAAAAAKKcRg",
    "task_status": "succeed",
    "task_result": {
      "videos": [
        {
          "id": "2e0bd237-31ac-464d-98b3-ab0535ea8fee",
          "url": "https://cdn.klingai.com/.../output.mp4",
          "duration": "5.1"
        }
      ]
    }
  }
}
```

---

## 错误码

| Code | HTTP | 含义 |
|------|------|------|
| 0 | 200 | 成功 |
| 1000 | 401 | 认证失败（JWT 错误或直接用了 AK/SK） |
| 404 | 404 | 资源不存在 |
| 500 | 500 | 内容安全审核不通过（prompt 或图片被拒绝） |
| 5xx | 502/503/504 | 服务端临时错误，建议指数退避重试 |

---

## 完整集成示例

```python
import time, jwt, requests, os

AK = os.environ["KLING_AK"]
SK = os.environ["KLING_SK"]
BASE = "https://api.klingai.com"

def get_token():
    return jwt.encode(
        {"iss": AK, "exp": int(time.time()) + 1800, "nbf": int(time.time()) - 5},
        SK, algorithm="HS256", headers={"alg": "HS256", "typ": "JWT"}
    )

def headers():
    return {"Authorization": f"Bearer {get_token()}", "Content-Type": "application/json"}

def text_to_video(prompt: str, model: str = "kling-v1-6") -> str:
    """提交文生视频任务，返回 task_id"""
    resp = requests.post(f"{BASE}/v1/videos/text2video", headers=headers(), json={
        "model_name": model,
        "prompt": prompt,
        "duration": "5",
        "aspect_ratio": "16:9"
    })
    data = resp.json()
    assert data["code"] == 0, f"Error: {data}"
    return data["data"]["task_id"]

def wait_for_video(task_id: str, timeout: int = 600) -> str:
    """轮询任务状态，返回视频 URL"""
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = requests.get(f"{BASE}/v1/videos/text2video/{task_id}", headers=headers())
        result = resp.json()["data"]
        status = result["task_status"]
        if status == "succeed":
            return result["task_result"]["videos"][0]["url"]
        elif status == "failed":
            raise RuntimeError(f"Task failed: {result}")
        time.sleep(5)
    raise TimeoutError(f"Task {task_id} timed out after {timeout}s")

# 使用示例
task_id = text_to_video("一只可爱的柴犬在雪地里奔跑，4K 写实风格")
video_url = wait_for_video(task_id)
print(f"视频已生成: {video_url}")
```

---

## 定价参考

| 套餐 | 月付 | Credits/月 |
|------|------|-----------|
| Free | $0 | 66–166/天 |
| Standard | $6.99 | 660 |
| Pro | $25.99 | 3,000 |
| Premier | $64.99 | 8,000 |

**Credits 消耗：**
- Standard 模式 5s 视频：10 credits
- Pro 模式 5s 视频：35 credits
- Pro 模式 10s 视频：~70 credits

**API 直接访问（企业版）：** 起步约 $4,200/月（需 3 个月订阅）

---

## 官方资源

| 资源 | 链接 |
|------|------|
| 开发者控制台 | https://app.klingai.com/global/dev |
| API 文档 | https://app.klingai.com/global/dev/document-api |
| 定价 | https://klingai.com/global/dev/pricing |
