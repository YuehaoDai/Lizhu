package librarian

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// RawEvidence 是 Librarian 从对话中提炼的单条能力证据（轻量中间结构体）。
type RawEvidence struct {
	Category   string `json:"category"`
	Tool       string `json:"tool"`
	Evidence   string `json:"evidence"`
	Confidence int    `json:"confidence"`
}

// RawTask 是 Librarian 生成的单条修炼任务（轻量中间结构体）。
type RawTask struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Category           string `json:"category"`
	SourceEvidence     string `json:"source_evidence"`
	TargetScoreHint    int    `json:"target_score_hint"`
}

// VerifyResult 是 Librarian 对任务验收的结果。
type VerifyResult struct {
	Passed   bool   `json:"passed"`
	Feedback string `json:"feedback"`
}

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

// ExtractEvidence 从完整多轮对话文本中提炼结构化能力证据条目。
func (a *Agent) ExtractEvidence(ctx context.Context, userName, conversation string) ([]*RawEvidence, error) {
	// 截断过长内容防止 token 溢出，保留前 5000 字符
	if len(conversation) > 5000 {
		conversation = conversation[:5000] + "……[对话过长，已截断]"
	}

	msgs := []*schema.Message{
		schema.SystemMessage(evidenceExtractPrompt),
		schema.UserMessage(buildEvidenceExtractPrompt(userName, conversation)),
	}

	resp, err := a.model.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("librarian extract evidence: %w", err)
	}

	raw := strings.TrimSpace(resp.Content)
	// 去掉可能的 markdown 代码块包裹
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		var inner []string
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				continue
			}
			inner = append(inner, line)
		}
		raw = strings.Join(inner, "\n")
	}

	var items []*RawEvidence
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("librarian extract evidence: parse json: %w (raw: %s)", err, raw)
	}
	return items, nil
}

// ExtractTasks 根据评估对话和当前档案，生成 1-2 条具体可执行的修炼任务。
// pendingCount 为当前未完成任务数量，超过 2 个时 LLM 会返回空数组。
// recentEvidence 为最近能力证据条目（反映最近在学什么），sessionSummaries 为近期会话摘要。
func (a *Agent) ExtractTasks(ctx context.Context, userName, conversation, profileSummary string, pendingCount int, recentEvidence, sessionSummaries string) ([]*RawTask, error) {
	if len(conversation) > 5000 {
		conversation = conversation[:5000] + "……[对话过长，已截断]"
	}

	msgs := []*schema.Message{
		schema.SystemMessage(taskExtractPrompt),
		schema.UserMessage(buildTaskExtractPrompt(userName, conversation, profileSummary, pendingCount, recentEvidence, sessionSummaries)),
	}

	resp, err := a.model.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("librarian extract tasks: %w", err)
	}

	raw := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		var inner []string
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				continue
			}
			inner = append(inner, line)
		}
		raw = strings.Join(inner, "\n")
	}

	var tasks []*RawTask
	if err := json.Unmarshal([]byte(raw), &tasks); err != nil {
		return nil, fmt.Errorf("librarian extract tasks: parse json: %w (raw: %s)", err, raw)
	}
	return tasks, nil
}

// VerifyTask 验收修行者对某条任务的完成汇报。
func (a *Agent) VerifyTask(ctx context.Context, taskTitle, criteria, userReport string) (*VerifyResult, error) {
	msgs := []*schema.Message{
		schema.SystemMessage(taskVerifyPrompt),
		schema.UserMessage(buildTaskVerifyPrompt(taskTitle, criteria, userReport)),
	}

	resp, err := a.model.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("librarian verify task: %w", err)
	}

	raw := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		var inner []string
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				continue
			}
			inner = append(inner, line)
		}
		raw = strings.Join(inner, "\n")
	}

	var result VerifyResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("librarian verify task: parse json: %w (raw: %s)", err, raw)
	}
	return &result, nil
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
