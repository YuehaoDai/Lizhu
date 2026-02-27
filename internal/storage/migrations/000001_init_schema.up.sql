-- 骊珠数据库初始化迁移

-- 修行档案表：记录用户双轨练气士 + 武夫境界评分
CREATE TABLE IF NOT EXISTS cultivation_profile (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL DEFAULT 'default',
    -- 主修路径
    active_path TEXT NOT NULL DEFAULT 'both', -- 'go', 'ai', 'both'
    -- Go开发练气士
    go_lianqi_score      INTEGER NOT NULL DEFAULT 0 CHECK (go_lianqi_score >= 0 AND go_lianqi_score <= 100),
    go_lianqi_level      INTEGER NOT NULL DEFAULT 1 CHECK (go_lianqi_level >= 1 AND go_lianqi_level <= 15),
    go_lianqi_level_name TEXT    NOT NULL DEFAULT '铜皮境',
    go_lianqi_branch     TEXT    NOT NULL DEFAULT '',
    -- AI应用练气士
    ai_lianqi_score      INTEGER NOT NULL DEFAULT 0 CHECK (ai_lianqi_score >= 0 AND ai_lianqi_score <= 100),
    ai_lianqi_level      INTEGER NOT NULL DEFAULT 1 CHECK (ai_lianqi_level >= 1 AND ai_lianqi_level <= 15),
    ai_lianqi_level_name TEXT    NOT NULL DEFAULT '铜皮境',
    ai_lianqi_branch     TEXT    NOT NULL DEFAULT '',
    -- 武夫
    wufu_score      INTEGER NOT NULL DEFAULT 0 CHECK (wufu_score >= 0 AND wufu_score <= 100),
    wufu_level      INTEGER NOT NULL DEFAULT 1 CHECK (wufu_level >= 1 AND wufu_level <= 11),
    wufu_level_name TEXT    NOT NULL DEFAULT '泥胚境',
    -- 心魔与破境
    xin_mo_records          TEXT[]  NOT NULL DEFAULT '{}',
    breakthrough_conditions JSONB   NOT NULL DEFAULT '[]',
    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

-- 会话记录表：每次对话结束后存储摘要与评估结果
CREATE TABLE IF NOT EXISTS sessions (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL DEFAULT 'default',
    -- 会话摘要
    summary TEXT NOT NULL DEFAULT '',
    -- 本次评估结果快照
    go_lianqi_score  INTEGER NOT NULL DEFAULT 0,
    ai_lianqi_score  INTEGER NOT NULL DEFAULT 0,
    wufu_score       INTEGER NOT NULL DEFAULT 0,
    -- 本次识别的心魔
    xin_mo_identified TEXT[] NOT NULL DEFAULT '{}',
    -- 破境条件更新
    breakthrough_updates JSONB NOT NULL DEFAULT '[]',
    -- 原始 LLM 输出（用于调试和知识库索引）
    raw_response TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 法器谱表：独立追踪每个工具的掌握程度
CREATE TABLE IF NOT EXISTS tool_mastery (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT NOT NULL DEFAULT 'default',
    tool_name  TEXT NOT NULL,
    category   TEXT NOT NULL, -- primary_weapon | fulu | zhenfa | ai_tool | telescope | quality | philosophy
    score      INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 100),
    level_name TEXT    NOT NULL DEFAULT '初识',
    evidence   TEXT    NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, tool_name)
);

-- Eino CheckPointStore 后端表
CREATE TABLE IF NOT EXISTS checkpoints (
    id         TEXT PRIMARY KEY,
    data       BYTEA       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_sessions_user_id     ON sessions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tool_mastery_user_id ON tool_mastery(user_id);
