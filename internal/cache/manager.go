package cache

import (
	"fmt"
	"log"
	"os"
	"time"
)

// CacheType represents the type of cache implementation
type CacheType string

const (
	MemoryCacheType CacheType = "memory"
	RedisCacheType  CacheType = "redis"
)

type Config struct {
	Type   CacheType         `json:"type"`
	Memory MemoryCacheConfig `json:"memory"`
	Redis  RedisConfig       `json:"redis"`
	TTL    TTLConfig         `json:"ttl"`
}

// RedisConfig holds Redis-specific configuration for future Redis implementation
type RedisConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Password     string        `json:"password"`
	Database     int           `json:"database"`
	MaxRetries   int           `json:"max_retries"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

// DefaultConfig returns a sensible default configuration for production
func DefaultConfig() Config {
	ttlConfig := GetTTLFromEnv()
	return Config{
		Type: MemoryCacheType,
		Memory: MemoryCacheConfig{
			MaxEntries:      1000,
			DefaultTTL:      ttlConfig.DefaultTTL,
			CleanupInterval: 30 * time.Second,
		},
		Redis: RedisConfig{
			Host:         "localhost",
			Port:         6379,
			Database:     0,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		TTL: ttlConfig,
	}
}

// PlayerStatsConfig returns cache configuration optimized for player statistics
func PlayerStatsConfig() Config {
	config := DefaultConfig()
	// Use configurable TTL for player stats
	config.Memory.DefaultTTL = config.TTL.PlayerStats
	// Increase capacity for player data but keep reasonable limits
	config.Memory.MaxEntries = 2000
	// More frequent cleanup for active usage
	config.Memory.CleanupInterval = 20 * time.Second
	return config
}

// DevelopmentConfig returns a configuration suitable for development/testing
func DevelopmentConfig() Config {
	config := DefaultConfig()
	config.Memory.MaxEntries = 100        // Small for testing
	config.Memory.DefaultTTL = 30 * time.Second // Short TTL for testing
	config.Memory.CleanupInterval = 5 * time.Second // Frequent cleanup
	// Override TTL config for development
	config.TTL = TTLConfig{
		PlayerStats:   30 * time.Second,
		PlayerSummary: 1 * time.Minute,
		SteamAPI:      30 * time.Second,
		DefaultTTL:    30 * time.Second,
	}
	return config
}

// Manager provides a factory and management layer for different cache implementations
type Manager struct {
	config         Config
	cache          Cache
	circuitBreaker *CircuitBreaker
}

// NewManager creates a new cache manager with the specified configuration
func NewManager(config Config) (*Manager, error) {
	manager := &Manager{
		config: config,
	}

	cache, err := manager.createCache()
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	manager.cache = cache
	
	// Create circuit breaker for upstream API protection
	circuitConfig := DefaultCircuitBreakerConfig()
	manager.circuitBreaker = NewCircuitBreaker(circuitConfig, cache)
	
	return manager, nil
}

// GetCache returns the underlying cache implementation
func (m *Manager) GetCache() Cache {
	return m.cache
}

// GetConfig returns the current cache configuration
func (m *Manager) GetConfig() Config {
	return m.config
}

// GetCircuitBreaker returns the circuit breaker for upstream API protection
func (m *Manager) GetCircuitBreaker() *CircuitBreaker {
	return m.circuitBreaker
}

// ExecuteWithFallback executes a function with circuit breaker and cache fallback
func (m *Manager) ExecuteWithFallback(key string, fn func() (interface{}, error)) (interface{}, error) {
	return m.circuitBreaker.ExecuteWithStaleCache(key, fn)
}

// GetCacheStatus returns comprehensive cache and circuit breaker status
func (m *Manager) GetCacheStatus() map[string]interface{} {
	status := map[string]interface{}{
		"cache_type": m.config.Type,
		"config":     m.config,
	}
	
	if m.circuitBreaker != nil {
		status["circuit_breaker"] = m.circuitBreaker.GetDetailedStatus()
	}
	
	// Add cache-specific stats if available
	if memCache, ok := m.cache.(*MemoryCache); ok {
		status["cache_stats"] = memCache.GetStats()
	}
	
	return status
}

// Close gracefully shuts down the cache
func (m *Manager) Close() error {
	if memCache, ok := m.cache.(*MemoryCache); ok {
		memCache.Close()
	}
	return nil
}

// createCache creates the appropriate cache implementation based on configuration
func (m *Manager) createCache() (Cache, error) {
	switch m.config.Type {
	case MemoryCacheType:
		return NewMemoryCache(m.config.Memory), nil
	case RedisCacheType:
		return nil, fmt.Errorf("redis cache not yet implemented - use memory cache for now. " +
			"consider Redis when you need: distributed caching, persistence, clustering, " +
			"or cache sizes > 10GB. current memory cache handles up to ~100k entries efficiently")
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", m.config.Type)
	}
}

// GenerateKey creates a consistent cache key for given parameters
func GenerateKey(prefix string, parts ...string) string {
	if len(parts) == 0 {
		return prefix
	}
	
	key := prefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

// Common cache key prefixes for different data types
const (
	PlayerStatsPrefix  = "player_stats"
	PlayerSummaryPrefix = "player_summary"
	SteamAPIPrefix     = "steam_api"
)

// Backward compatibility: TTL constants (deprecated - use TTLConfig instead)
const (
	PlayerStatsTTL   = 5 * time.Minute  // Deprecated: use config.TTL.PlayerStats
	PlayerSummaryTTL = 10 * time.Minute // Deprecated: use config.TTL.PlayerSummary
	SteamAPITTL      = 3 * time.Minute  // Deprecated: use config.TTL.SteamAPI
	DefaultTTL       = 3 * time.Minute  // Deprecated: use config.TTL.DefaultTTL
)

// TTLConfig holds configurable TTL values for different data types
type TTLConfig struct {
	PlayerStats   time.Duration `json:"player_stats_ttl"`
	PlayerSummary time.Duration `json:"player_summary_ttl"`
	SteamAPI      time.Duration `json:"steam_api_ttl"`
	DefaultTTL    time.Duration `json:"default_ttl"`
}

// GetTTLFromEnv returns TTL configuration from environment variables with fallbacks
func GetTTLFromEnv() TTLConfig {
	return TTLConfig{
		PlayerStats:   getEnvDuration("CACHE_PLAYER_STATS_TTL", 5*time.Minute),
		PlayerSummary: getEnvDuration("CACHE_PLAYER_SUMMARY_TTL", 10*time.Minute),
		SteamAPI:      getEnvDuration("CACHE_STEAM_API_TTL", 3*time.Minute),
		DefaultTTL:    getEnvDuration("CACHE_DEFAULT_TTL", 3*time.Minute),
	}
}

// getEnvDuration parses duration from environment variable with fallback
func getEnvDuration(envKey string, fallback time.Duration) time.Duration {
	if value := os.Getenv(envKey); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		// Log warning about invalid duration format but continue with fallback
		log.Printf("Warning: Invalid duration format for %s: %s, using fallback %v", envKey, value, fallback)
	}
	return fallback
}
