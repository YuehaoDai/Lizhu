// Package guardian 实现骊珠护道人 Agent。
// 基于 Eino ChatModel，构建"世界观系统Prompt + 历史记忆 + RAG知识"的完整上下文，
// 调用 LLM 生成结构化修行评估，并将结果持久化到 PostgreSQL。
package guardian

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/YuehaoDai/lizhu/internal/agent/librarian"
	"github.com/YuehaoDai/lizhu/internal/knowledge"
	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	"github.com/YuehaoDai/lizhu/internal/worldview"
	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Config 护道人 Agent 配置。
type Config struct {
	// LLM
	LLMProvider string
	APIKey      string
	Model       string
	BaseURL     string
	// 世界观
	WorldViewDir string
	ActivePath   worldview.ActivePath
	// 用户
	UserID   string
	UserName string
	// 护道人人格
	PersonaID   string // 人格ID，对应世界观 YAML 中的 persona_id 字段
	PersonaName string // 显示名称，用于 UI 展示
	// 会话
	HistoryWindow int // 注入的历史摘要数量
	// RAG 知识库（可选，Enabled=false 时跳过检索）
	KnowledgeCfg knowledge.Config
	// 搜索（可选，空字符串时禁用 search_web 工具）
	BraveAPIKey string
}

// Agent 护道人智能体。
type Agent struct {
	cfg          Config
	model        model.ChatModel
	toolModel    model.ToolCallingChatModel // 绑定 browse_web 工具后的模型实例（可为 nil）
	loader       *worldview.Loader
	repo         *episodic.Repository
	retriever    *knowledge.Retriever
	librarian    *librarian.Agent
	persistWg    sync.WaitGroup // 追踪后台持久化 goroutine，退出前等待完成
}

// PersonaName 返回护道人显示名称（空字符串表示使用默认"护道人"）。
func (a *Agent) PersonaName() string { return a.cfg.PersonaName }

// WaitPersist 等待所有后台持久化 goroutine 完成，应在程序退出前调用。
func (a *Agent) WaitPersist() { a.persistWg.Wait() }

