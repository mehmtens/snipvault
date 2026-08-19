package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type visitor struct {
	count   int
	resetAt time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]visitor
	limit    int
	window   time.Duration
	requests uint64
	now      func() time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{visitors: make(map[string]visitor), limit: limit, window: window, now: time.Now}
}

func (l *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, remaining, resetAt := l.allow(clientIP(r))
		w.Header().Set("RateLimit-Limit", strconv.Itoa(l.limit))
		w.Header().Set("RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int(time.Until(resetAt).Seconds()))))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests; try again later"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *RateLimiter) allow(ip string) (bool, int, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.requests++
	if l.requests%100 == 0 {
		for key, value := range l.visitors {
			if !value.resetAt.After(now) {
				delete(l.visitors, key)
			}
		}
	}
	value, exists := l.visitors[ip]
	if !exists || !value.resetAt.After(now) {
		value = visitor{resetAt: now.Add(l.window)}
	}
	if value.count >= l.limit {
		return false, 0, value.resetAt
	}
	value.count++
	l.visitors[ip] = value
	return true, l.limit - value.count, value.resetAt
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
