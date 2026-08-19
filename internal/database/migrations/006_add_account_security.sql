ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN;
UPDATE users SET email_verified = TRUE WHERE email_verified IS NULL;
ALTER TABLE users ALTER COLUMN email_verified SET DEFAULT FALSE;
ALTER TABLE users ALTER COLUMN email_verified SET NOT NULL;

CREATE TABLE IF NOT EXISTS account_tokens (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose VARCHAR(32) NOT NULL CHECK (purpose IN ('verify_email', 'reset_password')),
    token_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, purpose)
);

CREATE INDEX IF NOT EXISTS idx_account_tokens_expires_at ON account_tokens (expires_at);
