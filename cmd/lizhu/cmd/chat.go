package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/YuehaoDai/lizhu/internal/tui"
	"github.com/chzyer/readline"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/schema"
	"github.com/spf13/cobra"
)

var chatTUIMode bool

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "与护道人开始修行对话",
	Long: `启动与护道人的交互式对话。
默认使用经典 readline CLI 模式；加 --tui 标志启用 Bubble Tea 全屏界面（实验性）。

输入 /quit 或 /exit 结束对话，/clear 清空本次会话历史，/status 查看当前档案。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if chatTUIMode {
			return runChatTUI(cmd.Context())
		}
		return runChatCLI(cmd.Context())
	},
}

func init() {
	chatCmd.Flags().BoolVar(&chatTUIMode, "tui", false, "启用 Bubble Tea 全屏 TUI 界面（实验性）")
}

// guardianLabel 返回护道人的显示名称。
func guardianLabel(personaName string) string {
	if personaName != "" {
		return "护道人·" + personaName
	}
	return "护道人"
}

// ---- TUI 模式 ----

func runChatTUI(ctx context.Context) error {
	agent, err := newGuardianAgent(ctx)
	if err != nil {
		return err
	}

	label := guardianLabel(agent.PersonaName())

	epProfile, err := repo.GetOrCreateProfile(ctx, "default")
	if err != nil {
		return fmt.Errorf("读取修行档案失败: %w", err)
	}

	m := tui.New(tui.Config{
		Agent:         agent,
		Repo:          repo,
		Ctx:           ctx,
		GuardianLabel: label,
		Profile:       epProfile,
	})

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI 运行失败: %w", err)
	}
	return nil
}

// ---- CLI 模式（readline，稳定退路）----

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

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "修行者 › ",
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("初始化输入行失败: %w", err)
	}
	defer rl.Close()

	var history []*schema.Message

	for {
		raw, err := rl.Readline()
		if err != nil {
			break
		}
		input := strings.TrimSpace(raw)
		if input == "" {
			continue
		}

		switch input {
		case "/quit", "/exit", "/q":
			fmt.Printf("%s：修行路漫漫，保重。\n", label)
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

		fmt.Printf("\n%s › \n", label)
		fmt.Println(strings.Repeat("─", 60))

		// 滑动缓冲区过滤器：保留末尾 len("<eval_json>")-1 字节作为"预警窗口"，
		// 确保跨 token 分割的 <eval_json> 标签也能被完整检测到。
		const evalTag = "<eval_json>"
		var buf strings.Builder
		evalTagSeen := false
		_, newHistory, err := agent.ChatStream(ctx, history, input, func(token string) {
			if evalTagSeen {
				return
			}
			buf.WriteString(token)
			s := buf.String()
			if idx := strings.Index(s, evalTag); idx >= 0 {
				evalTagSeen = true
				if idx > 0 {
					fmt.Print(s[:idx])
				}
				buf.Reset()
				return
			}
			// 保留末尾 len(evalTag)-1 字节，剩余部分安全输出
			safe := len(s) - (len(evalTag) - 1)
			if safe > 0 {
				fmt.Print(s[:safe])
				tail := s[safe:]
				buf.Reset()
				buf.WriteString(tail)
			}
		})
		// 冲刷剩余缓冲（未出现 eval_json 标签时）
		if !evalTagSeen && buf.Len() > 0 {
			fmt.Print(buf.String())
		}
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n[错误] %v\n", err)
			continue
		}
		history = newHistory
		fmt.Println(strings.Repeat("─", 60))
		fmt.Fprintln(rl.Stdout())
	}

	return nil
}

func printWelcomeCLI(label, entranceScene string, firstTime bool) {
	bannerText := "骊珠 · " + label + " 已就位"
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Printf("║  %-46s  ║\n", bannerText)
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
	fmt.Println("\n输入 /help 查看可用命令。（加 --tui 可启用实验性全屏界面）")
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
