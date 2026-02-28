// Package tui 实现骊珠 Bubble Tea 终端界面。
package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap 定义全局快捷键。
type KeyMap struct {
	Send    key.Binding
	Quit    key.Binding
	Clear   key.Binding
	Status  key.Binding
	Assess  key.Binding
	Help    key.Binding
	ScrollUp   key.Binding
	ScrollDown key.Binding
}

// DefaultKeyMap 返回默认快捷键绑定。
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "发送"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "ctrl+q"),
			key.WithHelp("ctrl+c", "退出"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "清空历史"),
		),
		Status: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "修行档案"),
		),
		Assess: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "境界评估"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+?"),
			key.WithHelp("ctrl+?", "帮助"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "上翻"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "下翻"),
		),
	}
}
