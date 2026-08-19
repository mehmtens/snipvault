package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIsSecureBehindHTTPSProxy(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://snipvault.test/", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	if !requestIsSecure(request) {
		t.Fatal("expected forwarded HTTPS request to be secure")
	}
}

func TestRegisterAndLogin(t *testing.T) {
	handler := testHandler()
	register := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"username":"mehmet","email":"mehmet@example.com","password":"password123"}`))
	registerResponse := httptest.NewRecorder()
	handler.ServeHTTP(registerResponse, register)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", registerResponse.Code, registerResponse.Body.String())
	}
	if !strings.Contains(registerResponse.Body.String(), `"csrf_token"`) {
		t.Fatal("expected CSRF token")
	}
	var sessionCookie *http.Cookie
	for _, cookie := range registerResponse.Result().Cookies() {
		if cookie.Name == "snipvault_session" {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("expected HttpOnly SameSite=Strict session cookie")
	}

	login := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"mehmet@example.com","password":"password123"}`))
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", loginResponse.Code, loginResponse.Body.String())
	}
}
