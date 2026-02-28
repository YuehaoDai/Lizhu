package worldview

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader 负责从目录中读取所有世界观 YAML 配置，
// 并按用户激活路径组合成最终的系统 Prompt。
type Loader struct {
	dir string
}

// NewLoader 创建一个 Loader，dir 为世界观 YAML 文件所在目录。
func NewLoader(dir string) *Loader {
	return &Loader{dir: dir}
}

// BuildSystemPrompt 读取目录下所有 YAML，按路径和人格过滤，
// 按 order 排序后拼接成完整系统 Prompt。
// personaID 为空时只包含 persona_id 为空的节（通用节）；
// personaID 非空时额外包含 persona_id 与之匹配的节。
// assess=true 时包含所有节（含 assess_only 节，用于 /assess 评估命令）；
// assess=false 时跳过 assess_only 节，LLM 不会收到 eval_json 相关指令。
func (l *Loader) BuildSystemPrompt(path ActivePath, personaID string, assess bool) (string, error) {
	sections, err := l.loadAll()
	if err != nil {
		return "", fmt.Errorf("worldview: load sections: %w", err)
	}

	// 过滤：同时满足 assess_only、path_filter 和 persona_id 三个维度
	filtered := make([]Section, 0, len(sections))
	for _, s := range sections {
		// assess_only 过滤：仅评估模式才包含
		if s.AssessOnly && !assess {
			continue
		}

		// persona_id 过滤：空=通用；非空=仅匹配指定人格
		if s.PersonaID != "" && s.PersonaID != personaID {
			continue
		}

		// path_filter 过滤
		if s.PathFilter == "" {
			filtered = append(filtered, s)
			continue
		}
		switch path {
		case PathBoth:
			filtered = append(filtered, s)
		case PathGo:
			if s.PathFilter == "go" {
				filtered = append(filtered, s)
			}
		case PathAI:
			if s.PathFilter == "ai" {
				filtered = append(filtered, s)
			}
		}
	}

	// 按 order 排序
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Order < filtered[j].Order
	})

	var sb strings.Builder
	for _, s := range filtered {
		sb.WriteString(strings.TrimSpace(s.Content))
		sb.WriteString("\n\n")
	}

	if assess {
		// 评估模式：强制 LLM 立即输出完整评估结构，不得以"证据不足"为由回避
		sb.WriteString("【当前模式：强制修行评估（Mode A）】修行者已主动请求完整境界评估。你必须立即根据本次对话内容和历史档案给出完整的境界评估，按照上方【输出格式规范】中 Mode A 的全部模块依次输出，并在末尾附上完整的 eval_json 块。禁止要求修行者补充更多信息；若证据不足，直接标注[信息不足，按保守估计]并给出合理区间。\n\n")
	} else {
		// 普通对话模式：禁止生成任何结构化评分或 JSON 块
		sb.WriteString("【当前模式：护道对话】请以护道人人格自然回应修行者，不要输出任何结构化评分、JSON 块或格式化模块。\n\n")
	}

	return strings.TrimSpace(sb.String()), nil
}

// LoadEntrancePrompt 返回指定人格的出场描写生成系统提示。
// 若该人格未配置 entrance_prompt，返回空字符串。
func (l *Loader) LoadEntrancePrompt(personaID string) (string, error) {
	if personaID == "" {
		return "", nil
	}
	sections, err := l.loadAll()
	if err != nil {
		return "", fmt.Errorf("worldview: load sections: %w", err)
	}
	for _, s := range sections {
		if s.PersonaID == personaID && s.EntrancePrompt != "" {
			return s.EntrancePrompt, nil
		}
	}
	return "", nil
}

// loadAll 读取目录下所有 .yaml / .yml 文件并解析为 Section。
func (l *Loader) loadAll() ([]Section, error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", l.dir, err)
	}

	var sections []Section
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		fullPath := filepath.Join(l.dir, name)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", fullPath, err)
		}
		var s Section
		if err := yaml.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse %q: %w", fullPath, err)
		}
		if s.SectionID == "" {
			continue // 跳过无效文件
		}
		sections = append(sections, s)
	}
	return sections, nil
}
