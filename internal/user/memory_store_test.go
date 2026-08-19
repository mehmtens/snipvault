package user

import (
	"context"
	"errors"
	"testing"
	"time"
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

func TestMemoryStoreAccountLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	created, err := store.Create(ctx, User{Username: "first", Email: "owner@example.com", PasswordHash: "old"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetByID(ctx, created.ID)
	if err != nil || loaded.ID != created.ID {
		t.Fatal("GetByID failed")
	}
	if err := store.MarkEmailVerified(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := store.UpdateProfile(ctx, created.ID, "updated")
	if err != nil || updated.Username != "updated" {
		t.Fatal("UpdateProfile failed")
	}
	if err := store.UpdatePassword(ctx, created.ID, "new"); err != nil {
		t.Fatal(err)
	}
	loaded, _ = store.GetByID(ctx, created.ID)
	if loaded.PasswordHash != "new" || !loaded.EmailVerified {
		t.Fatal("account updates were not persisted")
	}
	token := AccountToken{UserID: created.ID, Purpose: "verify_email", TokenHash: "hash", ExpiresAt: time.Now().Add(time.Minute)}
	if err := store.SaveToken(ctx, token); err != nil {
		t.Fatal(err)
	}
	if valid, err := store.ConsumeToken(ctx, AccountToken{UserID: created.ID, Purpose: token.Purpose, TokenHash: "wrong"}); err != nil || valid {
		t.Fatal("wrong token must fail")
	}
	if valid, err := store.ConsumeToken(ctx, token); err != nil || !valid {
		t.Fatal("valid token must be consumed")
	}
	if valid, _ := store.ConsumeToken(ctx, token); valid {
		t.Fatal("token must be single use")
	}
	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetByID(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted user must not exist")
	}
	if err := store.Delete(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleting missing user must fail")
	}
}

func TestMemoryStoreRejectsExpiredTokenAndMissingUpdates(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	expired := AccountToken{UserID: 99, Purpose: "reset_password", TokenHash: "expired", ExpiresAt: time.Now().Add(-time.Minute)}
	_ = store.SaveToken(ctx, expired)
	if valid, _ := store.ConsumeToken(ctx, expired); valid {
		t.Fatal("expired token must fail")
	}
	if err := store.MarkEmailVerified(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing verification update must fail")
	}
	if _, err := store.UpdateProfile(ctx, 99, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing profile update must fail")
	}
	if err := store.UpdatePassword(ctx, 99, "hash"); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing password update must fail")
	}
}
