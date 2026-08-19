package auth

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"snipvault/internal/user"
)

type Service struct {
	store  user.Store
	secret []byte
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

func NewService(store user.Store, secret string) *Service {
	return &Service{store: store, secret: []byte(secret)}
}

func (s *Service) Register(ctx context.Context, username, email, password string) (user.User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return user.User{}, "", err
	}
	value, err := s.store.Create(ctx, user.User{Username: strings.TrimSpace(username), Email: normalizeEmail(email), PasswordHash: string(hash)})
	if err != nil {
		return user.User{}, "", err
	}
	token, err := s.token(value)
	return value, token, err
}

func (s *Service) Login(ctx context.Context, email, password string) (user.User, string, error) {
	value, err := s.store.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return user.User{}, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(value.PasswordHash), []byte(password)) != nil {
		return user.User{}, "", user.ErrInvalidCredentials
	}
	token, err := s.token(value)
	return value, token, err
}

func (s *Service) token(value user.User) (string, error) {
	claims := jwt.MapClaims{"sub": fmt.Sprint(value.ID), "username": value.Username, "iat": time.Now().Unix(), "exp": time.Now().Add(24 * time.Hour).Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
