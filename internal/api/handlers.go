package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings" 
	"time"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/cache"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
	"github.com/rgonzalez12/dbd-analytics/internal/models"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

const (
	DefaultRequestTimeout = 5 * time.Second
	SteamAPITimeout      = 3 * time.Second
	CacheTimeout         = 1 * time.Second
)

var (
	digitOnlyRegex = regexp.MustCompile(`^\d+$`)
	vanityURLRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

type Handler struct {
	steamClient      *steam.Client
	cacheManager     *cache.Manager
	lastEvictionTime time.Time
}

func NewHandler() *Handler {
	// Initialize cache with player stats configuration
	cacheManager, err := cache.NewManager(cache.PlayerStatsConfig())
	if err != nil {
		log.Error("Failed to initialize cache manager, proceeding without cache",
			"error", err,
			"fallback", "direct_steam_api_calls")
		return &Handler{
			steamClient: steam.NewClient(),
		}
	}

	log.Info("API handler initialized with caching enabled",
		"cache_type", string(cacheManager.GetConfig().Type),
		"max_entries", cacheManager.GetConfig().Memory.MaxEntries,
		"default_ttl", cacheManager.GetConfig().Memory.DefaultTTL)

	return &Handler{
		steamClient:  steam.NewClient(),
		cacheManager: cacheManager,
	}
}

func convertToPlayerStats(dbdStats steam.DBDPlayerStats) models.PlayerStats {
	return models.PlayerStats{
		// Core player identification
		SteamID:     dbdStats.SteamID,
		DisplayName: dbdStats.DisplayName,

		// Progression metrics
		KillerPips:   dbdStats.Killer.KillerPips,
		SurvivorPips: dbdStats.Survivor.SurvivorPips,

		// Killer statistics
		KilledCampers:     dbdStats.Killer.TotalKills,
		SacrificedCampers: dbdStats.Killer.SacrificedVictims,
		MoriKills:         dbdStats.Killer.MoriKills,
		HooksPerformed:    dbdStats.Killer.HooksPerformed,
		UncloakAttacks:    dbdStats.Killer.UncloakAttacks,

		// Survivor statistics
		GeneratorPct:         dbdStats.Survivor.GeneratorsCompleted,
		HealPct:              dbdStats.Survivor.HealingCompleted,
		EscapesKO:            dbdStats.Survivor.EscapesKnockedOut,
		Escapes:              dbdStats.Survivor.TotalEscapes,
		SkillCheckSuccess:    dbdStats.Survivor.SkillChecksHit,
		HookedAndEscape:      dbdStats.Survivor.HookedAndEscaped,
		UnhookOrHeal:         dbdStats.Survivor.UnhooksPerformed,
		HealsPerformed:       dbdStats.Survivor.HealsPerformed,
		UnhookOrHealPostExit: dbdStats.Survivor.PostExitActions,
		PostExitActions:      dbdStats.Survivor.PostExitActions,
		EscapeThroughHatch:   dbdStats.Survivor.EscapesThroughHatch,

		// Game progression
		BloodwebPoints: dbdStats.General.BloodwebPoints,

		// Achievement counters
		CamperPerfectGames: dbdStats.Survivor.PerfectGames,
		KillerPerfectGames: dbdStats.Killer.PerfectGames,

		// Equipment tracking
		CamperFullLoadout: dbdStats.Survivor.FullLoadoutGames,
		KillerFullLoadout: dbdStats.Killer.FullLoadoutGames,
		CamperNewItem:     dbdStats.Survivor.NewItemsFound,

		// General game statistics
		TotalMatches: dbdStats.General.TotalMatches,
		TimePlayed:   dbdStats.General.TimePlayed,

		// Metadata
		LastUpdated: dbdStats.General.LastUpdated,
	}
}

type ResponseBuilder struct {
	data      map[string]interface{}
	timestamp string
}

func NewResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{
		data:      make(map[string]interface{}),
		timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// AddData adds key-value data to the response
func (rb *ResponseBuilder) AddData(key string, value interface{}) *ResponseBuilder {
	rb.data[key] = value
	return rb
}

func (rb *ResponseBuilder) AddCacheStats(stats cache.CacheStats, cacheType string) *ResponseBuilder {
	rb.data["cache_stats"] = stats
	rb.data["cache_type"] = cacheType
	return rb
}

func (rb *ResponseBuilder) AddPerformanceMetrics(stats cache.CacheStats) *ResponseBuilder {
	totalRequests := stats.Hits + stats.Misses
	performance := "excellent"
	if stats.HitRate < 75 && totalRequests > 10 {
		performance = "good"
	}
	if stats.HitRate < 50 && totalRequests > 50 {
		performance = "critical"
	}

	rb.data["performance"] = map[string]interface{}{
		"assessment":      performance,
		"total_requests":  totalRequests,
		"memory_usage_mb": float64(stats.MemoryUsage) / 1024 / 1024,
		"uptime_hours":    float64(stats.UptimeSeconds) / 3600,
		"ops_per_second": func() float64 {
			if stats.UptimeSeconds > 0 {
				return float64(totalRequests) / float64(stats.UptimeSeconds)
			}
			return 0
		}(),
	}

	// Add performance recommendations
	var recs []string
	if stats.HitRate < 70 && totalRequests > 100 {
		recs = append(recs, "Consider increasing TTL values or reviewing cache key strategy")
	}
	if stats.LRUEvictions > stats.ExpiredKeys*2 {
		recs = append(recs, "High LRU eviction rate - consider increasing cache capacity")
	}
	if performance == "critical" {
		recs = append(recs, "Critical: Cache not providing benefits - review implementation")
	}
	if len(recs) == 0 {
		recs = append(recs, "Cache performance is optimal")
	}
	rb.data["recommendations"] = recs
	
	return rb
}

// AddTimestamp adds timestamp to response
func (rb *ResponseBuilder) AddTimestamp() *ResponseBuilder {
	rb.data["timestamp"] = rb.timestamp
	return rb
}

func (rb *ResponseBuilder) Build() map[string]interface{} {
	rb.AddTimestamp()
	return rb.data
}

func validateSteamID(steamID string) bool {
	// Steam ID must be exactly 17 digits in length
	if len(steamID) != 17 {
		return false
	}

	// Ensure all characters are numeric
	if _, err := strconv.ParseUint(steamID, 10, 64); err != nil {
		return false
	}

	// Steam IDs must start with 7656119 (Steam's standardized base)
	return steamID[:7] == "7656119"
}

func isValidVanityURL(vanity string) bool {
	// Steam vanity URLs must be 3-32 characters, alphanumeric plus underscore/hyphen
	if len(vanity) < 3 || len(vanity) > 32 {
		return false
	}

	// Use pre-compiled regex for optimal performance
	return vanityURLRegex.MatchString(vanity)
}

func validateSteamIDOrVanity(input string) *steam.APIError {
	if input == "" {
		return steam.NewValidationError("Steam ID or vanity URL required")
	}

	// Prevent excessively long inputs that could cause DoS
	if len(input) > 64 {
		return steam.NewValidationError("Input too long. Steam ID must be 17 digits or vanity URL 3-32 characters")
	}

	// If input starts with Steam ID prefix (7656119), validate as Steam ID
	if len(input) >= 7 && input[:7] == "7656119" {
		if !validateSteamID(input) {
			return steam.NewValidationError("Invalid Steam ID format. Must be 17 digits starting with 7656119")
		}
		return nil
	}

	// If it's all digits but doesn't start with Steam prefix, it's an invalid Steam ID
	if digitOnlyRegex.MatchString(input) {
		return steam.NewValidationError("Invalid Steam ID format. Must be 17 digits starting with 7656119")
	}

	// Otherwise validate as vanity URL
	if !isValidVanityURL(input) {
		return steam.NewValidationError("Invalid vanity URL format. Must be 3-32 characters, alphanumeric with underscore/hyphen only")
	}

	return nil
}

func (h *Handler) GetPlayerSummary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), DefaultRequestTimeout)
	defer cancel()
	
	steamID := mux.Vars(r)["steamid"]

	// Create structured logger with comprehensive request context
	requestLogger := log.HTTPRequestContext(r.Method, r.URL.Path, steamID, r.RemoteAddr)

	// Validate Steam ID format before making any external API calls
	if err := validateSteamIDOrVanity(steamID); err != nil {
		log.ErrorContext(string(err.Type), steamID).Warn("Invalid Steam ID format in GetPlayerSummary",
			"user_agent", r.UserAgent(),
			"error_message", err.Message,
			"validation_type", string(err.Type))
		writeSteamAPIError(w, r, err)
		return
	}

	// Check cache first if caching is enabled
	var cacheKey string
	var cacheHit bool
	if h.cacheManager != nil {
		cacheKey = cache.GenerateKey(cache.PlayerSummaryPrefix, steamID)
		if cached, found := h.cacheManager.GetCache().Get(cacheKey); found {
			if summary, ok := cached.(*steam.SteamPlayer); ok {
				cacheHit = true
				durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
				log.PerformanceContext("cache_hit", steamID, durationMs).Info("Cache hit for player summary",
					"persona_name", summary.PersonaName,
					"cache_key", cacheKey,
					"cache_status", "hit")
				writeJSONResponse(w, summary)
				return
			} else {
				// Invalid cache entry type - this indicates a cache corruption issue
				h.cacheManager.GetCache().Delete(cacheKey)
				log.ErrorContext("cache_corruption", steamID).Error("Cache corruption detected: invalid entry type",
					"cache_key", cacheKey,
					"expected_type", "*steam.SteamPlayer",
					"action", "cache_entry_removed")
			}
		} else {
			requestLogger.Debug("Cache miss for player summary",
				"cache_key", cacheKey,
				"cache_status", "miss")
		}
	}

	requestLogger.Info("Processing player summary request",
		"cache_hit", cacheHit,
		"operation", "steam_api_call")

	// Check context timeout before making Steam API call
	select {
	case <-ctx.Done():
		writeTimeoutError(w, r, "player_summary")
		return
	default:
	}

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		// Log Steam API errors with comprehensive context for debugging
		durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
		log.ErrorContext(string(err.Type), steamID).Error("Failed to get player summary",
			"error_message", err.Message,
			"retryable", err.Retryable,
			"duration_ms", durationMs)
		writeSteamAPIError(w, r, err)
		return
	}

	// Store in cache if caching is enabled
	if h.cacheManager != nil && cacheKey != "" {
		// Check context timeout before caching
		select {
		case <-ctx.Done():
			// Don't fail if we can't cache - just log and return the data we have
			requestLogger.Info("Context timeout during cache write", "steam_id", steamID)
		default:
			config := h.cacheManager.GetConfig()
			// Cache with timeout protection (but don't use the context in Set call for now)
			if err := h.cacheManager.GetCache().Set(cacheKey, summary, config.TTL.PlayerSummary); err != nil {
				requestLogger.Error("Failed to cache player summary",
					"error", err,
					"cache_key", cacheKey,
					"cache_status", "set_failed")
			} else {
				requestLogger.Debug("Player summary cached",
					"cache_key", cacheKey,
					"ttl", config.TTL.PlayerSummary,
					"cache_status", "set_success")
			}

			// Log cache performance stats periodically
			if stats := h.cacheManager.GetCache().Stats(); (stats.Hits+stats.Misses)%100 == 0 {
				requestLogger.Info("Cache performance snapshot",
					"hit_rate", fmt.Sprintf("%.1f%%", stats.HitRate),
					"total_operations", stats.Hits+stats.Misses,
					"entries", stats.Entries)
			}
		}
	}

	durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
	log.PerformanceContext("player_summary_success", steamID, durationMs).Info("Successfully processed player summary request",
		"persona_name", summary.PersonaName,
		"cache_hit", cacheHit)
	writeJSONResponse(w, summary)
}

