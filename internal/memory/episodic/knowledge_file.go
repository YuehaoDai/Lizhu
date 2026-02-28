package episodic

import (
	"context"
	"time"
)

// KnowledgeFile 代表一条已索引的知识库文件记录。
type KnowledgeFile struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	FilePath   string    `json:"file_path"`
	FileHash   string    `json:"file_hash"`
	ChunkCount int       `json:"chunk_count"`
	Summary    string    `json:"summary"`
	IndexedAt  time.Time `json:"indexed_at"`
}

// UpsertKnowledgeFile 插入或更新知识库文件记录（以 user_id+file_path 为唯一键）。
func (r *Repository) UpsertKnowledgeFile(ctx context.Context, kf *KnowledgeFile) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO knowledge_files (user_id, file_path, file_hash, chunk_count, summary)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, file_path) DO UPDATE SET
		    file_hash   = EXCLUDED.file_hash,
		    chunk_count = EXCLUDED.chunk_count,
		    summary     = EXCLUDED.summary,
		    indexed_at  = NOW()
	`, kf.UserID, kf.FilePath, kf.FileHash, kf.ChunkCount, kf.Summary)
	return err
}

// ListKnowledgeFiles 列出指定用户的所有已索引文件，按入库时间降序。
func (r *Repository) ListKnowledgeFiles(ctx context.Context, userID string) ([]*KnowledgeFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, file_path, file_hash, chunk_count, summary, indexed_at
		FROM knowledge_files
		WHERE user_id = $1
		ORDER BY indexed_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*KnowledgeFile
	for rows.Next() {
		var kf KnowledgeFile
		if err := rows.Scan(
			&kf.ID, &kf.UserID, &kf.FilePath, &kf.FileHash,
			&kf.ChunkCount, &kf.Summary, &kf.IndexedAt,
		); err != nil {
			return nil, err
		}
		files = append(files, &kf)
	}
	return files, rows.Err()
}
