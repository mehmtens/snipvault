package user

import (
	"context"
	"errors"
	"time"
)

var (
	ErrEmailTaken         = errors.New("email is already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotFound           = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid or expired code")
	ErrEmailNotVerified   = errors.New("email address is not verified")
)

type User struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

type AccountToken struct {
	UserID    int64
	Purpose   string
	TokenHash string
	ExpiresAt time.Time
}

type Store interface {
	Create(context.Context, User) (User, error)
	GetByEmail(context.Context, string) (User, error)
	GetByID(context.Context, int64) (User, error)
	MarkEmailVerified(context.Context, int64) error
	UpdateProfile(context.Context, int64, string) (User, error)
	UpdatePassword(context.Context, int64, string) error
	Delete(context.Context, int64) error
	SaveToken(context.Context, AccountToken) error
	ConsumeToken(context.Context, AccountToken) (bool, error)
}