func (h *Handler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), DefaultRequestTimeout)
	defer cancel()
	
	start := time.Now()
	steamID := mux.Vars(r)["steamid"]

	// Create structured logger with comprehensive request context
	requestLogger := log.HTTPRequestContext(r.Method, r.URL.Path, steamID, r.RemoteAddr)

	if err := validateSteamIDOrVanity(steamID); err != nil {
		log.ErrorContext(string(err.Type), steamID).Warn("Invalid Steam ID format in GetPlayerStats",
			"user_agent", r.UserAgent(),
			"error_message", err.Message,
			"validation_type", string(err.Type))
		writeValidationError(w, r, err.Message, "steam_id")
		return
	}

	// Check cache first if caching is enabled
	var cacheKey string
	var cacheHit bool
	if h.cacheManager != nil {
		cacheKey = cache.GenerateKey(cache.PlayerStatsPrefix, steamID)
		if cached, found := h.cacheManager.GetCache().Get(cacheKey); found {
			if playerStats, ok := cached.(models.PlayerStats); ok {
				cacheHit = true
				durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
				log.PerformanceContext("cache_hit", steamID, durationMs).Info("Cache hit for player stats",
					"display_name", playerStats.DisplayName,
					"cache_key", cacheKey)
				writeJSONResponse(w, playerStats)
				return
			} else {
				// Invalid cache entry type - log and remove
				requestLogger.Warn("Invalid cache entry type, removing",
					"cache_key", cacheKey,
					"expected", "models.PlayerStats",
					"actual", fmt.Sprintf("%T", cached))
				if err := h.cacheManager.GetCache().Delete(cacheKey); err != nil {
					requestLogger.Error("Failed to delete invalid cache entry", "error", err)
				}
			}
		}
	}

	requestLogger.Info("Processing player stats request", "cache_hit", cacheHit)

	// Check context timeout before making Steam API calls
	select {
	case <-ctx.Done():
		writeTimeoutError(w, r, "player_stats")
		return
	default:
	}

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		requestLogger.Error("Failed to get player summary for stats request",
			"error", err.Message,
			"error_type", string(err.Type),
			"duration", time.Since(start))
		writeSteamAPIError(w, r, err)
		return
	}

	// Check context timeout before second API call
	select {
	case <-ctx.Done():
		writeTimeoutError(w, r, "player_stats")
		return
	default:
	}

	rawStats, err := h.steamClient.GetPlayerStats(steamID)
	if err != nil {
		requestLogger.Error("Failed to get player stats",
			"persona_name", summary.PersonaName,
			"error", err.Message,
			"error_type", string(err.Type),
			"duration", time.Since(start))
		writeSteamAPIError(w, r, err)
		return
	}

	playerStats := steam.MapSteamStats(rawStats.Stats, summary.SteamID, summary.PersonaName)

	// Convert nested structure to flat API response format
	flatPlayerStats := convertToPlayerStats(playerStats)

	// Store in cache if caching is enabled
	if h.cacheManager != nil && cacheKey != "" {
		config := h.cacheManager.GetConfig()
		if err := h.cacheManager.GetCache().Set(cacheKey, flatPlayerStats, config.TTL.PlayerStats); err != nil {
			requestLogger.Error("Failed to cache player stats",
				"error", err,
				"cache_key", cacheKey,
				"stats_size", len(fmt.Sprintf("%+v", flatPlayerStats)))
			// Don't fail the request if caching fails - log and continue
		} else {
			requestLogger.Debug("Player stats cached successfully",
				"cache_key", cacheKey,
				"ttl", config.TTL.PlayerStats,
				"display_name", flatPlayerStats.DisplayName)
		}
	}

	requestLogger.Info("Successfully processed player stats request",
		"raw_stats_count", len(rawStats.Stats),
		"persona_name", summary.PersonaName,
		"duration", time.Since(start),
		"cache_hit", cacheHit)
	writeJSONResponse(w, flatPlayerStats)
}

