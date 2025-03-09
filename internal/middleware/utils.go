package middleware

import (
	"net/http"
	"strings"
)

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