// New 创建护道人 Agent。
func New(ctx context.Context, cfg Config, repo *episodic.Repository) (*Agent, error) {
	m, err := newChatModel(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("guardian: init model: %w", err)
	}

	// 若 RAG 已启用，快速探测 Milvus 连通性（3 秒超时）。
	// 不可达时自动降级为禁用，避免每次对话都打印超时警告。
	if cfg.KnowledgeCfg.Enabled {
		probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := knowledge.ProbeMilvus(probeCtx, cfg.KnowledgeCfg.Address); err != nil {
			fmt.Printf("[知识库] Milvus 不可达（%v），本次会话禁用 RAG。\n", err)
			cfg.KnowledgeCfg.Enabled = false
		} else {
			fmt.Println("[知识库] Milvus 连接正常，RAG 已启用。")
		}
	}

	libAgent, err := librarian.New(ctx, librarian.Config{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		// Librarian 初始化失败不应阻断主流程，降级为 nil（persistSession 会回退到截断摘要）
		fmt.Printf("[警告] 知识整理官初始化失败，会话摘要将使用截断文本: %v\n", err)
		libAgent = nil
	}

	// 尝试绑定网页工具，失败时降级为 nil（不影响核心对话功能）
	var toolModel model.ToolCallingChatModel
	if tc, ok := m.(model.ToolCallingChatModel); ok {
		tools := []*schema.ToolInfo{browseWebToolInfo}
		if cfg.BraveAPIKey != "" {
			tools = append(tools, searchWebToolInfo)
			fmt.Println("[联网] Brave Search 已启用，护道人可主动搜索互联网。")
		}
		tm, err := tc.WithTools(tools)
		if err != nil {
			fmt.Printf("[警告] 网页工具绑定失败，本次会话禁用联网功能: %v\n", err)
		} else {
			toolModel = tm
		}
	}

	return &Agent{
		cfg:       cfg,
		model:     m,
		toolModel: toolModel,
		loader:    worldview.NewLoader(cfg.WorldViewDir),
		repo:      repo,
		retriever: knowledge.NewRetriever(cfg.KnowledgeCfg),
		librarian: libAgent,
	}, nil
}

// GenerateEntrance 调用 LLM 生成护道人出场场景描写。
// 若该人格未配置 entrance_prompt，或 LLM 调用失败，返回空字符串（调用方应优雅降级）。
func (a *Agent) GenerateEntrance(ctx context.Context, firstTime bool) string {
	entrancePrompt, err := a.loader.LoadEntrancePrompt(a.cfg.PersonaID)
	if err != nil || entrancePrompt == "" {
		return ""
	}

	scenario := "修行者再次前来拜访，护道人正在做某件日常之事，请生成一段他的出场场景描写。"
	if firstTime {
		scenario = "修行者初次前来拜访，护道人正在做某件日常之事，请生成一段他的出场场景描写，略带初见的意味。"
	}

	genCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := a.model.Generate(genCtx, []*schema.Message{
		schema.SystemMessage(entrancePrompt),
		schema.UserMessage(scenario),
	})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(resp.Content)
}

// Chat 处理一轮对话（非流式）。assess=true 时进入评估模式（生成 eval_json）。
func (a *Agent) Chat(ctx context.Context, history []*schema.Message, userInput string, assess bool) (string, []*schema.Message, error) {
	systemMsg, err := a.buildSystemMessage(ctx, userInput, assess)
	if err != nil {
		return "", nil, fmt.Errorf("build system message: %w", err)
	}
	messages := buildMessages(systemMsg, history, userInput)

	resp, err := a.model.Generate(ctx, messages)
	if err != nil {
		return "", nil, fmt.Errorf("llm generate: %w", err)
	}
	reply := resp.Content

	if assess {
		if err := a.persistEvaluation(ctx, reply); err != nil {
			fmt.Printf("[警告] 修行档案持久化失败: %v\n", err)
		}
	}

	newHistory := append(history,
		schema.UserMessage(userInput),
		schema.AssistantMessage(reply, nil),
	)
	return reply, newHistory, nil
}

// ChatStream 处理一轮对话（流式）。assess=true 时进入评估模式（生成 eval_json）。
//
// Mode B（assess=false）：系统提示不含 eval_json 指令，LLM 不会生成结构化块，
// 流式 token 全部透传给 onToken，流结束后保存轻量级会话记录。
//
// Mode A（assess=true）：系统提示含完整评估指令，检测到 <eval_json 后立即返回
// 控制权，剩余 token 由后台 goroutine 消费并持久化，避免用户感知延迟。
func (a *Agent) ChatStream(ctx context.Context, history []*schema.Message, userInput string, onToken func(string), assess bool) (string, []*schema.Message, error) {
	systemMsg, err := a.buildSystemMessage(ctx, userInput, assess)
	if err != nil {
		return "", nil, fmt.Errorf("build system message: %w", err)
	}
	messages := buildMessages(systemMsg, history, userInput)

	// ── Mode B + tool calling：有 toolModel 时走 agentic loop ──
	// 仅在非评估模式下启用，以免干扰 eval_json 的解析流程。
	if !assess && a.toolModel != nil {
		return a.chatStreamWithTools(ctx, messages, history, userInput, onToken)
	}

	stream, err := a.model.Stream(ctx, messages)
	if err != nil {
		return "", nil, fmt.Errorf("llm stream: %w", err)
	}

	var fullReply strings.Builder

	if assess {
		// ── Mode A：评估模式，检测 eval_json 并后台持久化 ──
		for {
			chunk, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				stream.Close()
				return "", nil, fmt.Errorf("stream recv: %w", err)
			}
			token := chunk.Content
			if token == "" {
				continue
			}
			onToken(token)
			fullReply.WriteString(token)

			// 检测到 eval_json 段落起始（标题行或标签本身）：可见文字已完整，交给后台 goroutine
			replyStr := fullReply.String()
			if !strings.Contains(replyStr, "<eval_json") && !strings.Contains(replyStr, "修行档案JSON") {
				continue
			}

		visibleReply := fullReply.String()
		a.persistWg.Add(1)
		go func() {
			defer a.persistWg.Done()
			defer stream.Close()
			var tail strings.Builder
			for {
				c, e := stream.Recv()
				if e != nil {
					break
				}
				tail.WriteString(c.Content)
			}
			if perr := a.persistEvaluation(context.Background(), visibleReply+tail.String()); perr != nil {
				fmt.Printf("[警告] 修行档案持久化失败: %v\n", perr)
			}
		}()

			newHistory := append(history,
				schema.UserMessage(userInput),
				schema.AssistantMessage(visibleReply, nil),
			)
			return visibleReply, newHistory, nil
		}

		// 降级路径：eval_json 未出现（通常不应走到此处）
		reply := fullReply.String()
		stream.Close()
		if err := a.persistEvaluation(ctx, reply); err != nil {
			fmt.Printf("[警告] 修行档案持久化失败: %v\n", err)
		}
		newHistory := append(history,
			schema.UserMessage(userInput),
			schema.AssistantMessage(reply, nil),
		)
		return reply, newHistory, nil
	}

	// ── Mode B：普通护道对话，全量流式输出 ──
	// LLM 有时仍会生成 <eval_json> 块，用滑动缓冲区在触发前截断，保证输出干净。
	defer stream.Close()
	var modeBBuf strings.Builder
	modeBSuppressed := false
	const modeBTrig = "<eval_json>"
	const modeBWin = len(modeBTrig) - 1
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, fmt.Errorf("stream recv: %w", err)
		}
		token := chunk.Content
		if token == "" {
			continue
		}
		if modeBSuppressed {
			continue
		}
		modeBBuf.WriteString(token)
		s := modeBBuf.String()
		if idx := strings.Index(s, modeBTrig); idx >= 0 {
			modeBSuppressed = true
			// 从触发行的行首截断
			printUntil := idx
			if nl := strings.LastIndex(s[:idx], "\n"); nl >= 0 {
				printUntil = nl + 1
			}
			visible := s[:printUntil]
			if visible != "" {
				onToken(visible)
				fullReply.WriteString(visible)
			}
			modeBBuf.Reset()
			continue
		}
		// 保留末尾预警窗口，安全部分直接输出
		safe := len(s) - modeBWin
		if safe > 0 {
			onToken(s[:safe])
			fullReply.WriteString(s[:safe])
			tail := s[safe:]
			modeBBuf.Reset()
			modeBBuf.WriteString(tail)
		}
	}
	// 流结束，刷出缓冲区剩余（未触发则全部输出）
	if !modeBSuppressed && modeBBuf.Len() > 0 {
		onToken(modeBBuf.String())
		fullReply.WriteString(modeBBuf.String())
	}

	reply := fullReply.String()
	newHistory := append(history,
		schema.UserMessage(userInput),
		schema.AssistantMessage(reply, nil),
	)
	return reply, newHistory, nil
}

