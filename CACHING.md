# Caching Implementation

## Overview

In-memory caching layer that reduces Steam API calls and improves response times. Designed for easy migration to Redis.

## Architecture

### Request Flow
```
1. HTTP Request → Handler
2. Generate Cache Key (e.g., "player_stats:76561198000000000")
3. Check Cache
   ├─ HIT: Return cached data (< 1ms)
   └─ MISS: Fetch from Steam API
4. Steam API Call (200-500ms)
5. Store in Cache
6. Return to Client
```

### Cache Interface
```go
type Cache interface {
    Set(key string, value interface{}, ttl time.Duration) error
    Get(key string) (interface{}, bool)
    Delete(key string) error
    Clear() error
    Stats() CacheStats
}
```

## Configuration

### TTL Settings
```go
const (
    PlayerStatsTTL   = 5 * time.Minute
    PlayerSummaryTTL = 10 * time.Minute  
    SteamAPITTL      = 3 * time.Minute
)
```

### Production Config
```go
config := cache.PlayerStatsConfig()
// MaxEntries: 2000 (~2MB memory)
// DefaultTTL: 5 minutes
// CleanupInterval: 30 seconds
```

## Features

### Thread Safety
- Uses `sync.RWMutex` for concurrent access
- Read operations use `RLock()` for performance
- Write operations use `Lock()` for safety

### LRU Eviction
- Tracks `AccessedAt` timestamp for each entry
- Evicts least recently used entries when at capacity
- Prevents memory bloat under high load

### TTL Expiration
- Each entry has configurable expiration time
- Background cleanup of expired entries
- Lazy expiration during Get operations

### Metrics
```go
type CacheStats struct {
    Hits        int64
    Misses      int64  
    Evictions   int64
    Entries     int
    HitRate     float64
    MemoryUsage int64
}
```

## Implementation

### Handler Integration
```go
// Check cache first
if h.cacheManager != nil {
    cacheKey := cache.GenerateKey(cache.PlayerStatsPrefix, steamID)
    if cached, found := h.cacheManager.GetCache().Get(cacheKey); found {
        if playerStats, ok := cached.(models.PlayerStats); ok {
            // Cache HIT - return immediately
            writeJSONResponse(w, playerStats)
            return
        }
    }
}

// Cache MISS - fetch from Steam API
// ... Steam API calls ...

// Store in cache
if h.cacheManager != nil {
    h.cacheManager.GetCache().Set(cacheKey, playerStats, cache.PlayerStatsTTL)
}
```

### Key Generation
```go
// Generates keys like: "player_stats:76561198000000000"
cacheKey := cache.GenerateKey(cache.PlayerStatsPrefix, steamID)

const (
    PlayerStatsPrefix   = "player_stats"
    PlayerSummaryPrefix = "player_summary" 
    SteamAPIPrefix      = "steam_api"
)
```

## Performance Impact

### Before Caching
- Every request: Steam API call (200-500ms)
- High API usage, potential rate limiting
- Slower response times

### After Caching (Cache Hit)
- Response time: < 1ms (99.8% faster)
- Steam API calls reduced by 80-90%
- Near-instant responses

### Metrics Example
```
Cache Statistics:
├─ Hits: 245 (85.7% hit rate)
├─ Misses: 41
├─ Evictions: 12 
├─ Current Entries: 156
└─ Memory Usage: 1.2 MB
```

## Redis Migration

The interface design allows zero-downtime Redis migration:

### Step 1: Implement Redis Cache
```go
type RedisCache struct {
    client *redis.Client
    stats  CacheStats
}

func (r *RedisCache) Get(key string) (interface{}, bool) {
    // Redis implementation
}
```

### Step 2: Update Configuration
```go
config := cache.Config{
    Type: cache.RedisCacheType,  // Changed from MemoryCacheType
    Redis: cache.RedisConfig{
        Host: "localhost",
        Port: 6379,
    },
}
```

### Step 3: Deploy
- Zero application code changes in handlers
- Same interface, different backend
- Gradual rollout possible

## Production Considerations

### Memory Management
- Current setup: ~2000 entries = ~2MB RAM
- Monitor with `/api/cache/stats` endpoint
- Automatic limits prevent memory bloat

### Error Handling
- Cache failures gracefully fallback to Steam API
- Thread-safe operations with proper locking
- Graceful shutdown with resource cleanup

### Monitoring
```
GET  /api/cache/stats        # Performance metrics
POST /api/cache/evict        # Manual cleanup
```

## Testing

Comprehensive test coverage includes:
- Basic operations (Set/Get/Delete)
- TTL expiration and LRU eviction
- Thread safety and concurrent access
- Statistics and error handling
- Performance benchmarks

## Usage

### Basic Integration
```go
// Initialize
manager, err := cache.NewManager(cache.PlayerStatsConfig())
if err != nil {
    log.Fatal("Failed to initialize cache:", err)
}
defer manager.Close()

// Use in handlers
handler := &Handler{
    steamClient:  steam.NewClient(),
    cacheManager: manager,
}
```

### Manual Operations
```go
cache := manager.GetCache()

// Store data
err := cache.Set("player:123", playerData, 5*time.Minute)

// Retrieve data
if data, found := cache.Get("player:123"); found {
    playerStats := data.(models.PlayerStats)
}

// Get metrics
stats := cache.Stats()
fmt.Printf("Hit rate: %.1f%%", stats.HitRate)
```