// GetCacheStats returns comprehensive cache performance metrics for monitoring
func (h *Handler) GetCacheStats(w http.ResponseWriter, r *http.Request) {
	if h.cacheManager == nil {
		writeErrorResponse(w, steam.NewInternalError(fmt.Errorf("caching not enabled")))
		return
	}

	stats := h.cacheManager.GetCache().Stats()

	// Calculate additional derived metrics
	totalRequests := stats.Hits + stats.Misses

	// Performance assessment
	performance := "excellent"
	if stats.HitRate < 90 && totalRequests > 1000 {
		performance = "good"
	}
	if stats.HitRate < 70 && totalRequests > 100 {
		performance = "poor"
	}
	if stats.HitRate < 50 && totalRequests > 50 {
		performance = "critical"
	}

	// Create comprehensive response using ResponseBuilder
	response := NewResponseBuilder().
		AddCacheStats(stats, string(h.cacheManager.GetConfig().Type)).
		AddPerformanceMetrics(stats).
		Build()

	log.Info("Cache stats requested",
		"hits", stats.Hits,
		"misses", stats.Misses,
		"hit_rate", fmt.Sprintf("%.1f%%", stats.HitRate),
		"entries", stats.Entries,
		"performance", performance,
		"total_requests", totalRequests)

	writeJSONResponse(w, response)
}

