package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	routing "github.com/go-ozzo/ozzo-routing/v2"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	r        rate.Limit
	b        int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		r:        r,
		b:        b,
	}

	// Start cleanup routine
	go rl.cleanupVisitors()
	return rl
}

func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.r, rl.b)
		rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func getIP(r *http.Request) string {
	// Check X-Forwarded-For header
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	// Extract IP without port
	return strings.Split(r.RemoteAddr, ":")[0]
}

func RateLimitMiddleware(rl *RateLimiter) routing.Handler {

	// forceReject is a flag to reject all requests with 429 status code
	forceReject := false

	return func(c *routing.Context) error {
		if forceReject {
			return c.WriteWithStatus(map[string]string{
				"error":       "Too many requests",
				"retry-after": "60",
			}, http.StatusTooManyRequests)
		}

		ip := getIP(c.Request)
		limiter := rl.GetLimiter(ip)

		if !limiter.Allow() {
			return c.WriteWithStatus(map[string]string{
				"error":       "Too many requests",
				"retry-after": "60",
			}, http.StatusTooManyRequests)
		}
		return c.Next()
	}
}
