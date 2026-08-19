package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"snipvault/internal/paste"
)

type PasteStore struct {
	pool *pgxpool.Pool
}

func (s *PasteStore) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := s.pool.Exec(ctx, `DELETE FROM pastes WHERE expires_at IS NOT NULL AND expires_at <= NOW()`)
	return result.RowsAffected(), err
}

func NewPasteStore(pool *pgxpool.Pool) *PasteStore {
	return &PasteStore{pool: pool}
}

func (s *PasteStore) Create(ctx context.Context, value paste.Paste) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO pastes (slug, title, content, language, visibility, created_at, updated_at, user_id, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		value.Slug, value.Title, value.Content, value.Language, value.Visibility, value.CreatedAt, value.UpdatedAt, value.UserID, value.ExpiresAt,
	)
	return err
}

func (s *PasteStore) GetBySlug(ctx context.Context, slug string) (paste.Paste, error) {
	var value paste.Paste
	err := s.pool.QueryRow(ctx, `
		SELECT slug, title, content, language, visibility, created_at, updated_at, user_id, expires_at, is_favorite
		FROM pastes
		WHERE slug = $1 AND (expires_at IS NULL OR expires_at > NOW())`, slug,
	).Scan(&value.Slug, &value.Title, &value.Content, &value.Language, &value.Visibility, &value.CreatedAt, &value.UpdatedAt, &value.UserID, &value.ExpiresAt, &value.IsFavorite)
	if errors.Is(err, pgx.ErrNoRows) {
		return paste.Paste{}, paste.ErrNotFound
	}
	return value, err
}

func (s *PasteStore) ListByUser(ctx context.Context, userID int64, options paste.ListOptions) ([]paste.Paste, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slug, title, content, language, visibility, created_at, updated_at, user_id, expires_at, is_favorite
		FROM pastes
		WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		  AND ($2 = '' OR title ILIKE '%' || $2 || '%' OR content ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR language = $3)
		  AND ($4 = '' OR visibility = $4)
		  AND (NOT $5 OR is_favorite)
		ORDER BY is_favorite DESC, updated_at DESC`, userID, options.Query, options.Language, options.Visibility, options.FavoritesOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]paste.Paste, 0)
	for rows.Next() {
		var value paste.Paste
		if err := rows.Scan(&value.Slug, &value.Title, &value.Content, &value.Language, &value.Visibility, &value.CreatedAt, &value.UpdatedAt, &value.UserID, &value.ExpiresAt, &value.IsFavorite); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *PasteStore) SetFavorite(ctx context.Context, slug string, userID int64, favorite bool) (paste.Paste, error) {
	var value paste.Paste
	err := s.pool.QueryRow(ctx, `
		UPDATE pastes SET is_favorite = $1, updated_at = NOW()
		WHERE slug = $2 AND user_id = $3
		RETURNING slug, title, content, language, visibility, created_at, updated_at, user_id, expires_at, is_favorite`, favorite, slug, userID,
	).Scan(&value.Slug, &value.Title, &value.Content, &value.Language, &value.Visibility, &value.CreatedAt, &value.UpdatedAt, &value.UserID, &value.ExpiresAt, &value.IsFavorite)
	if errors.Is(err, pgx.ErrNoRows) {
		return paste.Paste{}, paste.ErrNotFound
	}
	return value, err
}

func (s *PasteStore) UpdateByUser(ctx context.Context, value paste.Paste, userID int64) (paste.Paste, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE pastes SET title=$1, content=$2, language=$3, visibility=$4, expires_at=$5, updated_at=NOW()
		WHERE slug=$6 AND user_id=$7
		RETURNING created_at, updated_at, user_id, is_favorite`, value.Title, value.Content, value.Language, value.Visibility, value.ExpiresAt, value.Slug, userID,
	).Scan(&value.CreatedAt, &value.UpdatedAt, &value.UserID, &value.IsFavorite)
	if errors.Is(err, pgx.ErrNoRows) {
		return paste.Paste{}, paste.ErrNotFound
	}
	return value, err
}

func (s *PasteStore) DeleteByUser(ctx context.Context, slug string, userID int64) (bool, error) {
	result, err := s.pool.Exec(ctx, `DELETE FROM pastes WHERE slug = $1 AND user_id = $2`, slug, userID)
	return result.RowsAffected() == 1, err
}
