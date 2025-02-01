package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	routing "github.com/go-ozzo/ozzo-routing/v2"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	visitors        map[string]*visitor
	blockedVisitors map[string]time.Time
	mu              sync.RWMutex
	r               rate.Limit
	b               int
	blockedDuration time.Duration
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

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		visitors:        make(map[string]*visitor),
		blockedVisitors: make(map[string]time.Time),
		r:               r,
		b:               b,
		blockedDuration: 30 * time.Second,
	}

	// Start cleanup routine
	go rl.cleanupVisitors()
	return rl
}

func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(30 * time.Second)

		rl.mu.Lock()
		now := time.Now()

		// Remove expired blocked IPs
		for ip, blockedTime := range rl.blockedVisitors {
			if now.Sub(blockedTime) > 30*time.Second {
				delete(rl.blockedVisitors, ip)
				log.Println("✅ IP: ", ip, " has been unblocked during cleanup routine")
			}
		}

		// Remove old visitors
		for ip, v := range rl.visitors {
			if now.Sub(v.lastSeen) > 3*time.Minute {
				log.Println("✅ IP: ", ip, " removed from visitors list due to inactivity")
				delete(rl.visitors, ip)
			}
		}

		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if _, blocked := rl.blockedVisitors[ip]; blocked {
		return nil // Indicate that this IP is temporarily blocked
	}

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.r, rl.b)
		rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func RateLimitMiddleware(rl *RateLimiter) routing.Handler {
	return func(c *routing.Context) error {
		ip := getIP(c.Request)

		rl.mu.Lock()
		blockedTime, isBlocked := rl.blockedVisitors[ip]
		now := time.Now()
		rl.mu.Unlock()

		if isBlocked {
			if now.Sub(blockedTime) < rl.blockedDuration {
				// Reset block time
				rl.mu.Lock()
				rl.blockedVisitors[ip] = now
				rl.mu.Unlock()

				log.Println("🚫 IP: ", ip, " block reinstated on line 114 due to successive requests")
				return c.WriteWithStatus(map[string]string{
					"error":       "Too many requests",
					"retry-after": rl.blockedDuration.String(),
				}, http.StatusTooManyRequests)
			}

			log.Println("🕰️ IP: ", ip, " will be unblocked during cleanup routine")
		}

		limiter := rl.GetLimiter(ip)
		if limiter == nil || !limiter.Allow() {
			rl.mu.Lock()
			rl.blockedVisitors[ip] = now
			rl.mu.Unlock()

			log.Println("🚫 IP: ", ip, " has been blocked due to rate limiting")
			return c.WriteWithStatus(map[string]string{
				"error":       "Too many requests",
				"retry-after": rl.blockedDuration.String(),
			}, http.StatusTooManyRequests)
		}

		log.Printf("✅ Allowed IP: %s, Rate Limit: %v", ip, limiter.Limit())
		return c.Next()
	}
}