// EvictExpiredEntries manually triggers cache cleanup and returns statistics
// This is an admin-only endpoint and should be protected in production
func (h *Handler) EvictExpiredEntries(w http.ResponseWriter, r *http.Request) {
	// Rate limiting: only allow one eviction per 30 seconds
	if !h.checkEvictionRateLimit(r) {
		w.Header().Set("Retry-After", "30")
		writeErrorResponse(w, steam.NewRateLimitErrorWithRetryAfter(30))
		return
	}

	// Basic admin check - in production, replace with proper auth
	adminToken := r.Header.Get("X-Admin-Token")
	if adminToken == "" {
		log.Warn("Unauthorized cache eviction attempt - missing token",
			"client_ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"path", r.URL.Path)
		writeErrorResponse(w, steam.NewUnauthorizedError("Admin token required"))
		return
	}

	if adminToken != "test-token" {
		log.Warn("Unauthorized cache eviction attempt - invalid token",
			"client_ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"path", r.URL.Path,
			"token_provided", true)
		writeErrorResponse(w, steam.NewForbiddenError("Invalid admin token"))
		return
	}

	if h.cacheManager == nil {
		writeErrorResponse(w, steam.NewInternalError(fmt.Errorf("caching not enabled")))
		return
	}

	start := time.Now()
	evicted := h.cacheManager.GetCache().EvictExpired()
	stats := h.cacheManager.GetCache().Stats()
	duration := time.Since(start)

	response := NewResponseBuilder().
		AddData("evicted_entries", evicted).
		AddData("remaining_entries", stats.Entries).
		AddData("duration_ms", duration.Milliseconds()).
		AddData("admin_initiated", true).
		Build()

	log.Info("Manual cache eviction completed",
		"evicted", evicted,
		"remaining", stats.Entries,
		"duration", duration,
		"admin_token_provided", adminToken != "",
		"client_ip", r.RemoteAddr)

	writeJSONResponse(w, response)
}

func (h *Handler) isMetricsAccessAllowed(r *http.Request) bool {
	// Production metrics endpoint security
	// In production, you'd configure these from environment variables
	allowedIPs := []string{
		"127.0.0.1",
		"::1",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16", // Private network ranges
	}

	clientIP := r.RemoteAddr
	// Extract IP from "IP:port" format
	if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
		clientIP = clientIP[:colon]
	}

	// Simple IP allowlist check (in production, use proper CIDR matching)
	for _, allowedIP := range allowedIPs {
		if strings.Contains(allowedIP, "/") {
			// CIDR range - simple prefix check for common private ranges
			if allowedIP == "10.0.0.0/8" && strings.HasPrefix(clientIP, "10.") {
				return true
			}
			if allowedIP == "172.16.0.0/12" && strings.HasPrefix(clientIP, "172.16.") {
				return true
			}
			if allowedIP == "192.168.0.0/16" && strings.HasPrefix(clientIP, "192.168.") {
				return true
			}
			continue
		}
		if clientIP == allowedIP {
			return true
		}
	}

	// For development/testing, allow all local traffic
	if strings.HasPrefix(clientIP, "127.") || strings.HasPrefix(clientIP, "192.168.") ||
		strings.HasPrefix(clientIP, "10.") || strings.HasPrefix(clientIP, "172.16.") ||
		clientIP == "::1" {
		return true
	}

	return false
}

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	// Security: Only allow metrics scraping from specific IPs in production
	if !h.isMetricsAccessAllowed(r) {
		log.Warn("Metrics access denied",
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"security_concern", "unauthorized_metrics_access")
		writeErrorResponse(w, steam.NewForbiddenError("Metrics access denied"))
		return
	}

	if h.cacheManager == nil {
		writeErrorResponse(w, steam.NewInternalError(fmt.Errorf("caching not enabled")))
		return
	}

	stats := h.cacheManager.GetCache().Stats()

	// Generate Prometheus-style metrics
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	fmt.Fprintf(w, "# HELP cache_hits_total Total number of cache hits\n")
	fmt.Fprintf(w, "# TYPE cache_hits_total counter\n")
	fmt.Fprintf(w, "cache_hits_total %d\n", stats.Hits)

	fmt.Fprintf(w, "# HELP cache_misses_total Total number of cache misses\n")
	fmt.Fprintf(w, "# TYPE cache_misses_total counter\n")
	fmt.Fprintf(w, "cache_misses_total %d\n", stats.Misses)

	fmt.Fprintf(w, "# HELP cache_evictions_total Total number of cache evictions\n")
	fmt.Fprintf(w, "# TYPE cache_evictions_total counter\n")
	fmt.Fprintf(w, "cache_evictions_total %d\n", stats.Evictions)

	fmt.Fprintf(w, "# HELP cache_lru_evictions_total Total number of LRU evictions\n")
	fmt.Fprintf(w, "# TYPE cache_lru_evictions_total counter\n")
	fmt.Fprintf(w, "cache_lru_evictions_total %d\n", stats.LRUEvictions)

	fmt.Fprintf(w, "# HELP cache_corruption_events_total Total number of corruption events detected\n")
	fmt.Fprintf(w, "# TYPE cache_corruption_events_total counter\n")
	fmt.Fprintf(w, "cache_corruption_events_total %d\n", stats.CorruptionEvents)

	fmt.Fprintf(w, "# HELP cache_recovery_events_total Total number of recovery operations performed\n")
	fmt.Fprintf(w, "# TYPE cache_recovery_events_total counter\n")
	fmt.Fprintf(w, "cache_recovery_events_total %d\n", stats.RecoveryEvents)

	fmt.Fprintf(w, "# HELP cache_entries Current number of cache entries\n")
	fmt.Fprintf(w, "# TYPE cache_entries gauge\n")
	fmt.Fprintf(w, "cache_entries %d\n", stats.Entries)

	fmt.Fprintf(w, "# HELP cache_memory_usage_bytes Current memory usage in bytes\n")
	fmt.Fprintf(w, "# TYPE cache_memory_usage_bytes gauge\n")
	fmt.Fprintf(w, "cache_memory_usage_bytes %d\n", stats.MemoryUsage)

	fmt.Fprintf(w, "# HELP cache_hit_rate_percent Current hit rate percentage\n")
	fmt.Fprintf(w, "# TYPE cache_hit_rate_percent gauge\n")
	fmt.Fprintf(w, "cache_hit_rate_percent %.2f\n", stats.HitRate)

	fmt.Fprintf(w, "# HELP cache_uptime_seconds Cache uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE cache_uptime_seconds gauge\n")
	fmt.Fprintf(w, "cache_uptime_seconds %d\n", stats.UptimeSeconds)

	log.Debug("Prometheus metrics served",
		"client_ip", r.RemoteAddr,
		"hit_rate", fmt.Sprintf("%.1f%%", stats.HitRate),
		"entries", stats.Entries)
}

