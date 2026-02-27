package guardian

import (
	"context"
	"fmt"

	"github.com/daiyh/lizhu/internal/memory/episodic"
)

// persistEvaluation 解析 LLM 响应中的评估 JSON，
// 更新修行档案、保存会话记录、更新法器谱。
func (a *Agent) persistEvaluation(ctx context.Context, response, userInput string) error {
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

	// 仅更新非零字段（允许本次对话未涉及某路径时保留原始值）
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

	// 2. 保存会话记录
	session := &episodic.Session{
		UserID:          a.cfg.UserID,
		Summary:         eval.SessionSummary,
		GoLianqiScore:   eval.GoLianqiScore,
		AILianqiScore:   eval.AILianqiScore,
		WufuScore:       eval.WufuScore,
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
			Category:  u.Category,
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
