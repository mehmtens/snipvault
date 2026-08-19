package mailer

import (
	"context"
	"errors"
	"testing"
)

func TestSMTPConfigurationAndCancelledContext(t *testing.T) {
	if _, ok := NewSMTP("mailpit:1025", "snipvault@local.test").(*SMTP); !ok {
		t.Fatal("valid SMTP configuration should create a sender")
	}
	if _, ok := NewSMTP("", "snipvault@local.test").(Unavailable); !ok {
		t.Fatal("missing SMTP address should disable email delivery")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sender := NewSMTP("mailpit:1025", "snipvault@local.test")
	if err := sender.SendVerification(ctx, "user@example.com", "123456"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancelled context, got %v", err)
	}
	if err := sender.SendPasswordReset(ctx, "user@example.com", "123456"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancelled context, got %v", err)
	}
}
