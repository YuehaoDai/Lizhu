package worldview

// Section 代表一个世界观配置节，对应一个 YAML 文件。
type Section struct {
	SectionID    string `yaml:"section_id"`
	SectionTitle string `yaml:"section_title"`
	Order        int    `yaml:"order"`
	// PathFilter 限定该节适用的练气士路径。
	// 空字符串表示所有路径均包含；"go" 仅 Go 路径；"ai" 仅 AI 路径。
	PathFilter string `yaml:"path_filter"`
	// PersonaID 限定该节仅在指定护道人人格下生效。
	// 空字符串表示所有人格均包含；非空时只有与配置匹配的人格才包含该节。
	PersonaID string `yaml:"persona_id"`
	Content   string `yaml:"content"`
	// EntrancePrompt 是专用于生成该人格出场场景描写的系统提示。
	// 仅在 persona YAML 文件中设置；供 Loader.LoadEntrancePrompt 读取。
	EntrancePrompt string `yaml:"entrance_prompt"`
}

// ActivePath 代表用户当前激活的练气士修行路径。
type ActivePath string

const (
	PathGo   ActivePath = "go"
	PathAI   ActivePath = "ai"
	PathBoth ActivePath = "both"
)
