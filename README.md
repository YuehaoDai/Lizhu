# 骊珠 (Lizhu)

<p align="center">
  <img src="assets/banner.png" alt="骊珠 Banner" width="480"/>
</p>

> 基于《剑来》修行世界观设计的 **Go & AI 开发者智能护道系统**

护道人持续做三件事：**诊断**你的修行境界、**规划**最短破境路径、**护道**避免走火入魔。

---

## 功能特性

| 特性 | 说明 |
|------|------|
| **双轨练气士评估** | Go开发练气士（十五境）与 AI应用练气士（十五境）独立评分，境界名称严格遵循原著 |
| **武夫底层内功** | 计算机底层硬功独立追踪（十一境）|
| **法器谱** | 七大类工具掌握程度客观量化（0-100，初识/熟用/精通/宗师四级制）|
| **按需评估** | 护道人智能判断对话模式：闲聊/问技术时自然对话，汇报成果/使用 `/assess` 时输出完整境界评估与破境方案 |
| **长期记忆** | 修行档案与会话摘要持久化，下次对话无需重复自我介绍 |
| **世界观热更新** | `configs/worldview/*.yaml` 随时补充设定，无需改代码重新编译 |
| **可扩展 Agent** | 基于 Eino Graph 编排，二期可扩展为多 Agent（知识整理官、对练陪练等）|

---

## 快速开始

### 前置要求

- Go 1.21+
- Docker & Docker Compose（用于 PostgreSQL + Milvus）
- OpenAI API Key（或兼容接口）

### 1. 启动基础设施

```bash
docker-compose up -d
# 等待 postgres 健康检查通过（约 10s）
docker-compose ps
```

### 2. 创建配置文件

```bash
cp configs/lizhu.yaml.example lizhu.yaml
```

编辑 `lizhu.yaml`，至少填写：

```yaml
llm:
  api_key: "sk-your-key"   # OpenAI API Key 或兼容接口 Key
  model: "gpt-4o"

user:
  name: "你的名字"
  active_path: "both"      # "go" | "ai" | "both"
```

### 3. 构建并运行

```bash
go build -o lizhu ./cmd/lizhu
./lizhu chat
```

---

## CLI 命令

```
lizhu chat                # 与护道人开始修行对话（主功能）
lizhu status              # 查看完整修行档案与法器谱
lizhu note add <文件>     # 将笔记/代码文件入库到知识库
lizhu note list           # 列出已索引文件（二期）
```

**对话内嵌命令**（chat 模式中输入）：

```
/assess   主动请求完整境界评估与破境方案（强制进入评估模式）
/status   查看当前修行档案与法器谱
/clear    清空本次会话历史（不影响已保存档案）
/quit     退出
/help     帮助
```

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
  persona_name: "齐静春"       # 对话框显示名称
```

启用后，对话提示符变为 `护道人·齐静春 ›`，系统 Prompt 自动注入齐静春的说话风格语料（儒家文圣亲传弟子，温润如玉，循循善诱）。

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
├── cmd/lizhu/              # CLI 入口（cobra）
├── internal/
│   ├── agent/guardian/     # 护道人 Agent（Eino ChatModel + 上下文构建）
│   ├── memory/
│   │   └── episodic/       # 情节记忆（PostgreSQL：档案、会话摘要、法器谱）
│   ├── worldview/          # 世界观 YAML 加载器
│   ├── knowledge/          # 知识库文件入库（Milvus，可选）
│   ├── checkpoint/         # Eino CheckPointStore（PostgreSQL 后端）
│   └── storage/            # DB 连接 + 迁移管理
└── configs/worldview/      # 世界观 YAML 配置目录
```

**技术选型**：Go · [Eino](https://github.com/cloudwego/eino) · PostgreSQL · Milvus（可选）· cobra/viper

---

## 开发指南

### 运行测试

```bash
# 全量测试
go test ./...

# 指定包（含详情）
go test ./internal/worldview/... -v
go test ./internal/agent/guardian/... -v
go test ./internal/memory/episodic/... -v
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

```bash
# 查看迁移状态（需安装 migrate CLI）
migrate -database "postgres://lizhu:lizhu@localhost:5432/lizhu?sslmode=disable" \
        -path internal/storage/migrations status
```

---

## 交付路线

- **一期（当前）**：CLI 对话 + 修行档案持久化 + 世界观 YAML 配置
- **二期**：TUI 终端界面 + Milvus RAG 知识库 + 知识整理官 Agent
- **三期**：Web API + 精美 Web UI + 酷炫修行报告
