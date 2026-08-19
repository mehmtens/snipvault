package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"snipvault/internal/mailer"
	"snipvault/internal/user"
)

type Service struct {
	store  user.Store
	secret []byte
	mailer mailer.Sender
}

func (s *Service) Parse(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return 0, user.ErrInvalidCredentials
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, user.ErrInvalidCredentials
	}
	subject, err := claims.GetSubject()
	if err != nil {
		return 0, user.ErrInvalidCredentials
	}
	userID, err := strconv.ParseInt(subject, 10, 64)
	if err != nil {
		return 0, user.ErrInvalidCredentials
	}
	return userID, nil
}

func NewService(store user.Store, secret string, senders ...mailer.Sender) *Service {
	sender := mailer.Sender(mailer.Unavailable{})
	if len(senders) > 0 && senders[0] != nil {
		sender = senders[0]
	}
	return &Service{store: store, secret: []byte(secret), mailer: sender}
}

func (s *Service) Register(ctx context.Context, username, email, password string) (user.User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return user.User{}, "", err
	}
	value, err := s.store.Create(ctx, user.User{Username: strings.TrimSpace(username), Email: normalizeEmail(email), PasswordHash: string(hash), EmailVerified: false})
	if err == user.ErrEmailTaken {
		value, err = s.store.GetByEmail(ctx, normalizeEmail(email))
		if err == nil && value.EmailVerified {
			return user.User{}, "", user.ErrEmailTaken
		}
	}
	if err != nil {
		return user.User{}, "", err
	}
	if err := s.sendCode(ctx, value, "verify_email"); err != nil {
		return user.User{}, "", err
	}
	return value, "", nil
}

func (s *Service) VerifyEmail(ctx context.Context, email, code string) (user.User, string, error) {
	value, err := s.store.GetByEmail(ctx, normalizeEmail(email))
	if err != nil || value.EmailVerified {
		return user.User{}, "", user.ErrInvalidToken
	}
	valid, err := s.store.ConsumeToken(ctx, user.AccountToken{UserID: value.ID, Purpose: "verify_email", TokenHash: s.codeHash(code)})
	if err != nil || !valid {
		return user.User{}, "", user.ErrInvalidToken
	}
	if err := s.store.MarkEmailVerified(ctx, value.ID); err != nil {
		return user.User{}, "", err
	}
	value.EmailVerified = true
	token, err := s.token(value)
	return value, token, err
}

func (s *Service) ResendVerification(ctx context.Context, email string) error {
	value, err := s.store.GetByEmail(ctx, normalizeEmail(email))
	if err != nil || value.EmailVerified {
		return nil
	}
	return s.sendCode(ctx, value, "verify_email")
}

func (s *Service) Login(ctx context.Context, email, password string) (user.User, string, error) {
	value, err := s.store.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return user.User{}, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(value.PasswordHash), []byte(password)) != nil {
		return user.User{}, "", user.ErrInvalidCredentials
	}
	if !value.EmailVerified {
		return user.User{}, "", user.ErrEmailNotVerified
	}
	token, err := s.token(value)
	return value, token, err
}

func (s *Service) Profile(ctx context.Context, userID int64) (user.User, error) {
	return s.store.GetByID(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, username string) (user.User, error) {
	return s.store.UpdateProfile(ctx, userID, strings.TrimSpace(username))
}

func (s *Service) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	value, err := s.store.GetByID(ctx, userID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(value.PasswordHash), []byte(currentPassword)) != nil {
		return user.ErrInvalidCredentials
	}
	return s.setPassword(ctx, userID, newPassword)
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	value, err := s.store.GetByEmail(ctx, normalizeEmail(email))
	if err != nil || !value.EmailVerified {
		return nil
	}
	return s.sendCode(ctx, value, "reset_password")
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	value, err := s.store.GetByEmail(ctx, normalizeEmail(email))
	if err != nil || !value.EmailVerified {
		return user.ErrInvalidToken
	}
	valid, err := s.store.ConsumeToken(ctx, user.AccountToken{UserID: value.ID, Purpose: "reset_password", TokenHash: s.codeHash(code)})
	if err != nil || !valid {
		return user.ErrInvalidToken
	}
	return s.setPassword(ctx, value.ID, newPassword)
}

func (s *Service) DeleteAccount(ctx context.Context, userID int64, password string) error {
	value, err := s.store.GetByID(ctx, userID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(value.PasswordHash), []byte(password)) != nil {
		return user.ErrInvalidCredentials
	}
	return s.store.Delete(ctx, userID)
}

func (s *Service) setPassword(ctx context.Context, userID int64, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.UpdatePassword(ctx, userID, string(hash))
}

func (s *Service) sendCode(ctx context.Context, value user.User, purpose string) error {
	code, err := verificationCode()
	if err != nil {
		return err
	}
	if err := s.store.SaveToken(ctx, user.AccountToken{UserID: value.ID, Purpose: purpose, TokenHash: s.codeHash(code), ExpiresAt: time.Now().Add(15 * time.Minute)}); err != nil {
		return err
	}
	if purpose == "verify_email" {
		return s.mailer.SendVerification(ctx, value.Email, code)
	}
	return s.mailer.SendPasswordReset(ctx, value.Email, code)
}

func (s *Service) codeHash(code string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(mac.Sum(nil))
}

func verificationCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func (s *Service) token(value user.User) (string, error) {
	claims := jwt.MapClaims{"sub": fmt.Sprint(value.ID), "username": value.Username, "iat": time.Now().Unix(), "exp": time.Now().Add(24 * time.Hour).Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
