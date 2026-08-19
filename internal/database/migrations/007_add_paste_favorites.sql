ALTER TABLE pastes ADD COLUMN IF NOT EXISTS is_favorite BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_pastes_user_favorite_updated
    ON pastes (user_id, is_favorite DESC, updated_at DESC);
