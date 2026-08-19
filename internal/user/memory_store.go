package user

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu      sync.Mutex
	nextID  int64
	byEmail map[string]User
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{nextID: 1, byEmail: make(map[string]User)}
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
