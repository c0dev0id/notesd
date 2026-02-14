package api

import (
	"net/http"
	"sync"
	"time"
)

// rateLimiter implements a simple fixed-window rate limiter keyed by IP.
type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
	limit   int
	period  time.Duration
}

type window struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(limit int, period time.Duration) *rateLimiter {
	return &rateLimiter{
		windows: make(map[string]*window),
		limit:   limit,
		period:  period,
	}
}

// allow checks if a request from the given key is allowed.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	w, ok := rl.windows[key]
	if !ok || now.After(w.resetAt) {
		rl.windows[key] = &window{count: 1, resetAt: now.Add(rl.period)}
		return true
	}

	w.count++
	return w.count <= rl.limit
}

// cleanup removes expired entries. Called periodically.
func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for k, w := range rl.windows {
		if now.After(w.resetAt) {
			delete(rl.windows, k)
		}
	}
}

// rateLimit wraps a handler with rate limiting.
func (rl *rateLimiter) rateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if !rl.allow(key) {
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next(w, r)
	}
}
