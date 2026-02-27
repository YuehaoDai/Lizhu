package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YuehaoDai/lizhu/internal/knowledge"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "管理知识库笔记",
	Long:  `管理骊珠知识库：将笔记、代码文件索引到向量数据库，供护道人对话时调用。`,
}

var noteAddCmd = &cobra.Command{
	Use:   "add <文件路径>",
	Short: "将文件入库到知识库",
	Long: `将指定文件（Markdown 笔记、Go 代码等）分块嵌入并存入知识库。
知识库内容将在护道人对话时自动检索，丰富评估上下文。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNoteAdd(cmd.Context(), args[0])
	},
}

var noteListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已索引的文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("[知识库] 文件列表功能将在 Milvus 集成后实现（二期）")
		return nil
	},
}

func init() {
	noteCmd.AddCommand(noteAddCmd)
	noteCmd.AddCommand(noteListCmd)
}

func runNoteAdd(ctx context.Context, filePath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("文件不存在: %s", filePath)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("解析文件路径失败: %w", err)
	}

	ingesterCfg := knowledge.Config{
		Enabled:        viper.GetBool("milvus.enabled"),
		Address:        viper.GetString("milvus.address"),
		Collection:     viper.GetString("milvus.collection"),
		EmbeddingModel: viper.GetString("milvus.embedding_model"),
		APIKey:         viper.GetString("llm.api_key"),
	}
	if ingesterCfg.Address == "" {
		ingesterCfg.Address = "localhost:19530"
	}
	if ingesterCfg.Collection == "" {
		ingesterCfg.Collection = "lizhu_knowledge"
	}

	ingester := knowledge.New(ingesterCfg)
	fmt.Printf("正在处理文件：%s\n", absPath)

	if err := ingester.IngestFile(ctx, absPath); err != nil {
		return fmt.Errorf("文件入库失败: %w", err)
	}

	fmt.Printf("完成：%s 已处理。\n", filepath.Base(absPath))
	if !ingesterCfg.Enabled {
		fmt.Println("\n[提示] 如需启用 RAG 知识库，请：")
		fmt.Println("  1. 在 docker-compose.yaml 中启动 Milvus")
		fmt.Println("  2. 在 lizhu.yaml 中设置 milvus.enabled: true")
	}
	return nil
}
