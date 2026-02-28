package tui

import (
	"fmt"
	"strings"

	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	"github.com/charmbracelet/lipgloss"
)

// ---- 样式 ----

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238"))

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("247"))

	barFillStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	barEmptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

// View 渲染整个 TUI 界面。
func (m Model) View() string {
	if m.width == 0 {
		return "骊珠加载中…"
	}

	var sections []string

	// 顶部状态栏
	sections = append(sections, m.renderHeader())

	// 对话区域（带边框）
	sections = append(sections, borderStyle.Render(m.viewport.View()))

	// 底部输入框
	sections = append(sections, m.renderFooter())

	return strings.Join(sections, "\n")
}

// renderHeader 渲染顶部标题栏。
func (m Model) renderHeader() string {
	title := fmt.Sprintf("  骊珠 · %s", m.guardianLabel)
	status := ""
	if m.state == stateStreaming {
		status = m.spinner.View() + " 思考中…"
	} else if m.lastErr != "" {
		status = errorStyle.Render("✗ " + m.lastErr)
	} else {
		status = helpStyle.Render("Enter 发送 · PgUp/PgDn 滚动 · Ctrl+C 退出")
	}

	width := m.width
	titleW := lipgloss.Width(title)
	statusW := lipgloss.Width(status)
	gap := width - titleW - statusW - 2
	if gap < 1 {
		gap = 1
	}
	line := title + strings.Repeat(" ", gap) + status
	return headerStyle.Width(width).Render(line)
}

// renderFooter 渲染底部输入区域。
func (m Model) renderFooter() string {
	hint := helpStyle.Render("/help /assess /status /clear /quit")
	inputView := inputStyle.Width(m.width - 2).Render(m.textarea.View())
	return inputView + "\n" + hint
}

// ---- 文本渲染辅助 ----

// welcomeText 生成欢迎页内容。
func welcomeText(profile *episodic.Profile, label string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("╔══════════════════════════════════════════════════╗\n"))
	sb.WriteString(fmt.Sprintf("║  骊珠 · %-41s ║\n", label+" 已就位"))
	sb.WriteString(fmt.Sprintf("╚══════════════════════════════════════════════════╝\n\n"))

	if profile != nil && (profile.GoLianqiScore > 0 || profile.AILianqiScore > 0) {
		sb.WriteString(fmt.Sprintf("%s已读取修行档案，可直接继续上次修行。\n", label))
		sb.WriteString(formatProfileTUI(profile))
	} else {
		sb.WriteString(fmt.Sprintf("这是你第一次与%s相见。\n", label))
		sb.WriteString("请先描述你的技术背景、工作经历与当前在学的内容，\n")
		sb.WriteString(fmt.Sprintf("%s将进行初次境界诊断并建立修行档案。\n", label))
	}

	sb.WriteString("\n输入 /help 查看可用命令。\n\n")
	return sb.String()
}

// formatProfileTUI 将修行档案渲染为 TUI 友好的文本。
func formatProfileTUI(p *episodic.Profile) string {
	var sb strings.Builder
	sb.WriteString("\n── 修行档案 ─────────────────────────────────────────\n")

	if p.GoLianqiScore > 0 || p.ActivePath == "go" || p.ActivePath == "both" {
		sb.WriteString(fmt.Sprintf("  Go练气士   %s %d/100  第%d境·%s\n",
			renderBarColored(p.GoLianqiScore), p.GoLianqiScore,
			p.GoLianqiLevel, p.GoLianqiLevelName))
	}
	if p.AILianqiScore > 0 || p.ActivePath == "ai" || p.ActivePath == "both" {
		sb.WriteString(fmt.Sprintf("  AI练气士   %s %d/100  第%d境·%s\n",
			renderBarColored(p.AILianqiScore), p.AILianqiScore,
			p.AILianqiLevel, p.AILianqiLevelName))
	}
	sb.WriteString(fmt.Sprintf("  武夫       %s %d/100  第%d境·%s\n",
		renderBarColored(p.WufuScore), p.WufuScore,
		p.WufuLevel, p.WufuLevelName))
	sb.WriteString("───────────────────────────────────────────────────\n\n")
	return sb.String()
}

// renderBarColored 渲染带颜色的进度条。
func renderBarColored(score int) string {
	const width = 10
	filled := score * width / 100
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	bar := barFillStyle.Render(strings.Repeat("█", filled)) +
		barEmptyStyle.Render(strings.Repeat("░", width-filled))
	return "[" + bar + "]"
}

// helpText 返回帮助文本。
func helpText() string {
	return `
可用命令：
  /assess  — 主动请求完整境界评估与破境方案
  /status  — 查看当前修行档案（也可按 Ctrl+S）
  /clear   — 清空本次会话历史（不影响已保存的档案）
  /quit    — 退出对话
  /help    — 显示此帮助

快捷键：
  Enter    — 发送消息
  PgUp     — 向上滚动对话
  PgDn     — 向下滚动对话
  Ctrl+C   — 退出

`
}

// statusBarStyle is defined above; this line avoids "declared and not used" in older Go.
var _ = statusBarStyle
