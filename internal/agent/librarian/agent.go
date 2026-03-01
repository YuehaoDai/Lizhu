package librarian

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Config 知识整理官配置。
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

// Agent 知识整理官智能体。
type Agent struct {
	cfg   Config
	model model.ChatModel
}

// New 创建知识整理官 Agent。
func New(ctx context.Context, cfg Config) (*Agent, error) {
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	maxTokens := 1024
	openaiCfg := &openaimodel.ChatModelConfig{
		Model:     cfg.Model,
		APIKey:    cfg.APIKey,
		MaxTokens: &maxTokens,
	}
	if cfg.BaseURL != "" {
		openaiCfg.BaseURL = cfg.BaseURL
	}
	m, err := openaimodel.NewChatModel(ctx, openaiCfg)
	if err != nil {
		return nil, fmt.Errorf("librarian: init model: %w", err)
	}
	return &Agent{cfg: cfg, model: m}, nil
}

// Summarize 让 LLM 提炼笔记要点，返回结构化摘要文本。
// filePath 用于获取文件名显示在摘要中；content 为文件全文。
func (a *Agent) Summarize(ctx context.Context, filePath, content string) (string, error) {
	// 截断过长内容（避免 token 溢出），保留前 6000 字符
	if len(content) > 6000 {
		content = content[:6000] + "\n\n[...内容过长，已截断...]"
	}

	fileName := filepath.Base(filePath)
	userMsg := buildSummarizePrompt(fileName, content)

	msgs := []*schema.Message{
		schema.SystemMessage(systemPromptTemplate),
		schema.UserMessage(userMsg),
	}

	resp, err := a.model.Generate(ctx, msgs)
	if err != nil {
		return "", fmt.Errorf("librarian summarize: %w", err)
	}
	return strings.TrimSpace(resp.Content), nil
}

// SummarizeSession 对完整多轮护道对话生成简短摘要，用于会话历史展示。
// userName 为修行者名字，conversation 为格式化后的完整对话文本。
func (a *Agent) SummarizeSession(ctx context.Context, userName, conversation string) (string, error) {
	// 截断过长内容防止 token 溢出，保留前 4000 字符
	if len(conversation) > 4000 {
		conversation = conversation[:4000] + "……[对话过长，已截断]"
	}

	msgs := []*schema.Message{
		schema.SystemMessage(sessionSummarizePrompt),
		schema.UserMessage(buildSessionSummarizePrompt(userName, conversation)),
	}

	resp, err := a.model.Generate(ctx, msgs)
	if err != nil {
		return "", fmt.Errorf("librarian summarize session: %w", err)
	}
	return strings.TrimSpace(resp.Content), nil
}
