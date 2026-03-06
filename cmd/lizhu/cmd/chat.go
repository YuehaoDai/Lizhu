package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/YuehaoDai/lizhu/internal/agent/guardian"
	"github.com/charmbracelet/glamour"
	"github.com/cloudwego/eino/schema"
	"github.com/peterh/liner"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// getTerminalWidth 获取终端列宽，失败时返回默认值 80。
func getTerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// streamRenderer 在流式输出的同时追踪已占用的终端行数，
// 输出完毕后用 ANSI 转义序列抹除原始文本，再用 glamour 重新渲染。
type streamRenderer struct {
	buf   strings.Builder
	rows  int // 已经"换行/折行"的次数（不含当前行）
	col   int // 当前列偏移
	termW int
}

func newStreamRenderer() *streamRenderer {
	return &streamRenderer{termW: getTerminalWidth()}
}

// print 打印字符串并同步更新行列计数。
func (sr *streamRenderer) print(s string) {
	fmt.Print(s)
	sr.buf.WriteString(s)
	for _, r := range s {
		if r == '\n' {
			sr.rows++
			sr.col = 0
		} else {
			sr.col += runeDisplayWidth(r)
			for sr.col >= sr.termW {
				sr.rows++
				sr.col -= sr.termW
			}
		}
	}
}

// finalize 用 ANSI 序列抹除已打印的原始流，再以 glamour 重新渲染。
// 调用后不需要再单独打印换行——glamour 输出末尾已含换行。
func (sr *streamRenderer) finalize() {
	text := sr.buf.String()
	if text == "" {
		return
	}

	// 回到流输出的起始行
	if sr.rows > 0 {
		fmt.Printf("\033[%dA", sr.rows)
	}
	fmt.Print("\r\033[J") // 回列首 + 清除到屏幕末尾

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(sr.termW-4),
	)
	if err != nil {
		fmt.Print(text)
		return
	}
	rendered, err := r.Render(text)
	if err != nil {
		fmt.Print(text)
		return
	}
	fmt.Print(rendered)
}