// checkEvictionRateLimit implements basic rate limiting for cache eviction
func (h *Handler) checkEvictionRateLimit(r *http.Request) bool {
	now := time.Now()
	if now.Sub(h.lastEvictionTime) < 30*time.Second {
		log.Warn("Cache eviction rate limited",
			"client_ip", r.RemoteAddr,
			"time_since_last", now.Sub(h.lastEvictionTime),
			"required_wait", 30*time.Second)
		return false
	}
	h.lastEvictionTime = now
	return true
}

func (h *Handler) Close() error {
	if h.cacheManager != nil {
		return h.cacheManager.Close()
	}
	return nil
}

func writeErrorResponse(w http.ResponseWriter, apiErr *steam.APIError) {
	// Generate a unique request ID for tracing
	requestID := generateRequestID()

	// Determine the appropriate HTTP status code
	statusCode := determineStatusCode(apiErr)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(statusCode)

	// Create enhanced error response format
	errorResponse := map[string]interface{}{
		"error":      apiErr.Message,
		"type":       string(apiErr.Type),
		"request_id": requestID,
	}

	// Add details for specific error types
	switch apiErr.Type {
	case steam.ErrorTypeRateLimit:
		errorResponse["details"] = "Steam API rate limit exceeded"
		// Use the actual retry_after value from the Steam API response
		retryAfter := 60 // Default fallback
		if apiErr.RetryAfter > 0 {
			retryAfter = apiErr.RetryAfter
		}
		errorResponse["retry_after"] = retryAfter

	case steam.ErrorTypeAPIError:
		if apiErr.StatusCode != 0 {
			errorResponse["details"] = fmt.Sprintf("Steam API returned %d %s", apiErr.StatusCode, http.StatusText(apiErr.StatusCode))
			// Differentiate client vs server errors from Steam
			if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
				errorResponse["source"] = "client_error"
			} else {
				errorResponse["source"] = "steam_api_error"
			}
		}
		if apiErr.Retryable {
			errorResponse["retry_after"] = 30 // Retry after 30 seconds for API errors
		}

	case steam.ErrorTypeNetwork:
		errorResponse["details"] = "Network connection to Steam API failed"
		errorResponse["source"] = "steam_api_error"
		errorResponse["retry_after"] = 30

	case steam.ErrorTypeNotFound:
		errorResponse["details"] = "Requested resource not found on Steam"
		errorResponse["source"] = "client_error"

	case steam.ErrorTypeValidation:
		errorResponse["details"] = "Invalid request parameters"
		errorResponse["source"] = "client_error"

	case steam.ErrorTypeInternal:
		errorResponse["details"] = "Internal server error occurred"
		errorResponse["source"] = "server_error"
	}

	// Add retryable flag for client guidance
	if apiErr.Retryable {
		errorResponse["retryable"] = true
	}

	// Log the error with request ID for tracing
	log.Error("API error response generated",
		"request_id", requestID,
		"error_type", string(apiErr.Type),
		"status_code", statusCode,
		"error_message", apiErr.Message)

	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Error("Failed to encode error response",
			"request_id", requestID,
			"error", err.Error(),
			"original_error", apiErr.Message)
		// Fallback to plain text if JSON encoding fails
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func determineStatusCode(apiErr *steam.APIError) int {
	// If the error already has a status code, use it but map appropriately
	if apiErr.StatusCode != 0 {
		switch apiErr.Type {
		case steam.ErrorTypeAPIError:
			// For Steam API errors, differentiate client vs server issues
			if apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusNotFound {
				// Steam returned 403/404 - pass through as client error
				return apiErr.StatusCode
			} else if apiErr.StatusCode >= 500 {
				// Steam server errors - return 502 Bad Gateway
				return http.StatusBadGateway
			} else if apiErr.StatusCode == http.StatusTooManyRequests {
				// Rate limiting - pass through
				return apiErr.StatusCode
			} else {
				// Other 4xx from Steam - return 502 as it's likely Steam API issue
				return http.StatusBadGateway
			}
		default:
			return apiErr.StatusCode
		}
	}

	// Map error types to status codes when no status code is set
	switch apiErr.Type {
	case steam.ErrorTypeValidation:
		return http.StatusBadRequest // 400
	case steam.ErrorTypeNotFound:
		return http.StatusNotFound // 404
	case steam.ErrorTypeRateLimit:
		return http.StatusTooManyRequests // 429
	case steam.ErrorTypeAPIError, steam.ErrorTypeNetwork:
		return http.StatusBadGateway // 502 - Steam API is down/unreachable
	case steam.ErrorTypeInternal:
		return http.StatusInternalServerError // 500
	default:
		return http.StatusInternalServerError // 500 - Safe default
	}
}

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	writeJSONResponseWithStatus(w, data, http.StatusOK)
}

func writeJSONResponseWithStatus(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Marshal to get response size for logging
	responseBytes, err := json.Marshal(data)
	if err != nil {
		log.Error("Failed to marshal JSON response",
			"error", err.Error())
		writeErrorResponse(w, steam.NewInternalError(err))
		return
	}

	// Set status code
	w.WriteHeader(statusCode)

	// Log successful response
	log.Info("successful_response_sent",
		"status_code", statusCode,
		"response_size", len(responseBytes),
		"content_type", "application/json")

	// Write the response
	if _, err := w.Write(responseBytes); err != nil {
		log.Error("Failed to write JSON response",
			"error", err.Error(),
			"response_size", len(responseBytes))
		// Can't call writeErrorResponse here as headers are already sent
		return
	}
}

