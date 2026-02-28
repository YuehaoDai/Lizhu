// Package tui 实现骊珠 Bubble Tea 终端界面。
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/YuehaoDai/lizhu/internal/agent/guardian"
	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/schema"
)

// ---- 消息类型 ----

// tokenMsg 流式 token 到达。
type tokenMsg struct{ token string }

// streamDoneMsg 流式回复完成。
type streamDoneMsg struct {
	newHistory []*schema.Message
	err        error
}

// statusTextMsg 来自 /status 命令的渲染文本。
type statusTextMsg struct{ text string }

// ---- 聊天状态 ----

type chatState int

const (
	stateIdle      chatState = iota // 等待用户输入
	stateStreaming                   // 正在接收 LLM 流式输出
)

// ---- Model ----

// Model 是 Bubble Tea 程序的核心状态机。
type Model struct {
	agent         *guardian.Agent
	repo          *episodic.Repository
	ctx           context.Context
	guardianLabel string
	keyMap        KeyMap

	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	state   chatState
	history []*schema.Message

	// viewport 原始内容（累积）
	content string
	// 当前流式回复缓冲（不含 <eval_json> 之后的部分）。
	// 使用指针避免 Bubble Tea 按值传递 Model 时 strings.Builder 被拷贝而 panic。
	streamBuf   *strings.Builder
	evalTagSeen bool

	// 流式通道（每次对话新建）
	tokenCh <-chan string
	doneCh  <-chan streamDoneMsg

	width  int
	height int

	lastErr string
}

// Config TUI 初始化配置。
type Config struct {
	Agent         *guardian.Agent
	Repo          *episodic.Repository
	Ctx           context.Context
	GuardianLabel string
	Profile       *episodic.Profile
}

// New 创建并初始化 TUI Model。
func New(cfg Config) Model {
	ta := textarea.New()
	ta.Placeholder = "输入修行感悟，或 /help 查看命令…"
	ta.Focus()
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.CharLimit = 2000
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(80, 20)
	welcome := welcomeText(cfg.Profile, cfg.GuardianLabel)
	vp.SetContent(welcome)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		agent:         cfg.Agent,
		repo:          cfg.Repo,
		ctx:           cfg.Ctx,
		guardianLabel: cfg.GuardianLabel,
		keyMap:        DefaultKeyMap(),
		viewport:      vp,
		textarea:      ta,
		spinner:       sp,
		state:         stateIdle,
		content:       welcome,
		streamBuf:     &strings.Builder{},
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.relayout()

	case tea.KeyMsg:
		if m.state == stateStreaming {
			// 流式输出期间忽略大部分键盘输入（仅允许 Ctrl+C）
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
			break
		}
		switch {
		case msg.Type == tea.KeyCtrlC:
			return m, tea.Quit
		case msg.Type == tea.KeyEnter:
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				break
			}
			m.textarea.Reset()
			cmd := m.handleInput(input)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case msg.Type == tea.KeyPgUp:
			m.viewport.LineUp(5)
		case msg.Type == tea.KeyPgDown:
			m.viewport.LineDown(5)
		}

	case tokenMsg:
		if !m.evalTagSeen {
			combined := m.streamBuf.String() + msg.token
			if idx := strings.Index(combined, "<eval_json>"); idx >= 0 {
				m.evalTagSeen = true
				m.streamBuf.Reset()
				m.streamBuf.WriteString(combined[:idx])
			} else {
				m.streamBuf.WriteString(msg.token)
			}
			m.updateViewportStream()
		}
		// 继续监听下一个 token
		cmds = append(cmds, waitToken(m.tokenCh, m.doneCh))

	case streamDoneMsg:
		m.state = stateIdle
		m.evalTagSeen = false
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.appendContent(fmt.Sprintf("[错误] %v\n", msg.err))
		} else {
			m.history = msg.newHistory
			m.lastErr = ""
			divider := "\n" + strings.Repeat("─", 60) + "\n\n"
			m.appendContent(m.streamBuf.String() + divider)
		}
		m.streamBuf.Reset()
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()

	case statusTextMsg:
		m.appendContent(msg.text)
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// 非流式状态下更新 textarea
	if m.state == stateIdle {
		var taCmd tea.Cmd
		m.textarea, taCmd = m.textarea.Update(msg)
		cmds = append(cmds, taCmd)
	}

	// viewport 始终更新（支持滚动）
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// ---- 内部辅助 ----

