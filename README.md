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
- [系统工作原理](#系统工作原理)
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

### 三期新增能力

| 特性 | 说明 |
|------|------|
| **修炼任务单** | `/assess` 结束后，Librarian 自动从弱点中提炼 1-2 条具体可执行任务（含明确验收标准），持久化至 `tasks` 表；护道人在下次会话中主动追问进展，用户汇报完成后自动验收并更新状态 |
| **任务积压保护** | 待完成任务 ≥ 3 条时不再生成新任务，防止任务越积越多带来压迫感 |
| **自然语言验收** | 护道人检测到回复中的 `[TASK_DONE:<标题>]` 标记后自动将任务标记为完成，无需修行者执行额外命令 |
| **`/tasks` 命令** | 随时在对话中输入 `/tasks`，列出全部待完成任务的标题、描述与验收标准 |
| **联网查阅** | 护道人通过 LLM Tool Calling 自主决定是否抓取网页；当你提供网址或问题需要实时信息时，自动拉取页面纯文本整合进回复，无需额外命令 |

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
    GA->>DB: persistEvaluation（更新档案、法宝库）
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
│   │   │   ├── agent.go     # 核心逻辑：上下文构建、RAG 注入、流式输出、工具调用 agentic loop
│   │   │   ├── browse.go    # 联网查阅工具：browse_web Tool 定义与 fetchWebContent 实现
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
│   │   └── repo.go          # 档案 CRUD、会话摘要、法宝库
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

## 系统工作原理

### 护道人每次对话看到什么

每次你发送一条消息，Guardian Agent 会把以下内容拼成一个完整的 System Prompt 发给 LLM：

```
[世界观 YAML 节]          ← configs/worldview/ 下所有 YAML 拼装
[修行者身份与档案]         ← cultivation_profile 表：境界分、法宝掌握度
[近期会话摘要]             ← sessions 表最近 N 条（history_window 控制）
[知识文件摘要列表]         ← knowledge_files 表：note add 过的文件名+摘要
[RAG 检索结果]             ← Milvus 按当前输入相似度检索 top-3 知识块（仅 enabled=true 时）
[待完成任务单]             ← tasks 表 status=pending 的任务（三期新增）
[历史能力证据]             ← ability_evidence 表最近 20 条（仅 /assess 模式注入）
```

这就是为什么护道人"认得你"——每次对话他都拿到了你所有的历史积累，而不是空白上下文。

### 记忆层架构

骊珠维护四类持久记忆，各司其职：

| 记忆类型 | 存储位置 | 更新时机 | 用途 |
|------|------|------|------|
| **修行档案** | `cultivation_profile` | `/assess` 评估后 | 境界分、法宝掌握度，护道人对你的"当前认知" |
| **会话摘要** | `sessions` | 每次退出后 | 近期对话主题，提供短期上下文连续性 |
| **能力证据** | `ability_evidence` | 每次退出后 | 具体技术事实（工具/置信度），评估时作为"证明材料"注入 |
| **知识文件** | `knowledge_files` + Milvus | `lizhu note add` 时 | 笔记/代码语义索引，对话时自动召回 |

### `/assess` 评估完整流程

```
你输入 /assess
    │
    ▼
Guardian 构建 System Prompt（加入 eval_json 格式规范 + 历史能力证据）
    │
    ▼
LLM 流式输出（可见部分打印给你）
检测到 <eval_json> 标签 → 立即截断打印，后台 goroutine 接管
    │
    ├─► 解析 eval_json → 更新 cultivation_profile（境界分 + 法宝）
    │
    └─► Librarian Agent 异步生成修炼任务（锚定最近弱点）→ 存入 tasks 表

你退出对话时（/quit）：
    ├─► 等待评估 goroutine 完成
    ├─► Librarian 生成会话摘要 → 存入 sessions
    └─► Librarian 提炼能力证据 → 存入 ability_evidence
```

### 修炼任务单工作机制

任务不是每次对话都生成，而是与评估绑定：

```
/assess 完成
    │ Librarian 收到：本次对话 + 最近 10 条能力证据 + 近 3 次会话摘要 + 当前档案分数
    │ → 锚定近期学习方向，找 1-2 个薄弱点
    │ → 生成带明确验收标准的具体任务（待完成任务 ≥ 3 时跳过）
    ▼
tasks 表写入（status = pending）

下次 lizhu chat
    │ System Prompt 自动注入待完成任务
    │ → 护道人主动追问进展
    ▼
你汇报完成情况
    │ 护道人对照 acceptance_criteria 判断
    │ → 通过：回复中附上 [TASK_DONE:<标题>]
    │ → 系统自动更新 tasks.status = done
    ▼
/tasks 命令随时查看剩余任务
```

**关键约束**：任务必须有明确的可验证产出（如"提交一段运行代码"），而非"学习X"这类无法验收的目标。

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
lizhu status                查看完整修行档案与法宝库（彩色进度条）
lizhu note add <文件路径>   将笔记/代码文件入库到 RAG 知识库
lizhu note list             列出所有已索引文件及摘要
```

### 对话内嵌命令

| 命令 | 功能 |
|------|------|
| `/assess` | 主动请求完整境界评估与破境方案（强制评估模式，LLM 必须给出完整报告）|
| `/tasks` | 查看当前修炼任务单（标题、描述、验收标准）|
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
| `000004_tasks.up.sql` | tasks 表（修炼任务单，含 acceptance_criteria / status / completed_at 字段）|

```bash
# 手动查看迁移状态（需安装 migrate CLI）
migrate -database "postgres://lizhu:lizhu@localhost:5432/lizhu?sslmode=disable" \
        -path internal/storage/migrations status
```

---

## 交付路线

骊珠的远期形态是 **Developer Growth OS**——不是一个等你开口的问答工具，而是主动感知你成长状态的开发者操作系统。

```
一期 ✅  CLI 交互式对话
        修行档案持久化（PostgreSQL）
        世界观 YAML 热更新
        护道人人格系统

二期 ✅  Milvus RAG 知识库
        知识整理官 Agent（笔记摘要 + 能力证据提炼）
        评估/对话双模式流式输出
        结构化能力证据体系（跨对话积累，/assess 时自动注入）
        整次会话摘要 + 退出进度展示

三期 🔜  [✅] 修炼任务单：评估后生成锚定弱点的具体任务，跨对话追踪验收
         [✅] 联网查阅：LLM Tool Calling 驱动，护道人自主决定是否实时查询网页
         [近] 修行周报：Librarian 聚合多次会话，生成成长复盘
         [中] MCP Server：将护道人记忆层暴露为标准接口，接入 Cursor / Claude Desktop
         [中] Web 仪表盘：成长曲线、境界时间线、法宝库热力图
         [远] 多 Agent 协作：考官出题、对练陪练（费曼技巧数字化）
         [远] 主动感知：Git hook + 沉默触发推送，无需你主动喂养
```

---

<p align="center">
  <sub>骊珠洞天曾有一位护道人，他选择留下来。</sub>
</p>