// runeDisplayWidth 返回单个字符的终端显示列宽（CJK 等宽字符返回 2，其余返回 1）。
func runeDisplayWidth(r rune) int {
	if r >= 0x1100 && (r <= 0x115F ||
		r == 0x2329 || r == 0x232A ||
		(r >= 0x2E80 && r <= 0x303E) ||
		(r >= 0x3040 && r <= 0x33FF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0xA000 && r <= 0xA4CF) ||
		(r >= 0xAC00 && r <= 0xD7AF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFE10 && r <= 0xFE1F) ||
		(r >= 0xFE30 && r <= 0xFE4F) ||
		(r >= 0xFF00 && r <= 0xFF60) ||
		(r >= 0xFFE0 && r <= 0xFFE6) ||
		(r >= 0x20000 && r <= 0x2A6DF)) {
		return 2
	}
	return 1
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "与护道人开始修行对话",
	Long: `启动与护道人的交互式对话。

输入 /quit 或 /exit 结束对话，/clear 清空本次会话历史，/status 查看当前档案，/tasks 查看修炼任务单。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChatCLI(cmd.Context())
	},
}

// guardianLabel 返回护道人的显示名称。
func guardianLabel(personaName string) string {
	if personaName != "" {
		return "护道人·" + personaName
	}
	return "护道人"
}

// ---- CLI 模式 ----

func runChatCLI(ctx context.Context) error {
	agent, err := newGuardianAgent(ctx)
	if err != nil {
		return err
	}

	label := guardianLabel(agent.PersonaName())

	priorSessions, _ := repo.GetRecentSessions(ctx, "default", 1)
	firstTime := len(priorSessions) == 0
	entranceScene := agent.GenerateEntrance(ctx, firstTime)
	printWelcomeCLI(label, entranceScene, firstTime)

	line := liner.NewLiner()
	line.SetCtrlCAborts(true)
	defer line.Close()

	var history []*schema.Message

	for {
		raw, err := line.Prompt("修行者 › ")
		if err != nil {
			// Ctrl+C 或 EOF
			runShutdownSequence(ctx, agent, history, label)
			break
		}
		input := strings.TrimSpace(raw)
		if input == "" {
			continue
		}

		switch input {
		case "/quit", "/exit", "/q":
			runShutdownSequence(ctx, agent, history, label)
			return nil
		case "/clear":
			history = nil
			fmt.Println("[本次会话历史已清空]")
			continue
		case "/status":
			printStatusInlineCLI(ctx)
			continue
		case "/tasks":
			printTasksInlineCLI(ctx)
			continue
		case "/help":
			printChatHelp()
			continue
		case "/assess":
			input = "[修行者请求完整境界评估] /assess"
		}

		assess := input == "[修行者请求完整境界评估] /assess"

		fmt.Printf("\n%s › \n", label)
		fmt.Println(strings.Repeat("─", 60))

		var (
			newHistory []*schema.Message
			chatErr    error
		)

		sr := newStreamRenderer()

		if assess {
			// 评估模式：LLM 会生成 eval_json（及其前的标题行），使用多触发器滑动缓冲区过滤。
			suppressTriggers := []string{"修行档案JSON", "<eval_json>"}
			maxTrigLen := 0
			for _, t := range suppressTriggers {
				if len(t) > maxTrigLen {
					maxTrigLen = len(t)
				}
			}
			var suppressBuf strings.Builder
			suppressed := false
			_, newHistory, chatErr = agent.ChatStream(ctx, history, input, func(token string) {
				if suppressed {
					return
				}
				suppressBuf.WriteString(token)
				s := suppressBuf.String()

				hitIdx := -1
				for _, trig := range suppressTriggers {
					if idx := strings.Index(s, trig); idx >= 0 {
						if hitIdx < 0 || idx < hitIdx {
							hitIdx = idx
						}
					}
				}
				if hitIdx >= 0 {
					suppressed = true
					printUntil := hitIdx
					if nl := strings.LastIndex(s[:hitIdx], "\n"); nl >= 0 {
						printUntil = nl + 1
					}
					if printUntil > 0 {
						sr.print(s[:printUntil])
					}
					suppressBuf.Reset()
					return
				}

				safe := len(s) - (maxTrigLen - 1)
				if safe > 0 {
					sr.print(s[:safe])
					tail := s[safe:]
					suppressBuf.Reset()
					suppressBuf.WriteString(tail)
				}
			}, true)
			if !suppressed && suppressBuf.Len() > 0 {
				sr.print(suppressBuf.String())
			}
		} else {
			// 普通护道对话：token 经 streamRenderer 输出，结束后 glamour 重渲染
			_, newHistory, chatErr = agent.ChatStream(ctx, history, input, func(token string) {
				sr.print(token)
			}, false)
		}

		sr.finalize()
		fmt.Println()
		if chatErr != nil {
			fmt.Fprintf(os.Stderr, "\n[错误] %v\n", chatErr)
			continue
		}
		history = newHistory
		line.AppendHistory(input)
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println()
	}

	return nil
}

// runShutdownSequence 在退出时依次等待后台持久化、保存会话概要与能力证据，
// 并在终端逐步展示进度，让修行者看到系统正在进行的操作。
func runShutdownSequence(ctx context.Context, agent *guardian.Agent, history []*schema.Message, label string) {
	fmt.Printf("\n%s：修行路漫漫，保重。\n", label)
	fmt.Println()
	fmt.Println("[骊珠正在封存本次修行记录...]")

	// 1. 等待 /assess 后台评估 goroutine 完成
	agent.WaitPersist()
	fmt.Println("  ✓ 评估档案已同步")

	// 2. 保存整次对话摘要 + 能力证据
	if len(history) == 0 {
		fmt.Println("  · 本次对话无记录，跳过封存")
		fmt.Println()
		return
	}

	fmt.Println("  · 正在生成会话概要与能力证据...")
	result, perr := agent.PersistFullSession(ctx, history)
	if perr != nil {
		fmt.Fprintf(os.Stderr, "  [警告] 会话记录保存失败: %v\n", perr)
	} else {
		if result.EvidenceCount > 0 {
			fmt.Printf("  ✓ 已封存 %d 条能力证据\n", result.EvidenceCount)
		} else {
			fmt.Println("  ✓ 会话概要已封存")
		}
	}

	fmt.Println()
	fmt.Println("本次修行已落入案牍。下次相见，护道人自会牢记今日所得。")
	fmt.Println()
}

// termWidth 计算字符串的终端显示宽度（CJK 全角字符算 2 列，其余算 1 列）。
func termWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeDisplayWidth(r)
	}
	return w
}

func printWelcomeCLI(label, entranceScene string, firstTime bool) {
	const borderWidth = 52 // ╔ + 50×═ + ╗
	bannerText := "骊珠 · " + label + " 已就位"
	textW := termWidth(bannerText)
	leftPad := (borderWidth - textW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Printf("%s%s\n", strings.Repeat(" ", leftPad), bannerText)
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	if entranceScene != "" {
		// LLM 生成的场景描写：散文叙述，整段输出
		fmt.Println(entranceScene)
	} else {
		// 降级：未配置或生成失败时显示简短提示
		if firstTime {
			fmt.Printf("%s：初次相见，且坐，你且说说看。\n", label)
		} else {
			fmt.Printf("%s：来了，坐。\n", label)
		}
	}
	fmt.Println("\n输入 /help 查看可用命令。")
	fmt.Println()
}

func printChatHelp() {
	fmt.Print(`
可用命令：
  /assess  — 主动请求完整境界评估与破境方案
  /status  — 查看当前修行档案与法宝库
  /tasks   — 查看当前修炼任务单
  /clear   — 清空本次会话历史（不影响已保存的档案）
  /quit    — 退出对话
  /help    — 显示此帮助
`)
}

func printTasksInlineCLI(ctx context.Context) {
	if repo == nil {
		fmt.Println("[错误] 仓库未初始化")
		return
	}
	tasks, err := repo.GetPendingTasks(ctx, "default")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取任务失败: %v\n", err)
		return
	}
	fmt.Println()
	fmt.Println("── 修炼任务单 ────────────────────────────────────")
	if len(tasks) == 0 {
		fmt.Println("  暂无待完成任务。完成一次 /assess 后，护道人会为你生成。")
	} else {
		for i, t := range tasks {
			fmt.Printf("\n  【%d】%s\n", i+1, t.Title)
			fmt.Printf("      %s\n", t.Description)
			fmt.Printf("      验收：%s\n", t.AcceptanceCriteria)
			if t.SourceEvidence != "" {
				fmt.Printf("      来源：%s\n", t.SourceEvidence)
			}
		}
	}
	fmt.Println("\n───────────────────────────────────────────────────")
}

func printStatusInlineCLI(ctx context.Context) {
	if repo == nil {
		fmt.Println("[错误] 仓库未初始化")
		return
	}
	p, err := repo.GetOrCreateProfile(ctx, "default")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取档案失败: %v\n", err)
		return
	}
	fmt.Println()
	fmt.Println("── 修行档案 ──────────────────────────────────────")
	if p.GoLianqiScore > 0 || p.ActivePath == "go" || p.ActivePath == "both" {
		fmt.Printf("  Go练气士  : %s %d/100  第%d境·%s\n",
			renderBar(p.GoLianqiScore), p.GoLianqiScore,
			p.GoLianqiLevel, p.GoLianqiLevelName)
	}
	if p.AILianqiScore > 0 || p.ActivePath == "ai" || p.ActivePath == "both" {
		fmt.Printf("  AI练气士  : %s %d/100  第%d境·%s\n",
			renderBar(p.AILianqiScore), p.AILianqiScore,
			p.AILianqiLevel, p.AILianqiLevelName)
	}
	fmt.Printf("  武夫      : %s %d/100  第%d境·%s\n",
		renderBar(p.WufuScore), p.WufuScore,
		p.WufuLevel, p.WufuLevelName)
	fmt.Println("───────────────────────────────────────────────────")
}
