# 后端配置指南

## ⚠️ 重要提醒

**同一个数字人项目必须全程使用同一个后端！**

不同平台的 avatar_id 不互通，选定后端后从创建到口播都用同一个。

---

## Kling 可灵（推荐）

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

- 数字人质量高
- 口型同步自然
- 支持图片生成数字人
- 国产首选

### API 文档

https://docs.qingque.cn/

---

## Jimeng 即梦

### 获取 API Key

1. 访问 https://jimeng.jianying.com/ai-tool/home
2. 登录（抖音/字节账号）
3. 开放平台 → 创建应用 → 获取 API Key

### 配置

```json
{
  "jimeng": {
    "api_key": "ak-xxxxxxxx"
  }
}
```

### 特点

- 速度快
- 中文口型好
- 有免费额度
- 和剪映生态打通

### API 文档

https://www.volcengine.com/docs/jimeng/

---

## HeyGen

### 获取 API Key

1. 访问 https://www.heygen.com/
2. 注册账号
3. Settings → API → Generate API Key

### 配置

```json
{
  "heygen": {
    "api_key": "your_api_key"
  }
}
```

### API 文档

https://docs.heygen.com/reference/

---

## D-ID

### 获取 API Key

1. 访问 https://www.d-id.com/
2. 注册账号
3. Studio → API → Create Key

### 配置

```json
{
  "d-id": {
    "api_key": "your_api_key"
  }
}
```

### API 文档

https://docs.d-id.com/reference/

---

## Synthesia

### 获取 API Key

1. 访问 https://www.synthesia.io/
2. 企业账号申请 API 访问
3. 联系销售获取 API Key

### 配置

```json
{
  "synthesia": {
    "api_key": "your_api_key"
  }
}
```

### API 文档

https://docs.synthesia.io/reference/

---

## 后端选择建议

| 场景 | 推荐后端 |
|------|----------|
| 国内抖音/小红书 | Jimeng |
| 出海/英文内容 | HeyGen |
| 快速测试 | D-ID |
| 企业正式商用 | Synthesia |
| 预算有限 | Jimeng / D-ID |
| 多语言需求 | Synthesia |
