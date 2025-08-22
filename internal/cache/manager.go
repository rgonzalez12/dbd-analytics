package cache

import (
	"fmt"
	"os"
	"time"

	internalLog "github.com/rgonzalez12/dbd-analytics/internal/log"
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

func DevelopmentConfig() Config {
	config := DefaultConfig()
	config.Memory.MaxEntries = 100
	config.Memory.DefaultTTL = 30 * time.Second
	config.Memory.CleanupInterval = 5 * time.Second
	config.TTL = TTLConfig{
		PlayerStats:        30 * time.Second,
		PlayerSummary:      1 * time.Minute,
		PlayerAchievements: 2 * time.Minute,
		PlayerCombined:     1 * time.Minute,
		SteamAPI:           30 * time.Second,
		DefaultTTL:         30 * time.Second,
	}
	return config
}

// Manager provides a factory and management layer for different cache implementations
type Manager struct {
	config         Config
	cache          Cache
	circuitBreaker *CircuitBreaker
}

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

// Cache key prefixes are defined in keys.go

// Backward compatibility: TTL constants (deprecated - use TTLConfig instead)
const (
	PlayerStatsTTL   = 5 * time.Minute
	PlayerSummaryTTL = 10 * time.Minute
	SteamAPITTL      = 3 * time.Minute
	DefaultTTL       = 3 * time.Minute
)

// TTLConfig holds configurable TTL values for different data types
type TTLConfig struct {
	PlayerStats        time.Duration `json:"player_stats_ttl"`
	PlayerSummary      time.Duration `json:"player_summary_ttl"`
	PlayerAchievements time.Duration `json:"player_achievements_ttl"`
	PlayerCombined     time.Duration `json:"player_combined_ttl"`
	SteamAPI           time.Duration `json:"steam_api_ttl"`
	DefaultTTL         time.Duration `json:"default_ttl"`
}

// GetTTLFromEnv returns TTL configuration from environment variables with fallbacks
// TTL Source Priority: Environment Variables > Deprecated Constants > Hardcoded Defaults
// This ensures production deployments can override TTL values without code changes
func GetTTLFromEnv() TTLConfig {
	config := TTLConfig{
		PlayerStats:        getEnvDuration("CACHE_PLAYER_STATS_TTL", PlayerStatsTTL),
		PlayerSummary:      getEnvDuration("CACHE_PLAYER_SUMMARY_TTL", PlayerSummaryTTL),
		PlayerAchievements: getEnvDuration("CACHE_PLAYER_ACHIEVEMENTS_TTL", 2*time.Minute),
		PlayerCombined:     getEnvDuration("CACHE_PLAYER_COMBINED_TTL", 10*time.Minute),
		SteamAPI:           getEnvDuration("CACHE_STEAM_API_TTL", SteamAPITTL), // Use deprecated constant for backward compatibility
		DefaultTTL:         getEnvDuration("CACHE_DEFAULT_TTL", DefaultTTL),    // Use deprecated constant for backward compatibility
	}

	internalLog.Info("Cache TTL configuration loaded",
		"player_stats_ttl", config.PlayerStats,
		"player_summary_ttl", config.PlayerSummary,
		"player_achievements_ttl", config.PlayerAchievements,
		"player_combined_ttl", config.PlayerCombined,
		"steam_api_ttl", config.SteamAPI,
		"default_ttl", config.DefaultTTL,
		"source_priority", "env_vars > deprecated_constants > defaults")

	return config
}

// getEnvDuration parses duration from environment variable with fallback
func getEnvDuration(envKey string, fallback time.Duration) time.Duration {
	if value := os.Getenv(envKey); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			internalLog.Info("TTL loaded from environment variable",
				"env_key", envKey,
				"value", duration,
				"source", "environment")
			return duration
		}
		internalLog.Warn("Invalid duration format in environment variable",
			"env_key", envKey,
			"value", value,
			"fallback", fallback)
	}
	return fallback
}
