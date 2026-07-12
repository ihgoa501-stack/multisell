package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter provides per-IP rate limiting for sensitive endpoints.
type RateLimiter struct {
	mu          sync.Mutex
	requests    map[string]*rateEntry
	limit       int
	window      time.Duration
	lastCleanup time.Time
	maxEntries  int
}

type rateEntry struct {
	count    int
	expireAt time.Time
}

// NewRateLimiter creates a rate limiter that allows `limit` requests per `window`.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:   make(map[string]*rateEntry),
		limit:      limit,
		window:     window,
		maxEntries: 10000,
	}
}

// Limit returns a Gin middleware that enforces per-IP rate limiting.
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()
		if rl.lastCleanup.IsZero() || now.Sub(rl.lastCleanup) >= rl.window {
			rl.cleanupExpiredLocked(now)
			rl.lastCleanup = now
		}

		entry, exists := rl.requests[ip]
		if !exists || now.After(entry.expireAt) {
			if !exists && len(rl.requests) >= rl.maxEntries {
				rl.mu.Unlock()
				abortRateLimited(c)
				return
			}
			// New window
			rl.requests[ip] = &rateEntry{count: 1, expireAt: now.Add(rl.window)}
			rl.mu.Unlock()
			c.Next()
			return
		}

		entry.count++
		if entry.count > rl.limit {
			rl.mu.Unlock()
			abortRateLimited(c)
			return
		}

		rl.mu.Unlock()
		c.Next()
	}
}

func abortRateLimited(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code": 429, "message": "rate limit exceeded, please try again later",
	})
}

func (rl *RateLimiter) cleanupExpiredLocked(now time.Time) {
	for ip, entry := range rl.requests {
		if now.After(entry.expireAt) {
			delete(rl.requests, ip)
		}
	}
}
