package episodic_test

import (
	"testing"

	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
)

func TestScoreToLevel(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{0, "初识"},
		{25, "初识"},
		{26, "熟用"},
		{50, "熟用"},
		{51, "精通"},
		{75, "精通"},
		{76, "宗师"},
		{100, "宗师"},
	}
	for _, c := range cases {
		got := episodic.ScoreToLevel(c.score)
		if got != c.want {
			t.Errorf("ScoreToLevel(%d) = %q, want %q", c.score, got, c.want)
		}
	}
}

// TestProfileDefaultValues 验证空 Profile 的默认字段。
func TestProfileDefaultValues(t *testing.T) {
	p := &episodic.Profile{
		UserID:        "test_user",
		ActivePath:    "both",
		GoLianqiLevel: 1,
		AILianqiLevel: 1,
		WufuLevel:     1,
	}
	if p.GoLianqiScore != 0 {
		t.Errorf("default GoLianqiScore should be 0, got %d", p.GoLianqiScore)
	}
	if p.ActivePath != "both" {
		t.Errorf("ActivePath = %q, want both", p.ActivePath)
	}
}
