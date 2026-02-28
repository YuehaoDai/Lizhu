package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/YuehaoDai/lizhu/internal/agent/librarian"
	"github.com/YuehaoDai/lizhu/internal/knowledge"
	"github.com/YuehaoDai/lizhu/internal/memory/episodic"
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
		return runNoteList(cmd.Context())
	},
}

func init() {
	noteCmd.AddCommand(noteAddCmd)
	noteCmd.AddCommand(noteListCmd)
}

func runNoteAdd(ctx context.Context, filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("文件不存在: %s", filePath)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("解析文件路径失败: %w", err)
	}

	cfg := buildIngesterConfig()
	ingester := knowledge.New(cfg)

	fmt.Printf("正在处理文件：%s\n", absPath)

	// 读取文件内容，供 Librarian 提炼摘要
	fileData, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// Librarian 提炼摘要（需要 API Key）
	summary := ""
	apiKey := viper.GetString("llm.api_key")
	if apiKey != "" {
		fmt.Println("正在提炼笔记摘要...")
		lib, libErr := librarian.New(ctx, librarian.Config{
			APIKey:  apiKey,
			Model:   viper.GetString("llm.model"),
			BaseURL: viper.GetString("llm.base_url"),
		})
		if libErr != nil {
			fmt.Fprintf(os.Stderr, "[警告] 知识整理官初始化失败: %v\n", libErr)
		} else {
			s, sumErr := lib.Summarize(ctx, absPath, string(fileData))
			if sumErr != nil {
				fmt.Fprintf(os.Stderr, "[警告] 摘要提炼失败: %v\n", sumErr)
			} else {
				summary = s
				fmt.Println("摘要提炼完成。")
			}
		}
	}

	result, err := ingester.IngestFile(ctx, absPath)
	if err != nil {
		return fmt.Errorf("文件入库失败: %w", err)
	}

	// 若 Milvus 未启用，result 为 nil，跳过 DB 记录
	if result != nil && repo != nil {
		kf := &episodic.KnowledgeFile{
			UserID:     "default",
			FilePath:   absPath,
			FileHash:   result.FileHash,
			ChunkCount: result.ChunkCount,
			Summary:    summary,
		}
		if err := repo.UpsertKnowledgeFile(ctx, kf); err != nil {
			fmt.Fprintf(os.Stderr, "[警告] 记录知识库文件失败: %v\n", err)
		}
	} else if result == nil && repo != nil && summary != "" {
		// Milvus 未启用但有摘要，仍记录到 DB（chunk_count=0）
		kf := &episodic.KnowledgeFile{
			UserID:     "default",
			FilePath:   absPath,
			FileHash:   fmt.Sprintf("%x", fileData[:min(32, len(fileData))]),
			ChunkCount: 0,
			Summary:    summary,
		}
		if err := repo.UpsertKnowledgeFile(ctx, kf); err != nil {
			fmt.Fprintf(os.Stderr, "[警告] 记录知识库文件失败: %v\n", err)
		}
	}

	fmt.Printf("完成：%s 已处理。\n", filepath.Base(absPath))
	if !cfg.Enabled {
		fmt.Println("\n[提示] 如需启用 RAG 知识库，请：")
		fmt.Println("  1. 运行 docker-compose up -d 确保 Milvus 已启动")
		fmt.Println("  2. 在 lizhu.yaml 中设置 milvus.enabled: true")
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func runNoteList(ctx context.Context) error {
	if repo == nil {
		return fmt.Errorf("数据库未初始化")
	}

	files, err := repo.ListKnowledgeFiles(ctx, "default")
	if err != nil {
		return fmt.Errorf("查询知识库文件失败: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("知识库暂无已索引文件。使用 `lizhu note add <文件>` 添加。")
		return nil
	}

	fmt.Printf("\n已索引文件（共 %d 个）：\n\n", len(files))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "文件路径\t分块数\t入库时间")
	fmt.Fprintln(w, "--------\t------\t--------")
	for _, f := range files {
		fmt.Fprintf(w, "%s\t%d\t%s\n",
			f.FilePath,
			f.ChunkCount,
			f.IndexedAt.Local().Format(time.DateTime),
		)
	}
	w.Flush()
	return nil
}