// relayout 调整各组件尺寸。
// 实际渲染行数 = 1(header) + 1(sep) + 1(border-top) + vpH + 1(border-bot)
//               + 1(sep) + 5(inputStyle=textarea3+border2) + 1(\n) + 1(hint)
//             = vpH + 12
// 因此 vpH = m.height - 12，viewport 宽度需减去 borderStyle 左右各 1 字符。
func (m *Model) relayout() {
	const overhead = 12
	vpH := m.height - overhead
	if vpH < 5 {
		vpH = 5
	}
	m.viewport.Width = m.width - 2 // 减去 borderStyle 左右边框
	m.viewport.Height = vpH
	m.textarea.SetWidth(m.width - 4)
	m.viewport.SetContent(m.content)
}

// appendContent 向累积内容追加文本并更新 viewport。
func (m *Model) appendContent(text string) {
	m.content += text
}

// updateViewportStream 在流式输出时实时显示当前缓冲内容。
func (m *Model) updateViewportStream() {
	m.viewport.SetContent(m.content + m.streamBuf.String())
	m.viewport.GotoBottom()
}

// handleInput 处理用户输入（命令路由）。
func (m *Model) handleInput(input string) tea.Cmd {
	m.appendContent(fmt.Sprintf("修行者 › %s\n", input))
	m.viewport.SetContent(m.content)
	m.viewport.GotoBottom()

	switch input {
	case "/quit", "/exit", "/q":
		return tea.Quit
	case "/clear":
		m.history = nil
		m.appendContent("[本次会话历史已清空]\n\n")
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()
		return nil
	case "/status":
		return m.cmdStatus()
	case "/help":
		m.appendContent(helpText())
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()
		return nil
	case "/assess":
		input = "[修行者请求完整境界评估] /assess"
	}

	return m.startStream(input)
}

// startStream 发起流式 LLM 调用，返回首个等待 token 的 Cmd。
func (m *Model) startStream(input string) tea.Cmd {
	m.state = stateStreaming
	m.evalTagSeen = false
	m.streamBuf.Reset()

	header := fmt.Sprintf("\n%s › \n%s\n", m.guardianLabel, strings.Repeat("─", 60))
	m.appendContent(header)
	m.viewport.SetContent(m.content)
	m.viewport.GotoBottom()

	tokenCh := make(chan string, 256)
	doneCh := make(chan streamDoneMsg, 1)
	m.tokenCh = tokenCh
	m.doneCh = doneCh

	history := m.history
	agent := m.agent
	ctx := m.ctx

	// 启动 goroutine 调用 LLM
	go func() {
		_, newHistory, err := agent.ChatStream(ctx, history, input, func(token string) {
			tokenCh <- token
		}, false)
		close(tokenCh)
		doneCh <- streamDoneMsg{newHistory: newHistory, err: err}
	}()

	// 立即返回一个监听第一个 token 的 Cmd
	return waitToken(tokenCh, doneCh)
}

// waitToken 返回一个 tea.Cmd，监听下一个 token 或完成信号。
func waitToken(tokenCh <-chan string, doneCh <-chan streamDoneMsg) tea.Cmd {
	return func() tea.Msg {
		token, ok := <-tokenCh
		if !ok {
			// channel 已关闭，等待 done 信号
			return <-doneCh
		}
		return tokenMsg{token: token}
	}
}

// cmdStatus 查询修行档案并返回格式化文本。
func (m *Model) cmdStatus() tea.Cmd {
	repo := m.repo
	ctx := m.ctx
	return func() tea.Msg {
		if repo == nil {
			return statusTextMsg{text: "[错误] 数据库未初始化\n\n"}
		}
		p, err := repo.GetOrCreateProfile(ctx, "default")
		if err != nil {
			return statusTextMsg{text: fmt.Sprintf("[错误] 读取档案失败: %v\n\n", err)}
		}
		return statusTextMsg{text: formatProfileTUI(p)}
	}
}
