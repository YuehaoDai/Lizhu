package guardian

import (
	"encoding/json"
	"regexp"
	"strings"
)

// EvalResult 是 LLM 在修行档案 JSON 块中输出的结构化评估结果。
type EvalResult struct {
	GoLianqiScore     int    `json:"go_lianqi_score"`
	GoLianqiLevel     int    `json:"go_lianqi_level"`
	GoLianqiLevelName string `json:"go_lianqi_level_name"`
	GoLianqiBranch    string `json:"go_lianqi_branch"`

	AILianqiScore     int    `json:"ai_lianqi_score"`
	AILianqiLevel     int    `json:"ai_lianqi_level"`
	AILianqiLevelName string `json:"ai_lianqi_level_name"`
	AILianqiBranch    string `json:"ai_lianqi_branch"`

	WufuScore     int    `json:"wufu_score"`
	WufuLevel     int    `json:"wufu_level"`
	WufuLevelName string `json:"wufu_level_name"`

	ToolMasteryUpdates []ToolMasteryUpdate `json:"tool_mastery_updates"`
	XinMoIdentified    []string            `json:"xin_mo_identified"`
	SessionSummary     string              `json:"session_summary"`
}

// ToolMasteryUpdate 代表单条工具掌握度更新。
type ToolMasteryUpdate struct {
	Tool     string `json:"tool"`
	Category string `json:"category"`
	Score    int    `json:"score"`
	Evidence string `json:"evidence"`
}

var evalJSONRe = regexp.MustCompile(`(?s)<eval_json>\s*(\{.*?\})\s*</eval_json>`)

// ParseEvalResult 从 LLM 响应文本中提取并解析 <eval_json>...</eval_json> 块。
// 如果未找到，返回 nil, nil（不视为错误）。
func ParseEvalResult(response string) (*EvalResult, error) {
	matches := evalJSONRe.FindStringSubmatch(response)
	if len(matches) < 2 {
		return nil, nil
	}
	raw := strings.TrimSpace(matches[1])
	var result EvalResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	return &result, nil
}
