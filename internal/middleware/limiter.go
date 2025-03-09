package middleware

import (
	"log"
	"sync"
	"time"

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

var maxBlockedVisitors = 100

// GetLimiter returns the existing limiter for an IP or creates a new one
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {

	//Cleaver way rate limit the rate limiter
	// if the number of blocked visitors is greater than the maxBlockedVisitors
	// then block the IP

	if len(rl.blockedVisitors) > maxBlockedVisitors {
		rl.blockedVisitors[ip] = time.Now()
		log.Println("🚫 IP", ip, "blocked due to rate limiting")
		return nil
	}

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

// cleanupVisitors removes inactive visitors and expired blocked IPs
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(15 * time.Second)

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

// NewRateLimiter initializes the rate limiter instance
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		visitors:        make(map[string]*visitor),
		blockedVisitors: make(map[string]time.Time),
		r:               r,
		b:               b,
		blockedDuration: 15 * time.Second,
	}

	// Start a background cleanup routine
	go rl.cleanupVisitors()
	return rl
}
