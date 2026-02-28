// Package knowledge — 知识库语义检索。
package knowledge

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// Retriever 基于 Milvus 的语义检索器。
type Retriever struct {
	cfg Config
}

// SearchResult 检索结果。
type SearchResult struct {
	Text     string
	FilePath string
	Score    float32
}

// NewRetriever 创建检索器。
func NewRetriever(cfg Config) *Retriever {
	if cfg.Address == "" {
		cfg.Address = "localhost:19530"
	}
	if cfg.Collection == "" {
		cfg.Collection = "lizhu_knowledge"
	}
	return &Retriever{cfg: cfg}
}

// Search 对查询文本进行语义检索，返回 topK 个最相关的知识块。
// 当 Milvus 未启用时返回空结果，不报错。
func (r *Retriever) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if !r.cfg.Enabled {
		return nil, nil
	}

	// 向量化查询
	vecs, err := embedTexts(ctx, r.cfg.APIKey, r.cfg.BaseURL, r.cfg.EmbeddingModel, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	c, err := newMilvusClient(ctx, r.cfg.Address)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	// 确保 collection 已加载
	if err := c.LoadCollection(ctx, r.cfg.Collection, false); err != nil {
		return nil, fmt.Errorf("load collection: %w", err)
	}

	sp, err := entity.NewIndexHNSWSearchParam(64)
	if err != nil {
		return nil, fmt.Errorf("search param: %w", err)
	}

	results, err := c.Search(
		ctx,
		r.cfg.Collection,
		nil,
		"",
		[]string{fieldText, fieldFilePath},
		[]entity.Vector{entity.FloatVector(vecs[0])},
		fieldVector,
		entity.COSINE,
		topK,
		sp,
		client.WithSearchQueryConsistencyLevel(entity.ClEventually),
	)
	if err != nil {
		return nil, fmt.Errorf("milvus search: %w", err)
	}

	var out []SearchResult
	for _, res := range results {
		texts := res.Fields.GetColumn(fieldText)
		if texts == nil {
			continue
		}
		filePaths := res.Fields.GetColumn(fieldFilePath)
		if filePaths == nil {
			continue
		}
		for j := 0; j < res.ResultCount; j++ {
			text, _ := texts.(*entity.ColumnVarChar).ValueByIdx(j)
			fp, _ := filePaths.(*entity.ColumnVarChar).ValueByIdx(j)
			score := float32(0)
			if j < len(res.Scores) {
				score = res.Scores[j]
			}
			out = append(out, SearchResult{
				Text:     text,
				FilePath: fp,
				Score:    score,
			})
		}
	}
	return out, nil
}
