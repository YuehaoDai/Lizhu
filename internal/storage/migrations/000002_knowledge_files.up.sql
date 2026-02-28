CREATE TABLE IF NOT EXISTS knowledge_files (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    file_path   TEXT NOT NULL,
    file_hash   TEXT NOT NULL,
    chunk_count INT  NOT NULL DEFAULT 0,
    summary     TEXT NOT NULL DEFAULT '',
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, file_path)
);

CREATE INDEX IF NOT EXISTS idx_knowledge_files_user_id ON knowledge_files(user_id);
