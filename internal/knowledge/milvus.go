// Package knowledge — Milvus Collection 初始化与管理。
package knowledge

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	fieldID       = "id"
	fieldFilePath = "file_path"
	fieldChunkIdx = "chunk_idx"
	fieldText     = "text"
	fieldVector   = "vector"
	maxTextLen    = 2048
)

// newMilvusClient 创建 Milvus gRPC 客户端。
func newMilvusClient(ctx context.Context, address string) (client.Client, error) {
	c, err := client.NewDefaultGrpcClient(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("milvus connect %s: %w", address, err)
	}
	return c, nil
}

// ProbeMilvus 尝试建立连接并立即关闭，用于启动时的连通性探测。
// ctx 应携带超时，调用方负责设置。
func ProbeMilvus(ctx context.Context, address string) error {
	if address == "" {
		address = "localhost:19530"
	}
	c, err := newMilvusClient(ctx, address)
	if err != nil {
		return err
	}
	c.Close()
	return nil
}

// ensureCollection 确保 collection 存在；不存在则创建并建索引。
func ensureCollection(ctx context.Context, c client.Client, collName string) error {
	exists, err := c.HasCollection(ctx, collName)
	if err != nil {
		return fmt.Errorf("check collection: %w", err)
	}
	if exists {
		return c.LoadCollection(ctx, collName, false)
	}

	schema := &entity.Schema{
		CollectionName: collName,
		Description:    "骊珠知识库",
		Fields: []*entity.Field{
			{
				Name:       fieldID,
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     true,
			},
			{
				Name:     fieldFilePath,
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "512",
				},
			},
			{
				Name:     fieldChunkIdx,
				DataType: entity.FieldTypeInt32,
			},
			{
				Name:     fieldText,
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": fmt.Sprintf("%d", maxTextLen),
				},
			},
			{
				Name:     fieldVector,
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", embeddingDim),
				},
			},
		},
	}

	if err := c.CreateCollection(ctx, schema, 1); err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	// HNSW 索引
	idx, err := entity.NewIndexHNSW(entity.COSINE, 8, 64)
	if err != nil {
		return fmt.Errorf("create index param: %w", err)
	}
	if err := c.CreateIndex(ctx, collName, fieldVector, idx, false); err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	return c.LoadCollection(ctx, collName, false)
}
