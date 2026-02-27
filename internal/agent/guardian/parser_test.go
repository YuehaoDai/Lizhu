package guardian_test

import (
	"testing"

	"github.com/daiyh/lizhu/internal/agent/guardian"
)

const sampleResponse = `
【修行总评】
Go练气士：45/100；第5境·筑庐境；主修分支：剑修

【证据】
1. 用户能独立搭建 HTTP 服务
2. 理解 error 流转但尚未涉及并发内观

【今日修行】
代码任务：实现一个带超时的 HTTP 客户端封装

<eval_json>
{
  "go_lianqi_score": 45,
  "go_lianqi_level": 5,
  "go_lianqi_level_name": "筑庐境",
  "go_lianqi_branch": "剑修",
  "ai_lianqi_score": 0,
  "ai_lianqi_level": 0,
  "ai_lianqi_level_name": "",
  "ai_lianqi_branch": "",
  "wufu_score": 30,
  "wufu_level": 2,
  "wufu_level_name": "水银境",
  "tool_mastery_updates": [
    {"tool": "Go", "category": "primary_weapon", "score": 45, "evidence": "能独立搭建HTTP服务"}
  ],
  "xin_mo_identified": ["过早优化"],
  "session_summary": "用户展示了基础Go工程能力，已达筑庐境，主要瓶颈在并发内观。"
}
</eval_json>
`

func TestParseEvalResult_Valid(t *testing.T) {
	result, err := guardian.ParseEvalResult(sampleResponse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.GoLianqiScore != 45 {
		t.Errorf("GoLianqiScore = %d, want 45", result.GoLianqiScore)
	}
	if result.GoLianqiLevel != 5 {
		t.Errorf("GoLianqiLevel = %d, want 5", result.GoLianqiLevel)
	}
	if result.GoLianqiLevelName != "筑庐境" {
		t.Errorf("GoLianqiLevelName = %q, want 筑庐境", result.GoLianqiLevelName)
	}
	if result.GoLianqiBranch != "剑修" {
		t.Errorf("GoLianqiBranch = %q, want 剑修", result.GoLianqiBranch)
	}
	if result.WufuScore != 30 {
		t.Errorf("WufuScore = %d, want 30", result.WufuScore)
	}
	if result.AILianqiScore != 0 {
		t.Errorf("AILianqiScore = %d, want 0 (not evaluated)", result.AILianqiScore)
	}
	if len(result.ToolMasteryUpdates) != 1 {
		t.Errorf("ToolMasteryUpdates len = %d, want 1", len(result.ToolMasteryUpdates))
	} else {
		u := result.ToolMasteryUpdates[0]
		if u.Tool != "Go" {
			t.Errorf("tool name = %q, want Go", u.Tool)
		}
		if u.Score != 45 {
			t.Errorf("tool score = %d, want 45", u.Score)
		}
	}
	if len(result.XinMoIdentified) != 1 || result.XinMoIdentified[0] != "过早优化" {
		t.Errorf("XinMoIdentified = %v, want [过早优化]", result.XinMoIdentified)
	}
	if result.SessionSummary == "" {
		t.Error("SessionSummary should not be empty")
	}
}

func TestParseEvalResult_NoJSON(t *testing.T) {
	result, err := guardian.ParseEvalResult("这是一段没有eval_json块的普通文本")
	if err != nil {
		t.Fatalf("unexpected error for missing block: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no eval_json block present")
	}
}

func TestParseEvalResult_MalformedJSON(t *testing.T) {
	bad := `<eval_json>{ "go_lianqi_score": "not_a_number" }</eval_json>`
	_, err := guardian.ParseEvalResult(bad)
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestParseEvalResult_EmptyBlock(t *testing.T) {
	result, err := guardian.ParseEvalResult("<eval_json></eval_json>")
	if err == nil && result != nil {
		t.Error("expected nil or error for empty block")
	}
}
