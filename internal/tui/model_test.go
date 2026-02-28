package tui

import (
	"strings"
	"testing"

	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	tea "github.com/charmbracelet/bubbletea"
)

// ---- Model 初始化 ----

func TestNew_InitializesCorrectly(t *testing.T) {
	m := New(Config{
		GuardianLabel: "齐静春",
		Profile:       &episodic.Profile{},
	})
	if m.guardianLabel != "齐静春" {
		t.Errorf("guardianLabel = %q, want 齐静春", m.guardianLabel)
	}
	if m.state != stateIdle {
		t.Errorf("initial state should be stateIdle, got %v", m.state)
	}
	if m.history != nil {
		t.Error("initial history should be nil")
	}
	if m.content == "" {
		t.Error("initial content should contain welcome text")
	}
}

func TestNew_ContentContainsLabel(t *testing.T) {
	m := New(Config{
		GuardianLabel: "三角竹笋",
		Profile:       nil,
	})
	if !strings.Contains(m.content, "三角竹笋") {
		t.Errorf("initial content should contain guardian label, got: %q", m.content[:min2(len(m.content), 200)])
	}
}

// ---- 键盘快捷键 / 命令处理 ----

func TestUpdate_CtrlC_Quits(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("Ctrl+C should return a Quit command")
	}
	// tea.Quit 是一个特定函数，通过执行结果检查
	msg := cmd()
	if msg != tea.Quit() {
		t.Error("Ctrl+C should result in tea.Quit")
	}
}

func TestUpdate_WindowSize_UpdatesDimensions(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m2 := updated.(Model)
	if m2.width != 120 {
		t.Errorf("width = %d, want 120", m2.width)
	}
	if m2.height != 40 {
		t.Errorf("height = %d, want 40", m2.height)
	}
}

func TestUpdate_ClearCommand_ClearsHistory(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	// 模拟用户输入 /clear
	m.textarea.SetValue("/clear")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.history != nil {
		t.Error("/clear should reset history to nil")
	}
	if !strings.Contains(m2.content, "已清空") {
		t.Error("/clear should add confirmation message to content")
	}
}

func TestUpdate_HelpCommand_ShowsHelp(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	m.textarea.SetValue("/help")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if !strings.Contains(m2.content, "/assess") {
		t.Error("/help should add help text containing /assess")
	}
}

func TestUpdate_EmptyEnter_NoAction(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	contentBefore := m.content
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.content != contentBefore {
		t.Error("empty Enter should not modify content")
	}
}

func TestUpdate_StreamingIgnoresInput(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	m.state = stateStreaming

	// 流式输出时 Enter 输入应被忽略
	contentBefore := m.content
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.content != contentBefore {
		t.Error("Enter during streaming should not change content")
	}
	if m2.state != stateStreaming {
		t.Error("state should remain stateStreaming after ignored key")
	}
}

// ---- tokenMsg 处理 ----

func TestUpdate_TokenMsg_AppendToStreamBuf(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	m.state = stateStreaming
	m.tokenCh = make(chan string)
	m.doneCh = make(chan streamDoneMsg, 1)

	updated, _ := m.Update(tokenMsg{token: "修行"})
	m2 := updated.(Model)
	if !strings.Contains(m2.streamBuf.String(), "修行") {
		t.Errorf("streamBuf should contain token, got: %q", m2.streamBuf.String())
	}
}

func TestUpdate_TokenMsg_StopsAtEvalTag(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	m.state = stateStreaming
	m.tokenCh = make(chan string)
	m.doneCh = make(chan streamDoneMsg, 1)

	// 先发送普通 token
	updated, _ := m.Update(tokenMsg{token: "正文内容"})
	m2 := updated.(Model)
	// 再发送包含 eval_json 标记的 token
	updated2, _ := m2.Update(tokenMsg{token: "<eval_json>隐藏内容"})
	m3 := updated2.(Model)

	if m3.evalTagSeen != true {
		t.Error("evalTagSeen should be true after <eval_json> received")
	}
	// eval_json 之后的内容不应在 streamBuf 中
	if strings.Contains(m3.streamBuf.String(), "隐藏内容") {
		t.Error("content after <eval_json> should not appear in streamBuf")
	}
	// 标记前的正文应存在
	if !strings.Contains(m3.streamBuf.String(), "正文内容") {
		t.Error("content before <eval_json> should be in streamBuf")
	}
}

// ---- streamDoneMsg 处理 ----

func TestUpdate_StreamDone_ResetsState(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	m.state = stateStreaming
	m.streamBuf.WriteString("完整回复内容")

	updated, _ := m.Update(streamDoneMsg{newHistory: nil, err: nil})
	m2 := updated.(Model)

	if m2.state != stateIdle {
		t.Error("state should return to stateIdle after streamDone")
	}
	if m2.streamBuf.Len() != 0 {
		t.Error("streamBuf should be cleared after streamDone")
	}
	if !strings.Contains(m2.content, "完整回复内容") {
		t.Error("completed reply should be persisted to content")
	}
}

func TestUpdate_StreamDone_WithError(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	m.state = stateStreaming

	updated, _ := m.Update(streamDoneMsg{err: errTest("LLM 调用失败")})
	m2 := updated.(Model)

	if m2.lastErr == "" {
		t.Error("lastErr should be set on stream error")
	}
	if !strings.Contains(m2.content, "错误") {
		t.Error("error message should appear in content")
	}
}

// errTest 是 error 接口的简单实现，用于测试。
type errTest string

func (e errTest) Error() string { return string(e) }

// ---- View ----

func TestView_BeforeWindowSize_ReturnsLoadingText(t *testing.T) {
	m := New(Config{GuardianLabel: "护道人"})
	// width=0（未收到 WindowSizeMsg）时应返回加载提示
	v := m.View()
	if v == "" {
		t.Error("View should not return empty string")
	}
}

func TestView_AfterWindowSize_ContainsLabel(t *testing.T) {
	m := New(Config{GuardianLabel: "齐静春"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(Model)
	v := m2.View()
	if !strings.Contains(v, "齐静春") {
		t.Errorf("View should contain guardian label after resize, got partial: %q", v[:min2(len(v), 100)])
	}
}

// min2 避免与 built-in min 冲突
func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}
