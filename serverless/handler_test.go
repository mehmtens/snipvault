package serverless

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestHandlerRejectsMissingConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	initialize = sync.Once{}
	app = nil
	startupErr = nil

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	Handler(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "initialization failed") {
		t.Fatalf("unexpected response body: %q", response.Body.String())
	}
}
