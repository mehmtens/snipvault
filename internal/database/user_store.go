package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"snipvault/internal/user"
)

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) Create(ctx context.Context, value user.User) (user.User, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, email_verified)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`, value.Username, value.Email, value.PasswordHash, value.EmailVerified,
	).Scan(&value.ID, &value.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return user.User{}, user.ErrEmailTaken
	}
	return value, err
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (user.User, error) {
	var value user.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, email_verified, created_at
		FROM users WHERE email = $1`, email,
	).Scan(&value.ID, &value.Username, &value.Email, &value.PasswordHash, &value.EmailVerified, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrInvalidCredentials
	}
	return value, err
}

func (s *UserStore) GetByID(ctx context.Context, id int64) (user.User, error) {
	var value user.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, email_verified, created_at
		FROM users WHERE id = $1`, id,
	).Scan(&value.ID, &value.Username, &value.Email, &value.PasswordHash, &value.EmailVerified, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	return value, err
}

func (s *UserStore) MarkEmailVerified(ctx context.Context, id int64) error {
	result, err := s.pool.Exec(ctx, `UPDATE users SET email_verified = TRUE WHERE id = $1`, id)
	if err == nil && result.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return err
}

func (s *UserStore) UpdateProfile(ctx context.Context, id int64, username string) (user.User, error) {
	var value user.User
	err := s.pool.QueryRow(ctx, `
		UPDATE users SET username = $1 WHERE id = $2
		RETURNING id, username, email, password_hash, email_verified, created_at`, username, id,
	).Scan(&value.ID, &value.Username, &value.Email, &value.PasswordHash, &value.EmailVerified, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	return value, err
}

func (s *UserStore) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	result, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, passwordHash, id)
	if err == nil && result.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return err
}

func (s *UserStore) Delete(ctx context.Context, id int64) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err == nil && result.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return err
}

func (s *UserStore) SaveToken(ctx context.Context, token user.AccountToken) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO account_tokens (user_id, purpose, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, purpose) DO UPDATE
		SET token_hash = EXCLUDED.token_hash, expires_at = EXCLUDED.expires_at, created_at = NOW()`,
		token.UserID, token.Purpose, token.TokenHash, token.ExpiresAt)
	return err
}

func (s *UserStore) ConsumeToken(ctx context.Context, token user.AccountToken) (bool, error) {
	result, err := s.pool.Exec(ctx, `
		DELETE FROM account_tokens
		WHERE user_id = $1 AND purpose = $2 AND token_hash = $3 AND expires_at > NOW()`,
		token.UserID, token.Purpose, token.TokenHash)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, err
}