// buildSystemMessage 构建包含世界观 + 用户档案 + 历史摘要 + RAG 知识的系统消息。
// assess=true 时加载含 eval_json 指令的评估节；assess=false 时跳过，保持纯对话。
func (a *Agent) buildSystemMessage(ctx context.Context, userInput string, assess bool) (string, error) {
	// 世界观系统提示（透传人格ID，加载对应语料）
	worldviewPrompt, err := a.loader.BuildSystemPrompt(a.cfg.ActivePath, a.cfg.PersonaID, assess)
	if err != nil {
		return "", err
	}

	// 用户修行档案
	profile, err := a.repo.GetOrCreateProfile(ctx, a.cfg.UserID)
	if err != nil {
		return "", fmt.Errorf("get profile: %w", err)
	}

	// 最近 N 次会话摘要
	sessions, err := a.repo.GetRecentSessions(ctx, a.cfg.UserID, a.cfg.HistoryWindow)
	if err != nil {
		return "", fmt.Errorf("get sessions: %w", err)
	}

	// 工具掌握度
	toolMastery, err := a.repo.GetToolMastery(ctx, a.cfg.UserID)
	if err != nil {
		return "", fmt.Errorf("get tool mastery: %w", err)
	}

	contextBlock := buildContextBlock(a.cfg.UserName, profile, sessions, toolMastery)

	// 知识库笔记摘要：让护道人始终了解用户已学习过的主题
	knowledgeFiles, _ := a.repo.ListKnowledgeFiles(ctx, a.cfg.UserID)
	knowledgeSummaryBlock := buildKnowledgeSummaryBlock(knowledgeFiles)

	// RAG 知识检索（仅当 Milvus 启用且用户输入非空时）
	// 使用独立的 5 秒超时 context，避免 Milvus/embedding 无响应时阻塞对话
	ragBlock := ""
	if a.cfg.KnowledgeCfg.Enabled && userInput != "" {
		ragCtx, ragCancel := context.WithTimeout(ctx, 5*time.Second)
		defer ragCancel()
		chunks, err := a.retriever.Search(ragCtx, userInput, 3)
		if err != nil {
			// RAG 失败不阻塞对话，仅打印警告
			fmt.Printf("[警告] 知识库检索失败: %v\n", err)
		} else if len(chunks) > 0 {
			ragBlock = buildRAGBlock(chunks)
		}
	}

	// 历史能力证据块（仅评估模式注入，帮助护道人跨对话追踪能力积累）
	evidenceBlock := ""
	if assess {
		evidenceItems, evErr := a.repo.GetRecentEvidence(ctx, a.cfg.UserID, 20)
		if evErr != nil {
			fmt.Printf("[警告] 能力证据读取失败: %v\n", evErr)
		} else if len(evidenceItems) > 0 {
			evidenceBlock = buildEvidenceBlock(evidenceItems)
		}
	}

	// 待完成任务块：始终注入，护道人负责追问进展和验收
	taskBlock := ""
	pendingTasks, taskErr := a.repo.GetPendingTasks(ctx, a.cfg.UserID)
	if taskErr != nil {
		fmt.Printf("[警告] 任务读取失败: %v\n", taskErr)
	} else if len(pendingTasks) > 0 {
		taskBlock = buildTaskBlock(pendingTasks)
	}

	parts := []string{worldviewPrompt, contextBlock}
	if knowledgeSummaryBlock != "" {
		parts = append(parts, knowledgeSummaryBlock)
	}
	if ragBlock != "" {
		parts = append(parts, ragBlock)
	}
	if evidenceBlock != "" {
		parts = append(parts, evidenceBlock)
	}
	if taskBlock != "" {
		parts = append(parts, taskBlock)
	}
	return strings.Join(parts, "\n\n"), nil
}

