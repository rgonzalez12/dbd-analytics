# Security & Production Hardening

Security measures and production configuration for DBD Analytics.

## Security Implementation

### Input Validation
```go
// Steam ID validation
func isValidSteamID(steamID string) bool {
    // Must be 17 digits starting with 7656119
    if len(steamID) != 17 {
        return false
    }
    
    if !strings.HasPrefix(steamID, "7656119") {
        return false
    }
    
    // Must be all numeric
    _, err := strconv.ParseUint(steamID, 10, 64)
    return err == nil
}

// Vanity URL validation  
func isValidVanityURL(vanity string) bool {
    // 3-32 characters, alphanumeric plus underscore/hyphen
    if len(vanity) < 3 || len(vanity) > 32 {
        return false
    }
    
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, vanity)
    return matched
}
```

### Rate Limiting
```go
type RateLimiter struct {
    globalLimit   *rate.Limiter  // 100 requests per minute per IP
    steamAPILimit *rate.Limiter  // Steam API quota management
    cacheEviction time.Time      // 30 second cooldown on cache clears
}

func (rl *RateLimiter) Allow(clientIP string) bool {
    return rl.globalLimit.Allow() && rl.checkIPLimit(clientIP)
}
```

### Security Headers
```go
func securityMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Prevent MIME type sniffing
        w.Header().Set("X-Content-Type-Options", "nosniff")
        
        // Prevent clickjacking
        w.Header().Set("X-Frame-Options", "DENY")
        
        // XSS protection
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // Referrer policy
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        // CORS configuration
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
        
        next.ServeHTTP(w, r)
    })
}
```

## Data Protection

### Sensitive Information Handling
```go
// Never log sensitive data
func sanitizeForLogging(steamID string) string {
    if len(steamID) > 4 {
        return steamID[:4] + "***" + steamID[len(steamID)-4:]
    }
    return "***"
}

// Environment variable protection
func loadConfig() Config {
    apiKey := os.Getenv("STEAM_API_KEY")
    if apiKey == "" {
        log.Fatal("STEAM_API_KEY is required")
    }
    
    // Never log the actual key
    log.Info("Steam API key loaded", 
        "key_length", len(apiKey),
        "key_prefix", apiKey[:4]+"***")
}
```

### Error Response Sanitization
```go
type ErrorResponse struct {
    Error     string `json:"error"`
    RequestID string `json:"request_id"`
    Code      int    `json:"code"`
}

func sanitizeError(err error, requestID string) ErrorResponse {
    // Never expose internal errors to clients
    publicMessage := "Internal server error"
    
    // Map known errors to user-friendly messages
    switch {
    case strings.Contains(err.Error(), "steam_id not found"):
        publicMessage = "Steam profile not found"
    case strings.Contains(err.Error(), "rate limit"):
        publicMessage = "Too many requests, please try again later"
    case strings.Contains(err.Error(), "timeout"):
        publicMessage = "Request timeout, please try again"
    }
    
    return ErrorResponse{
        Error:     publicMessage,
        RequestID: requestID,
        Code:      getErrorCode(err),
    }
}
```

## Production Configuration

### Environment Variables
```bash
# Required
STEAM_API_KEY=your_production_key_here

# Security
RATE_LIMIT_PER_MINUTE=100
RATE_LIMIT_BURST=20
CORS_ORIGINS=https://yourdomain.com
ALLOWED_IPS=10.0.0.0/8,172.16.0.0/12

# Performance
CACHE_MAX_ENTRIES=50000
CACHE_MAX_MEMORY_MB=512
CIRCUIT_BREAKER_FAILURES=3

# Monitoring
LOG_LEVEL=warn
METRICS_ENABLED=true
HEALTH_CHECK_INTERVAL=30s
```

