package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func registerToken(t *testing.T, handler http.Handler) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"username":"owner","email":"owner@example.com","password":"password123"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("register failed: %s", response.Body.String())
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "snipvault_session" {
			return cookie.Value
		}
	}
	t.Fatal("session cookie not found")
	return ""
}

func TestPrivatePasteOwnership(t *testing.T) {
	handler := testHandler()
	token := registerToken(t, handler)

	create := httptest.NewRequest(http.MethodPost, "/pastes", strings.NewReader(`{"title":"Secret","content":"private text","visibility":"private"}`))
	create.Header.Set("Authorization", "Bearer "+token)
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", created.Code, created.Body.String())
	}
	var value struct {
		Slug string `json:"slug"`
	}
	_ = json.Unmarshal(created.Body.Bytes(), &value)

	privateRead := httptest.NewRequest(http.MethodGet, "/pastes/"+value.Slug, nil)
	privateResponse := httptest.NewRecorder()
	handler.ServeHTTP(privateResponse, privateRead)
	if privateResponse.Code != http.StatusNotFound {
		t.Fatalf("anonymous user should not read private paste, got %d", privateResponse.Code)
	}

	rawAnonymous := httptest.NewRequest(http.MethodGet, "/raw/"+value.Slug, nil)
	rawAnonymousResponse := httptest.NewRecorder()
	handler.ServeHTTP(rawAnonymousResponse, rawAnonymous)
	if rawAnonymousResponse.Code != http.StatusNotFound {
		t.Fatalf("anonymous user should not read private raw paste, got %d", rawAnonymousResponse.Code)
	}

	rawOwner := httptest.NewRequest(http.MethodGet, "/raw/"+value.Slug, nil)
	rawOwner.Header.Set("Authorization", "Bearer "+token)
	rawOwnerResponse := httptest.NewRecorder()
	handler.ServeHTTP(rawOwnerResponse, rawOwner)
	if rawOwnerResponse.Code != http.StatusOK || rawOwnerResponse.Body.String() != "private text" {
		t.Fatalf("owner should read raw paste: %s", rawOwnerResponse.Body.String())
	}

	list := httptest.NewRequest(http.MethodGet, "/me/pastes", nil)
	list.Header.Set("Authorization", "Bearer "+token)
	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, list)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), "Secret") {
		t.Fatalf("owner should list paste: %s", listed.Body.String())
	}

	update := httptest.NewRequest(http.MethodPut, "/pastes/"+value.Slug, strings.NewReader(`{"title":"Updated secret","content":"updated text","language":"text","visibility":"private","expires_in":"never"}`))
	update.Header.Set("Authorization", "Bearer "+token)
	updated := httptest.NewRecorder()
	handler.ServeHTTP(updated, update)
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), "Updated secret") {
		t.Fatalf("owner should update paste: %s", updated.Body.String())
	}

	remove := httptest.NewRequest(http.MethodDelete, "/pastes/"+value.Slug, nil)
	remove.Header.Set("Authorization", "Bearer "+token)
	removed := httptest.NewRecorder()
	handler.ServeHTTP(removed, remove)
	if removed.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", removed.Code)
	}
}

func TestCookieAuthenticationRequiresCSRF(t *testing.T) {
	handler := testHandler()
	register := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"username":"cookieuser","email":"cookie@example.com","password":"password123"}`))
	registered := httptest.NewRecorder()
	handler.ServeHTTP(registered, register)
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	_ = json.Unmarshal(registered.Body.Bytes(), &body)
	cookies := registered.Result().Cookies()

	withoutCSRF := httptest.NewRequest(http.MethodPost, "/pastes", strings.NewReader(`{"content":"protected","visibility":"private"}`))
	for _, cookie := range cookies {
		withoutCSRF.AddCookie(cookie)
	}
	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, withoutCSRF)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without CSRF, got %d", denied.Code)
	}

	withCSRF := httptest.NewRequest(http.MethodPost, "/pastes", strings.NewReader(`{"content":"protected","visibility":"private"}`))
	for _, cookie := range cookies {
		withCSRF.AddCookie(cookie)
	}
	withCSRF.Header.Set("X-CSRF-Token", body.CSRFToken)
	allowed := httptest.NewRecorder()
	handler.ServeHTTP(allowed, withCSRF)
	if allowed.Code != http.StatusCreated {
		t.Fatalf("expected 201 with CSRF, got %d: %s", allowed.Code, allowed.Body.String())
	}
}
