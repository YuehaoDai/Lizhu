package guardian

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	"github.com/cloudwego/eino/schema"
)

// persistEvaluation 解析 LLM 响应中的评估 JSON，
// 更新修行档案、保存会话记录、更新法器谱。
func (a *Agent) persistEvaluation(ctx context.Context, response string) error {
	eval, err := ParseEvalResult(response)
	if err != nil {
		return fmt.Errorf("parse eval json: %w", err)
	}
	if eval == nil {
		// LLM 未输出评估块，跳过持久化
		return nil
	}

	// 1. 更新修行档案
	profile, err := a.repo.GetOrCreateProfile(ctx, a.cfg.UserID)
	if err != nil {
		return err
	}

	// 仅更新非零且非 -1 的字段（-1 表示模式 B "本次未更新"，保留历史档案值）
	if eval.GoLianqiScore > 0 {
		profile.GoLianqiScore = eval.GoLianqiScore
		profile.GoLianqiLevel = eval.GoLianqiLevel
		profile.GoLianqiLevelName = eval.GoLianqiLevelName
		if eval.GoLianqiBranch != "" {
			profile.GoLianqiBranch = eval.GoLianqiBranch
		}
	}
	if eval.AILianqiScore > 0 {
		profile.AILianqiScore = eval.AILianqiScore
		profile.AILianqiLevel = eval.AILianqiLevel
		profile.AILianqiLevelName = eval.AILianqiLevelName
		if eval.AILianqiBranch != "" {
			profile.AILianqiBranch = eval.AILianqiBranch
		}
	}
	if eval.WufuScore > 0 {
		profile.WufuScore = eval.WufuScore
		profile.WufuLevel = eval.WufuLevel
		profile.WufuLevelName = eval.WufuLevelName
	}

	// 合并心魔记录（去重）
	profile.XinMoRecords = mergeUnique(profile.XinMoRecords, eval.XinMoIdentified)

	if err := a.repo.UpdateProfile(ctx, profile); err != nil {
		return fmt.Errorf("update profile: %w", err)
	}

	// 2. 保存会话记录（-1 表示本次未更新，存入会话时归零）
	session := &episodic.Session{
		UserID:          a.cfg.UserID,
		Summary:         eval.SessionSummary,
		GoLianqiScore:   zeroIfNeg(eval.GoLianqiScore),
		AILianqiScore:   zeroIfNeg(eval.AILianqiScore),
		WufuScore:       zeroIfNeg(eval.WufuScore),
		XinMoIdentified: eval.XinMoIdentified,
		RawResponse:     response,
	}
	if err := a.repo.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	// 3. 更新法器谱
	for _, u := range eval.ToolMasteryUpdates {
		if u.Tool == "" || u.Score <= 0 {
			continue
		}
		tm := &episodic.ToolMastery{
			UserID:    a.cfg.UserID,
			ToolName:  u.Tool,
			Category:  normCategory(u.Category),
			Score:     u.Score,
			LevelName: episodic.ScoreToLevel(u.Score),
			Evidence:  u.Evidence,
		}
		if err := a.repo.UpsertToolMastery(ctx, tm); err != nil {
			return fmt.Errorf("upsert tool mastery %q: %w", u.Tool, err)
		}
	}
	return nil
}

// PersistResult 保存整次会话后返回的结果摘要，用于退出时展示进度。
type PersistResult struct {
	SummaryGenerated bool
	EvidenceCount    int
}

// PersistFullSession 在整次 chat 会话结束后保存会话概要记录并提炼能力证据条目。
// history 为本次 chat 从开始到结束的完整多轮消息列表。
func (a *Agent) PersistFullSession(ctx context.Context, history []*schema.Message) (PersistResult, error) {
	if len(history) == 0 {
		return PersistResult{}, nil
	}

	// 将完整 history 拼装为可读对话文本
	var conv strings.Builder
	for _, msg := range history {
		switch msg.Role {
		case schema.User:
			conv.WriteString("【修行者】\n" + msg.Content + "\n\n")
		case schema.Assistant:
			conv.WriteString("【护道人】\n" + msg.Content + "\n\n")
		}
	}
	conversation := strings.TrimSpace(conv.String())

	// 取最后一条 Assistant 消息作为降级 rawResponse
	lastReply := ""
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == schema.Assistant {
			lastReply = history[i].Content
			break
		}
	}

	var result PersistResult
	var summary string

	// 优先调用 Librarian 生成整个会话的语义摘要
	if a.librarian != nil {
		sumCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		s, err := a.librarian.SummarizeSession(sumCtx, a.cfg.UserName, conversation)
		if err == nil && s != "" {
			summary = s
			result.SummaryGenerated = true
		} else {
			fmt.Printf("[警告] 会话摘要生成失败，回退到截断文本: %v\n", err)
		}
	}

	// 降级：Librarian 不可用时截取最后一条回复前 80 字
	if summary == "" {
		runes := []rune(lastReply)
		summary = string(runes)
		if len(runes) > 80 {
			summary = string(runes[:80]) + "……"
		}
	}

	session := &episodic.Session{
		UserID:          a.cfg.UserID,
		Summary:         summary,
		XinMoIdentified: []string{},
		RawResponse:     lastReply,
	}
	if err := a.repo.SaveSession(ctx, session); err != nil {
		return result, fmt.Errorf("save session: %w", err)
	}

	// 提炼能力证据条目（独立 timeout，失败不影响摘要保存）
	if a.librarian != nil {
		evCtx, evCancel := context.WithTimeout(ctx, 15*time.Second)
		defer evCancel()
		rawItems, err := a.librarian.ExtractEvidence(evCtx, a.cfg.UserName, conversation)
		if err != nil {
			fmt.Printf("[警告] 能力证据提炼失败: %v\n", err)
		} else {
			var items []*episodic.EvidenceItem
			for _, r := range rawItems {
				items = append(items, &episodic.EvidenceItem{
					UserID:     a.cfg.UserID,
					Category:   r.Category,
					Tool:       r.Tool,
					Evidence:   r.Evidence,
					Confidence: r.Confidence,
				})
			}
			if saveErr := a.repo.SaveEvidenceItems(ctx, items); saveErr != nil {
				fmt.Printf("[警告] 能力证据保存失败: %v\n", saveErr)
			} else {
				result.EvidenceCount = len(items)
			}
		}
	}

	return result, nil
}

// normCategory 将 LLM 可能输出的中/英文类别名统一映射为 status.go 使用的英文 key。
func normCategory(c string) string {
	switch c {
	case "本命飞剑", "primary_weapon":
		return "primary_weapon"
	case "绘卷", "juanjuan":
		return "juanjuan"
	case "符箓", "fulu":
		return "fulu"
	case "方寸物", "fangcun":
		return "fangcun"
	case "护山大阵", "zhenfa":
		return "zhenfa"
	case "灵宠", "AI法器", "ai_tool", "linchong":
		return "linchong"
	case "观星镜", "telescope":
		return "telescope"
	case "法家戒尺", "quality":
		return "quality"
	case "三教修为", "philosophy":
		return "philosophy" // 已移出法器谱，保留映射避免旧数据报错
	default:
		return c
	}
}

// zeroIfNeg 将负数归零，用于过滤模式 B 中填写的 -1 占位值。
func zeroIfNeg(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// mergeUnique 合并两个字符串切片，结果去重。
func mergeUnique(existing, newItems []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(newItems))
	for _, s := range existing {
		seen[s] = struct{}{}
	}
	result := append([]string{}, existing...)
	for _, s := range newItems {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	return result
}
