package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"snipvault/internal/auth"
	"snipvault/internal/paste"
	"snipvault/internal/user"
)

func testHandler() http.Handler {
	return NewHandler(paste.NewMemoryStore(), auth.NewService(user.NewMemoryStore(), "test-secret-that-is-long-enough-for-tests"))
}

func TestHomePage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "SnipVault") {
		t.Fatal("expected SnipVault home page")
	}
}

func TestDocumentationEndpoints(t *testing.T) {
	handler := testHandler()
	for _, path := range []string{"/docs", "/openapi.yaml"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "SnipVault") {
			t.Fatalf("documentation endpoint %s failed", path)
		}
	}
}

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

func TestCreateAndGetPaste(t *testing.T) {
	handler := testHandler()
	request := httptest.NewRequest(http.MethodPost, "/pastes", strings.NewReader(`{"title":"Hello","content":"fmt.Println(\"Hello\")","language":"go"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
}

func TestExpirationTime(t *testing.T) {
	if value, err := expirationTime("never"); err != nil || value != nil {
		t.Fatal("never should not create an expiration time")
	}
	value, err := expirationTime("10m")
	if err != nil || value == nil {
		t.Fatal("10m should create an expiration time")
	}
	if _, err := expirationTime("2years"); err == nil {
		t.Fatal("unsupported expiration should fail")
	}
}