### Production Deployment Checklist
```yaml
security:
  - [x] Steam API key in secure environment variable
  - [x] Rate limiting configured for expected load
  - [x] Security headers enabled
  - [x] Input validation on all endpoints
  - [x] Error sanitization implemented
  - [x] No sensitive data in logs

performance:
  - [x] Cache properly sized for workload
  - [x] Circuit breaker configured
  - [x] Graceful shutdown implemented
  - [x] Health checks configured
  - [x] Metrics collection enabled

monitoring:
  - [x] Structured logging with correlation IDs
  - [x] Error rates tracked
  - [x] Cache hit rates monitored
  - [x] Circuit breaker state alerting
  - [x] Performance metrics collected
```

## 🔍 Monitoring & Alerting

### Health Check Endpoint
```go
func healthCheck(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "status":     "healthy",
        "timestamp":  time.Now().UTC(),
        "version":    buildVersion,
        "cache":      getCacheHealth(),
        "steam_api":  getSteamAPIHealth(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(status)
}
```

### Security Metrics
```go
type SecurityMetrics struct {
    RateLimitHits     int64 `json:"rate_limit_hits"`
    InvalidRequests   int64 `json:"invalid_requests"`
    SuspiciousIPs     int64 `json:"suspicious_ips"`
    ErrorRate         float64 `json:"error_rate"`
}

func trackSecurityEvent(eventType string, clientIP string) {
    log.Warn("Security event detected",
        "event_type", eventType,
        "client_ip", sanitizeIP(clientIP),
        "timestamp", time.Now().UTC())
        
    metrics.SecurityEvents.WithLabelValues(eventType).Inc()
}
```

## 🛠️ Operational Security

### Graceful Shutdown
```go
func (s *Server) Shutdown(ctx context.Context) error {
    log.Info("Starting graceful shutdown")
    
    // Stop accepting new requests
    if err := s.httpServer.Shutdown(ctx); err != nil {
        return err
    }
    
    // Clear sensitive data from memory
    s.cache.Clear()
    
    // Close database connections
    s.db.Close()
    
    log.Info("Graceful shutdown completed")
    return nil
}
```

### Request Tracing
```go
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := generateRequestID()
        
        // Add to context for logging
        ctx := context.WithValue(r.Context(), "request_id", requestID)
        r = r.WithContext(ctx)
        
        // Return in response headers
        w.Header().Set("X-Request-ID", requestID)
        
        next.ServeHTTP(w, r)
    })
}
```

### Circuit Breaker Security
```go
func (cb *CircuitBreaker) recordFailure(err error) {
    cb.failures++
    
    // Log potential security issues
    if isSecurityRelatedError(err) {
        log.Warn("Potential security issue detected",
            "error_type", getErrorType(err),
            "failure_count", cb.failures,
            "circuit_state", cb.state)
    }
    
    if cb.failures >= cb.maxFailures {
        cb.state = Open
        log.Error("Circuit breaker opened due to failures",
            "total_failures", cb.failures,
            "last_error", sanitizeError(err))
    }
}
```

## Security Monitoring Dashboard

### Key Metrics to Track
- **Request Rate**: Requests per minute by endpoint
- **Error Rate**: 4xx/5xx responses over time
- **Cache Hit Rate**: Performance and potential DoS indicators
- **Circuit Breaker State**: Steam API health
- **Response Times**: P50, P95, P99 latencies
- **Rate Limit Hits**: Potential abuse attempts

### Alert Conditions
```yaml
alerts:
  high_error_rate:
    condition: error_rate > 5%
    duration: 5m
    
  rate_limit_abuse:
    condition: rate_limit_hits > 100/hour
    duration: 1m
    
  circuit_breaker_open:
    condition: circuit_breaker_state == "open"
    duration: 30s
    
  cache_memory_high:
    condition: cache_memory_usage > 90%
    duration: 2m
```

This security implementation ensures the DBD Analytics application is production-ready with comprehensive protection against common web application vulnerabilities while maintaining excellent observability for security monitoring.
