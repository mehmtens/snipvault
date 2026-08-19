package user

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MemoryStore struct {
	mu      sync.Mutex
	nextID  int64
	byEmail map[string]User
	tokens  map[string]AccountToken
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{nextID: 1, byEmail: make(map[string]User), tokens: make(map[string]AccountToken)}
}

func (s *MemoryStore) GetByID(_ context.Context, id int64) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, value := range s.byEmail {
		if value.ID == id {
			return value, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *MemoryStore) MarkEmailVerified(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for email, value := range s.byEmail {
		if value.ID == id {
			value.EmailVerified = true
			s.byEmail[email] = value
			return nil
		}
	}
	return ErrNotFound
}

func (s *MemoryStore) UpdateProfile(_ context.Context, id int64, username string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for email, value := range s.byEmail {
		if value.ID == id {
			value.Username = username
			s.byEmail[email] = value
			return value, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *MemoryStore) UpdatePassword(_ context.Context, id int64, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for email, value := range s.byEmail {
		if value.ID == id {
			value.PasswordHash = passwordHash
			s.byEmail[email] = value
			return nil
		}
	}
	return ErrNotFound
}

func (s *MemoryStore) Delete(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for email, value := range s.byEmail {
		if value.ID == id {
			delete(s.byEmail, email)
			delete(s.tokens, tokenKey(id, "verify_email"))
			delete(s.tokens, tokenKey(id, "reset_password"))
			return nil
		}
	}
	return ErrNotFound
}

func (s *MemoryStore) SaveToken(_ context.Context, token AccountToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[tokenKey(token.UserID, token.Purpose)] = token
	return nil
}

func (s *MemoryStore) ConsumeToken(_ context.Context, token AccountToken) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tokenKey(token.UserID, token.Purpose)
	stored, exists := s.tokens[key]
	if !exists || stored.TokenHash != token.TokenHash || !stored.ExpiresAt.After(time.Now()) {
		return false, nil
	}
	delete(s.tokens, key)
	return true, nil
}

func tokenKey(userID int64, purpose string) string {
	return fmt.Sprintf("%d:%s", userID, purpose)
}

func (s *MemoryStore) Create(_ context.Context, value User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byEmail[value.Email]; exists {
		return User{}, ErrEmailTaken
	}
	value.ID = s.nextID
	value.CreatedAt = time.Now().UTC()
	s.nextID++
	s.byEmail[value.Email] = value
	return value, nil
}

func (s *MemoryStore) GetByEmail(_ context.Context, email string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, exists := s.byEmail[email]
	if !exists {
		return User{}, ErrInvalidCredentials
	}
	return value, nil
}
