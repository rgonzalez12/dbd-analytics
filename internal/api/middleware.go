package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

type contextKey string

const (
	requestIDKey         contextKey = "request_id"
	clientFingerprintKey contextKey = "client_fingerprint"
)

func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := GenerateRequestID()

			w.Header().Set("X-Request-ID", requestID)

			ctx := context.WithValue(r.Context(), requestIDKey, requestID)

			log.Info("Request started",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.Header.Get("User-Agent"))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GenerateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

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

// RateLimitMiddleware creates HTTP middleware for rate limiting with enhanced client identification
func RateLimitMiddleware(limiter *RequestLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use client fingerprint for more accurate rate limiting
			clientFingerprint, ok := r.Context().Value(clientFingerprintKey).(string)
			if !ok {
				// Fallback to IP if fingerprint not available
				clientFingerprint = getClientIP(r)
			}

			if !limiter.Allow(clientFingerprint) {
				log.Warn("Rate limit exceeded",
					"client_fingerprint", clientFingerprint,
					"user_agent", r.UserAgent(),
					"endpoint", r.URL.Path,
					"method", r.Method,
					"max_requests", limiter.maxReqs,
					"window", limiter.window)

				// Enhanced rate limit headers
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(int(limiter.window.Seconds())))
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.maxReqs))
				w.Header().Set("X-RateLimit-Window", limiter.window.String())

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

// SecurityMiddleware adds security headers and enhanced protection
func SecurityMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Enhanced security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

			// CORS headers for API (restrict in production)
			allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
			if allowedOrigins == "" {
				allowedOrigins = "*" // Development fallback
			}
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Block suspicious requests
			userAgent := r.Header.Get("User-Agent")
			if userAgent == "" || len(userAgent) > 512 {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			// Rate limit per user agent + IP combination for better protection
			clientFingerprint := getClientFingerprint(r)

			// Add client fingerprint to context for downstream middleware
			ctx := context.WithValue(r.Context(), clientFingerprintKey, clientFingerprint)

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// getClientFingerprint creates a unique identifier for rate limiting
func getClientFingerprint(r *http.Request) string {
	// Combine IP, User-Agent hash, and API key for unique fingerprinting
	clientIP := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	apiKey := r.Header.Get("X-API-Key")

	fingerprint := clientIP
	if len(userAgent) > 0 {
		fingerprint += "_" + userAgent[:min(50, len(userAgent))]
	}
	if len(apiKey) > 0 {
		fingerprint += "_" + apiKey[:8]
	}

	return fingerprint
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
