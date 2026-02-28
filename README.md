# 骊珠 (Lizhu)

<p align="center">
  <img src="assets/banner.png" alt="骊珠 Banner" width="480"/>
</p>

> 基于《剑来》修行世界观设计的 **Go & AI 开发者智能护道系统**

护道人持续做三件事：**诊断**你的修行境界、**规划**最短破境路径、**护道**避免走火入魔。

---

## 功能特性

### 核心能力（一期 + 二期）

| 特性 | 说明 |
|------|------|
| **双轨练气士评估** | Go开发练气士（十五境）与 AI应用练气士（十五境）独立评分，境界名称严格遵循原著 |
| **武夫底层内功** | 计算机底层硬功独立追踪（十一境）|
| **法器谱** | 七大类工具掌握程度客观量化（0-100，初识/熟用/精通/宗师四级制）|
| **按需评估** | 护道人智能判断对话模式：闲聊/问技术时自然对话，汇报成果/使用 `/assess` 时输出完整境界评估 |
| **长期记忆** | 修行档案与会话摘要持久化，下次对话无需重复自我介绍 |
| **世界观热更新** | `configs/worldview/*.yaml` 随时补充设定，无需改代码重新编译 |

### 二期新增能力

| 特性 | 说明 |
|------|------|
| **Bubble Tea TUI** | 全屏终端界面替代 readline，三层布局（标题栏 + 可滚动对话区 + 输入框），彩色进度条 |
| **流式输出** | LLM token 逐字追加到 TUI viewport，`<eval_json>` 块静默过滤，不污染对话区 |
| **RAG 知识库** | 笔记/代码文件向量化写入 Milvus，对话时自动检索 top-3 相关片段注入系统提示 |
| **知识整理官 Agent** | `lizhu note add` 时调用 Librarian 对文件内容提炼结构化摘要（要点列表 + 关键词），摘要随文件记录持久化 |
| **lipgloss 彩色档案** | `lizhu status` 输出彩色境界进度条、分区标题高亮展示 |

---

## 快速开始

### 前置要求

- Go 1.21+
- Docker & Docker Compose（PostgreSQL + Milvus）
- OpenAI API Key（或兼容接口，如 DashScope、deepseek 等）

### 1. 启动基础设施

```bash
docker-compose up -d
# Milvus 冷启动较慢（约 30-60s），等待健康检查通过
docker-compose ps
```

`docker-compose.yaml` 包含两个服务：

| 服务 | 镜像 | 端口 | 说明 |
|------|------|------|------|
| `lizhu-postgres` | postgres:16-alpine | 5432 | 修行档案、会话摘要、知识文件记录 |
| `lizhu-milvus` | milvusdb/milvus:v2.4.15 | 19530 / 9091 | 向量知识库（RAG，可选启用）|

> **仅使用核心对话功能**：只需要 postgres 正常运行，Milvus 可以在配置中保持 `enabled: false`。

### 2. 创建配置文件

```bash
cp configs/lizhu.yaml.example lizhu.yaml
```

