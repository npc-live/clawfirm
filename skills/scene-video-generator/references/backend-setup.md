# 后端配置指南

## 即梦 Jimeng

### 获取 API Key

1. 访问 https://jimeng.jianying.com/ai-tool/home
2. 登录（抖音/字节账号）
3. 开放平台 → 创建应用 → 获取 API Key

### 配置

```json
{
  "jimeng": {
    "api_key": "ak-xxxxxxxxxxxxxxxx"
  }
}
```

### 特点

- 国内访问快
- 中文理解好
- 支持文生视频、图生视频
- 有免费额度

### API 文档

https://www.volcengine.com/docs/jimeng/

---

## 可灵 Kling

### 获取 API Key

1. 访问 https://klingai.kuaishou.com/
2. 登录（快手账号）
3. 开放平台 → 创建应用
4. 获取 **Access Key** 和 **Secret Key**

### 配置

```json
{
  "kling": {
    "access_key": "your_access_key",
    "secret_key": "your_secret_key"
  }
}
```

### 特点

- 质量高，运动自然
- 支持 5s/10s 视频
- 支持图生视频
- 国内首选

### API 文档

https://docs.qingque.cn/d/home/eZQClV8BFVPVr2FVHI_0p0FUu

---

## Runway

### 获取 API Key

1. 访问 https://runwayml.com/
2. 注册账号
3. Settings → API Keys

### 配置

```json
{
  "runway": {
    "api_key": "your_api_key"
  }
}
```

### 特点

- Gen-3 Alpha 质量顶级
- 国际标杆
- 价格较贵
- 需要科学上网

### API 文档

https://docs.runwayml.com/

---

## Pika

### 获取 API Key

1. 访问 https://pika.art/
2. 注册账号
3. API Access 申请

### 配置

```json
{
  "pika": {
    "api_key": "your_api_key"
  }
}
```

### 特点

- 风格化强
- 速度快
- 3-5s 短视频
- 适合创意内容

---

## Vidu

### 获取 API Key

1. 访问 https://www.vidu.com/
2. 注册账号
3. 开放平台获取 API

### 配置

```json
{
  "vidu": {
    "api_key": "your_api_key"
  }
}
```

### 特点

- 国产，性价比高
- 支持多种风格
- 4-8s 视频

---

## 后端选择建议

| 场景 | 推荐 |
|------|------|
| 国内平台/快速测试 | Jimeng |
| 高质量需求 | Kling |
| 国际项目/顶级质量 | Runway |
| 风格化/创意 | Pika |
| 预算有限 | Vidu |

---

## 完整配置示例

```json
{
  "jimeng": {
    "api_key": "ak-xxxxxxxxxxxxxxxx"
  },
  "kling": {
    "access_key": "your_access_key",
    "secret_key": "your_secret_key"
  },
  "runway": {
    "api_key": "your_api_key"
  },
  "pika": {
    "api_key": "your_api_key"
  },
  "vidu": {
    "api_key": "your_api_key"
  }
}
```
