package paste

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("paste not found")

type Paste struct {
	Slug       string     `json:"slug"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Language   string     `json:"language"`
	Visibility string     `json:"visibility"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	UserID     *int64     `json:"-"`
	IsFavorite bool       `json:"is_favorite"`
}

type ListOptions struct {
	Query         string
	Language      string
	Visibility    string
	FavoritesOnly bool
}

type Store interface {
	Create(context.Context, Paste) error
	GetBySlug(context.Context, string) (Paste, error)
	ListByUser(context.Context, int64, ListOptions) ([]Paste, error)
	SetFavorite(context.Context, string, int64, bool) (Paste, error)
	DeleteByUser(context.Context, string, int64) (bool, error)
	UpdateByUser(context.Context, Paste, int64) (Paste, error)
}
