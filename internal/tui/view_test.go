package tui

import (
	"strings"
	"testing"

	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
)

// ---- renderBarColored ----

func TestRenderBarColored_Zero(t *testing.T) {
	bar := renderBarColored(0)
	if !strings.HasPrefix(bar, "[") || !strings.HasSuffix(bar, "]") {
		t.Errorf("bar should be wrapped in [], got: %q", bar)
	}
}

func TestRenderBarColored_Full(t *testing.T) {
	bar := renderBarColored(100)
	if !strings.Contains(bar, "█") {
		t.Error("full bar should contain filled blocks")
	}
}

func TestRenderBarColored_Half(t *testing.T) {
	bar := renderBarColored(50)
	// 50分 → 5个填充 + 5个空白
	if !strings.Contains(bar, "█") || !strings.Contains(bar, "░") {
		t.Error("half bar should contain both filled and empty blocks")
	}
}

func TestRenderBarColored_Clamp(t *testing.T) {
	// 超出范围不应 panic
	_ = renderBarColored(-10)
	_ = renderBarColored(200)
}

// ---- welcomeText ----

func TestWelcomeText_WithNilProfile(t *testing.T) {
	text := welcomeText(nil, "齐静春")
	if !strings.Contains(text, "齐静春") {
		t.Error("welcome text should contain guardian label")
	}
	if !strings.Contains(text, "骊珠") {
		t.Error("welcome text should contain 骊珠")
	}
	if !strings.Contains(text, "/help") {
		t.Error("welcome text should mention /help command")
	}
}

func TestWelcomeText_WithExistingProfile(t *testing.T) {
	p := &episodic.Profile{
		GoLianqiScore:     45,
		GoLianqiLevel:     5,
		GoLianqiLevelName: "筑庐境",
		ActivePath:        "go",
	}
	text := welcomeText(p, "护道人")
	if !strings.Contains(text, "修行档案") {
		t.Error("welcome text should mention 修行档案 for existing profile")
	}
}

func TestWelcomeText_FirstTime(t *testing.T) {
	p := &episodic.Profile{
		GoLianqiScore: 0,
		AILianqiScore: 0,
	}
	text := welcomeText(p, "护道人")
	if !strings.Contains(text, "第一次") {
		t.Error("welcome text should say 第一次 for new profile")
	}
}

// ---- formatProfileTUI ----

func TestFormatProfileTUI_ContainsFields(t *testing.T) {
	p := &episodic.Profile{
		GoLianqiScore:     60,
		GoLianqiLevel:     6,
		GoLianqiLevelName: "崩山境",
		GoLianqiBranch:    "气盛",
		AILianqiScore:     30,
		AILianqiLevel:     3,
		AILianqiLevelName: "水银境",
		WufuScore:         20,
		WufuLevel:         2,
		WufuLevelName:     "水银境",
		ActivePath:        "both",
	}
	text := formatProfileTUI(p)
	if !strings.Contains(text, "Go练气士") {
		t.Error("profile should contain Go练气士")
	}
	if !strings.Contains(text, "AI练气士") {
		t.Error("profile should contain AI练气士")
	}
	if !strings.Contains(text, "武夫") {
		t.Error("profile should contain 武夫")
	}
	if !strings.Contains(text, "60") {
		t.Error("profile should contain score 60")
	}
}

func TestFormatProfileTUI_OnlyGoPath(t *testing.T) {
	p := &episodic.Profile{
		GoLianqiScore:     50,
		GoLianqiLevel:     5,
		GoLianqiLevelName: "筑庐境",
		ActivePath:        "go",
	}
	text := formatProfileTUI(p)
	if !strings.Contains(text, "Go练气士") {
		t.Error("should show Go练气士 for go path")
	}
}

// ---- helpText ----

func TestHelpText_ContainsCommands(t *testing.T) {
	h := helpText()
	commands := []string{"/assess", "/status", "/clear", "/quit", "/help"}
	for _, cmd := range commands {
		if !strings.Contains(h, cmd) {
			t.Errorf("helpText should contain %q", cmd)
		}
	}
}

func TestHelpText_ContainsShortcuts(t *testing.T) {
	h := helpText()
	if !strings.Contains(h, "Enter") {
		t.Error("helpText should mention Enter key")
	}
	if !strings.Contains(h, "Ctrl+C") {
		t.Error("helpText should mention Ctrl+C")
	}
}
