package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看修行档案与法器谱",
	Long:  `展示当前修行者的完整档案，包括双轨练气士境界、武夫境界、法器谱及近期心魔记录。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(cmd.Context())
	},
}

func runStatus(ctx context.Context) error {
	profile, err := repo.GetOrCreateProfile(ctx, "default")
	if err != nil {
		return fmt.Errorf("读取档案失败: %w", err)
	}

	toolMastery, err := repo.GetToolMastery(ctx, "default")
	if err != nil {
		return fmt.Errorf("读取法器谱失败: %w", err)
	}

	sessions, err := repo.GetRecentSessions(ctx, "default", 3)
	if err != nil {
		return fmt.Errorf("读取会话记录失败: %w", err)
	}

	printFullProfile(profile, toolMastery, sessions)
	return nil
}

// printProfileSummary 打印简洁的修行档案摘要（用于 chat 内联显示）。
func printProfileSummary(p *episodic.Profile) {
	fmt.Println()
	fmt.Println("── 修行档案 ──────────────────────────────────────")
	if p.GoLianqiScore > 0 || p.ActivePath == "go" || p.ActivePath == "both" {
		fmt.Printf("  Go练气士  : %d/100  第%d境·%s  分支：%s\n",
			p.GoLianqiScore, p.GoLianqiLevel, p.GoLianqiLevelName,
			ifEmptyStr(p.GoLianqiBranch, "待定"))
	}
	if p.AILianqiScore > 0 || p.ActivePath == "ai" || p.ActivePath == "both" {
		fmt.Printf("  AI练气士  : %d/100  第%d境·%s  分支：%s\n",
			p.AILianqiScore, p.AILianqiLevel, p.AILianqiLevelName,
			ifEmptyStr(p.AILianqiBranch, "待定"))
	}
	fmt.Printf("  武夫      : %d/100  第%d境·%s\n",
		p.WufuScore, p.WufuLevel, p.WufuLevelName)
	fmt.Println("───────────────────────────────────────────────────")
}

// printFullProfile 打印完整档案（status 命令使用）。
func printFullProfile(p *episodic.Profile, tools []*episodic.ToolMastery, sessions []*episodic.Session) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║              骊珠 · 修行档案                       ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")

	fmt.Println("\n【修行境界】")
	if p.GoLianqiScore > 0 || p.ActivePath == "go" || p.ActivePath == "both" {
		fmt.Printf("  Go开发练气士  %s %d/100\n",
			renderBar(p.GoLianqiScore), p.GoLianqiScore)
		fmt.Printf("  第%d境·%-8s  主修分支：%s\n",
			p.GoLianqiLevel, p.GoLianqiLevelName, ifEmptyStr(p.GoLianqiBranch, "待定"))
	}
	if p.AILianqiScore > 0 || p.ActivePath == "ai" || p.ActivePath == "both" {
		fmt.Printf("\n  AI应用练气士  %s %d/100\n",
			renderBar(p.AILianqiScore), p.AILianqiScore)
		fmt.Printf("  第%d境·%-8s  主修分支：%s\n",
			p.AILianqiLevel, p.AILianqiLevelName, ifEmptyStr(p.AILianqiBranch, "待定"))
	}
	fmt.Printf("\n  武夫          %s %d/100\n",
		renderBar(p.WufuScore), p.WufuScore)
	fmt.Printf("  第%d境·%s\n", p.WufuLevel, p.WufuLevelName)

	// 法器谱
	if len(tools) > 0 {
		fmt.Println("\n【法器谱】")
		groupedTools := groupByCategory(tools)
		categories := []string{"primary_weapon", "ai_tool", "fulu", "zhenfa", "telescope", "quality", "philosophy"}
		catNames := map[string]string{
			"primary_weapon": "本命法宝",
			"ai_tool":        "AI法器",
			"fulu":           "符箓",
			"zhenfa":         "护山大阵",
			"telescope":      "观星镜",
			"quality":        "法家戒尺",
			"philosophy":     "三教修为",
		}
		for _, cat := range categories {
			items, ok := groupedTools[cat]
			if !ok || len(items) == 0 {
				continue
			}
			fmt.Printf("  [%s]\n", catNames[cat])
			for _, t := range items {
				fmt.Printf("    %-20s %s %-4d  %s\n",
					t.ToolName, renderBar(t.Score), t.Score, t.LevelName)
			}
		}
	}

	// 心魔记录
	if len(p.XinMoRecords) > 0 {
		fmt.Println("\n【已记录心魔】")
		for _, xm := range p.XinMoRecords {
			fmt.Printf("  ✗ %s\n", xm)
		}
	}

	// 近期修行记录
	if len(sessions) > 0 {
		fmt.Println("\n【近期修行记录】")
		for i, s := range sessions {
			fmt.Printf("  %d. [%s] %s\n", i+1,
				s.CreatedAt.Format("01-02 15:04"), s.Summary)
		}
	}

	fmt.Println()
}

// renderBar 将 0-100 的分数渲染为简单进度条。
func renderBar(score int) string {
	const width = 10
	filled := score * width / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return "[" + bar + "]"
}

// groupByCategory 按类别分组工具掌握度。
func groupByCategory(tools []*episodic.ToolMastery) map[string][]*episodic.ToolMastery {
	result := make(map[string][]*episodic.ToolMastery)
	for _, t := range tools {
		result[t.Category] = append(result[t.Category], t)
	}
	return result
}

func ifEmptyStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
