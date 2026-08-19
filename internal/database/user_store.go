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
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`, value.Username, value.Email, value.PasswordHash,
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
		SELECT id, username, email, password_hash, created_at
		FROM users WHERE email = $1`, email,
	).Scan(&value.ID, &value.Username, &value.Email, &value.PasswordHash, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrInvalidCredentials
	}
	return value, err
}