// chatStreamWithTools 实现带 browse_web 工具的 agentic loop（仅 Mode B 调用）。
//
// 流式优先架构：
// 第一步：以流式方式发出请求（toolModel.Stream）。
//   - 若模型输出 content：直接流式输出（应用 Mode B eval_json 抑制），无额外 LLM 调用。
//   - 若模型输出 tool_calls delta：静默累积，流结束后执行工具，进入第二步。
// 第二步（仅有工具调用时）：将工具结果注入消息链，再次流式输出最终回复。
func (a *Agent) chatStreamWithTools(
	ctx context.Context,
	messages []*schema.Message,
	history []*schema.Message,
	userInput string,
	onToken func(string),
) (string, []*schema.Message, error) {

	stream, err := a.toolModel.Stream(ctx, messages)
	if err != nil {
		return "", nil, fmt.Errorf("llm stream (tool): %w", err)
	}
	defer stream.Close()

	// Mode B eval_json 抑制状态（文本路径）
	var textBuf strings.Builder
	textSuppressed := false
	const evalTrig = "<eval_json>"
	const evalWin = len(evalTrig) - 1
	var fullTextReply strings.Builder

	// Tool call 增量累积（tool call 路径）
	type tcAccum struct {
		id, typ, name string
		argsBuf       strings.Builder
	}
	tcAccumMap := map[int]*tcAccum{}
	isToolMode := false

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, fmt.Errorf("stream recv (tool first): %w", err)
		}

		// 累积 tool call delta（增量 JSON 参数）
		for _, tc := range chunk.ToolCalls {
			isToolMode = true
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			acc := tcAccumMap[idx]
			if acc == nil {
				acc = &tcAccum{}
				tcAccumMap[idx] = acc
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Type != "" {
				acc.typ = tc.Type
			}
			if tc.Function.Name != "" {
				acc.name = tc.Function.Name
			}
			acc.argsBuf.WriteString(tc.Function.Arguments)
		}

		// 文本路径：仅在非 tool mode 时流式输出，同时抑制 eval_json
		if !isToolMode && chunk.Content != "" {
			if textSuppressed {
				continue
			}
			textBuf.WriteString(chunk.Content)
			s := textBuf.String()
			if idx := strings.Index(s, evalTrig); idx >= 0 {
				textSuppressed = true
				printUntil := idx
				if nl := strings.LastIndex(s[:idx], "\n"); nl >= 0 {
					printUntil = nl + 1
				}
				visible := s[:printUntil]
				if visible != "" {
					onToken(visible)
					fullTextReply.WriteString(visible)
				}
				textBuf.Reset()
				continue
			}
			safe := len(s) - evalWin
			if safe > 0 {
				onToken(s[:safe])
				fullTextReply.WriteString(s[:safe])
				tail := s[safe:]
				textBuf.Reset()
				textBuf.WriteString(tail)
			}
		}
	}

	// 刷出文本路径缓冲区尾部
	if !isToolMode && !textSuppressed && textBuf.Len() > 0 {
		onToken(textBuf.String())
		fullTextReply.WriteString(textBuf.String())
	}

	// ── 文本路径：已流式输出完成 ──
	if !isToolMode {
		reply := strings.TrimSpace(fullTextReply.String())
		newHistory := append(history,
			schema.UserMessage(userInput),
			schema.AssistantMessage(reply, nil),
		)
		return reply, newHistory, nil
	}

	// ── 工具调用路径：重建完整 ToolCalls，执行工具，第二步流式输出 ──
	onToken("\n[护道人正在查阅资料...]\n\n")

	toolCalls := make([]schema.ToolCall, 0, len(tcAccumMap))
	for i := 0; i < len(tcAccumMap); i++ {
		acc := tcAccumMap[i]
		toolCalls = append(toolCalls, schema.ToolCall{
			ID:   acc.id,
			Type: acc.typ,
			Function: schema.FunctionCall{
				Name:      acc.name,
				Arguments: acc.argsBuf.String(),
			},
		})
	}

	assistantMsg := schema.AssistantMessage("", toolCalls)
	toolMessages := make([]*schema.Message, 0, len(messages)+1+len(toolCalls))
	toolMessages = append(toolMessages, messages...)
	toolMessages = append(toolMessages, assistantMsg)

	for _, tc := range toolCalls {
		var toolResult string
		switch tc.Function.Name {
		case "browse_web":
			u := extractJSONString(tc.Function.Arguments, "url")
			if u == "" {
				toolResult = "错误：url 参数为空"
			} else {
				content, fetchErr := fetchWebContent(u)
				if fetchErr != nil {
					toolResult = fmt.Sprintf("网页抓取失败: %v", fetchErr)
				} else {
					toolResult = content
				}
			}
		case "search_web":
			query := extractJSONString(tc.Function.Arguments, "query")
			if query == "" {
				toolResult = "错误：query 参数为空"
			} else {
				result, searchErr := searchWeb(query, a.cfg.BraveAPIKey)
				if searchErr != nil {
					toolResult = fmt.Sprintf("搜索失败: %v", searchErr)
				} else {
					toolResult = result
				}
			}
		default:
			toolResult = fmt.Sprintf("未知工具: %s", tc.Function.Name)
		}
		toolMessages = append(toolMessages, schema.ToolMessage(toolResult, tc.ID))
	}

	// 第二步：流式输出含工具结果的最终回复（同样抑制 eval_json）
	stream2, err := a.toolModel.Stream(ctx, toolMessages)
	if err != nil {
		return "", nil, fmt.Errorf("llm stream (after tool): %w", err)
	}
	defer stream2.Close()

	var fullReply strings.Builder
	var s2Buf strings.Builder
	s2Suppressed := false

	for {
		chunk, err := stream2.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, fmt.Errorf("stream recv (after tool): %w", err)
		}
		if chunk.Content == "" || s2Suppressed {
			continue
		}
		s2Buf.WriteString(chunk.Content)
		s := s2Buf.String()
		if idx := strings.Index(s, evalTrig); idx >= 0 {
			s2Suppressed = true
			printUntil := idx
			if nl := strings.LastIndex(s[:idx], "\n"); nl >= 0 {
				printUntil = nl + 1
			}
			visible := s[:printUntil]
			if visible != "" {
				onToken(visible)
				fullReply.WriteString(visible)
			}
			s2Buf.Reset()
			continue
		}
		safe := len(s) - evalWin
		if safe > 0 {
			onToken(s[:safe])
			fullReply.WriteString(s[:safe])
			s2Buf.Reset()
			s2Buf.WriteString(s[safe:])
		}
	}
	if !s2Suppressed && s2Buf.Len() > 0 {
		onToken(s2Buf.String())
		fullReply.WriteString(s2Buf.String())
	}

	reply := fullReply.String()
	newHistory := append(history,
		schema.UserMessage(userInput),
		schema.AssistantMessage(reply, nil),
	)
	return reply, newHistory, nil
}

