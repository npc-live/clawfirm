# 后端配置指南

## MiniMax

### 获取 API Key

1. 访问 https://www.minimax.chat/
2. 注册/登录
3. 开放平台 → API Key

### 配置

```json
{
  "minimax": {
    "api_key": "your_api_key",
    "group_id": "your_group_id"
  }
}
```

### 特点

- 中文效果极好
- 支持声纹克隆
- 支持情绪控制
- 国内访问快

### API 示例

```bash
# 语音合成
curl -X POST "https://api.minimax.chat/v1/text_to_speech" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "你好，世界",
    "voice_id": "female-tianmei",
    "speed": 1.0
  }'
```

---

## ElevenLabs

### 获取 API Key

1. 访问 https://elevenlabs.io/
2. 注册账号
3. Profile → API Key

### 配置

```json
{
  "elevenlabs": {
    "api_key": "your_api_key"
  }
}
```

### 特点

- 质量顶级
- 30+ 语言
- 情绪控制精细
- 克隆效果好

### API 文档

https://docs.elevenlabs.io/api-reference

### 预设声音

| Voice ID | 名称 | 特点 |
|----------|------|------|
| 21m00Tcm4TlvDq8ikWAM | Rachel | 美式女声 |
| AZnzlk1XvdvUeBnXmlld | Domi | 活泼女声 |
| EXAVITQu4vr4xnSDxMaL | Bella | 温柔女声 |
| ErXwobaYiN019PkySvjV | Antoni | 成熟男声 |
| MF3mGyEYCl7XYWbV9V6O | Elli | 年轻女声 |

---

## Fish Audio

### 获取 API Key

1. 访问 https://fish.audio/
2. 注册账号
3. Settings → API Key

### 配置

```json
{
  "fish_audio": {
    "api_key": "your_api_key"
  }
}
```

### 特点

- 开源友好
- 价格便宜
- 中文/英文/日文
- 克隆质量不错

### API 文档

https://docs.fish.audio/

---

## Azure TTS

### 获取 API Key

1. Azure Portal → 创建语音服务资源
2. 获取 Key 和 Region

### 配置

```json
{
  "azure_tts": {
    "api_key": "your_api_key",
    "region": "eastasia"
  }
}
```

### 特点

- 100+ 语言
- 极其稳定
- 企业级
- 无克隆（用 Custom Voice 需另申请）

### 中文推荐声音

| Voice | 特点 |
|-------|------|
| zh-CN-XiaoxiaoNeural | 年轻女声，自然 |
| zh-CN-YunxiNeural | 年轻男声，活泼 |
| zh-CN-XiaoyiNeural | 成熟女声，专业 |
| zh-CN-YunyangNeural | 新闻播报风 |

---

## OpenAI TTS

### 获取 API Key

1. https://platform.openai.com/
2. API Keys

### 配置

```json
{
  "openai": {
    "api_key": "your_api_key"
  }
}
```

### 特点

- 简单易用
- 质量不错
- 无克隆
- 6 种预设声音

### 预设声音

| Voice | 特点 |
|-------|------|
| alloy | 中性 |
| echo | 男声 |
| fable | 英式 |
| onyx | 深沉男声 |
| nova | 女声 |
| shimmer | 温暖女声 |

### API 示例

```bash
curl https://api.openai.com/v1/audio/speech \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tts-1",
    "input": "Hello world",
    "voice": "nova"
  }' \
  --output speech.mp3
```

---

## 后端选择建议

| 场景 | 推荐 |
|------|------|
| 中文内容 | MiniMax |
| 最高质量 | ElevenLabs |
| 预算有限 | Fish Audio |
| 多语言 | Azure |
| 快速测试 | OpenAI |
| 需要克隆 | MiniMax / ElevenLabs / Fish |

---

## 价格参考（2024）

| 后端 | 免费额度 | 付费价格 |
|------|----------|----------|
| MiniMax | 有 | ~¥0.1/千字 |
| ElevenLabs | 10k 字符/月 | $5/月起 |
| Fish Audio | 有 | 便宜 |
| Azure | 500k 字符/月 | $4/百万字符 |
| OpenAI | 无 | $15/百万字符 |
