// Package knowledge 实现知识库文件入库功能（Milvus RAG，可选）。
// 当 Milvus 未配置时，该包提供 stub 实现，不影响核心功能。
package knowledge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config Milvus 知识库配置。
type Config struct {
	Enabled        bool
	Address        string
	Collection     string
	EmbeddingModel string
	APIKey         string // 用于 embedding API
}

// Ingester 负责将本地文件分块并入库到 Milvus。
type Ingester struct {
	cfg Config
}

// New 创建 Ingester。若 cfg.Enabled 为 false，则所有操作均为 no-op。
func New(cfg Config) *Ingester {
	return &Ingester{cfg: cfg}
}

// IngestFile 读取文件内容，分块后嵌入向量并存入 Milvus。
func (i *Ingester) IngestFile(ctx context.Context, path string) error {
	if !i.cfg.Enabled {
		fmt.Printf("[知识库] Milvus 未启用，跳过索引文件: %s\n", path)
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	chunks := chunkText(string(data), ext)

	fmt.Printf("[知识库] 文件 %s 已分为 %d 个片段，准备入库...\n", filepath.Base(path), len(chunks))

	// TODO: Phase 2 接入 Eino milvus2 Indexer
	// indexer, err := milvus2.NewIndexer(ctx, &milvus2.IndexerConfig{...})
	// docs := chunksToDocuments(chunks, path)
	// _, err = indexer.Store(ctx, docs)

	fmt.Printf("[知识库] Milvus 集成将在二期实现，当前仅统计分块数量。\n")
	return nil
}

// chunkText 将文本按段落或固定大小分块。
func chunkText(content, ext string) []string {
	const maxChunkSize = 512

	var chunks []string
	switch ext {
	case ".md", ".txt":
		// 按段落分块
		paragraphs := strings.Split(content, "\n\n")
		for _, p := range paragraphs {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if len(p) <= maxChunkSize {
				chunks = append(chunks, p)
			} else {
				// 超长段落按行再分
				for _, line := range strings.Split(p, "\n") {
					if line = strings.TrimSpace(line); line != "" {
						chunks = append(chunks, line)
					}
				}
			}
		}
	default:
		// 代码文件：按固定大小滑动窗口分块
		for start := 0; start < len(content); start += maxChunkSize / 2 {
			end := start + maxChunkSize
			if end > len(content) {
				end = len(content)
			}
			chunks = append(chunks, content[start:end])
			if end == len(content) {
				break
			}
		}
	}
	return chunks
}
