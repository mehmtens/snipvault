package auth

import (
	"context"
	"errors"
	"testing"

	"snipvault/internal/user"
)

type captureMailer struct{ verificationCode, resetCode string }
func (m *captureMailer) SendVerification(_ context.Context, _ string, code string) error { m.verificationCode = code; return nil }
func (m *captureMailer) SendPasswordReset(_ context.Context, _ string, code string) error { m.resetCode = code; return nil }

func TestRegisterLoginAndParse(t *testing.T) {
	sender := &captureMailer{}
	service := NewService(user.NewMemoryStore(), "test-secret-that-is-definitely-long-enough", sender)
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
	if token != "" { t.Fatal("registration must not create a session before verification") }
	created, token, err = service.VerifyEmail(context.Background(), created.Email, sender.verificationCode)
	if err != nil { t.Fatal(err) }
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
	sender := &captureMailer{}
	service := NewService(user.NewMemoryStore(), "test-secret-that-is-definitely-long-enough", sender)
	_, _, _ = service.Register(context.Background(), "first", "same@example.com", "password123")
	_, _, _ = service.VerifyEmail(context.Background(), "same@example.com", sender.verificationCode)
	if _, _, err := service.Register(context.Background(), "second", "same@example.com", "password123"); !errors.Is(err, user.ErrEmailTaken) {
		t.Fatal("duplicate email should fail")
	}
}

func TestPasswordRecoveryAndAccountManagement(t *testing.T) {
	sender := &captureMailer{}
	service := NewService(user.NewMemoryStore(), "test-secret-that-is-definitely-long-enough", sender)
	created, _, err := service.Register(context.Background(), "owner", "owner@example.com", "password123")
	if err != nil { t.Fatal(err) }
	if _, _, err = service.VerifyEmail(context.Background(), created.Email, sender.verificationCode); err != nil { t.Fatal(err) }
	updated, err := service.UpdateProfile(context.Background(), created.ID, "new-owner")
	if err != nil || updated.Username != "new-owner" { t.Fatal("profile update failed") }
	if err = service.ChangePassword(context.Background(), created.ID, "password123", "changed-password"); err != nil { t.Fatal(err) }
	if err = service.ForgotPassword(context.Background(), created.Email); err != nil { t.Fatal(err) }
	if err = service.ResetPassword(context.Background(), created.Email, sender.resetCode, "reset-password"); err != nil { t.Fatal(err) }
	if _, _, err = service.Login(context.Background(), created.Email, "reset-password"); err != nil { t.Fatal("reset password login failed") }
	if err = service.DeleteAccount(context.Background(), created.ID, "reset-password"); err != nil { t.Fatal(err) }
	if _, err = service.Profile(context.Background(), created.ID); !errors.Is(err, user.ErrNotFound) { t.Fatal("deleted user still exists") }
}
