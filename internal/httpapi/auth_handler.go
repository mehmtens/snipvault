package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"snipvault/internal/mailer"
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
	address, err := mail.ParseAddress(input.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "enter a valid email address"})
		return
	}
	input.Email = address.Address
	if len(input.Password) < 8 || len(input.Password) > 72 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be between 8 and 72 characters"})
		return
	}

	value, _, err := h.auth.Register(r.Context(), input.Username, input.Email, input.Password)
	if errors.Is(err, user.ErrEmailTaken) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	if errors.Is(err, mailer.ErrUnavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "email delivery is temporarily unavailable"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "account could not be created"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"email": value.Email, "verification_required": true, "message": "verification code sent"})
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var input authRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	value, token, err := h.auth.Login(r.Context(), input.Email, input.Password)
	if errors.Is(err, user.ErrEmailNotVerified) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
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

type codeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}
type resetRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}
type profileRequest struct {
	Username string `json:"username"`
}
type passwordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
type deleteAccountRequest struct {
	Password string `json:"password"`
}

func (h *handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var input codeRequest
	if decodeJSON(w, r, &input) != nil || !validCode(input.Code) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "enter the 6-digit code"})
		return
	}
	value, token, err := h.auth.VerifyEmail(r.Context(), input.Email, input.Code)
	if errors.Is(err, user.ErrInvalidToken) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "email could not be verified"})
		return
	}
	h.writeAuthSession(w, r, http.StatusOK, value, token)
}

func (h *handler) resendVerification(w http.ResponseWriter, r *http.Request) {
	var input codeRequest
	if decodeJSON(w, r, &input) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if err := h.auth.ResendVerification(r.Context(), input.Email); errors.Is(err, mailer.ErrUnavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "email delivery is temporarily unavailable"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "code could not be sent"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "if the account needs verification, a new code was sent"})
}

func (h *handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var input codeRequest
	if decodeJSON(w, r, &input) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if err := h.auth.ForgotPassword(r.Context(), input.Email); errors.Is(err, mailer.ErrUnavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "email delivery is temporarily unavailable"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "request could not be completed"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "if that account exists, a reset code was sent"})
}

func (h *handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var input resetRequest
	if decodeJSON(w, r, &input) != nil || !validCode(input.Code) || !validPassword(input.Password) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "enter a valid code and an 8 to 72 character password"})
		return
	}
	if err := h.auth.ResetPassword(r.Context(), input.Email, input.Code, input.Password); errors.Is(err, user.ErrInvalidToken) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "password could not be reset"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) profile(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	value, err := h.auth.Profile(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "profile could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireProtectedUser(w, r)
	if !ok {
		return
	}
	var input profileRequest
	if decodeJSON(w, r, &input) != nil || len(strings.TrimSpace(input.Username)) < 3 || len(strings.TrimSpace(input.Username)) > 50 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username must be between 3 and 50 characters"})
		return
	}
	value, err := h.auth.UpdateProfile(r.Context(), userID, input.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "profile could not be updated"})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *handler) changePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireProtectedUser(w, r)
	if !ok {
		return
	}
	var input passwordRequest
	if decodeJSON(w, r, &input) != nil || !validPassword(input.NewPassword) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new password must be between 8 and 72 characters"})
		return
	}
	if err := h.auth.ChangePassword(r.Context(), userID, input.CurrentPassword, input.NewPassword); errors.Is(err, user.ErrInvalidCredentials) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "current password is incorrect"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "password could not be changed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireProtectedUser(w, r)
	if !ok {
		return
	}
	var input deleteAccountRequest
	if decodeJSON(w, r, &input) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if err := h.auth.DeleteAccount(r.Context(), userID, input.Password); errors.Is(err, user.ErrInvalidCredentials) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "password is incorrect"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "account could not be deleted"})
		return
	}
	h.clearAuthSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) requireUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok, err := h.currentUserID(r)
	if err != nil || !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "sign in required"})
		return 0, false
	}
	return id, true
}
func (h *handler) requireProtectedUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := h.requireUser(w, r)
	if !ok {
		return 0, false
	}
	if !h.validCSRF(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid CSRF token"})
		return 0, false
	}
	return id, true
}
func validCode(value string) bool {
	if len(strings.TrimSpace(value)) != 6 {
		return false
	}
	for _, char := range strings.TrimSpace(value) {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
func validPassword(value string) bool { return len(value) >= 8 && len(value) <= 72 }

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
	h.clearAuthSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) clearAuthSession(w http.ResponseWriter, r *http.Request) {
	secure := requestIsSecure(r)
	http.SetCookie(w, &http.Cookie{Name: "snipvault_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(w, &http.Cookie{Name: "snipvault_csrf", Value: "", Path: "/", MaxAge: -1, Secure: secure, SameSite: http.SameSiteStrictMode})
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
