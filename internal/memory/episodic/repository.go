package episodic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Profile 代表用户当前修行档案。
type Profile struct {
	UserID     string    `json:"user_id"`
	ActivePath string    `json:"active_path"`
	// Go开发练气士
	GoLianqiScore     int    `json:"go_lianqi_score"`
	GoLianqiLevel     int    `json:"go_lianqi_level"`
	GoLianqiLevelName string `json:"go_lianqi_level_name"`
	GoLianqiBranch    string `json:"go_lianqi_branch"`
	// AI应用练气士
	AILianqiScore     int    `json:"ai_lianqi_score"`
	AILianqiLevel     int    `json:"ai_lianqi_level"`
	AILianqiLevelName string `json:"ai_lianqi_level_name"`
	AILianqiBranch    string `json:"ai_lianqi_branch"`
	// 武夫
	WufuScore     int    `json:"wufu_score"`
	WufuLevel     int    `json:"wufu_level"`
	WufuLevelName string `json:"wufu_level_name"`
	// 心魔 & 破境条件
	XinMoRecords           []string        `json:"xin_mo_records"`
	BreakthroughConditions json.RawMessage `json:"breakthrough_conditions"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

// Session 代表一次历史会话记录。
type Session struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Summary          string    `json:"summary"`
	GoLianqiScore    int       `json:"go_lianqi_score"`
	AILianqiScore    int       `json:"ai_lianqi_score"`
	WufuScore        int       `json:"wufu_score"`
	XinMoIdentified  []string  `json:"xin_mo_identified"`
	BreakthroughUpdates json.RawMessage `json:"breakthrough_updates"`
	RawResponse      string    `json:"raw_response"`
	CreatedAt        time.Time `json:"created_at"`
}

// ToolMastery 代表某个工具的掌握程度记录。
type ToolMastery struct {
	UserID    string    `json:"user_id"`
	ToolName  string    `json:"tool_name"`
	Category  string    `json:"category"`
	Score     int       `json:"score"`
	LevelName string    `json:"level_name"`
	Evidence  string    `json:"evidence"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Repository 封装所有情节记忆的数据库操作。
type Repository struct {
	pool *pgxpool.Pool
}

// New 创建 Repository。
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetOrCreateProfile 获取用户档案，不存在则创建默认档案。
func (r *Repository) GetOrCreateProfile(ctx context.Context, userID string) (*Profile, error) {
	p, err := r.GetProfile(ctx, userID)
	if err == nil {
		return p, nil
	}

	// 创建默认档案
	_, err = r.pool.Exec(ctx, `
		INSERT INTO cultivation_profile (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}
	return r.GetProfile(ctx, userID)
}

// GetProfile 查询用户修行档案。
func (r *Repository) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT user_id, active_path,
		       go_lianqi_score, go_lianqi_level, go_lianqi_level_name, go_lianqi_branch,
		       ai_lianqi_score, ai_lianqi_level, ai_lianqi_level_name, ai_lianqi_branch,
		       wufu_score, wufu_level, wufu_level_name,
		       xin_mo_records, breakthrough_conditions, updated_at
		FROM cultivation_profile
		WHERE user_id = $1
	`, userID)

	var p Profile
	var bc []byte
	err := row.Scan(
		&p.UserID, &p.ActivePath,
		&p.GoLianqiScore, &p.GoLianqiLevel, &p.GoLianqiLevelName, &p.GoLianqiBranch,
		&p.AILianqiScore, &p.AILianqiLevel, &p.AILianqiLevelName, &p.AILianqiBranch,
		&p.WufuScore, &p.WufuLevel, &p.WufuLevelName,
		&p.XinMoRecords, &bc, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	p.BreakthroughConditions = bc
	return &p, nil
}

// UpdateProfile 更新用户修行档案（来自 LLM 评估结果）。
func (r *Repository) UpdateProfile(ctx context.Context, p *Profile) error {
	bc := p.BreakthroughConditions
	if bc == nil {
		bc = json.RawMessage("[]")
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE cultivation_profile SET
		    go_lianqi_score      = $2,
		    go_lianqi_level      = $3,
		    go_lianqi_level_name = $4,
		    go_lianqi_branch     = $5,
		    ai_lianqi_score      = $6,
		    ai_lianqi_level      = $7,
		    ai_lianqi_level_name = $8,
		    ai_lianqi_branch     = $9,
		    wufu_score           = $10,
		    wufu_level           = $11,
		    wufu_level_name      = $12,
		    xin_mo_records       = $13,
		    breakthrough_conditions = $14,
		    updated_at           = NOW()
		WHERE user_id = $1
	`,
		p.UserID,
		p.GoLianqiScore, p.GoLianqiLevel, p.GoLianqiLevelName, p.GoLianqiBranch,
		p.AILianqiScore, p.AILianqiLevel, p.AILianqiLevelName, p.AILianqiBranch,
		p.WufuScore, p.WufuLevel, p.WufuLevelName,
		p.XinMoRecords, bc,
	)
	return err
}

// SaveSession 保存一次会话记录。
func (r *Repository) SaveSession(ctx context.Context, s *Session) error {
	bu := s.BreakthroughUpdates
	if bu == nil {
		bu = json.RawMessage("[]")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions
		    (user_id, summary, go_lianqi_score, ai_lianqi_score, wufu_score,
		     xin_mo_identified, breakthrough_updates, raw_response)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		s.UserID, s.Summary,
		s.GoLianqiScore, s.AILianqiScore, s.WufuScore,
		s.XinMoIdentified, bu, s.RawResponse,
	)
	return err
}

// GetRecentSessions 获取最近 n 条会话摘要。
func (r *Repository) GetRecentSessions(ctx context.Context, userID string, n int) ([]*Session, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, summary, go_lianqi_score, ai_lianqi_score, wufu_score,
		       xin_mo_identified, breakthrough_updates, raw_response, created_at
		FROM sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, n)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var s Session
		var bu []byte
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Summary,
			&s.GoLianqiScore, &s.AILianqiScore, &s.WufuScore,
			&s.XinMoIdentified, &bu, &s.RawResponse, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.BreakthroughUpdates = bu
		sessions = append(sessions, &s)
	}
	return sessions, rows.Err()
}

// UpsertToolMastery 更新或插入工具掌握度。
func (r *Repository) UpsertToolMastery(ctx context.Context, tm *ToolMastery) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tool_mastery (user_id, tool_name, category, score, level_name, evidence)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, tool_name) DO UPDATE SET
		    category   = EXCLUDED.category,
		    score      = EXCLUDED.score,
		    level_name = EXCLUDED.level_name,
		    evidence   = EXCLUDED.evidence,
		    updated_at = NOW()
	`, tm.UserID, tm.ToolName, tm.Category, tm.Score, tm.LevelName, tm.Evidence)
	return err
}

// GetToolMastery 获取用户所有工具掌握度。
func (r *Repository) GetToolMastery(ctx context.Context, userID string) ([]*ToolMastery, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, tool_name, category, score, level_name, evidence, updated_at
		FROM tool_mastery
		WHERE user_id = $1
		ORDER BY score DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ToolMastery
	for rows.Next() {
		var tm ToolMastery
		if err := rows.Scan(
			&tm.UserID, &tm.ToolName, &tm.Category,
			&tm.Score, &tm.LevelName, &tm.Evidence, &tm.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, &tm)
	}
	return result, rows.Err()
}

// ScoreToLevel 根据分数返回法器掌握级别名称。
func ScoreToLevel(score int) string {
	switch {
	case score <= 25:
		return "初识"
	case score <= 50:
		return "熟用"
	case score <= 75:
		return "精通"
	default:
		return "宗师"
	}
}