func writePartialDataResponse(w http.ResponseWriter, data interface{}, warnings []string) {
	// Convert to map to add warnings
	var responseData map[string]interface{}
	
	// Convert data to map
	dataBytes, _ := json.Marshal(data)
	json.Unmarshal(dataBytes, &responseData)
	
	if responseData == nil {
		responseData = make(map[string]interface{})
		responseData["data"] = data
	}
	
	// Add warnings to response
	if len(warnings) > 0 {
		responseData["warnings"] = warnings
		responseData["status"] = "partial_success"
		// Use 206 Partial Content for partial data scenarios
		writeJSONResponseWithStatus(w, responseData, http.StatusPartialContent)
	} else {
		writeJSONResponseWithStatus(w, data, http.StatusOK)
	}
}

// GetPlayerStatsWithAchievements fetches both player stats and achievements in a single endpoint
func (h *Handler) GetPlayerStatsWithAchievements(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), DefaultRequestTimeout)
	defer cancel()
	
	start := time.Now()
	steamID := mux.Vars(r)["steamid"]

	// Create structured logger with comprehensive request context
	requestLogger := log.HTTPRequestContext(r.Method, r.URL.Path, steamID, r.RemoteAddr)

	if err := validateSteamIDOrVanity(steamID); err != nil {
		log.ErrorContext(string(err.Type), steamID).Warn("Invalid Steam ID format in GetPlayerStatsWithAchievements",
			"user_agent", r.UserAgent(),
			"error_message", err.Message,
			"validation_type", string(err.Type))
		writeValidationError(w, r, err.Message, "steam_id")
		return
	}

	// Resolve vanity URL first to ensure consistent cache keys
	resolvedSteamID, resolveErr := h.steamClient.ResolveSteamID(steamID)
	if resolveErr != nil {
		requestLogger.Error("Failed to resolve Steam ID/vanity URL",
			"error", resolveErr.Message,
			"error_type", string(resolveErr.Type),
			"duration", time.Since(start))
		writeErrorResponse(w, resolveErr)
		return
	}

	// Check combined cache using resolved Steam ID
	var combinedCacheKey string
	var combinedCacheHit bool
	if h.cacheManager != nil {
		combinedCacheKey = cache.GenerateKey(cache.PlayerCombinedPrefix, resolvedSteamID)
		if cached, found := h.cacheManager.GetCache().Get(combinedCacheKey); found {
			if response, ok := cached.(models.PlayerStatsWithAchievements); ok {
				combinedCacheHit = true
				requestLogger.Info("Combined cache hit",
					"display_name", response.DisplayName,
					"has_achievements", response.Achievements != nil,
					"duration", time.Since(start))
				writeJSONResponse(w, response)
				return
			} else {
				// Invalid cache entry - remove it
				requestLogger.Warn("Invalid combined cache entry type, removing",
					"expected", "models.PlayerStatsWithAchievements",
					"actual", fmt.Sprintf("%T", cached))
				h.cacheManager.GetCache().Delete(combinedCacheKey)
			}
		}
	}

	requestLogger.Info("Processing combined player data request",
		"combined_cache_hit", combinedCacheHit)

	requestLogger.Info("Steam ID resolution completed",
		"original_input", steamID,
		"resolved_steam_id", resolvedSteamID,
		"was_vanity_url", steamID != resolvedSteamID)

	// Fetch data in parallel for better performance using resolved Steam ID
	type fetchResult struct {
		stats        models.PlayerStats
		achievements *models.AchievementData
		statsError   error
		achError     error
		statsSource  string
		achSource    string
	}

	select {
	case <-ctx.Done():
		writeTimeoutError(w, r, "player_stats_with_achievements")
		return
	default:
	}

	result := fetchResult{}
	resultChan := make(chan struct{}, 2)

	// Fetch player stats using resolved Steam ID
	go func() {
		defer func() { resultChan <- struct{}{} }()
		result.stats, result.statsError, result.statsSource = h.fetchPlayerStatsWithSource(resolvedSteamID)
	}()

	// Fetch achievements using resolved Steam ID
	go func() {
		defer func() { resultChan <- struct{}{} }()
		result.achievements, result.achError, result.achSource = h.fetchPlayerAchievementsWithSource(resolvedSteamID)
	}()

	timeout := time.After(SteamAPITimeout)
	completedCount := 0
	for completedCount < 2 {
		select {
		case <-resultChan:
			completedCount++
		case <-ctx.Done():
			writeTimeoutError(w, r, "player_stats_with_achievements")
			return
		case <-timeout:
			writeTimeoutError(w, r, "player_stats_with_achievements")
			return
		}
	}

	// Determine response strategy based on what succeeded
	response := models.PlayerStatsWithAchievements{
		PlayerStats: result.stats,
		DataSources: models.DataSourceStatus{
			Stats: models.DataSourceInfo{
				Success:   result.statsError == nil,
				Source:    result.statsSource,
				FetchedAt: time.Now(),
			},
			Achievements: models.DataSourceInfo{
				Success:   result.achError == nil,
				Source:    result.achSource,
				FetchedAt: time.Now(),
			},
		},
	}

	// Handle different failure scenarios with detailed classification
	if result.statsError != nil {
		// Stats failed - this is critical, return error
		response.DataSources.Stats.Error = result.statsError.Error()
		requestLogger.Error("Failed to fetch player stats - critical failure",
			"error", result.statsError,
			"error_type", classifyError(result.statsError),
			"original_steam_id", steamID,
			"resolved_steam_id", resolvedSteamID,
			"duration", time.Since(start))
		writeErrorResponse(w, steam.NewInternalError(result.statsError))
		return
	}

	// Always initialize achievements to prevent frontend errors
	response.Achievements = &models.AchievementData{
		AdeptSurvivors: make(map[string]bool),
		AdeptKillers:   make(map[string]bool),
		LastUpdated:    time.Now(),
	}

	if result.achError != nil {
		// Achievements failed but stats succeeded - return partial data with empty achievements
		errorType := classifyError(result.achError)
		response.DataSources.Achievements.Error = result.achError.Error()

		// Log with different severity based on error type
		if errorType == "steam_api_down" || errorType == "rate_limited" {
			requestLogger.Error("Steam achievements API unavailable - returning stats only",
				"error", result.achError,
				"error_type", errorType,
				"steam_id", steamID,
				"persona_name", result.stats.DisplayName,
				"impact", "partial_data_served")
		} else if errorType == "private_profile" || errorType == "no_achievements" {
			requestLogger.Info("Player achievements not accessible - returning stats only",
				"error", result.achError,
				"error_type", errorType,
				"steam_id", steamID,
				"persona_name", result.stats.DisplayName,
				"reason", "expected_user_privacy_or_no_data")
		} else {
			requestLogger.Warn("Unexpected achievement fetch error - returning stats only",
				"error", result.achError,
				"error_type", errorType,
				"steam_id", steamID,
				"persona_name", result.stats.DisplayName)
		}
	} else {
		// Both succeeded - populate with actual data
		response.Achievements = result.achievements
		requestLogger.Debug("Successfully fetched both stats and achievements",
			"steam_id", steamID,
			"persona_name", result.stats.DisplayName,
			"survivor_unlocks", countUnlocked(result.achievements.AdeptSurvivors),
			"killer_unlocks", countUnlocked(result.achievements.AdeptKillers))
	}

	// Cache the combined result
	if h.cacheManager != nil && combinedCacheKey != "" {
		config := h.cacheManager.GetConfig()
		if err := h.cacheManager.GetCache().Set(combinedCacheKey, response, config.TTL.PlayerCombined); err != nil {
			requestLogger.Error("Failed to cache combined response",
				"error", err,
				"cache_key", combinedCacheKey)
		} else {
			requestLogger.Debug("Combined response cached successfully",
				"cache_key", combinedCacheKey,
				"ttl", config.TTL.PlayerCombined)
		}
	}

	requestLogger.Info("Successfully processed combined player data request",
		"persona_name", result.stats.DisplayName,
		"original_steam_id", steamID,
		"resolved_steam_id", resolvedSteamID,
		"stats_success", result.statsError == nil,
		"achievements_success", result.achError == nil,
		"duration", time.Since(start))

	// Use appropriate response based on whether we have partial data
	if result.achError != nil {
		// Partial data - stats succeeded but achievements failed
		warnings := []string{
			"Achievement data unavailable: " + result.achError.Error(),
		}
		writePartialDataResponse(w, response, warnings)
	} else {
		// Complete data - both stats and achievements succeeded
		writeJSONResponse(w, response)
	}
}

