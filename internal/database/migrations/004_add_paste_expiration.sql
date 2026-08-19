ALTER TABLE pastes
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_pastes_expires_at
    ON pastes (expires_at) WHERE expires_at IS NOT NULL;
