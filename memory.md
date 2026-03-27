这个项目的记忆系统是一套基于 Markdown 文件 + SQLite +                                        
  嵌入向量的语义搜索引擎，设计得相当完整。                        
                                                                                               
  整体架构                                                                                     
                                                                                               
  1. 存储层：Markdown 文件作为真相源                                                           
                                                                                               
  记忆的持久化源不是 LLM 状态，而是纯 Markdown 文件：                                          
  - memory/YYYY-MM-DD.md — 按日期的追加式日志                                                  
  - MEMORY.md — 精选的长期记忆

  2. 索引层：SQLite + 向量搜索 (src/memory/)

  核心类是 src/memory/manager.ts 的 MemoryIndexManager，SQLite
  中有以下表（src/memory/memory-schema.ts）：

  ┌─────────────────┬────────────────────────────────────────────────────┐
  │       表        │                        作用                        │
  ├─────────────────┼────────────────────────────────────────────────────┤
  │ files           │ 跟踪已索引的 Markdown 文件（路径、hash、修改时间） │
  ├─────────────────┼────────────────────────────────────────────────────┤
  │ chunks          │ 分块后的文本 + embedding 向量                      │
  ├─────────────────┼────────────────────────────────────────────────────┤
  │ chunks_fts      │ FTS5 虚拟表，做关键词 BM25 搜索                    │
  ├─────────────────┼────────────────────────────────────────────────────┤
  │ chunks_vec      │ sqlite-vec 向量表，做余弦相似度搜索                │
  ├─────────────────┼────────────────────────────────────────────────────┤
  │ embedding_cache │ 缓存 embedding 结果，减少 API 调用                 │
  └─────────────────┴────────────────────────────────────────────────────┘

  分块策略：默认 400 token 一块，80 token 重叠。

  3. Embedding 提供者 (src/memory/embeddings.ts)

  支持 6 种 embedding 后端，自动回退：
  1. OpenAI — text-embedding-3-small（默认远程）
  2. Google Gemini — gemini-embedding-001
  3. Voyage AI — voyage-4-large
  4. Mistral — mistral-embed
  5. Ollama — nomic-embed-text（本地自托管）
  6. 本地 GGUF — node-llama-cpp + embeddinggemma-300m

  4. 搜索算法 (src/memory/hybrid.ts)

  默认使用混合搜索：

  最终得分 = 0.7 × 向量余弦相似度 + 0.3 × BM25 关键词得分

  可选增强：
  - MMR（最大边际相关性）— 多样性重排序
  - 时间衰减 — 近期记忆权重更高（半衰期 30 天）

  5. Agent 工具集成 (src/agents/tools/memory-tool.ts)

  注册了两个 agent 工具：
  - memory_search — 语义搜索记忆，返回带路径、行号、分数的片段
  - memory_get — 根据搜索结果精确读取文件特定行

  系统提示词要求 agent 回答关于过往工作/决策/偏好的问题前，必须先调用 memory_search。

  6. 自动记忆写入

  会话压缩前的自动 flush（src/auto-reply/reply/memory-flush.ts）：
  - 当 context token 接近窗口上限时触发
  - 让 agent 静默执行一轮，把重要信息写入 memory/YYYY-MM-DD.md
  - 用户看不到这个 flush 过程

  会话重置时的捕获（src/hooks/bundled/session-memory/handler.ts）：
  - /new 或 /reset 时触发
  - 提取当前会话的关键上下文保存到记忆文件

  7. 同步触发机制

  ┌─────────────────────┬─────────────────────────────┐
  │      触发方式         │            说明             │
  ├─────────────────────┼─────────────────────────────┤
  │ 文件监听 (chokidar)  │ memory/ 目录变更，1.5s 防抖 │
  ├─────────────────────┼─────────────────────────────┤
  │ 会话开始             │ warmSession() 异步预同步    │
  ├─────────────────────┼─────────────────────────────┤
  │ 定时                 │ 每 60 分钟周期同步          │
  ├─────────────────────┼─────────────────────────────┤
  │ 搜索时               │ 搜索前自动增量同步          │
  └─────────────────────┴─────────────────────────────┘

  8. 插件架构

  记忆是一个插件槽位（extensions/memory-core/），可替换：
  - memory-core — 默认的 SQLite + embedding 实现
  - memory-lancedb — LanceDB 向量存储替代方案
  - none — 禁用记忆

  ---
  总结：数据流是 Markdown 文件 → 分块 → embedding → SQLite 存储 → 混合搜索（BM25 + 向量）→
  agent 工具调用。整个系统强调 Markdown
  文件作为持久化真相源，配合语义向量搜索实现高质量的长期记忆召回。