package user

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(context.Background(), User{Username: "user", Email: "user@example.com", PasswordHash: "hash"})
	if err != nil || created.ID == 0 || created.CreatedAt.IsZero() {
		t.Fatal("user was not created")
	}
	loaded, err := store.GetByEmail(context.Background(), created.Email)
	if err != nil || loaded.ID != created.ID {
		t.Fatal("user was not loaded")
	}
	if _, err := store.Create(context.Background(), User{Email: created.Email}); !errors.Is(err, ErrEmailTaken) {
		t.Fatal("duplicate email should fail")
	}
	if _, err := store.GetByEmail(context.Background(), "missing@example.com"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("missing user should fail")
	}
}