编辑 `lizhu.yaml`，填写必要参数（详见[完整配置说明](#完整配置说明)）：

```yaml
llm:
  api_key: "sk-your-key"   # LLM API Key（必填）
  model: "gpt-4o"
  base_url: ""              # 自定义 OpenAI 兼容接口端点，空则用官方

user:
  name: "你的道号"
  active_path: "both"       # "go" | "ai" | "both"
```

### 3. 构建并运行

```bash
go build -o lizhu ./cmd/lizhu
./lizhu chat
```

---

## CLI 命令

```
lizhu chat                  与护道人开始修行对话（全屏 TUI 界面）
lizhu status                查看完整修行档案与法器谱（彩色进度条）
lizhu note add <文件路径>   将笔记/代码文件入库到 RAG 知识库
lizhu note list             列出所有已索引的文件及摘要信息
```

### TUI 对话界面

`lizhu chat` 启动后进入全屏 Bubble Tea TUI 界面：

```
┌─────────────────────────────────────────────┐
│  骊珠 · 护道人·齐静春          ● 思考中…   │
├─────────────────────────────────────────────┤
│                                             │
│  [对话区 - 可上下滚动]                      │
│                                             │
│  护道人·齐静春 ›                            │
│  ...                                        │
│                                             │
├─────────────────────────────────────────────┤
│  输入修行感悟，或 /help 查看命令…           │
│  /help /assess /status /clear /quit         │
└─────────────────────────────────────────────┘
```

**键盘操作**：

| 按键 | 功能 |
|------|------|
| `Enter` | 发送消息 |
| `PgUp` / `PgDn` | 上下滚动对话记录 |
| `Ctrl+C` | 退出 |

**内嵌命令**（在输入框中输入）：

| 命令 | 功能 |
|------|------|
| `/assess` | 主动请求完整境界评估与破境方案（强制进入评估模式）|
| `/status` | 在对话区内查看当前修行档案 |
| `/clear` | 清空本次会话历史（不影响已保存档案）|
| `/quit` | 退出对话 |
| `/help` | 显示帮助信息 |

### 知识库命令

```bash
# 将 Markdown 笔记入库（自动提炼摘要 + 向量化写入 Milvus）
lizhu note add ./notes/golang-concurrency.md

# 将 Go 代码文件入库
lizhu note add ./src/main.go

# 查看已索引文件列表
lizhu note list
```

`note add` 完整流程：

```
读取文件 → 知识整理官提炼摘要 → 分块向量化 → 写入 Milvus → 记录到 PostgreSQL
```

护道人对话时会自动从 Milvus 检索与用户输入最相关的 top-3 知识块，注入系统提示中的「修行参考资料」节。

---

## 完整配置说明

`lizhu.yaml` 完整字段：

```yaml
# ── LLM 配置 ──────────────────────────────────────
llm:
  provider: "openai"         # 目前支持 openai 兼容接口
  api_key: "sk-your-key"    # LLM API Key（必填）
  model: "gpt-4o"            # 对话模型；推荐 gpt-4o / gpt-4-turbo / qwen-max
  base_url: ""               # 自定义端点（DashScope / deepseek 等），空则用 OpenAI 官方

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
  enabled: false             # true = 启用 RAG；需先 docker-compose up -d 并等待 Milvus 就绪
  address: "localhost:19530" # Milvus gRPC 地址
  collection: "lizhu_knowledge"   # Collection 名称（首次运行自动创建）
  embedding_model: "text-embedding-3-small"  # 向量化模型（与 llm.base_url 同端点）

# ── 世界观 ────────────────────────────────────────
worldview:
  path: "./configs/worldview"   # 世界观 YAML 配置目录

# ── 用户 ──────────────────────────────────────────
user:
  name: "修行者"             # 你的道号（显示在档案和护道人对话中）
  active_path: "both"        # "go"（只练 Go）| "ai"（只练 AI）| "both"（双修）

# ── 会话 ──────────────────────────────────────────
session:
  history_window: 5          # 注入系统提示的历次会话摘要数量（越大 token 消耗越多）
  max_tokens: 4096           # 单次对话最大输出 token

# ── 护道人人格 ────────────────────────────────────
guardian:
  persona_id: ""             # 人格ID，空=无名护道人；内置: "qi_jingchun"
  persona_name: ""           # 对话框显示名称，空=显示"护道人"
```

### 启用 RAG 知识库

1. 确保 Milvus 正在运行：`docker-compose ps`（状态为 healthy）
2. 在 `lizhu.yaml` 中设置 `milvus.enabled: true`
3. 确认 `milvus.embedding_model` 与你的 LLM 端点支持的 embedding 模型一致
4. 将笔记文件入库：`lizhu note add <文件路径>`
5. 重启 `lizhu chat`，对话中将自动检索相关知识

> **注意**：Milvus standalone 模式运行时内存占用约 2-4 GB。不使用 RAG 时保持 `enabled: false` 即可，所有核心功能正常工作。

---

## 世界观配置

`configs/worldview/` 目录下所有 YAML 文件共同构成护道人的"道法体系"，支持热更新：

| 文件 | 内容 |
|------|------|
| `base.yaml` | 总纲、护道人职责、两大道结构、禁止行为 |
| `go_lianqishi_levels.yaml` | Go开发练气士十五境（铜皮→至高）|
| `ai_lianqishi_levels.yaml` | AI应用练气士十五境（铜皮→至高）|
| `wufu_levels.yaml` | 武夫十一境（泥胚→武神）|
| `go_branches.yaml` | Go路径分支：剑修 / 符箓 / 阵法 / 炼丹 |
| `ai_branches.yaml` | AI路径分支：符咒宗师 / 藏书楼主 / Agent统领 / 模型驯化师 |
| `sanjiaozhuzi.yaml` | 三教诸子哲学映射（儒/道/佛/兵/墨/法）|
| `tool_mastery.yaml` | 法器谱七大类定义与四级评分标准 |
| `output_format.yaml` | 自适应输出格式：评估模式（模式 A）完整输出，对话模式（模式 B）精简输出，含 eval_json 块供系统解析 |

**新增世界观设定**：在 `configs/worldview/` 下新建 YAML 文件，设置 `section_id`、`order` 和 `content` 即可，无需改代码。

---

## 护道人人格配置

护道人默认以无名状态出现，你可以在 `lizhu.yaml` 中配置人格，让护道人化身为《剑来》中的特定角色，带来专属的说话风格与语料。

### 启用齐静春人格

```yaml
guardian:
  persona_id: "qi_jingchun"   # 人格ID，对应 configs/worldview/ 中的 persona_xxx.yaml
  persona_name: "齐静春"       # TUI 标题栏和对话区显示名称
```

启用后，TUI 标题栏变为 `骊珠 · 护道人·齐静春`，系统 Prompt 自动注入齐静春的说话风格语料（儒家文圣亲传弟子，温润如玉，循循善诱）。

### 关闭人格（无名护道人）

```yaml
guardian:
  persona_id: ""
  persona_name: ""
```

### 内置人格列表

| persona_id | 人物 | 风格描述 |
|---|---|---|
| `qi_jingchun` | 齐静春 | 儒家文圣弟子，温润如玉，春风化雨，循循善诱 |

### 自定义人格（无需改代码）

1. 在 `configs/worldview/` 下新建 `persona_xxx.yaml`：

```yaml
section_id: "persona_xxx"
section_title: "护道人人格：某角色"
order: 5
persona_id: "xxx"          # 自定义唯一ID
content: |
  ========================
  【护道人人格设定：某角色】
  ========================
  （填写角色背景、说话风格、经典语句等语料）
```

2. 在 `lizhu.yaml` 中配置：

```yaml
guardian:
  persona_id: "xxx"
  persona_name: "角色显示名"
```

3. 重新编译或重启即生效，无需改任何代码。

---

## 架构概览

```
lizhu/
├── cmd/lizhu/
│   └── cmd/                # CLI 命令（cobra）
│       ├── root.go         # 根命令、依赖初始化
│       ├── chat.go         # chat 命令 → 启动 Bubble Tea TUI
│       ├── note.go         # note add / note list 命令
│       └── status.go       # status 命令（lipgloss 彩色输出）
├── internal/
│   ├── agent/
│   │   ├── guardian/       # 护道人 Agent（Eino ChatModel + 上下文构建 + RAG 注入）
│   │   └── librarian/      # 知识整理官 Agent（笔记摘要提炼）
│   ├── tui/                # Bubble Tea TUI（model / view / keymap）
│   ├── knowledge/          # RAG 知识库（Milvus 向量写入 + 语义检索 + Embedding）
│   ├── memory/
│   │   └── episodic/       # 情节记忆（PostgreSQL：档案、会话摘要、法器谱、知识文件）
│   ├── worldview/          # 世界观 YAML 加载器
│   ├── checkpoint/         # Eino CheckPointStore（PostgreSQL 后端）
│   └── storage/            # DB 连接 + 迁移管理
└── configs/worldview/      # 世界观 YAML 配置目录
```

### 数据流

```
用户输入
  │
  ▼
TUI（internal/tui）            ← Bubble Tea 事件循环
  │
  ▼
Guardian Agent（internal/agent/guardian）
  ├── 世界观 Loader → 系统提示词
  ├── Episodic Memory → 用户档案 + 会话摘要 + 法器谱
  └── Knowledge Retriever → Milvus top-3 知识块（可选）
  │
  ▼
LLM 流式输出 → token 逐字回传 TUI → <eval_json> 块静默解析 → 持久化
```

```
lizhu note add <文件>
  │
  ▼
Librarian Agent → 结构化摘要（LLM 提炼）
  │
  ▼
Knowledge Ingester → 分块 → Embedding → Milvus
  │
  ▼
PostgreSQL（knowledge_files 表，含摘要字段）
```

**技术选型**：Go · [Eino](https://github.com/cloudwego/eino) · [Bubble Tea](https://github.com/charmbracelet/bubbletea) · [lipgloss](https://github.com/charmbracelet/lipgloss) · PostgreSQL · Milvus（可选）· cobra/viper

---

## 开发指南

### 运行测试

```bash
# 全量测试
go test ./...

# 指定包（含详情）
go test -v ./internal/knowledge/...        # chunkText、Embedding mock、Ingester/Retriever stub
go test -v ./internal/agent/librarian/...  # prompt 构建、Summarize mock LLM
go test -v ./internal/tui/...              # TUI Model 状态机、View 渲染辅助函数
go test -v ./internal/agent/guardian/...   # 评估 JSON 解析、mergeUnique
go test -v ./internal/memory/episodic/...  # ScoreToLevel
go test -v ./internal/worldview/...        # 世界观 Loader
```

### 静态检查

```bash
go vet ./...
```

### 新增世界观节

1. 在 `configs/worldview/` 新建 `my_section.yaml`
2. 设置字段：`section_id`、`order`（决定在系统 Prompt 中的位置）、`content`
3. 若仅适用特定路径：设置 `path_filter: "go"` 或 `path_filter: "ai"`
4. 重启 `lizhu chat` 即生效，无需重新编译

### 数据库迁移

迁移文件在 `internal/storage/migrations/`，程序启动时自动执行。

| 文件 | 内容 |
|------|------|
| `000001_init.up.sql` | 初始表结构（profiles、sessions、tool_mastery）|
| `000002_knowledge_files.up.sql` | 知识库文件记录表（knowledge_files，含 summary 字段）|

```bash
# 查看迁移状态（需安装 migrate CLI）
migrate -database "postgres://lizhu:lizhu@localhost:5432/lizhu?sslmode=disable" \
        -path internal/storage/migrations status
```

---

## 交付路线

- **一期（已完成）**：CLI readline 对话 + 修行档案持久化 + 世界观 YAML 配置 + 护道人人格系统
- **二期（已完成）**：Bubble Tea TUI 界面 + Milvus RAG 知识库 + 知识整理官 Agent + lipgloss 彩色档案
- **三期（计划中）**：Web API + 精美 Web UI + 酷炫修行报告 + 多 Agent 协作（对练陪练等）
