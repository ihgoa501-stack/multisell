package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter provides per-IP rate limiting for sensitive endpoints.
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]*rateEntry
	limit    int
	window   time.Duration
}

type rateEntry struct {
	count    int
	expireAt time.Time
}

// NewRateLimiter creates a rate limiter that allows `limit` requests per `window`.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]*rateEntry),
		limit:    limit,
		window:   window,
	}
}

// Limit returns a Gin middleware that enforces per-IP rate limiting.
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()

		entry, exists := rl.requests[ip]
		if !exists || now.After(entry.expireAt) {
			// New window
			rl.requests[ip] = &rateEntry{count: 1, expireAt: now.Add(rl.window)}
			rl.mu.Unlock()
			c.Next()
			return
		}

		entry.count++
		if entry.count > rl.limit {
			rl.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "rate limit exceeded, please try again later",
			})
			return
		}

		rl.mu.Unlock()
		c.Next()
	}
}

// CleanupPeriodic removes expired entries from the rate limiter.
// Call this in a background goroutine.
func (rl *RateLimiter) CleanupPeriodic(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, entry := range rl.requests {
			if now.After(entry.expireAt) {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}
