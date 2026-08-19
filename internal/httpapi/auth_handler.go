package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"snipvault/internal/user"
)

type authRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *handler) register(w http.ResponseWriter, r *http.Request) {
	var input authRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if len(input.Username) < 3 || len(input.Username) > 50 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username must be between 3 and 50 characters"})
		return
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "enter a valid email address"})
		return
	}
	if len(input.Password) < 8 || len(input.Password) > 72 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be between 8 and 72 characters"})
		return
	}

	value, token, err := h.auth.Register(r.Context(), input.Username, input.Email, input.Password)
	if errors.Is(err, user.ErrEmailTaken) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "account could not be created"})
		return
	}
	h.writeAuthSession(w, r, http.StatusCreated, value, token)
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var input authRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	value, token, err := h.auth.Login(r.Context(), input.Email, input.Password)
	if errors.Is(err, user.ErrInvalidCredentials) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login failed"})
		return
	}
	h.writeAuthSession(w, r, http.StatusOK, value, token)
}

func (h *handler) writeAuthSession(w http.ResponseWriter, r *http.Request, status int, value user.User, token string) {
	csrfToken := randomToken()
	secure := requestIsSecure(r)
	http.SetCookie(w, &http.Cookie{Name: "snipvault_session", Value: token, Path: "/", MaxAge: 86400, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(w, &http.Cookie{Name: "snipvault_csrf", Value: csrfToken, Path: "/", MaxAge: 86400, HttpOnly: false, Secure: secure, SameSite: http.SameSiteStrictMode})
	writeJSON(w, status, map[string]any{"user": value, "csrf_token": csrfToken})
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	if !h.validCSRF(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid CSRF token"})
		return
	}
	secure := requestIsSecure(r)
	http.SetCookie(w, &http.Cookie{Name: "snipvault_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(w, &http.Cookie{Name: "snipvault_csrf", Value: "", Path: "/", MaxAge: -1, HttpOnly: false, Secure: secure, SameSite: http.SameSiteStrictMode})
	w.WriteHeader(http.StatusNoContent)
}

func requestIsSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func randomToken() string {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(value)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
