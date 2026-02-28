package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "与护道人开始修行对话",
	Long: `启动与护道人的交互式对话。
护道人将根据你的描述评估修行境界、给出破境路径、更新修行档案。

输入 /quit 或 /exit 结束对话，/clear 清空本次会话历史，/status 查看当前档案。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChat(cmd.Context())
	},
}

// guardianLabel 返回护道人的显示名称。
// personaName 非空时返回 "护道人·<personaName>"，否则返回 "护道人"。
func guardianLabel(personaName string) string {
	if personaName != "" {
		return "护道人·" + personaName
	}
	return "护道人"
}

func runChat(ctx context.Context) error {
	agent, err := newGuardianAgent(ctx)
	if err != nil {
		return err
	}

	label := guardianLabel(agent.PersonaName())

	profile, err := repo.GetOrCreateProfile(ctx, "default")
	if err != nil {
		return fmt.Errorf("读取修行档案失败: %w", err)
	}

	printWelcome(profile, label)

	var history []*schema.Message
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for {
		fmt.Print("\n修行者 › ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch input {
		case "/quit", "/exit", "/q":
			fmt.Printf("\n%s：修行路漫漫，保重。\n", label)
			return nil
		case "/clear":
			history = nil
			fmt.Println("[本次会话历史已清空]")
			continue
		case "/status":
			printStatusInline(ctx)
			continue
		case "/help":
			printChatHelp()
			continue
		}

		fmt.Printf("\n%s › ", label)
		fmt.Println()
		fmt.Println(strings.Repeat("─", 60))

		// 流式输出：遇到 <eval_json> 标记前的内容实时显示，之后的 token 静默缓冲
		var printed strings.Builder
		evalTagSeen := false
		_, newHistory, err := agent.ChatStream(ctx, history, input, func(token string) {
			if evalTagSeen {
				return
			}
			combined := printed.String() + token
			if idx := strings.Index(combined, "<eval_json>"); idx >= 0 {
				evalTagSeen = true
				visible := combined[:idx]
				fmt.Print(visible[printed.Len():])
				printed.Reset()
				printed.WriteString(visible)
				return
			}
			fmt.Print(token)
			printed.WriteString(token)
		})
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n[错误] %v\n", err)
			continue
		}
		history = newHistory
		fmt.Println(strings.Repeat("─", 60))
	}

	return scanner.Err()
}

func printWelcome(profile *episodic.Profile, label string) {
	// 横幅宽度固定 50 字符，居中显示 label
	bannerText := "骊珠 · " + label + " 已就位"
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Printf("║  %-46s  ║\n", bannerText)
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	if profile.GoLianqiScore > 0 || profile.AILianqiScore > 0 {
		fmt.Printf("%s已读取修行档案，可直接继续上次修行。\n", label)
		printProfileSummary(profile)
	} else {
		fmt.Printf("这是你第一次与%s相见。\n", label)
		fmt.Println("请先描述你的技术背景、工作经历与当前在学的内容，")
		fmt.Printf("%s将进行初次境界诊断并建立修行档案。\n", label)
	}
	fmt.Println("\n输入 /help 查看可用命令。")
	fmt.Println()
}

func printChatHelp() {
	fmt.Print(`
可用命令：
  /status  — 查看当前修行档案与法器谱
  /clear   — 清空本次会话历史（不影响已保存的档案）
  /quit    — 退出对话
  /help    — 显示此帮助
`)
}

func printStatusInline(ctx context.Context) {
	if repo == nil {
		fmt.Println("[错误] 仓库未初始化")
		return
	}
	p, err := repo.GetOrCreateProfile(ctx, "default")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取档案失败: %v\n", err)
		return
	}
	printProfileSummary(p)
}

// filterEvalJSON 从显示文本中移除 <eval_json>...</eval_json> 块。
func filterEvalJSON(text string) string {
	start := strings.Index(text, "<eval_json>")
	end := strings.Index(text, "</eval_json>")
	if start < 0 || end < 0 || end < start {
		return text
	}
	before := strings.TrimRight(text[:start], "\n ")
	after := strings.TrimLeft(text[end+len("</eval_json>"):], "\n ")
	if after == "" {
		return before
	}
	return before + "\n" + after
}
