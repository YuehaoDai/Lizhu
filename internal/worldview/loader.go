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

// BuildSystemPrompt 读取目录下所有 YAML，过滤路径，
// 按 order 排序后拼接成完整系统 Prompt。
func (l *Loader) BuildSystemPrompt(path ActivePath) (string, error) {
	sections, err := l.loadAll()
	if err != nil {
		return "", fmt.Errorf("worldview: load sections: %w", err)
	}

	// 过滤：保留 path_filter 为空（通用）或与激活路径匹配的节
	filtered := make([]Section, 0, len(sections))
	for _, s := range sections {
		if s.PathFilter == "" {
			filtered = append(filtered, s)
			continue
		}
		switch path {
		case PathBoth:
			// 两条并修：包含所有节
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
	return strings.TrimSpace(sb.String()), nil
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