// fetchPlayerStatsWithSource fetches player stats and returns the data source information
func (h *Handler) fetchPlayerStatsWithSource(steamID string) (models.PlayerStats, error, string) {
	// Check stats cache first
	if h.cacheManager != nil {
		cacheKey := cache.GenerateKey(cache.PlayerStatsPrefix, steamID)
		if cached, found := h.cacheManager.GetCache().Get(cacheKey); found {
			if playerStats, ok := cached.(models.PlayerStats); ok {
				return playerStats, nil, "cache"
			}
		}
	}

	// Fetch from Steam API
	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		return models.PlayerStats{}, fmt.Errorf("steam summary failed: %w", err), "api"
	}

	rawStats, err := h.steamClient.GetPlayerStats(steamID)
	if err != nil {
		return models.PlayerStats{}, fmt.Errorf("steam stats failed: %w", err), "api"
	}

	playerStats := steam.MapSteamStats(rawStats.Stats, summary.SteamID, summary.PersonaName)
	flatPlayerStats := convertToPlayerStats(playerStats)

	// Cache the stats
	if h.cacheManager != nil {
		cacheKey := cache.GenerateKey(cache.PlayerStatsPrefix, steamID)
		config := h.cacheManager.GetConfig()
		h.cacheManager.GetCache().Set(cacheKey, flatPlayerStats, config.TTL.PlayerStats)
	}

	return flatPlayerStats, nil, "api"
}

