package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("email delivery is not configured")

type Sender interface {
	SendVerification(context.Context, string, string) error
	SendPasswordReset(context.Context, string, string) error
}

type Unavailable struct{}

func (Unavailable) SendVerification(context.Context, string, string) error  { return ErrUnavailable }
func (Unavailable) SendPasswordReset(context.Context, string, string) error { return ErrUnavailable }

type Brevo struct {
	apiKey    string
	fromEmail string
	fromName  string
	client    *http.Client
}

type SMTP struct {
	address   string
	fromEmail string
}

func NewSMTP(address, fromEmail string) Sender {
	if strings.TrimSpace(address) == "" || strings.TrimSpace(fromEmail) == "" {
		return Unavailable{}
	}
	return &SMTP{address: strings.TrimSpace(address), fromEmail: strings.TrimSpace(fromEmail)}
}

func (s *SMTP) SendVerification(ctx context.Context, to, code string) error {
	return s.send(ctx, to, "Verify your SnipVault email", fmt.Sprintf(
		`<h1>Verify your email</h1><p>Your SnipVault verification code is:</p><p style="font:700 32px monospace;letter-spacing:8px">%s</p><p>This code expires in 15 minutes.</p>`, html.EscapeString(code)))
}

func (s *SMTP) SendPasswordReset(ctx context.Context, to, code string) error {
	return s.send(ctx, to, "Reset your SnipVault password", fmt.Sprintf(
		`<h1>Reset your password</h1><p>Your SnipVault reset code is:</p><p style="font:700 32px monospace;letter-spacing:8px">%s</p><p>This code expires in 15 minutes.</p>`, html.EscapeString(code)))
}

func (s *SMTP) send(ctx context.Context, to, subject, htmlContent string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	message := []byte("From: SnipVault <" + s.fromEmail + ">\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n" + htmlContent)
	return smtp.SendMail(s.address, nil, s.fromEmail, []string{to}, message)
}

func NewBrevo(apiKey, fromEmail, fromName string) Sender {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(fromEmail) == "" {
		return Unavailable{}
	}
	if strings.TrimSpace(fromName) == "" {
		fromName = "SnipVault"
	}
	return &Brevo{apiKey: apiKey, fromEmail: fromEmail, fromName: fromName, client: &http.Client{Timeout: 8 * time.Second}}
}

func (b *Brevo) SendVerification(ctx context.Context, to, code string) error {
	return b.send(ctx, to, "Verify your SnipVault email", fmt.Sprintf(
		`<h1 style="font-family:monospace">Verify your email</h1><p>Use this code to finish creating your SnipVault account:</p><p style="font:700 32px monospace;letter-spacing:8px">%s</p><p>This code expires in 15 minutes. If you did not request it, you can ignore this email.</p>`,
		html.EscapeString(code)))
}

func (b *Brevo) SendPasswordReset(ctx context.Context, to, code string) error {
	return b.send(ctx, to, "Reset your SnipVault password", fmt.Sprintf(
		`<h1 style="font-family:monospace">Reset your password</h1><p>Use this code to choose a new SnipVault password:</p><p style="font:700 32px monospace;letter-spacing:8px">%s</p><p>This code expires in 15 minutes. If you did not request it, you can ignore this email.</p>`,
		html.EscapeString(code)))
}

func (b *Brevo) send(ctx context.Context, to, subject, htmlContent string) error {
	payload := map[string]any{
		"sender":      map[string]string{"name": b.fromName, "email": b.fromEmail},
		"to":          []map[string]string{{"email": to}},
		"subject":     subject,
		"htmlContent": htmlContent,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("api-key", b.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := b.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("email provider returned %s", response.Status)
	}
	return nil
}
