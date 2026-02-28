package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YuehaoDai/lizhu/internal/agent/guardian"
	"github.com/YuehaoDai/lizhu/internal/knowledge"
	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
	"github.com/YuehaoDai/lizhu/internal/storage"
	"github.com/YuehaoDai/lizhu/internal/worldview"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// buildIngesterConfig 从 viper 读取知识库配置（note.go 和 root.go 共用）。
func buildIngesterConfig() knowledge.Config {
	cfg := knowledge.Config{
		Enabled:        viper.GetBool("milvus.enabled"),
		Address:        viper.GetString("milvus.address"),
		Collection:     viper.GetString("milvus.collection"),
		EmbeddingModel: viper.GetString("milvus.embedding_model"),
		BaseURL:        viper.GetString("llm.base_url"),
		APIKey:         viper.GetString("llm.api_key"),
	}
	if cfg.Address == "" {
		cfg.Address = "localhost:19530"
	}
	if cfg.Collection == "" {
		cfg.Collection = "lizhu_knowledge"
	}
	return cfg
}

var (
	cfgFile string
	dbPool  *pgxpool.Pool
	repo    *episodic.Repository
)

var rootCmd = &cobra.Command{
	Use:   "lizhu",
	Short: "骊珠 — Go & AI 开发者智能护道系统",
	Long: `骊珠 (Lizhu) 是基于《剑来》修行世界观设计的 Go & AI 开发者智能护道系统。
护道人将评估你的修行境界，规划破境路径，守护你不走弯路。`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "help" || cmd.Name() == "lizhu" {
			return nil
		}
		return initDependencies(cmd.Context())
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if dbPool != nil {
			dbPool.Close()
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径（默认：./lizhu.yaml 或 ~/.lizhu/config.yaml）")
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(noteCmd)
	rootCmd.AddCommand(statusCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("lizhu")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, ".lizhu"))
		}
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "[提示] 未读取到配置文件，将使用默认值（%v）\n", err)
	}
}

func initDependencies(ctx context.Context) error {
	dbCfg := storage.Config{
		Host:     viper.GetString("database.host"),
		Port:     viper.GetInt("database.port"),
		Name:     viper.GetString("database.name"),
		User:     viper.GetString("database.user"),
		Password: viper.GetString("database.password"),
		SSLMode:  viper.GetString("database.ssl_mode"),
	}
	// 填充默认值
	if dbCfg.Host == "" {
		dbCfg.Host = "localhost"
	}
	if dbCfg.Port == 0 {
		dbCfg.Port = 5432
	}
	if dbCfg.Name == "" {
		dbCfg.Name = "lizhu"
	}
	if dbCfg.User == "" {
		dbCfg.User = "lizhu"
	}
	if dbCfg.Password == "" {
		dbCfg.Password = "lizhu"
	}

	if err := storage.RunMigrations(dbCfg); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	pool, err := storage.Connect(ctx, dbCfg)
	if err != nil {
		return fmt.Errorf("数据库连接失败（请确认 docker-compose 已启动）: %w", err)
	}
	dbPool = pool
	repo = episodic.New(pool)
	return nil
}

func newGuardianAgent(ctx context.Context) (*guardian.Agent, error) {
	wvPath := viper.GetString("worldview.path")
	if wvPath == "" {
		wvPath = "./configs/worldview"
	}

	activePath := toActivePath(viper.GetString("user.active_path"))

	historyWindow := viper.GetInt("session.history_window")
	if historyWindow == 0 {
		historyWindow = 5
	}

	userName := viper.GetString("user.name")
	if userName == "" {
		userName = "修行者"
	}

	model := viper.GetString("llm.model")
	if model == "" {
		model = "gpt-4o"
	}

	apiKey := viper.GetString("llm.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("未配置 LLM API Key，请在 lizhu.yaml 中设置 llm.api_key")
	}

	cfg := guardian.Config{
		LLMProvider:   viper.GetString("llm.provider"),
		APIKey:        apiKey,
		Model:         model,
		BaseURL:       viper.GetString("llm.base_url"),
		WorldViewDir:  wvPath,
		ActivePath:    activePath,
		UserID:        "default",
		UserName:      userName,
		PersonaID:     viper.GetString("guardian.persona_id"),
		PersonaName:   viper.GetString("guardian.persona_name"),
		HistoryWindow: historyWindow,
		KnowledgeCfg:  buildIngesterConfig(),
	}
	return guardian.New(ctx, cfg, repo)
}

func toActivePath(s string) worldview.ActivePath {
	switch s {
	case "go":
		return worldview.PathGo
	case "ai":
		return worldview.PathAI
	default:
		return worldview.PathBoth
	}
}
