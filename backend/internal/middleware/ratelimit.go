package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Per-IP token bucket (docs/08 M8: rate limits verified). In-memory map is
// fine for a single API replica; swap for Redis if we ever scale out.
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
}

type visitor struct {
	lim  *rate.Limiter
	seen int64 // last unix nano, for cleanup
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{visitors: make(map[string]*visitor)}
}

func (rl *rateLimiter) get(ip string, r rate.Limit, b int) *rate.Limiter {
	now := time.Now().UnixNano()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if len(rl.visitors) > maxVisitors {
		for k, v := range rl.visitors {
			if now-v.seen > staleVisitorNs {
				delete(rl.visitors, k)
			}
		}
	}

	v, ok := rl.visitors[ip]
	if !ok {
		v = &visitor{lim: rate.NewLimiter(r, b), seen: now}
		rl.visitors[ip] = v
	}
	v.seen = now
	return v.lim
}

// RateLimit returns a middleware allowing r events per second with burst b.
// On limit exceed it responds 429 TOO_MANY_REQUESTS via the app error envelope.
func RateLimit(r rate.Limit, b int) gin.HandlerFunc {
	rl := newRateLimiter()
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.get(ip, r, b).Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Too many requests. Slow down and try again shortly.",
				},
			})
			return
		}
		c.Next()
	}
}

const maxVisitors = 10_000
const staleVisitorNs = int64(3_600_000_000_000) // 1h
