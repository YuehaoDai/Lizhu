// Package guardian 实现骊珠护道人 Agent。
// 基于 Eino ChatModel，构建"世界观系统Prompt + 历史记忆 + RAG知识"的完整上下文，
// 调用 LLM 生成结构化修行评估，并将结果持久化到 PostgreSQL。
package guardian

import (
	"context"
	"fmt"
	"io"
	"strings"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	"github.com/YuehaoDai/lizhu/internal/worldview"
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
}

// Agent 护道人智能体。
type Agent struct {
	cfg    Config
	model  model.ChatModel
	loader *worldview.Loader
	repo   *episodic.Repository
}

// PersonaName 返回护道人显示名称（空字符串表示使用默认"护道人"）。
func (a *Agent) PersonaName() string { return a.cfg.PersonaName }

// New 创建护道人 Agent。
func New(ctx context.Context, cfg Config, repo *episodic.Repository) (*Agent, error) {
	m, err := newChatModel(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("guardian: init model: %w", err)
	}
	return &Agent{
		cfg:    cfg,
		model:  m,
		loader: worldview.NewLoader(cfg.WorldViewDir),
		repo:   repo,
	}, nil
}

// Chat 处理一轮对话（非流式）：构建上下文 → 调用 LLM → 解析结果 → 持久化。
func (a *Agent) Chat(ctx context.Context, history []*schema.Message, userInput string) (string, []*schema.Message, error) {
	systemMsg, err := a.buildSystemMessage(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("build system message: %w", err)
	}
	messages := buildMessages(systemMsg, history, userInput)

	resp, err := a.model.Generate(ctx, messages)
	if err != nil {
		return "", nil, fmt.Errorf("llm generate: %w", err)
	}
	reply := resp.Content

	if err := a.persistEvaluation(ctx, reply, userInput); err != nil {
		fmt.Printf("[警告] 修行档案持久化失败: %v\n", err)
	}

	newHistory := append(history,
		schema.UserMessage(userInput),
		schema.AssistantMessage(reply, nil),
	)
	return reply, newHistory, nil
}

// ChatStream 处理一轮对话（流式）：逐 token 回调 onToken，返回完整回复与新历史。
func (a *Agent) ChatStream(ctx context.Context, history []*schema.Message, userInput string, onToken func(string)) (string, []*schema.Message, error) {
	systemMsg, err := a.buildSystemMessage(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("build system message: %w", err)
	}
	messages := buildMessages(systemMsg, history, userInput)

	stream, err := a.model.Stream(ctx, messages)
	if err != nil {
		return "", nil, fmt.Errorf("llm stream: %w", err)
	}
	defer stream.Close()

	var fullReply strings.Builder
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, fmt.Errorf("stream recv: %w", err)
		}
		token := chunk.Content
		if token != "" {
			onToken(token)
			fullReply.WriteString(token)
		}
	}

	reply := fullReply.String()

	if err := a.persistEvaluation(ctx, reply, userInput); err != nil {
		fmt.Printf("[警告] 修行档案持久化失败: %v\n", err)
	}

	newHistory := append(history,
		schema.UserMessage(userInput),
		schema.AssistantMessage(reply, nil),
	)
	return reply, newHistory, nil
}

// buildSystemMessage 构建包含世界观 + 用户档案 + 历史摘要的系统消息。
func (a *Agent) buildSystemMessage(ctx context.Context) (string, error) {
	// 世界观系统提示（透传人格ID，加载对应语料）
	worldviewPrompt, err := a.loader.BuildSystemPrompt(a.cfg.ActivePath, a.cfg.PersonaID)
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

	return worldviewPrompt + "\n\n" + contextBlock, nil
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
