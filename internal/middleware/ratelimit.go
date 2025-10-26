/*
internal/middleware/ratelimit.go
Package middleware provides rate limiting middleware using token bucket algorithm.
*/

package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for the gateway
type RateLimiter struct {
	limiter *rate.Limiter
	mu      sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond int, burst int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
	}
}

// RateLimitMiddleware creates a middleware that enforces rate limiting
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil {
			c.Next()
			return
		}

		limiter.mu.RLock()
		allowed := limiter.limiter.Allow()
		limiter.mu.RUnlock()

		if !allowed {
			c.Header("X-RateLimit-Limit", "100")
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PerRouteRateLimiter manages rate limiters for individual routes
type PerRouteRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
}

// NewPerRouteRateLimiter creates a new per-route rate limiter
func NewPerRouteRateLimiter() *PerRouteRateLimiter {
	return &PerRouteRateLimiter{
		limiters: make(map[string]*RateLimiter),
	}
}

// AddRoute adds a rate limiter for a specific route
func (prl *PerRouteRateLimiter) AddRoute(path string, requestsPerSecond int, burst int) {
	prl.mu.Lock()
	defer prl.mu.Unlock()
	prl.limiters[path] = NewRateLimiter(requestsPerSecond, burst)
}

// PerRouteRateLimitMiddleware creates a middleware that enforces per-route rate limiting
func PerRouteRateLimitMiddleware(prl *PerRouteRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if prl == nil {
			c.Next()
			return
		}

		prl.mu.RLock()
		limiter, exists := prl.limiters[c.FullPath()]
		prl.mu.RUnlock()

		if !exists {
			c.Next()
			return
		}

		limiter.mu.RLock()
		allowed := limiter.limiter.Allow()
		limiter.mu.RUnlock()

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded for this route",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
