package cache

import (
	"fmt"
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
	return Config{
		Type: MemoryCacheType,
		Memory: MemoryCacheConfig{
			MaxEntries:      1000,
			DefaultTTL:      3 * time.Minute,
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
	}
}

// PlayerStatsConfig returns cache configuration optimized for player statistics
func PlayerStatsConfig() Config {
	config := DefaultConfig()
	// Player stats change infrequently, can cache longer
	config.Memory.DefaultTTL = 5 * time.Minute
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
	return config
}

// Manager provides a factory and management layer for different cache implementations
type Manager struct {
	config Config
	cache  Cache
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
		return nil, fmt.Errorf("Redis cache not yet implemented - use memory cache for now")
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

// TTL constants for different data types
const (
	PlayerStatsTTL   = 5 * time.Minute  // Player stats change infrequently
	PlayerSummaryTTL = 10 * time.Minute // Profile info changes very rarely
	SteamAPITTL      = 3 * time.Minute  // General Steam API data
)
