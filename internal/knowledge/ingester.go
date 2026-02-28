// Package knowledge 实现知识库文件入库功能（Milvus RAG，可选）。
// 当 Milvus 未配置时，该包提供 stub 实现，不影响核心功能。
package knowledge

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// Config Milvus 知识库配置。
type Config struct {
	Enabled        bool
	Address        string
	Collection     string
	EmbeddingModel string
	BaseURL        string // OpenAI 兼容接口 base URL（空则用官方）
	APIKey         string // 用于 embedding API
}

// Ingester 负责将本地文件分块并入库到 Milvus。
type Ingester struct {
	cfg Config
}

// IngestResult 入库结果。
type IngestResult struct {
	FilePath   string
	FileHash   string
	ChunkCount int
}

// New 创建 Ingester。若 cfg.Enabled 为 false，则所有操作均为 no-op。
func New(cfg Config) *Ingester {
	if cfg.Address == "" {
		cfg.Address = "localhost:19530"
	}
	if cfg.Collection == "" {
		cfg.Collection = "lizhu_knowledge"
	}
	return &Ingester{cfg: cfg}
}

// IngestFile 读取文件内容，分块 → 向量化 → 存入 Milvus。
// 返回入库结果（含 hash 与分块数），供调用方持久化到 PostgreSQL。
func (i *Ingester) IngestFile(ctx context.Context, path string) (*IngestResult, error) {
	if !i.cfg.Enabled {
		fmt.Printf("[知识库] Milvus 未启用，跳过索引文件: %s\n", path)
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	hash := fmt.Sprintf("%x", md5.Sum(data))
	ext := strings.ToLower(filepath.Ext(path))
	chunks := chunkText(string(data), ext)

	if len(chunks) == 0 {
		return &IngestResult{FilePath: path, FileHash: hash, ChunkCount: 0}, nil
	}

	fmt.Printf("[知识库] 文件 %s → %d 个片段，正在向量化...\n", filepath.Base(path), len(chunks))

	vecs, err := embedTexts(ctx, i.cfg.APIKey, i.cfg.BaseURL, i.cfg.EmbeddingModel, chunks)
	if err != nil {
		return nil, fmt.Errorf("embedding: %w", err)
	}

	fmt.Printf("[知识库] 向量化完成，正在写入 Milvus...\n")

	if err := i.storeVectors(ctx, path, chunks, vecs); err != nil {
		return nil, fmt.Errorf("store vectors: %w", err)
	}

	fmt.Printf("[知识库] 文件 %s 已成功入库（%d 个片段）。\n", filepath.Base(path), len(chunks))
	return &IngestResult{FilePath: path, FileHash: hash, ChunkCount: len(chunks)}, nil
}

// storeVectors 将文本与向量批量写入 Milvus。
func (i *Ingester) storeVectors(ctx context.Context, filePath string, chunks []string, vecs [][]float32) error {
	c, err := newMilvusClient(ctx, i.cfg.Address)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := ensureCollection(ctx, c, i.cfg.Collection); err != nil {
		return err
	}

	// 截断过长的文本块防止超出 VarChar 限制
	truncatedChunks := make([]string, len(chunks))
	for j, ch := range chunks {
		if len(ch) > maxTextLen {
			truncatedChunks[j] = ch[:maxTextLen]
		} else {
			truncatedChunks[j] = ch
		}
	}

	// 组装列数据
	filePaths := make([]string, len(chunks))
	chunkIdxs := make([]int32, len(chunks))
	for j := range chunks {
		filePaths[j] = filePath
		chunkIdxs[j] = int32(j)
	}

	cols := []entity.Column{
		entity.NewColumnVarChar(fieldFilePath, filePaths),
		entity.NewColumnInt32(fieldChunkIdx, chunkIdxs),
		entity.NewColumnVarChar(fieldText, truncatedChunks),
		entity.NewColumnFloatVector(fieldVector, embeddingDim, vecs),
	}

	_, err = c.Insert(ctx, i.cfg.Collection, "", cols...)
	if err != nil {
		return fmt.Errorf("milvus insert: %w", err)
	}

	return c.Flush(ctx, i.cfg.Collection, false)
}

// DeleteByFilePath 删除某文件的全部向量（重新入库前调用）。
func (i *Ingester) DeleteByFilePath(ctx context.Context, filePath string) error {
	if !i.cfg.Enabled {
		return nil
	}
	c, err := newMilvusClient(ctx, i.cfg.Address)
	if err != nil {
		return err
	}
	defer c.Close()

	expr := fmt.Sprintf(`%s == "%s"`, fieldFilePath, strings.ReplaceAll(filePath, `"`, `\"`))
	return c.Delete(ctx, i.cfg.Collection, "", expr)
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
