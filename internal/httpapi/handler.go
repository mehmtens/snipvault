package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"snipvault/internal/auth"
	"snipvault/internal/middleware"
	"snipvault/internal/paste"
)

var errInvalidToken = fmt.Errorf("invalid token")

type createPasteRequest struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	Language   string `json:"language"`
	Visibility string `json:"visibility"`
	ExpiresIn  string `json:"expires_in"`
}

type handler struct {
	store paste.Store
	auth  *auth.Service
}

func NewHandler(store paste.Store, authService *auth.Service) http.Handler {
	h := &handler{store: store, auth: authService}
	mux := http.NewServeMux()
	authLimiter := middleware.NewRateLimiter(10, time.Minute)
	registerUI(mux)
	mux.HandleFunc("GET /health", h.health)
	mux.Handle("POST /register", authLimiter.Handler(http.HandlerFunc(h.register)))
	mux.Handle("POST /login", authLimiter.Handler(http.HandlerFunc(h.login)))
	mux.HandleFunc("POST /logout", h.logout)
	mux.HandleFunc("POST /pastes", h.createPaste)
	mux.HandleFunc("GET /pastes/{slug}", h.getPaste)
	mux.HandleFunc("GET /raw/{slug}", h.rawPaste)
	mux.HandleFunc("DELETE /pastes/{slug}", h.deletePaste)
	mux.HandleFunc("PUT /pastes/{slug}", h.updatePaste)
	mux.HandleFunc("GET /me/pastes", h.myPastes)
	generalLimiter := middleware.NewRateLimiter(120, time.Minute)
	return middleware.SecurityHeaders(middleware.RequestID(middleware.Logging(generalLimiter.Handler(mux))))
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) createPaste(w http.ResponseWriter, r *http.Request) {
	var input createPasteRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}
	if input.Visibility == "" {
		input.Visibility = "unlisted"
	}
	if input.Visibility != "public" && input.Visibility != "unlisted" && input.Visibility != "private" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "visibility must be public, unlisted or private"})
		return
	}
	userID, authenticated, err := h.currentUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
		return
	}
	if authenticated && !h.validCSRF(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid CSRF token"})
		return
	}
	if input.Visibility == "private" && !authenticated {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "sign in to create a private paste"})
		return
	}
	expiresAt, err := expirationTime(input.ExpiresIn)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	slug, err := newSlug()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create paste"})
		return
	}

	value := paste.Paste{
		Slug: slug, Title: strings.TrimSpace(input.Title), Content: input.Content,
		Language: strings.TrimSpace(input.Language), Visibility: input.Visibility,
		CreatedAt: time.Now().UTC(), ExpiresAt: expiresAt,
	}
	value.UpdatedAt = value.CreatedAt
	if authenticated {
		value.UserID = &userID
	}
	if err := h.store.Create(r.Context(), value); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save paste"})
		return
	}

	writeJSON(w, http.StatusCreated, value)
}

func expirationTime(value string) (*time.Time, error) {
	var duration time.Duration
	switch strings.TrimSpace(value) {
	case "", "never":
		return nil, nil
	case "10m":
		duration = 10 * time.Minute
	case "1h":
		duration = time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	default:
		return nil, fmt.Errorf("expires_in must be never, 10m, 1h, 24h or 7d")
	}
	expiresAt := time.Now().UTC().Add(duration)
	return &expiresAt, nil
}

func (h *handler) getPaste(w http.ResponseWriter, r *http.Request) {
	value, status, err := h.visiblePaste(r)
	if err != nil {
		writeJSON(w, status, map[string]string{"error": "paste not found"})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *handler) visiblePaste(r *http.Request) (paste.Paste, int, error) {
	value, err := h.store.GetBySlug(r.Context(), r.PathValue("slug"))
	if errors.Is(err, paste.ErrNotFound) {
		return paste.Paste{}, http.StatusNotFound, err
	}
	if err != nil {
		return paste.Paste{}, http.StatusInternalServerError, err
	}
	if value.Visibility == "private" {
		userID, authenticated, authErr := h.currentUserID(r)
		if authErr != nil || !authenticated || value.UserID == nil || *value.UserID != userID {
			return paste.Paste{}, http.StatusNotFound, paste.ErrNotFound
		}
	}
	return value, http.StatusOK, nil
}

func (h *handler) rawPaste(w http.ResponseWriter, r *http.Request) {
	value, status, err := h.visiblePaste(r)
	if err != nil {
		http.Error(w, "paste not found", status)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.txt"`, value.Slug))
	}
	_, _ = w.Write([]byte(value.Content))
}

func newSlug() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
