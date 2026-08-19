package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerDelegatesToServerlessApp(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	Handler(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected delegated 503, got %d", response.Code)
	}
}