// extractJSONString 从简单 JSON 对象字符串中提取指定 key 的字符串值。
// 仅用于解析 tool call arguments，不依赖 encoding/json 以减少开销。
func extractJSONString(jsonStr, key string) string {
	needle := `"` + key + `"`
	idx := strings.Index(jsonStr, needle)
	if idx < 0 {
		return ""
	}
	rest := jsonStr[idx+len(needle):]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// buildMessages 组装 [system, ...history, user] 消息列表。
func buildMessages(systemPrompt string, history []*schema.Message, userInput string) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(history)+2)
	msgs = append(msgs, schema.SystemMessage(systemPrompt))
	msgs = append(msgs, history...)
	msgs = append(msgs, schema.UserMessage(userInput))
	return msgs
}

// newChatModel 根据配置创建 Eino ChatModel。
func newChatModel(ctx context.Context, cfg Config) (model.ChatModel, error) {
	maxTokens := 4096
	baseURL := cfg.BaseURL

	openaiCfg := &openaimodel.ChatModelConfig{
		Model:     cfg.Model,
		APIKey:    cfg.APIKey,
		MaxTokens: &maxTokens,
	}
	if baseURL != "" {
		openaiCfg.BaseURL = baseURL
	}
	return openaimodel.NewChatModel(ctx, openaiCfg)
}
