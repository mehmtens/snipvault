package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterRejectsExcessRequests(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for attempt := 1; attempt <= 3; attempt++ {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.RemoteAddr = "192.0.2.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if attempt <= 2 && response.Code != http.StatusOK {
			t.Fatalf("attempt %d should pass", attempt)
		}
		if attempt == 3 && response.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429, got %d", response.Code)
		}
	}
}

func TestSecurityHeadersAndRequestID(t *testing.T) {
	handler := SecurityHeaders(RequestID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing CSP")
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing request ID")
	}
	if response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing frame protection")
	}
}
