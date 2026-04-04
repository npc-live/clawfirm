# WhipFlow 示例

## 1. 最简 workflow
```whip
# hello.whip
# 功能：向用户打招呼
let reply = session "用一句话介绍 WhipFlow 是什么"
print reply
```

## 2. 带用户输入的 workflow
```whip
# research.whip
# 功能：对指定主题做研究报告
# 输入：research_topic（用户输入）
# 输出：打印报告

ask research_topic: "请输入要研究的主题"
ask depth: "研究深度（简短/详细）"

let outline = session "为'{research_topic}'创建研究大纲，深度：{depth}"
let report = session "根据以下大纲，撰写完整的研究报告：\n{outline}"
print report
```

## 3. 多 Agent 协作
```whip
# content-pipeline.whip
# 功能：研究 → 写作 → 审核的内容生产流水线

ask topic: "内容主题"

agent researcher:
  tools: ["web_search", "read"]
  prompt: """
  专业研究员。擅长从网络获取最新信息，提炼关键事实和数据。
  输出结构化的研究摘要，包含来源。
  """

agent writer:
  prompt: """
  资深内容写手。将研究材料转化为引人入胜的文章。
  语言流畅自然，观点清晰，有故事性。
  """

agent editor:
  prompt: """
  严格的编辑。检查事实准确性、逻辑连贯性、语言质量。
  给出具体的修改意见。
  """

# 第一步：研究
let research = session: researcher
  prompt: "深入研究'{topic}'，收集最新信息、数据和观点"

# 第二步：写作
let draft = session: writer
  prompt: "基于以下研究，写一篇1500字的文章：\n\n{research}"

# 第三步：审核
let review = session: editor
  prompt: "审核以下文章，给出修改建议：\n\n{draft}"

# 第四步：修订
let final = session: writer
  prompt: "根据编辑意见修改文章：\n\n原稿：{draft}\n\n编辑意见：{review}"

print final
```

## 4. 并行处理
```whip
# parallel-analysis.whip
# 功能：同时从多个维度分析一份报告

ask report_path: "报告文件路径"

agent analyst:
  tools: ["read"]
  prompt: "专业分析师，擅长从不同角度深入分析文档"

# 并行做三个维度的分析（节省时间）
parallel:
  let financial = session: analyst
    prompt: "阅读 {report_path}，分析财务状况和盈利能力"
  let market = session: analyst
    prompt: "阅读 {report_path}，分析市场地位和竞争格局"
  let risk = session: analyst
    prompt: "阅读 {report_path}，识别主要风险因素"
end

# 汇总
let summary = session "整合以下三份分析，生成综合报告：
财务分析：{financial}
市场分析：{market}
风险分析：{risk}"

print summary
```

## 5. 带循环和条件的 workflow
```whip
# qa-loop.whip
# 功能：持续优化直到质量达标

ask requirement: "需求描述"

agent developer:
  tools: ["bash", "read", "write"]
  prompt: "资深开发者，根据需求和反馈迭代改进代码"

agent reviewer:
  tools: ["read", "bash"]
  prompt: "代码审查专家，严格评估代码质量，给出具体改进意见"

let code = session: developer
  prompt: "根据需求实现功能：{requirement}"

loop max: 3:
  let review = session: reviewer
    prompt: "审查以下代码，评估是否满足需求：{requirement}\n\n代码：{code}"

  until [review 表明代码质量合格，无重大问题]

  let code = session: developer
    prompt: "根据审查意见改进代码：\n\n当前代码：{code}\n\n审查意见：{review}"
end

print code
```

## 6. 批量处理（foreach + 文件持久化）
```whip
# batch-translate.whip
# 功能：批量翻译文章列表

agent translator:
  tools: ["read", "write"]
  prompt: "专业翻译，中英双语，保留原文风格和格式"

# 读取待翻译文件列表
let files = session: translator
  prompt: """
  读取 data/to-translate.json（格式：["file1.md", "file2.md", ...]）
  输出文件路径列表
  """

# 逐个翻译
foreach file in files:
  let translated = session: translator
    prompt: "将 {file} 翻译成中文，保存到 data/translated/{file}"
end

print "翻译完成"
```

## 7. 使用 choice 动态路由
```whip
# smart-router.whip
# 功能：根据输入自动选择处理方式

ask input: "请输入要处理的内容（文件路径或文字描述）"

choice [根据 {input} 判断内容类型]:
  "是本地文件路径":
    let content = session "读取文件 {input} 的内容"
    let result = session "分析这份文件：{content}"
  "是 URL":
    let content = session "抓取网页 {input} 的内容"
    let result = session "总结这个网页：{content}"
  "是文字描述":
    let result = session "回答这个问题：{input}"
end

print result
```
