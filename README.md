# 骊珠 (Lizhu)

<p align="center">
  <img src="assets/banner.png" alt="骊珠 Banner" width="480"/>
</p>

<p align="center">
  <a href="#快速开始"><img src="https://img.shields.io/badge/快速开始-→-brightgreen?style=flat-square" alt="快速开始"/></a>
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version"/>
  <img src="https://img.shields.io/badge/LLM-OpenAI%20Compatible-412991?style=flat-square&logo=openai" alt="LLM"/>
  <img src="https://img.shields.io/badge/RAG-Milvus-00B2FF?style=flat-square" alt="Milvus"/>
  <img src="https://img.shields.io/badge/DB-PostgreSQL-4169E1?style=flat-square&logo=postgresql" alt="PostgreSQL"/>
  <img src="https://img.shields.io/badge/version-v2.1.0-blue?style=flat-square" alt="Version"/>
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"/>
</p>

> 天道崩塌，我陈平安，唯有一剑，可搬山，断江，倒海，降妖，镇魔，敕神，摘星，摧城，开天！
>
<p align="right">——《剑来》</p>

---

## 目录

- [骊珠是什么](#骊珠是什么)
- [功能特性](#功能特性)
- [系统架构](#系统架构)
- [快速开始](#快速开始)
- [CLI 命令](#cli-命令)
- [完整配置说明](#完整配置说明)
- [世界观配置](#世界观配置)
- [护道人人格配置](#护道人人格配置)
- [开发指南](#开发指南)
- [交付路线](#交付路线)

---

## 骊珠是什么

骊珠，取自《剑来》骊珠洞天。这个项目借了那片洞天的名字，给 Go 和 AI 开发者配了一位护道人。

> 练气士追求长生不老，所以最是惜命。他们破境，可能只是看了一场大雨，或是喝了一杯醇酒，甚至只是跟人吵了一架，心结一开，便能跻身下一个境界。但是我们武夫不行。我们破境，那是拿命去换，是在死人堆里爬出来的，是一拳一脚打出来的。练气士的境界，多是求老天爷赏饭吃，顺天而行；我们武夫的境界，是跟老天爷抢饭吃，是逆流而上。
>
> ——宋长镜

这两条路放到开发者身上同样成立。**练气士**对应应用层技术——Go 生态与 AI 生态，靠积累与顿悟破境；**武夫**对应计算机底层硬功——OS、网络、算法，没有捷径，是一道题一道题啃出来的。

护道人三大核心功能：

- **客观评级**：主动从对话中判断你的当前水平，不需要你自我介绍，也不会只说"挺好的"
- **持久记忆**：笔记、代码、历史对话都留存，下次打开还认得你，不用重新说一遍
- **破境指引**：告诉你距离下一境还差什么，给出具体的修炼方向

护道人支持自定义角色人格，内置《剑来》原著人物齐静春。

---

## 修行档案示例

<p align="center">
  <img src="assets/preview_status.png" alt="骊珠修行档案示例" width="780"/>
</p>

---

## 功能特性

### 核心能力（一期）

| 特性 | 说明 |
|------|------|
| **双轨练气士评估** | Go 开发练气士（十五境）与 AI 应用练气士（十五境）独立评分，境界名称严格遵循原著 |
| **武夫底层内功** | 计算机底层硬功独立追踪，十一境从泥胚到武神 |
| **法宝体系** | 多类生态工具掌握程度客观量化（0–100，初识 / 熟用 / 精通 / 宗师四级制）；类别遵循《剑来》世界观命名（本命飞剑、绘卷、符箓、方寸物、护山大阵、灵宠、观星镜、法家戒尺）|
| **按需评估** | 护道人智能识别对话意图：闲聊/技术问答自然对话，汇报成果/`/assess` 时输出完整境界报告 |
| **长期记忆** | 修行档案、整次会话摘要、结构化能力证据条目均持久化至 PostgreSQL，重启后无需重复自我介绍 |
| **世界观热更新** | `configs/worldview/*.yaml` 随时补充设定，无需改代码重新编译 |

### 二期新增能力

| 特性 | 说明 |
|------|------|
| **流式输出** | LLM token 逐字流式打印，评估模式下 `<eval_json>` 块在后台静默解析，不污染对话区 |
| **RAG 知识库** | 笔记/代码文件向量化写入 Milvus，对话时自动检索 top-3 相关片段注入系统提示 |
| **知识整理官 Agent** | `lizhu note add` 调用独立 Librarian Agent，提炼结构化摘要（要点列表 + 关键词），随文件持久化 |
| **lipgloss 彩色档案** | `lizhu status` 输出彩色境界进度条与分区标题高亮 |
| **能力证据体系** | 对话结束时 Librarian 自动提炼 3~5 条结构化证据（工具/类别/置信度 1-5），存入 `ability_evidence` 表；`/assess` 时自动注入系统提示，实现跨对话长期记忆 |
| **退出进度展示** | `/quit` 或 Ctrl+C 后逐行展示封存步骤（等待评估同步→生成摘要→封存证据），操作完毕显示提示后关闭 |

---

## 系统架构

### 整体架构图

<p align="center">
  <img src="assets/architecture.png" alt="骊珠系统架构图" width="860"/>
</p>

### 对话数据流

```mermaid
sequenceDiagram
    participant U as 修行者
    participant CLI as lizhu chat
    participant GA as Guardian Agent
    participant WL as Worldview Loader
    participant EM as Episodic Memory
    participant KR as Knowledge Retriever
    participant LLM as LLM Provider
    participant DB as PostgreSQL

    U->>CLI: 输入对话内容
    CLI->>GA: ChatStream(input, history)
    GA->>WL: BuildSystemPrompt(persona, assess=false)
    WL-->>GA: 系统提示词（世界观 + 人格语料）
    GA->>EM: GetOrCreateProfile + GetRecentSessions
    EM-->>GA: 修行档案 + 近期会话摘要
    GA->>KR: Search(input, top=3)
    KR-->>GA: 相关知识块（Milvus 检索，超时保护）
    GA->>LLM: 流式请求（system + history + input）
    LLM-->>CLI: token stream（逐字打印）
    CLI-->>U: 护道人回复完整呈现
    Note over CLI,U: /quit 或 Ctrl+C 触发退出序列
    CLI->>GA: WaitPersist() + PersistFullSession(history)
    GA->>DB: SaveSession（整次会话摘要）+ SaveEvidenceItems（能力证据）
```

### 评估模式数据流（`/assess`）

```mermaid
sequenceDiagram
    participant U as 修行者
    participant CLI as lizhu chat
    participant GA as Guardian Agent
    participant WL as Worldview Loader
    participant LLM as LLM Provider
    participant DB as PostgreSQL

    U->>CLI: 输入 /assess
    CLI->>GA: ChatStream(input, assess=true)
    GA->>DB: GetRecentEvidence（最近 20 条能力证据）
    DB-->>GA: 历史能力证据条目
    GA->>WL: BuildSystemPrompt(persona, assess=true)
    WL-->>GA: 完整系统提示词（含 eval_json 格式规范 + 历史证据块）
    GA->>LLM: 流式请求
    LLM-->>CLI: 可见文字 token（逐字打印）
    LLM-->>GA: <eval_json>...</eval_json>（后台静默接收）
    Note over CLI: 检测到 <eval_json> 立即截断打印
    GA->>GA: 后台 goroutine 解析 eval_json
    GA->>DB: persistEvaluation（更新档案、法器谱）
    DB-->>GA: OK
```

### 知识入库数据流

```mermaid
flowchart LR
    F[📄 笔记/代码文件] --> LA[Librarian Agent\nLLM 提炼摘要]
    LA --> KI[Knowledge Ingester\n文本分块]
    KI --> EM2[Embedding Model\n向量化]
    EM2 --> MV[(Milvus\n向量存储)]
    LA --> PG[(PostgreSQL\nknowledge_files)]
    MV -.检索.-> GA[Guardian Agent\n对话注入]
    PG -.摘要注入.-> GA
```

### 目录结构

```
lizhu/
├── cmd/lizhu/cmd/
│   ├── root.go          # 根命令、全局依赖初始化（DB、Repo、Config）
│   ├── chat.go          # chat 命令：liner 交互、双模式流式输出
│   ├── note.go          # note add / note list 命令
│   └── status.go        # status 命令（lipgloss 彩色渲染）
│
├── internal/
│   ├── agent/
│   │   ├── guardian/    # 护道人 Agent
│   │   │   ├── agent.go     # 核心逻辑：上下文构建、RAG 注入、流式输出
│   │   │   ├── context.go   # 系统提示组装
│   │   │   └── persist.go   # 评估结果与会话摘要持久化
│   │   └── librarian/   # 知识整理官 Agent
│   │       ├── agent.go     # Summarize：LLM 提炼结构化摘要
│   │       └── prompt.go    # 摘要提示词模板
│   │
│   ├── knowledge/       # RAG 知识库层
│   │   ├── ingester.go      # 文件入库：分块 → Embedding → Milvus
│   │   ├── retriever.go     # 语义检索：向量相似度 top-k
│   │   ├── embedding.go     # Embedding HTTP 客户端
│   │   └── milvus.go        # Collection 初始化与连通性探测
│   │
│   ├── memory/episodic/ # 情节记忆（PostgreSQL）
│   │   └── repo.go          # 档案 CRUD、会话摘要、法器谱
│   │
│   ├── worldview/       # 世界观 YAML 加载器
│   │   ├── loader.go        # 多文件加载、路径过滤、评估/对话双模式 Prompt
│   │   └── schema.go        # Section 结构体定义
│   │
│   ├── checkpoint/      # Eino CheckPointStore（PostgreSQL 后端）
│   └── storage/         # DB 连接 + golang-migrate 自动迁移
│
├── configs/worldview/   # 世界观 YAML 配置（可热更新）
├── notes/               # 用户笔记目录（lizhu note add 入库源）
└── lizhu.yaml           # 用户配置文件（不入 Git）
```

---

## 快速开始

### 前置要求

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.21+ | 编译运行 |
| Docker & Compose | 任意新版 | 启动 PostgreSQL（必须）和 Milvus（可选）|
| LLM API Key | — | OpenAI 或任意兼容接口（DashScope、deepseek 等）|

### 1. 启动基础设施

```bash
docker-compose up -d
docker-compose ps   # 确认 postgres healthy，Milvus 约需 30-60s 冷启动
```

| 服务 | 镜像 | 端口 | 说明 |
|------|------|------|------|
| `lizhu-postgres` | postgres:16-alpine | 5432 | 档案、会话、知识文件元数据 |
| `lizhu-milvus` | milvusdb/milvus:v2.4.15 | 19530 / 9091 | 向量知识库（RAG，可选）|

> **仅使用对话功能**：只需 postgres 正常，Milvus 保持 `enabled: false` 即可。

### 2. 创建配置文件

```bash
cp configs/lizhu.yaml.example lizhu.yaml
```

填写最少必要字段：

```yaml
llm:
  api_key: "sk-your-key"   # 必填
  model: "gpt-4o"
  base_url: ""              # 使用 DashScope 等时填写自定义端点

user:
  name: "你的名字"
  active_path: "both"       # "go" | "ai" | "both"
```

### 3. 构建并运行

```bash
go build -o lizhu ./cmd/lizhu
./lizhu chat
```

启动后你会看到：

```
╔══════════════════════════════════════════════════╗
         骊珠 · 护道人·齐静春 已就位
╚══════════════════════════════════════════════════╝

  书院后山，晚风徐来。一位白衣儒生负手而立，目光
  温润，如春日远山……

输入 /help 查看可用命令。
```

---

## CLI 命令

```
lizhu chat                  与护道人开始修行对话
lizhu status                查看完整修行档案与法器谱（彩色进度条）
lizhu note add <文件路径>   将笔记/代码文件入库到 RAG 知识库
lizhu note list             列出所有已索引文件及摘要
```

### 对话内嵌命令

| 命令 | 功能 |
|------|------|
| `/assess` | 主动请求完整境界评估与破境方案（强制评估模式，LLM 必须给出完整报告）|
| `/status` | 在对话中查看当前修行档案 |
| `/clear` | 清空本次会话历史（已保存档案不受影响）|
| `/quit` / `/exit` | 退出对话 |
| `/help` | 显示帮助 |

### 知识库命令

```bash
# Markdown 笔记入库（自动提炼摘要 + 向量化）
lizhu note add ./notes/go/context_timeout.md

# Go 代码入库
lizhu note add ./src/worker_pool.go

# 查看已索引文件
lizhu note list
```

`note add` 内部流程：

```
① 读取文件内容
     │
     ▼
② Librarian Agent（LLM）
   提炼结构化摘要：要点列表 + 关键词
     │
     ├─────────────────────────────► PostgreSQL
     │         存储摘要 + 文件路径    knowledge_files 表
     ▼
③ Knowledge Ingester
   按 512 token 分块（支持中英文）
     │
     ▼
④ Embedding Model
   文本 → 向量（1536 / 1024 维）
     │
     ▼
⑤ Milvus 写入
   向量 + 原文 chunk 存储
```

对话时，Guardian Agent 自动用用户输入检索 Milvus，top-3 结果注入系统提示的「修行参考资料」节，护道人无需你手动提及笔记内容便能信手拈来。

---

## 完整配置说明

`lizhu.yaml` 完整字段：

```yaml
# ── LLM 配置 ──────────────────────────────────────
llm:
  provider: "openai"          # 目前支持 openai 兼容接口
  api_key: "sk-your-key"     # LLM API Key（必填）
  model: "gpt-4o"             # 推荐 gpt-4o / qwen-max / deepseek-chat
  base_url: ""                # 自定义端点，空则用 OpenAI 官方

# ── 数据库配置 ─────────────────────────────────────
database:
  host: "localhost"
  port: 5432
  name: "lizhu"
  user: "lizhu"
  password: "lizhu"
  ssl_mode: "disable"

# ── Milvus RAG 知识库（可选）──────────────────────
milvus:
  enabled: false              # true = 启用 RAG
  address: "localhost:19530"
  collection: "lizhu_knowledge"
  embedding_model: "text-embedding-v2"  # 与 llm.base_url 同端点的 embedding 模型

# ── 笔记目录 ──────────────────────────────────────
notes:
  path: "./notes"             # note add 时的文件根路径

# ── 世界观 ────────────────────────────────────────
worldview:
  path: "./configs/worldview"

# ── 用户 ──────────────────────────────────────────
user:
  name: "修行者"
  active_path: "both"         # "go" | "ai" | "both"

# ── 会话 ──────────────────────────────────────────
session:
  history_window: 5           # 注入上下文的历史会话数（影响 token 消耗）
  max_tokens: 4096

# ── 护道人人格 ────────────────────────────────────
guardian:
  persona_id: ""              # 空 = 无名护道人；内置: "qi_jingchun"
  persona_name: ""            # 对话显示名称
```

### 启用 RAG 知识库

```
1. docker-compose ps  →  lizhu-milvus 状态为 healthy
2. lizhu.yaml: milvus.enabled: true
3. 确认 milvus.embedding_model 与 llm.base_url 端点一致
4. lizhu note add <你的笔记>
5. 重启 lizhu chat  →  护道人对话自动检索笔记
```

> Milvus standalone 内存占用约 2–4 GB，不使用 RAG 时保持 `enabled: false` 即可，所有核心功能不受影响。

---

## 世界观配置

`configs/worldview/` 下所有 YAML 文件共同构成护道人的"道法体系"，支持运行时热更新：

```
configs/worldview/
├── base.yaml                  # 总纲：护道人职责、两大道结构、禁止行为
├── go_lianqishi_levels.yaml   # Go 练气士十五境（铜皮 → 至高）
├── ai_lianqishi_levels.yaml   # AI 练气士十五境（铜皮 → 至高）
├── wufu_levels.yaml           # 武夫十一境（泥胚 → 武神）
├── go_branches.yaml           # Go 路径分支：剑修 / 符箓 / 阵法 / 炼丹
├── ai_branches.yaml           # AI 路径分支：符咒宗师 / 藏书楼主 / Agent 统领 / 模型驯化师
├── sanjiaozhuzi.yaml          # 三教诸子哲学映射（儒 / 道 / 佛 / 兵 / 墨 / 法）
├── tool_mastery.yaml          # 法宝体系定义与四级评分标准（绘卷/方寸物/灵宠等《剑来》命名）
├── output_format.yaml         # 输出格式规范（assess_only: true，仅评估模式注入）
└── persona_qi_jingchun.yaml   # 齐静春人格语料与出场提示词
```

**新增世界观节**：新建 YAML，设置 `section_id`、`order`、`content` 即可，无需改代码。

**条件注入**：设置 `assess_only: true` 的节仅在 `/assess` 评估模式下注入系统提示，日常对话不占用 token。

---

## 护道人人格配置

护道人默认以无名状态出现。配置人格后，系统 Prompt 自动注入角色语料，并由 LLM 即兴生成符合角色气质的文学出场描写。

### 内置人格

| persona_id | 人物 | 风格 |
|---|---|---|
| `qi_jingchun` | 齐静春 | 儒家文圣弟子，温润如玉，春风化雨，循循善诱 |

### 启用人格

```yaml
# lizhu.yaml
guardian:
  persona_id: "qi_jingchun"
  persona_name: "齐静春"
```

### 自定义人格（无需改代码）

在 `configs/worldview/` 下新建 `persona_xxx.yaml`：

```yaml
section_id: "persona_xxx"
section_title: "护道人人格：某角色"
order: 5
persona_id: "xxx"
entrance_prompt: |
  请以某角色的口吻，用 80 字以内的第三人称散文描写他的出场场景……
content: |
  【护道人人格设定：某角色】
  背景：……
  说话风格：……
  经典语句：……
```

然后在 `lizhu.yaml` 中指定 `persona_id: "xxx"`，重启即生效。

---

## 开发指南

### 运行测试

```bash
go test ./...                                    # 全量
go test -v ./internal/knowledge/...              # RAG：分块、Embedding、Ingester/Retriever
go test -v ./internal/agent/librarian/...        # 摘要 Prompt 构建 + mock LLM
go test -v ./internal/agent/guardian/...         # eval_json 解析、mergeUnique
go test -v ./internal/memory/episodic/...        # ScoreToLevel 境界换算
go test -v ./internal/worldview/...              # 世界观 Loader、双模式 Prompt
```

### 静态检查

```bash
go vet ./...
```

### 数据库迁移

程序启动时自动执行，迁移文件位于 `internal/storage/migrations/`：

| 文件 | 内容 |
|------|------|
| `000001_init.up.sql` | profiles、sessions、tool_mastery 表 |
| `000002_knowledge_files.up.sql` | knowledge_files 表（含 summary 字段）|
| `000003_ability_evidence.up.sql` | ability_evidence 表（结构化能力证据，含 category / tool / confidence 字段）|

```bash
# 手动查看迁移状态（需安装 migrate CLI）
migrate -database "postgres://lizhu:lizhu@localhost:5432/lizhu?sslmode=disable" \
        -path internal/storage/migrations status
```

---

## 交付路线

```
一期 ✅  CLI 交互式对话
        修行档案持久化（PostgreSQL）
        世界观 YAML 热更新
        护道人人格系统

二期 ✅  Milvus RAG 知识库
        知识整理官 Agent（笔记摘要提炼）
        评估/对话双模式流式输出
        lipgloss 彩色档案展示
        法宝体系世界观精确分类
        整次会话摘要（替代逐轮摘要）
        结构化能力证据体系（跨对话积累，/assess 时自动注入）
        退出进度展示（封存步骤逐行打印）

三期 🔜  Web API + 精美 Web UI
        酷炫修行周报/月报
        多 Agent 协作（对练陪练、考官等）
        移动端 / 微信小程序
```

---

<p align="center">
  <sub>骊珠洞天曾有一位护道人，他选择留下来。</sub>
</p>
