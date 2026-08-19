package httpapi

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"snipvault/internal/paste"
)

func (h *handler) currentUserID(r *http.Request) (int64, bool, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	var tokenString string
	if header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return 0, false, errInvalidToken
		}
		tokenString = strings.TrimSpace(parts[1])
	} else {
		cookie, err := r.Cookie("snipvault_session")
		if errors.Is(err, http.ErrNoCookie) {
			return 0, false, nil
		}
		if err != nil || cookie.Value == "" {
			return 0, false, errInvalidToken
		}
		tokenString = cookie.Value
	}
	userID, err := h.auth.Parse(tokenString)
	if err != nil {
		return 0, false, errInvalidToken
	}
	return userID, true, nil
}

func (h *handler) validCSRF(r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		return true
	}
	session, sessionErr := r.Cookie("snipvault_session")
	if sessionErr != nil || session.Value == "" {
		return true
	}
	cookie, err := r.Cookie("snipvault_csrf")
	header := r.Header.Get("X-CSRF-Token")
	if err != nil || cookie.Value == "" || header == "" || len(cookie.Value) != len(header) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) == 1
}

func (h *handler) updatePaste(w http.ResponseWriter, r *http.Request) {
	userID, authenticated, err := h.currentUserID(r)
	if err != nil || !authenticated {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	if !h.validCSRF(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid CSRF token"})
		return
	}
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
	if input.Visibility != "public" && input.Visibility != "unlisted" && input.Visibility != "private" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "visibility must be public, unlisted or private"})
		return
	}
	expiresAt, err := expirationTime(input.ExpiresIn)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	value, err := h.store.UpdateByUser(r.Context(), paste.Paste{
		Slug: r.PathValue("slug"), Title: strings.TrimSpace(input.Title), Content: input.Content,
		Language: strings.TrimSpace(input.Language), Visibility: input.Visibility, ExpiresAt: expiresAt,
	}, userID)
	if errors.Is(err, paste.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "paste not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "paste could not be updated"})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *handler) myPastes(w http.ResponseWriter, r *http.Request) {
	userID, authenticated, err := h.currentUserID(r)
	if err != nil || !authenticated {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	values, err := h.store.ListByUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pastes could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, values)
}

func (h *handler) deletePaste(w http.ResponseWriter, r *http.Request) {
	userID, authenticated, err := h.currentUserID(r)
	if err != nil || !authenticated {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	if !h.validCSRF(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid CSRF token"})
		return
	}
	deleted, err := h.store.DeleteByUser(r.Context(), r.PathValue("slug"), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "paste could not be deleted"})
		return
	}
	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "paste not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
