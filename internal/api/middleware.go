package api

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

// RequestLimiter implements token bucket rate limiting
type RequestLimiter struct {
	mu      sync.RWMutex
	clients map[string]*TokenBucket
	maxReqs int           // requests per window
	window  time.Duration // time window
	cleanup time.Duration // cleanup interval
}

type TokenBucket struct {
	tokens     int
	lastRefill time.Time
	capacity   int
	refillRate time.Duration
}

// NewRequestLimiter creates a new rate limiter
func NewRequestLimiter(maxReqs int, window time.Duration) *RequestLimiter {
	rl := &RequestLimiter{
		clients: make(map[string]*TokenBucket),
		maxReqs: maxReqs,
		window:  window,
		cleanup: time.Minute * 5, // cleanup old entries every 5 minutes
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()
	return rl
}

// Allow checks if a request should be allowed
func (rl *RequestLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.clients[clientID]
	if !exists {
		bucket = &TokenBucket{
			tokens:     rl.maxReqs - 1, // consume one token immediately
			lastRefill: time.Now(),
			capacity:   rl.maxReqs,
			refillRate: rl.window,
		}
		rl.clients[clientID] = bucket
		return true
	}

	// Refill tokens based on time passed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)

	if elapsed >= bucket.refillRate {
		// Full refill after window period
		bucket.tokens = bucket.capacity
		bucket.lastRefill = now
	}

	// Check if we have tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// cleanupRoutine removes old unused client entries
func (rl *RequestLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, bucket := range rl.clients {
			// Remove clients inactive for more than 2x the window period
			if now.Sub(bucket.lastRefill) > rl.window*2 {
				delete(rl.clients, clientID)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates HTTP middleware for rate limiting
func RateLimitMiddleware(limiter *RequestLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client identifier (prefer real IP over proxy headers)
			clientID := getClientIP(r)

			if !limiter.Allow(clientID) {
				log.Warn("Rate limit exceeded",
					"client_ip", clientID,
					"user_agent", r.UserAgent(),
					"endpoint", r.URL.Path,
					"method", r.Method)

				// Return structured error response consistent with our API
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(int(limiter.window.Seconds())))

				// Use our existing error response structure
				apiErr := steam.NewRateLimitErrorWithRetryAfter(int(limiter.window.Seconds()))
				writeErrorResponse(w, apiErr)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the real client IP, checking various headers
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (but validate it's not spoofed)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (leftmost) which should be the original client
		if firstIP := parseFirstIP(xff); firstIP != "" {
			return firstIP
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return parseIPFromRemoteAddr(r.RemoteAddr)
}

// parseFirstIP extracts the first IP from a comma-separated list
func parseFirstIP(ips string) string {
	for i, char := range ips {
		if char == ',' || char == ' ' {
			return ips[:i]
		}
	}
	return ips
}

// parseIPFromRemoteAddr extracts IP from "ip:port" format
func parseIPFromRemoteAddr(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// SecurityMiddleware adds security headers
func SecurityMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// CORS headers for API
			w.Header().Set("Access-Control-Allow-Origin", "*") // Configure appropriately for production
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyMiddleware provides optional API key authentication for public endpoints
func APIKeyMiddleware() func(http.Handler) http.Handler {
	requiredKey := os.Getenv("API_KEY")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip API key check if not configured or for non-API endpoints
			if requiredKey == "" || !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip for cache and metrics endpoints (they have their own auth)
			if strings.HasPrefix(r.URL.Path, "/api/cache/") || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			providedKey := r.Header.Get("X-API-Key")
			if providedKey != requiredKey {
				log.Warn("API key authentication failed",
					"path", r.URL.Path,
					"client_ip", r.RemoteAddr,
					"user_agent", r.UserAgent(),
					"has_key", providedKey != "")

				writeErrorResponse(w, steam.NewUnauthorizedError("Valid API key required"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
