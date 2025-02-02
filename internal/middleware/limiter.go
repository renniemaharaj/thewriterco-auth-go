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

// visitor stores rate limiter and last activity time
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter tracks active visitors and blocked IPs
type RateLimiter struct {
	visitors        map[string]*visitor
	blockedVisitors map[string]time.Time
	mu              sync.RWMutex
	r               rate.Limit
	b               int
	blockedDuration time.Duration
}

// getIP extracts the real IP address from request headers or RemoteAddr
func getIP(r *http.Request) string {
	// Check X-Forwarded-For header (useful for proxies and load balancers)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0] // Take first IP in the chain
	}
	// Extract IP without port
	return strings.Split(r.RemoteAddr, ":")[0]
}

// NewRateLimiter initializes the rate limiter instance
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		visitors:        make(map[string]*visitor),
		blockedVisitors: make(map[string]time.Time),
		r:               r,
		b:               b,
		blockedDuration: 30 * time.Second,
	}

	// Start a background cleanup routine
	go rl.cleanupVisitors()
	return rl
}

// cleanupVisitors removes inactive visitors and expired blocked IPs
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(30 * time.Second)

		rl.mu.Lock()
		now := time.Now()

		// Unblock IPs if the block duration has expired
		for ip, blockedTime := range rl.blockedVisitors {
			if now.Sub(blockedTime) > rl.blockedDuration {
				delete(rl.blockedVisitors, ip)
				log.Println("✅ IP unblocked:", ip)
			}
		}

		// Remove inactive visitors to free memory
		for ip, v := range rl.visitors {
			if now.Sub(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
				log.Println("✅ Removed inactive visitor:", ip)
			}
		}

		rl.mu.Unlock()
	}
}

// GetLimiter returns the existing limiter for an IP or creates a new one
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// If IP is blocked, return nil
	if _, blocked := rl.blockedVisitors[ip]; blocked {
		return nil
	}

	// Return existing limiter if available
	if v, exists := rl.visitors[ip]; exists {
		v.lastSeen = time.Now()
		return v.limiter
	}

	// Create and store a new limiter for this IP
	limiter := rate.NewLimiter(rl.r, rl.b)
	rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

// RateLimitMiddleware applies rate limiting to incoming requests
func RateLimitMiddleware(rl *RateLimiter) routing.Handler {
	return func(c *routing.Context) error {
		ip := getIP(c.Request)

		// Check if IP is blocked
		rl.mu.RLock()
		blockedTime, isBlocked := rl.blockedVisitors[ip]
		now := time.Now()
		rl.mu.RUnlock()

		if isBlocked {
			if now.Sub(blockedTime) < rl.blockedDuration {
				// Reset block time on further attempts
				rl.mu.Lock()
				rl.blockedVisitors[ip] = now
				rl.mu.Unlock()

				log.Println("🚫 IP", ip, "block reinstated due to repeated requests")
				return c.WriteWithStatus(map[string]string{
					"response":    "Hold reinstated, retry after 30 seconds",
					"retry-after": rl.blockedDuration.String(),
				}, http.StatusTooManyRequests)
			}
		}

		// Get the rate limiter for this IP
		limiter := rl.GetLimiter(ip)
		if limiter == nil || !limiter.Allow() {
			rl.mu.Lock()
			rl.blockedVisitors[ip] = now
			rl.mu.Unlock()

			log.Println("🚫 IP", ip, "blocked due to rate limiting")
			c.WriteWithStatus(map[string]string{
				"response":    "Too many requests, retry after 30 seconds",
				"retry-after": rl.blockedDuration.String(),
			}, http.StatusTooManyRequests)
			c.Abort() // 🔥 Prevents further execution
			return nil
		}

		log.Printf("✅ Allowed IP: %s, Rate Limit: %v", ip, limiter.Limit())
		return c.Next()
	}
}
