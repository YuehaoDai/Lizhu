package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ---- 样式定义 ----

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	barFilledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	barEmptyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看修行档案与法宝库",
	Long:  `展示当前修行者的完整档案，包括双轨练气士境界、武夫境界、法宝库及近期心魔记录。`,
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
		return fmt.Errorf("读取法宝库失败: %w", err)
	}

	sessions, err := repo.GetRecentSessions(ctx, "default", 3)
	if err != nil {
		return fmt.Errorf("读取会话记录失败: %w", err)
	}

	userName := viper.GetString("user.name")
	if userName == "" {
		userName = "修行者"
	}
	printFullProfile(userName, profile, toolMastery, sessions)
	return nil
}

// printFullProfile 打印完整档案（status 命令使用）。
func printFullProfile(userName string, p *episodic.Profile, tools []*episodic.ToolMastery, sessions []*episodic.Session) {
	const borderWidth = 52 // ╔ + 50×═ + ╗，内容区 50 列
	bannerText := "骊珠 · " + userName + "的修行档案"
	textW := termWidth(bannerText)
	leftPad := (borderWidth - textW) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	fmt.Println()
	fmt.Println(titleStyle.Render("╔══════════════════════════════════════════════════╗"))
	fmt.Printf("%s%s\n", strings.Repeat(" ", leftPad), titleStyle.Render(bannerText))
	fmt.Println(titleStyle.Render("╚══════════════════════════════════════════════════╝"))

	fmt.Println("\n" + sectionStyle.Render("【修行境界】"))
	if p.GoLianqiScore > 0 || p.ActivePath == "go" || p.ActivePath == "both" {
		fmt.Printf("  Go开发练气士  %s %d/100\n",
			renderBar(p.GoLianqiScore), p.GoLianqiScore)
		fmt.Printf("  第%d境·%-8s  主修分支：%s\n",
			p.GoLianqiLevel, highlightStyle.Render(p.GoLianqiLevelName),
			ifEmptyStr(p.GoLianqiBranch, "待定"))
	}
	if p.AILianqiScore > 0 || p.ActivePath == "ai" || p.ActivePath == "both" {
		fmt.Printf("\n  AI应用练气士  %s %d/100\n",
			renderBar(p.AILianqiScore), p.AILianqiScore)
		fmt.Printf("  第%d境·%-8s  主修分支：%s\n",
			p.AILianqiLevel, highlightStyle.Render(p.AILianqiLevelName),
			ifEmptyStr(p.AILianqiBranch, "待定"))
	}
	fmt.Printf("\n  武夫          %s %d/100\n",
		renderBar(p.WufuScore), p.WufuScore)
	fmt.Printf("  第%d境·%s\n", p.WufuLevel, highlightStyle.Render(p.WufuLevelName))

	// 法宝库
	if len(tools) > 0 {
		fmt.Println("\n" + sectionStyle.Render("【法宝库】"))
		groupedTools := groupByCategory(tools)
		categories := []string{"primary_weapon", "juanjuan", "fulu", "fangcun", "zhenfa", "linchong", "telescope", "quality"}
		catNames := map[string]string{
			"primary_weapon": "本命飞剑",
			"juanjuan":       "绘卷",
			"fulu":           "符箓",
			"fangcun":        "方寸物",
			"zhenfa":         "护山大阵",
			"linchong":       "灵宠",
			"telescope":      "观星镜",
			"quality":        "法家戒尺",
		}
		for _, cat := range categories {
			items, ok := groupedTools[cat]
			if !ok || len(items) == 0 {
				continue
			}
			fmt.Printf("  %s\n", dimStyle.Render("["+catNames[cat]+"]"))
			for _, t := range items {
				fmt.Printf("    %-20s %s %-4d  %s\n",
					t.ToolName, renderBar(t.Score), t.Score,
					highlightStyle.Render(t.LevelName))
			}
		}
	}

	// 心魔记录
	if len(p.XinMoRecords) > 0 {
		fmt.Println("\n" + sectionStyle.Render("【已记录心魔】"))
		for _, xm := range p.XinMoRecords {
			fmt.Printf("  ✗ %s\n", xm)
		}
	}

	// 近期修行记录
	if len(sessions) > 0 {
		fmt.Println("\n" + sectionStyle.Render("【近期修行记录】"))
		for i, s := range sessions {
			fmt.Printf("  %d. %s %s\n", i+1,
				dimStyle.Render("["+s.CreatedAt.Format("01-02 15:04")+"]"),
				s.Summary)
		}
	}

	fmt.Println()
}

// renderBar 将 0-100 的分数渲染为带颜色的进度条。
func renderBar(score int) string {
	const width = 10
	filled := score * width / 100
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	bar := barFilledStyle.Render(strings.Repeat("█", filled)) +
		barEmptyStyle.Render(strings.Repeat("░", width-filled))
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
