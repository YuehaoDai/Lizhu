package guardian

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/YuehaoDai/lizhu/internal/knowledge"
	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
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
	sb.WriteString(fmt.Sprintf("修行者名字：%s\n", userName))
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

	// 法宝库（前10项）
	if len(toolMastery) > 0 {
		sb.WriteString("\n当前法宝库（部分）：\n")
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

// buildKnowledgeSummaryBlock 将用户已入库的笔记摘要注入上下文，
// 让护道人始终了解用户系统性学习过哪些主题，即使当前对话未涉及。
func buildKnowledgeSummaryBlock(files []*episodic.KnowledgeFile) string {
	if len(files) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("========================\n")
	sb.WriteString("【修行者知识库（已索引笔记，代表修行者已系统学习的主题）】\n")
	sb.WriteString("========================\n\n")
	for _, f := range files {
		name := filepath.Base(f.FilePath)
		if f.Summary != "" {
			sb.WriteString(fmt.Sprintf("- %s：%s\n", name, f.Summary))
		} else {
			sb.WriteString(fmt.Sprintf("- %s（暂无摘要）\n", name))
		}
	}
	sb.WriteString("\n以上为修行者主动整理入库的学习笔记，评估时应纳入参考，说明其对这些主题有过系统学习。\n")
	return sb.String()
}

// buildEvidenceBlock 将历史能力证据条目格式化为系统提示的注入块。
// 仅在 assess=true 时调用，帮助护道人综合评估跨对话积累的能力。
func buildEvidenceBlock(items []*episodic.EvidenceItem) string {
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("========================\n")
	sb.WriteString("【历史能力证据库（跨对话积累，评估时优先参考）】\n")
	sb.WriteString("========================\n")
	for _, item := range items {
		date := item.CreatedAt.Format("01-02")
		toolPart := ""
		if item.Tool != "" {
			toolPart = fmt.Sprintf("[%s/%s]", item.Category, item.Tool)
		} else {
			toolPart = fmt.Sprintf("[%s]", item.Category)
		}
		sb.WriteString(fmt.Sprintf("[%s] %s %s（置信度%d/5）\n", date, toolPart, item.Evidence, item.Confidence))
	}
	sb.WriteString("以上为历史对话中提炼的客观能力证据，应与本次对话内容综合评估。\n")
	return sb.String()
}

// buildRAGBlock 将检索到的知识块格式化为系统提示中的参考资料节。
func buildRAGBlock(chunks []knowledge.SearchResult) string {
	var sb strings.Builder
	sb.WriteString("========================\n")
	sb.WriteString("【修行参考资料（来自知识库，仅供参考）】\n")
	sb.WriteString("========================\n\n")
	for i, c := range chunks {
		sb.WriteString(fmt.Sprintf("参考资料 %d（来源：%s）：\n%s\n\n", i+1, c.FilePath, c.Text))
	}
	sb.WriteString("以上为从知识库检索到的相关内容，请结合修行者当前描述综合评估。\n")
	return sb.String()
}