// fetchPlayerAchievementsWithSource fetches player achievements and returns the data source information
func (h *Handler) fetchPlayerAchievementsWithSource(steamID string) (*models.AchievementData, error, string) {
	// Check achievements cache first - longer TTL since achievements change infrequently
	if h.cacheManager != nil {
		cacheKey := cache.GenerateKey(cache.PlayerAchievementsPrefix, steamID)
		if cached, found := h.cacheManager.GetCache().Get(cacheKey); found {
			if achievements, ok := cached.(*models.AchievementData); ok {
				// Log cache hit with age information for monitoring
				age := time.Since(achievements.LastUpdated)
				log.Debug("Achievement cache hit",
					"steam_id", steamID,
					"cache_age", age,
					"cache_key", cacheKey)
				return achievements, nil, "cache"
			} else {
				// Invalid cache entry - clean it up
				log.Warn("Invalid achievement cache entry type, removing",
					"steam_id", steamID,
					"cache_key", cacheKey,
					"expected", "*models.AchievementData",
					"actual", fmt.Sprintf("%T", cached))
				h.cacheManager.GetCache().Delete(cacheKey)
			}
		}
	}

	// Use circuit breaker for Steam API protection
	var rawAchievements *steam.PlayerAchievements
	var apiErr error

	if h.cacheManager != nil && h.cacheManager.GetCircuitBreaker() != nil {
		// Execute with circuit breaker protection
		result, err := h.cacheManager.GetCircuitBreaker().ExecuteWithStaleCache(
			cache.GenerateKey(cache.PlayerAchievementsPrefix, steamID),
			func() (interface{}, error) {
				achievements, apiErr := h.steamClient.GetPlayerAchievements(steamID, 381210)
				if apiErr != nil {
					return nil, fmt.Errorf("steam API error: %s", apiErr.Message)
				}
				return achievements, nil
			},
		)

		if err != nil {
			apiErr = err
		} else if achievements, ok := result.(*steam.PlayerAchievements); ok {
			rawAchievements = achievements
		} else {
			apiErr = fmt.Errorf("circuit breaker returned unexpected type: %T", result)
		}
	} else {
		// Fallback to direct API call if no circuit breaker
		var steamErr *steam.APIError
		rawAchievements, steamErr = h.steamClient.GetPlayerAchievements(steamID, 381210)
		if steamErr != nil {
			apiErr = fmt.Errorf("steam API error: %s", steamErr.Message)
		}
	}

	if apiErr != nil {
		// Enhanced error logging with context
		log.Error("Steam achievements API failed",
			"steam_id", steamID,
			"error", apiErr,
			"error_type", classifyError(apiErr),
			"circuit_breaker_active", h.cacheManager != nil && h.cacheManager.GetCircuitBreaker() != nil)
		return nil, fmt.Errorf("steam achievements failed: %w", apiErr), "api"
	}

	// Get adept map from schema
	ctx := context.Background()
	adeptMap, err := h.steamClient.GetAdeptMapCached(ctx, h.cacheManager.GetCache())
	if err != nil {
		log.Warn("Failed to get adept map from schema, falling back to hardcoded mapping",
			"error", err)
		// Continue without schema-based adepts - use fallback
		adeptMap = make(map[string]steam.AdeptEntry)
	}

	// Map achievements using the new schema-based approach
	mappedData := steam.GetMappedAchievementsWithCache(rawAchievements, h.cacheManager.GetCache())
	mappedAchievements := mappedData["achievements"].([]steam.AchievementMapping)
	summary := mappedData["summary"].(map[string]interface{})

	// Process adepts using schema-based mapping
	adeptSurv := make(map[string]bool)
	adeptKill := make(map[string]bool)

	// Process raw achievements to build adept maps using schema
	for _, rawAch := range rawAchievements.Achievements {
		if entry, ok := adeptMap[rawAch.APIName]; ok {
			if entry.Kind == "killer" {
				adeptKill[entry.Character] = rawAch.Achieved == 1
			} else {
				adeptSurv[entry.Character] = rawAch.Achieved == 1
			}
		}
	}

	// Convert to models format
	processedAchievements := &models.AchievementData{
		AdeptSurvivors:     adeptSurv,
		AdeptKillers:       adeptKill,
		MappedAchievements: make([]models.MappedAchievement, len(mappedAchievements)),
		Summary: models.AchievementSummary{
			TotalAchievements: summary["total_achievements"].(int),
			UnlockedCount:     summary["unlocked_count"].(int),
			SurvivorCount:     summary["survivor_count"].(int),
			KillerCount:       summary["killer_count"].(int),
			GeneralCount:      summary["general_count"].(int),
			AdeptSurvivors:    summary["adept_survivors"].([]string),
			AdeptKillers:      summary["adept_killers"].([]string),
			CompletionRate:    summary["completion_rate"].(float64),
		},
		LastUpdated: time.Now(),
	}

	// Convert mapped achievements to models format
	for i, mapped := range mappedAchievements {
		processedAchievements.MappedAchievements[i] = models.MappedAchievement{
			ID:          mapped.ID,
			Name:        mapped.Name,
			DisplayName: mapped.DisplayName,
			Description: mapped.Description,
			Character:   mapped.Character,
			Type:        mapped.Type,
			Unlocked:    mapped.Unlocked,
			UnlockTime:  mapped.UnlockTime,
		}
		// Note: Legacy AdeptSurvivors/AdeptKillers are now populated directly from schema above
	}

	// Cache the achievements with longer TTL
	if h.cacheManager != nil {
		cacheKey := cache.GenerateKey(cache.PlayerAchievementsPrefix, steamID)
		config := h.cacheManager.GetConfig()

		if err := h.cacheManager.GetCache().Set(cacheKey, processedAchievements, config.TTL.PlayerAchievements); err != nil {
			log.Error("Failed to cache achievements",
				"steam_id", steamID,
				"error", err,
				"cache_key", cacheKey,
				"ttl", config.TTL.PlayerAchievements)
		} else {
			log.Debug("Achievements cached successfully",
				"steam_id", steamID,
				"cache_key", cacheKey,
				"ttl", config.TTL.PlayerAchievements,
				"survivor_count", len(processedAchievements.AdeptSurvivors),
				"killer_count", len(processedAchievements.AdeptKillers))
		}
	}

	return processedAchievements, nil, "api"
}

// classifyError categorizes errors for better logging and monitoring
func classifyError(err error) string {
	if err == nil {
		return "none"
	}

	// Handle interface{} with nil underlying value
	if err == (*steam.APIError)(nil) {
		return "none"
	}

	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "too many requests"):
		return "rate_limited"
	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded"):
		return "timeout"
	case strings.Contains(errStr, "private") || strings.Contains(errStr, "not found"):
		return "private_profile"
	case strings.Contains(errStr, "achievements not found") || strings.Contains(errStr, "no achievements"):
		return "no_achievements"
	case strings.Contains(errStr, "network") || strings.Contains(errStr, "connection"):
		return "network_error"
	case strings.Contains(errStr, "steam") && (strings.Contains(errStr, "api") || strings.Contains(errStr, "server")):
		return "steam_api_down"
	case strings.Contains(errStr, "invalid") || strings.Contains(errStr, "validation"):
		return "validation_error"
	default:
		return "unknown_error"
	}
}

// countUnlocked counts the number of unlocked achievements in a map
func countUnlocked(achievements map[string]bool) int {
	count := 0
	for _, unlocked := range achievements {
		if unlocked {
			count++
		}
	}
	return count
}

// HealthCheck provides a simple health check endpoint for load balancers
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"services": map[string]string{
			"steam_api": "available",
			"cache":     "available",
		},
	}

	// Check cache health if enabled
	if h.cacheManager != nil {
		cacheStatus := h.cacheManager.GetCacheStatus()
		status["services"].(map[string]string)["cache"] = "available"
		status["cache_status"] = cacheStatus
	} else {
		status["services"].(map[string]string)["cache"] = "disabled"
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}
