CREATE TABLE IF NOT EXISTS tasks (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             TEXT        NOT NULL DEFAULT 'default',
    title               TEXT        NOT NULL,
    description         TEXT        NOT NULL,
    acceptance_criteria TEXT        NOT NULL,
    category            TEXT        NOT NULL,
    source_evidence     TEXT,
    target_score_hint   INTEGER,
    status              TEXT        NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tasks_user_status ON tasks (user_id, status);
