---
project: "抖音爆款分析"
step: "01 - 搜索结果"
agent: social-cli-agent
date: 2026-04-06
---

# 抖音「test」关键词搜索结果

> ⚠️ **注意**：本次搜索未能通过 CDP 实时抓取（沙盒环境限制了网络访问）。
> 以下为执行计划 + 重新执行指令。请在完整环境中运行以获取实时数据。

## 执行方式

```bash
# 1. 确保 Chrome 以远程调试模式运行
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome \
  --remote-debugging-port=9222 &

# 2. 使用 browser_shortcut 执行搜索
browser-shortcut ~/.clawfirm/shortcuts/douyin.yaml search test
```

## 搜索配置

| 参数 | 值 |
|------|-----|
| 关键词 | test |
| 搜索 URL | https://www.douyin.com/search/test?type=video |
| YAML 适配器 | ~/.clawfirm/shortcuts/douyin.yaml |
| 提取方式 | CDP → `a[href*='/video/']` 选择器 |
| 等待时间 | 8000ms（页面加载） |

## douyin.yaml 搜索流程

1. **打开页面**：`https://www.douyin.com/search/test?type=video`
2. **等待加载**：8 秒
3. **抓取链接**：提取所有 `a[href*='/video/']` 元素
4. **提取字段**：
   - `link` → 视频链接（href 属性）
   - `text` → 标题/描述（`.kgnD1hJB` 等选择器）
5. **返回**：页面标题 + 所有抖音链接列表

## 预期输出格式

搜索完成后，数据将以如下格式整理：

| 排名 | 时长 | 点赞 | 标题 | 链接 |
|------|------|------|------|------|
| 1 | -- | -- | [待抓取] | https://www.douyin.com/video/... |
| 2 | -- | -- | [待抓取] | https://www.douyin.com/video/... |
| ... | | | | |

## 数据摘要（待填充）

- **总视频数**：待抓取
- **爆款门槛**：10万+ 点赞
- **内容类型分布**：待分析
- **热门标签**：待提取

## 环境要求

- ✅ douyin.yaml 适配器已就绪：`~/.clawfirm/shortcuts/douyin.yaml`
- ✅ browser-shortcut 二进制已编译：`pi-go/bin/browser-shortcut`
- ❌ Chrome CDP 连接：需要启动 `--remote-debugging-port=9222`
- ❌ 沙盒网络限制：当前 Claude Code 沙盒环境无法访问 localhost:9222

## 重新执行命令

在 Clawfirm 桌面应用或非沙盒终端中执行：

```
browser_shortcut(file="douyin.yaml", command="search", args=["test"])
```

---
*数据来源：抖音搜索「test」| 计划执行时间：2026-04-06 | 状态：⏳ 待重新执行*
