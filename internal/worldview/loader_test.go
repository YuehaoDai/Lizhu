package worldview_test

import (
	"strings"
	"testing"

	"github.com/daiyh/lizhu/internal/worldview"
)

const testDataDir = "testdata/worldview"

func TestBuildSystemPrompt_Both(t *testing.T) {
	loader := worldview.NewLoader(testDataDir)
	prompt, err := loader.BuildSystemPrompt(worldview.PathBoth)
	if err != nil {
		t.Fatalf("BuildSystemPrompt(both) error: %v", err)
	}

	// PathBoth 应包含所有节（base + go + ai + common）
	wantContains := []string{
		"你是护道人",
		"Go练气士十五境内容",
		"AI练气士十五境内容",
		"三教诸子内容",
	}
	for _, want := range wantContains {
		if !strings.Contains(prompt, want) {
			t.Errorf("PathBoth: prompt should contain %q, got:\n%s", want, prompt)
		}
	}
}

func TestBuildSystemPrompt_GoOnly(t *testing.T) {
	loader := worldview.NewLoader(testDataDir)
	prompt, err := loader.BuildSystemPrompt(worldview.PathGo)
	if err != nil {
		t.Fatalf("BuildSystemPrompt(go) error: %v", err)
	}

	if !strings.Contains(prompt, "Go练气士十五境内容") {
		t.Error("PathGo: should contain Go levels")
	}
	if strings.Contains(prompt, "AI练气士十五境内容") {
		t.Error("PathGo: should NOT contain AI levels")
	}
	// 通用节（无 path_filter）应始终包含
	if !strings.Contains(prompt, "你是护道人") {
		t.Error("PathGo: should contain base section")
	}
	if !strings.Contains(prompt, "三教诸子内容") {
		t.Error("PathGo: should contain common section")
	}
}

func TestBuildSystemPrompt_AIOnly(t *testing.T) {
	loader := worldview.NewLoader(testDataDir)
	prompt, err := loader.BuildSystemPrompt(worldview.PathAI)
	if err != nil {
		t.Fatalf("BuildSystemPrompt(ai) error: %v", err)
	}

	if !strings.Contains(prompt, "AI练气士十五境内容") {
		t.Error("PathAI: should contain AI levels")
	}
	if strings.Contains(prompt, "Go练气士十五境内容") {
		t.Error("PathAI: should NOT contain Go levels")
	}
}

func TestBuildSystemPrompt_OrderRespected(t *testing.T) {
	loader := worldview.NewLoader(testDataDir)
	prompt, err := loader.BuildSystemPrompt(worldview.PathBoth)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// base (order=0) 应出现在 Go (order=1) 之前
	baseIdx := strings.Index(prompt, "你是护道人")
	goIdx := strings.Index(prompt, "Go练气士十五境内容")
	if baseIdx < 0 || goIdx < 0 {
		t.Fatal("sections not found in prompt")
	}
	if baseIdx > goIdx {
		t.Errorf("base section (order=0) should appear before go section (order=1)")
	}
}

func TestBuildSystemPrompt_InvalidDir(t *testing.T) {
	loader := worldview.NewLoader("/nonexistent/path")
	_, err := loader.BuildSystemPrompt(worldview.PathBoth)
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}
}
