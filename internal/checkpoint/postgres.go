// Package checkpoint 实现 Eino compose.CheckPointStore 接口，
// 以 PostgreSQL 作为持久化后端（约 30 行核心逻辑）。
package checkpoint

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore 实现 Eino CheckPointStore 接口。
type PostgresStore struct {
	pool *pgxpool.Pool
}

// New 创建 PostgresStore。
func New(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Get 根据 checkpointID 读取已保存的图执行状态。
func (s *PostgresStore) Get(ctx context.Context, checkpointID string) ([]byte, bool, error) {
	var data []byte
	err := s.pool.QueryRow(ctx,
		"SELECT data FROM checkpoints WHERE id = $1", checkpointID,
	).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("checkpoint get: %w", err)
	}
	return data, true, nil
}

// Set 保存图执行状态，id 冲突时覆盖。
func (s *PostgresStore) Set(ctx context.Context, checkpointID string, data []byte) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO checkpoints (id, data) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, updated_at = NOW()
	`, checkpointID, data)
	if err != nil {
		return fmt.Errorf("checkpoint set: %w", err)
	}
	return nil
}
