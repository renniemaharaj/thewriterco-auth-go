package middleware

import (
	"log"
	"net/http"
	"time"

	routing "github.com/go-ozzo/ozzo-routing/v2"
)

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
