ALTER TABLE pastes
    ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE pastes DROP CONSTRAINT IF EXISTS pastes_visibility_check;
ALTER TABLE pastes ADD CONSTRAINT pastes_visibility_check
    CHECK (visibility IN ('public', 'unlisted', 'private'));

CREATE INDEX IF NOT EXISTS idx_pastes_user_id_created_at
    ON pastes (user_id, created_at DESC);
