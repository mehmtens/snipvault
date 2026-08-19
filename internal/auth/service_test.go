package auth

import (
	"context"
	"errors"
	"testing"

	"snipvault/internal/user"
)

func TestRegisterLoginAndParse(t *testing.T) {
	service := NewService(user.NewMemoryStore(), "test-secret-that-is-definitely-long-enough")
	created, token, err := service.Register(context.Background(), " Mehmet ", "MEHMET@example.com ", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if created.Username != "Mehmet" || created.Email != "mehmet@example.com" {
		t.Fatalf("unexpected user: %+v", created)
	}
	if created.PasswordHash == "password123" {
		t.Fatal("password was not hashed")
	}
	userID, err := service.Parse(token)
	if err != nil || userID != created.ID {
		t.Fatalf("unexpected token result: %d %v", userID, err)
	}

	loggedIn, _, err := service.Login(context.Background(), "MEHMET@example.com", "password123")
	if err != nil || loggedIn.ID != created.ID {
		t.Fatal("login failed")
	}
	if _, _, err := service.Login(context.Background(), created.Email, "wrong-password"); !errors.Is(err, user.ErrInvalidCredentials) {
		t.Fatal("wrong password should fail")
	}
	if _, err := service.Parse("not-a-token"); !errors.Is(err, user.ErrInvalidCredentials) {
		t.Fatal("invalid token should fail")
	}
}

func TestDuplicateRegistration(t *testing.T) {
	service := NewService(user.NewMemoryStore(), "test-secret-that-is-definitely-long-enough")
	_, _, _ = service.Register(context.Background(), "first", "same@example.com", "password123")
	if _, _, err := service.Register(context.Background(), "second", "same@example.com", "password123"); !errors.Is(err, user.ErrEmailTaken) {
		t.Fatal("duplicate email should fail")
	}
}
