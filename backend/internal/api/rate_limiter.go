package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type clientBucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

type IPRateLimiter struct {
	rate       float64 // tokens per second
	capacity   float64 // max burst capacity
	clients    sync.Map
	stopSweep  chan struct{}
}

func NewIPRateLimiter(rate float64, capacity float64) *IPRateLimiter {
	limiter := &IPRateLimiter{
		rate:      rate,
		capacity:  capacity,
		stopSweep: make(chan struct{}),
	}

	// Periodic cleanup of stale IPs every 10 minutes
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cutoff := time.Now().Add(-15 * time.Minute)
				limiter.clients.Range(func(key, value interface{}) bool {
					bucket := value.(*clientBucket)
					bucket.mu.Lock()
					if bucket.lastRefill.Before(cutoff) {
						limiter.clients.Delete(key)
					}
					bucket.mu.Unlock()
					return true
				})
			case <-limiter.stopSweep:
				return
			}
		}
	}()

	return limiter
}

func (l *IPRateLimiter) getIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (l *IPRateLimiter) Allow(r *http.Request) bool {
	ip := l.getIP(r)
	now := time.Now()

	val, _ := l.clients.LoadOrStore(ip, &clientBucket{
		tokens:     l.capacity,
		lastRefill: now,
	})
	bucket := val.(*clientBucket)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.lastRefill = now
	bucket.tokens += elapsed * l.rate
	if bucket.tokens > l.capacity {
		bucket.tokens = l.capacity
	}

	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

func (l *IPRateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(r) {
			RespondError(w, http.StatusTooManyRequests, "too many requests, please try again shortly")
			return
		}
		next(w, r)
	}
}
