-- 能力证据表：每次对话结束后由 Librarian 提炼的结构化能力证据条目
CREATE TABLE IF NOT EXISTS ability_evidence (
    id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT    NOT NULL DEFAULT 'default',
    category   TEXT    NOT NULL,  -- go_lianqi | ai_lianqi | wufu | general
    tool       TEXT    NOT NULL DEFAULT '',
    evidence   TEXT    NOT NULL,
    confidence INTEGER NOT NULL DEFAULT 3 CHECK (confidence BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ability_evidence_user
    ON ability_evidence(user_id, created_at DESC);
