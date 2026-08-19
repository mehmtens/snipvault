package paste

import (
	"context"
	"sort"
	"strings"
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
	value.IsFavorite = current.IsFavorite
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

func (s *MemoryStore) ListByUser(_ context.Context, userID int64, options ListOptions) ([]Paste, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make([]Paste, 0)
	for _, value := range s.pastes {
		matchesQuery := options.Query == "" || strings.Contains(strings.ToLower(value.Title), strings.ToLower(options.Query)) || strings.Contains(strings.ToLower(value.Content), strings.ToLower(options.Query))
		if value.UserID != nil && *value.UserID == userID && (value.ExpiresAt == nil || value.ExpiresAt.After(time.Now())) && matchesQuery && (options.Language == "" || value.Language == options.Language) && (options.Visibility == "" || value.Visibility == options.Visibility) && (!options.FavoritesOnly || value.IsFavorite) {
			values = append(values, value)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].IsFavorite != values[j].IsFavorite {
			return values[i].IsFavorite
		}
		return values[i].UpdatedAt.After(values[j].UpdatedAt)
	})
	return values, nil
}

func (s *MemoryStore) SetFavorite(_ context.Context, slug string, userID int64, favorite bool) (Paste, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, exists := s.pastes[slug]
	if !exists || value.UserID == nil || *value.UserID != userID {
		return Paste{}, ErrNotFound
	}
	value.IsFavorite = favorite
	value.UpdatedAt = time.Now().UTC()
	s.pastes[slug] = value
	return value, nil
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
