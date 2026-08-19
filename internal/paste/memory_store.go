package paste

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu     sync.RWMutex
	pastes map[string]Paste
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{pastes: make(map[string]Paste)}
}

func (s *MemoryStore) Create(_ context.Context, value Paste) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pastes[value.Slug] = value
	return nil
}

func (s *MemoryStore) UpdateByUser(_ context.Context, value Paste, userID int64) (Paste, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.pastes[value.Slug]
	if !exists || current.UserID == nil || *current.UserID != userID {
		return Paste{}, ErrNotFound
	}
	value.CreatedAt = current.CreatedAt
	value.UpdatedAt = time.Now().UTC()
	value.UserID = current.UserID
	s.pastes[value.Slug] = value
	return value, nil
}

func (s *MemoryStore) GetBySlug(_ context.Context, slug string) (Paste, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.pastes[slug]
	if !ok || (value.ExpiresAt != nil && !value.ExpiresAt.After(time.Now())) {
		return Paste{}, ErrNotFound
	}
	return value, nil
}

func (s *MemoryStore) ListByUser(_ context.Context, userID int64) ([]Paste, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make([]Paste, 0)
	for _, value := range s.pastes {
		if value.UserID != nil && *value.UserID == userID && (value.ExpiresAt == nil || value.ExpiresAt.After(time.Now())) {
			values = append(values, value)
		}
	}
	return values, nil
}

func (s *MemoryStore) DeleteByUser(_ context.Context, slug string, userID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, exists := s.pastes[slug]
	if !exists || value.UserID == nil || *value.UserID != userID {
		return false, nil
	}
	delete(s.pastes, slug)
	return true, nil
}
