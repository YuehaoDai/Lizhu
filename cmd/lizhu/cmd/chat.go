package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/peterh/liner"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "与护道人开始修行对话",
	Long: `启动与护道人的交互式对话。

输入 /quit 或 /exit 结束对话，/clear 清空本次会话历史，/status 查看当前档案。`,
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
			// Ctrl+C 或 EOF，先等后台持久化完成，再保存完整会话概要
			agent.WaitPersist()
			if len(history) > 0 {
				if perr := agent.PersistFullSession(ctx, history); perr != nil {
					fmt.Fprintf(os.Stderr, "[警告] 会话记录保存失败: %v\n", perr)
				}
			}
			break
		}
		input := strings.TrimSpace(raw)
		if input == "" {
			continue
		}

		switch input {
		case "/quit", "/exit", "/q":
			fmt.Printf("%s：修行路漫漫，保重。\n", label)
			agent.WaitPersist()
			if len(history) > 0 {
				if perr := agent.PersistFullSession(ctx, history); perr != nil {
					fmt.Fprintf(os.Stderr, "[警告] 会话记录保存失败: %v\n", perr)
				}
			}
			return nil
		case "/clear":
			history = nil
			fmt.Println("[本次会话历史已清空]")
			continue
		case "/status":
			printStatusInlineCLI(ctx)
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

		if assess {
			// 评估模式：LLM 会生成 eval_json（及其前的标题行），使用多触发器滑动缓冲区过滤。
			// 触发器按优先级：先检测 "修行档案JSON"（标题行），再检测 "<eval_json>"（标签本身）。
			// 检测到任一触发器时，从所在行的行首开始截断（不打印该行及之后的内容）。
			suppressTriggers := []string{"修行档案JSON", "<eval_json>"}
			maxTrigLen := 0
			for _, t := range suppressTriggers {
				if len(t) > maxTrigLen {
					maxTrigLen = len(t)
				}
			}
			var buf strings.Builder
			suppressed := false
			_, newHistory, chatErr = agent.ChatStream(ctx, history, input, func(token string) {
				if suppressed {
					return
				}
				buf.WriteString(token)
				s := buf.String()

				// 检查是否命中任一触发器
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
					// 从行首截断：找到 hitIdx 之前最近的换行符
					printUntil := hitIdx
					if nl := strings.LastIndex(s[:hitIdx], "\n"); nl >= 0 {
						printUntil = nl + 1 // 保留换行符本身，但不打印触发行
					}
					if printUntil > 0 {
						fmt.Print(s[:printUntil])
					}
					buf.Reset()
					return
				}

				// 保留末尾 maxTrigLen-1 字节作为预警窗口
				safe := len(s) - (maxTrigLen - 1)
				if safe > 0 {
					fmt.Print(s[:safe])
					tail := s[safe:]
					buf.Reset()
					buf.WriteString(tail)
				}
			}, true)
			if !suppressed && buf.Len() > 0 {
				fmt.Print(buf.String())
			}
		} else {
			// 普通护道对话：LLM 不生成 eval_json，token 直接输出
			_, newHistory, chatErr = agent.ChatStream(ctx, history, input, func(token string) {
				fmt.Print(token)
			}, false)
		}

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

// termWidth 计算字符串的终端显示宽度（CJK 全角字符算 2 列，其余算 1 列）。
func termWidth(s string) int {
	w := 0
	for _, r := range s {
		if r >= 0x1100 && (r <= 0x115F || // Hangul Jamo
			r == 0x2329 || r == 0x232A ||
			(r >= 0x2E80 && r <= 0x303E) || // CJK Radicals Supplement .. CJK Symbols
			(r >= 0x3040 && r <= 0x33FF) || // Japanese
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Unified Ideographs Extension A
			(r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
			(r >= 0xA000 && r <= 0xA4CF) || // Yi
			(r >= 0xAC00 && r <= 0xD7AF) || // Hangul Syllables
			(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
			(r >= 0xFE10 && r <= 0xFE1F) || // Vertical forms
			(r >= 0xFE30 && r <= 0xFE4F) || // CJK Compatibility Forms
			(r >= 0xFF00 && r <= 0xFF60) || // Fullwidth Forms
			(r >= 0xFFE0 && r <= 0xFFE6) ||
			(r >= 0x1F300 && r <= 0x1F64F) || // Misc Symbols and Pictographs
			(r >= 0x20000 && r <= 0x2A6DF)) { // CJK Extension B
			w += 2
		} else {
			w++
		}
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
  /status  — 查看当前修行档案与法器谱
  /clear   — 清空本次会话历史（不影响已保存的档案）
  /quit    — 退出对话
  /help    — 显示此帮助
`)
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
