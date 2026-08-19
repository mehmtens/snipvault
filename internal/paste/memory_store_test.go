package paste

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	ownerID := int64(7)
	now := time.Now().UTC()
	value := Paste{Slug: "abc", Title: "Original", Content: "hello", Visibility: "private", UserID: &ownerID, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(ctx, value); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetBySlug(ctx, "abc")
	if err != nil || loaded.Title != "Original" {
		t.Fatal("paste was not loaded")
	}
	listed, _ := store.ListByUser(ctx, ownerID, ListOptions{})
	if len(listed) != 1 {
		t.Fatalf("expected one paste, got %d", len(listed))
	}
	updated, err := store.UpdateByUser(ctx, Paste{Slug: "abc", Title: "Updated", Content: "new", Visibility: "unlisted"}, ownerID)
	if err != nil || updated.Title != "Updated" || updated.UpdatedAt.Before(now) {
		t.Fatal("paste was not updated")
	}
	if deleted, err := store.DeleteByUser(ctx, "abc", ownerID+1); err != nil || deleted {
		t.Fatal("different owner should not delete")
	}
	if deleted, err := store.DeleteByUser(ctx, "abc", ownerID); err != nil || !deleted {
		t.Fatal("owner should delete")
	}
	if _, err := store.GetBySlug(ctx, "abc"); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted paste should be missing")
	}
}

func TestMemoryStoreFiltersAndFavorites(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	ownerID := int64(7)
	otherID := int64(8)
	now := time.Now().UTC()
	values := []Paste{
		{Slug: "go1", Title: "Go helper", Content: "package main", Language: "go", Visibility: "private", UserID: &ownerID, CreatedAt: now, UpdatedAt: now},
		{Slug: "js1", Title: "Frontend", Content: "const helper = true", Language: "javascript", Visibility: "unlisted", UserID: &ownerID, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
		{Slug: "other", Title: "Other", Language: "go", Visibility: "private", UserID: &otherID, CreatedAt: now, UpdatedAt: now},
	}
	for _, value := range values {
		_ = store.Create(ctx, value)
	}
	favorite, err := store.SetFavorite(ctx, "go1", ownerID, true)
	if err != nil || !favorite.IsFavorite {
		t.Fatal("favorite failed")
	}
	filtered, _ := store.ListByUser(ctx, ownerID, ListOptions{Query: "helper", Language: "go", Visibility: "private", FavoritesOnly: true})
	if len(filtered) != 1 || filtered[0].Slug != "go1" {
		t.Fatalf("unexpected filters: %+v", filtered)
	}
	all, _ := store.ListByUser(ctx, ownerID, ListOptions{})
	if len(all) != 2 || all[0].Slug != "go1" {
		t.Fatal("favorite must be sorted first")
	}
	if _, err := store.SetFavorite(ctx, "go1", otherID, true); !errors.Is(err, ErrNotFound) {
		t.Fatal("other owner must not favorite")
	}
}

func TestExpiredPasteIsHidden(t *testing.T) {
	store := NewMemoryStore()
	expired := time.Now().Add(-time.Minute)
	_ = store.Create(context.Background(), Paste{Slug: "expired", Content: "gone", ExpiresAt: &expired})
	if _, err := store.GetBySlug(context.Background(), "expired"); !errors.Is(err, ErrNotFound) {
		t.Fatal("expired paste should be hidden")
	}
}
