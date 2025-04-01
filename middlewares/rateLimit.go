package middlewares

import (
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var (
	limiters = make(map[string]*rate.Limiter)
	mu       sync.Mutex
)

func getLimiter(key string, r rate.Limit, b int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := limiters[key]
	if !exists {
		limiter = rate.NewLimiter(r, b)
		limiters[key] = limiter
	}
	return limiter
}

func RateLimitMiddleware(r rate.Limit, b int, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)
		limiter := getLimiter(key, r, b)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(429, gin.H{"error": "Too many requests. Please slow down :("})
			return
		}

		c.Next()
	}
}
