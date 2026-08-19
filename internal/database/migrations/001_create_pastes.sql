CREATE TABLE IF NOT EXISTS pastes (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(16) NOT NULL UNIQUE,
    title VARCHAR(200) NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    language VARCHAR(50) NOT NULL DEFAULT '',
    visibility VARCHAR(20) NOT NULL DEFAULT 'unlisted'
        CHECK (visibility IN ('public', 'unlisted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pastes_created_at ON pastes (created_at DESC);
