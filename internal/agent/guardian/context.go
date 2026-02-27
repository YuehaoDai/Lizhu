package guardian

import (
	"fmt"
	"strings"

	"github.com/daiyh/lizhu/internal/memory/episodic"
)

// buildContextBlock 将用户档案、历史摘要、工具掌握度
// 格式化为注入系统消息的上下文块。
func buildContextBlock(
	userName string,
	profile *episodic.Profile,
	sessions []*episodic.Session,
	toolMastery []*episodic.ToolMastery,
) string {
	var sb strings.Builder

	sb.WriteString("========================\n")
	sb.WriteString("【修行者当前档案（系统注入，请据此评估）】\n")
	sb.WriteString("========================\n\n")

	// 修行者信息
	sb.WriteString(fmt.Sprintf("修行者道号：%s\n", userName))
	sb.WriteString(fmt.Sprintf("主修路径：%s\n\n", profile.ActivePath))

	// Go练气士状态
	if profile.GoLianqiScore > 0 || profile.ActivePath == "go" || profile.ActivePath == "both" {
		sb.WriteString(fmt.Sprintf("Go开发练气士：%d/100，第%d境·%s，主修分支：%s\n",
			profile.GoLianqiScore,
			profile.GoLianqiLevel,
			profile.GoLianqiLevelName,
			ifEmpty(profile.GoLianqiBranch, "待定"),
		))
	}

	// AI练气士状态
	if profile.AILianqiScore > 0 || profile.ActivePath == "ai" || profile.ActivePath == "both" {
		sb.WriteString(fmt.Sprintf("AI应用练气士：%d/100，第%d境·%s，主修分支：%s\n",
			profile.AILianqiScore,
			profile.AILianqiLevel,
			profile.AILianqiLevelName,
			ifEmpty(profile.AILianqiBranch, "待定"),
		))
	}

	// 武夫状态
	sb.WriteString(fmt.Sprintf("武夫：%d/100，第%d境·%s\n",
		profile.WufuScore,
		profile.WufuLevel,
		profile.WufuLevelName,
	))

	// 已记录的心魔
	if len(profile.XinMoRecords) > 0 {
		sb.WriteString(fmt.Sprintf("\n已记录心魔：%s\n", strings.Join(profile.XinMoRecords, "、")))
	}

	// 历史会话摘要
	if len(sessions) > 0 {
		sb.WriteString("\n近期修行记录：\n")
		for i, s := range sessions {
			sb.WriteString(fmt.Sprintf("  第%d次：%s\n", i+1, s.Summary))
		}
	}

	// 法器谱（前10项）
	if len(toolMastery) > 0 {
		sb.WriteString("\n当前法器谱（部分）：\n")
		limit := 10
		if len(toolMastery) < limit {
			limit = len(toolMastery)
		}
		for _, tm := range toolMastery[:limit] {
			sb.WriteString(fmt.Sprintf("  %s：%d/100·%s\n", tm.ToolName, tm.Score, tm.LevelName))
		}
	}

	sb.WriteString("\n以上为修行者历史档案，请结合本次对话内容进行综合评估。\n")
	return sb.String()
}

func ifEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
