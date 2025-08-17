# Caching Strategy

Multi-layer caching implementation with circuit breaker protection for reliable Steam API access.

## Problem Overview

Steam API challenges that require caching:

- **Slow Response Times**: 200-500ms average response times
- **Rate Limiting**: Strict API quotas and request throttling
- **Service Reliability**: Frequent timeouts and temporary outages
- **API Costs**: Each request counts against daily quotas

## Caching Architecture

The system uses a multi-layer approach to minimize Steam API calls:

## Request Flow

```
HTTP Request
      │
      ▼
┌─────────────┐    HIT     ┌─────────────────────────┐
│ Memory Cache│──────────▶│    Return Data          │
│   (L1)      │   <1ms     │    (Fastest Path)       │
└─────────────┘            └─────────────────────────┘
      │ MISS
      ▼
┌─────────────┐    OPEN    ┌─────────────────────────┐
│   Circuit   │──────────▶│   Return Stale Data     │
│   Breaker   │   <5ms     │   (Graceful Fallback)  │
└─────────────┘            └─────────────────────────┘
      │ CLOSED
      ▼
┌─────────────┐  SUCCESS   ┌─────────────────────────┐
│  Steam API  │──────────▶│  Cache + Return Data    │
│    Call     │ 200-500ms  │                         │
└─────────────┘            └─────────────────────────┘
      │ FAILURE
      ▼
┌─────────────┐            ┌─────────────────────────┐
│   Circuit   │            │    Open Circuit +       │
│   Opens     │            │   Return Stale Data     │
└─────────────┘            └─────────────────────────┘
```

## Cache Layers
```go
type MemoryCache struct {
    data    map[string]CacheEntry
    mutex   sync.RWMutex
    maxSize int
    lru     *list.List // Least Recently Used tracking
}

type CacheEntry struct {
    Value     interface{}
    CreatedAt time.Time
    TTL       time.Duration
    AccessCount int
}
```

### TTL Configuration by Data Type
```go
const (
    PlayerStatsTTL    = 5 * time.Minute   // Frequently changing
    AchievementsTTL   = 10 * time.Minute  // Semi-static 
    PlayerSummaryTTL  = 15 * time.Minute  // Stable data
    SchemaTTL         = 1 * time.Hour     // Rarely changes
)
```

## Circuit Breaker Pattern

### State Management
```go
type CircuitState int

const (
    Closed   CircuitState = iota // Normal operation
    Open                         // Steam API down - serve cache
    HalfOpen                     // Testing recovery
)

type CircuitBreaker struct {
    state           CircuitState
    failures        int
    maxFailures     int           // Default: 5
    resetTimeout    time.Duration // Default: 60s
    lastFailureTime time.Time
}
```

### Failure Detection
```go
func (cb *CircuitBreaker) shouldTripBreaker(err error) bool {
    // Network timeouts
    if isTimeoutError(err) { return true }
    
    // Steam API rate limiting  
    if isRateLimitError(err) { return true }
    
    // 5xx server errors
    if isServerError(err) { return true }
    
    // Don't trip on 4xx client errors (bad Steam ID, etc.)
    return false
}
```

## Performance Optimizations

### Cache Key Strategy
```go
func generateCacheKey(steamID, dataType string) string {
    return fmt.Sprintf("%s:%s", dataType, steamID)
}

// Examples:
// "player_stats:76561198215615835"
// "achievements:counteredspell" 
// "player_summary:76561198215615835"
```

### LRU Eviction
```go
func (c *MemoryCache) evictLRU() {
    if c.lru.Len() > c.maxSize {
        // Remove least recently used entries
        oldest := c.lru.Back()
        c.lru.Remove(oldest)
        delete(c.data, oldest.Value.(string))
    }
}
```

### Batch Operations
```go
func (c *MemoryCache) SetBatch(entries map[string]CacheEntry) error {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    for key, entry := range entries {
        c.data[key] = entry
    }
    return nil
}
```

## Reliability Features

### Graceful Degradation
```go
func GetPlayerData(steamID string) (*PlayerData, error) {
    // Try fresh data first
    if cb.state == Closed {
        data, err := fetchFromSteamAPI(steamID)
        if err == nil {
            cache.Set(generateKey(steamID), data, PlayerStatsTTL)
            return data, nil
        }
        
        // Handle failure
        cb.recordFailure(err)
    }
    
    // Fallback to cached data
    if cached, exists := cache.Get(generateKey(steamID)); exists {
        log.Warn("Serving stale data due to Steam API failure",
            "steam_id", steamID,
            "age", time.Since(cached.CreatedAt))
        return cached.Value.(*PlayerData), nil
    }
    
    return nil, ErrNoDataAvailable
}
```

### Cache Corruption Detection
```go
func (c *MemoryCache) validateEntry(key string, entry CacheEntry) error {
    // Check TTL expiration
    if time.Since(entry.CreatedAt) > entry.TTL {
        return ErrEntryExpired
    }
    
    // Validate data integrity
    if entry.Value == nil {
        return ErrCorruptedData
    }
    
    // Type validation
    if !isValidDataType(entry.Value) {
        return ErrInvalidDataType
    }
    
    return nil
}
```

## Cache Metrics & Monitoring

### Real-Time Statistics
```go
type CacheStats struct {
    Hits        int64   `json:"hits"`
    Misses      int64   `json:"misses"`
    HitRate     float64 `json:"hit_rate"`
    Entries     int     `json:"entries"`
    MemoryUsage int64   `json:"memory_usage"`
    Evictions   int64   `json:"evictions"`
}
```

### Monitoring Endpoint
```bash
curl http://localhost:8080/api/cache/status
```

```json
{
  "cache_stats": {
    "hits": 1247,
    "misses": 83,
    "hit_rate": 93.8,
    "entries": 342,
    "memory_usage": 2457600,
    "evictions": 15
  },
  "circuit_breaker": {
    "state": "closed",
    "failures": 0,
    "last_success": "2025-08-17T20:35:30Z",
    "uptime_percent": 99.2
  }
}
```

## 🔧 Configuration

### Environment Variables
```bash
# Cache TTL Settings
CACHE_PLAYER_STATS_TTL=5m
CACHE_ACHIEVEMENTS_TTL=10m
CACHE_PLAYER_SUMMARY_TTL=15m
CACHE_SCHEMA_TTL=1h

# Circuit Breaker Settings  
CIRCUIT_BREAKER_MAX_FAILURES=5
CIRCUIT_BREAKER_RESET_TIMEOUT=60s
CIRCUIT_BREAKER_HALF_OPEN_MAX_CALLS=3

# Memory Limits
CACHE_MAX_ENTRIES=10000
CACHE_MAX_MEMORY_MB=100
```

### Runtime Configuration
```go
type CacheConfig struct {
    MaxEntries      int           `default:"10000"`
    MaxMemoryMB     int           `default:"100"`
    DefaultTTL      time.Duration `default:"5m"`
    CleanupInterval time.Duration `default:"1m"`
}
```

## Production Performance

### Typical Metrics
- **Cache Hit Rate**: 94-98% in production
- **Memory Usage**: ~50MB for 5000 cached players
- **Response Time**: < 1ms for cache hits
- **Fallback Response**: < 5ms for stale data

### Scaling Characteristics
```go
// Memory usage scales linearly
MemoryPerPlayer = ~10KB  // Serialized player data
MaxPlayers = MaxMemoryMB * 1024 / 10  // Approximate capacity

// Example: 100MB cache = ~10,000 players
```

## 🔍 Observability

### Structured Logging
```json
{
  "timestamp": "2025-08-17T20:35:30Z",
  "level": "INFO", 
  "msg": "cache_operation",
  "operation": "set",
  "key": "player_stats:76561198215615835",
  "ttl_seconds": 300,
  "data_size_bytes": 8192
}
```

### Alert Conditions
- **Low Hit Rate**: < 85% (indicates TTL too aggressive)
- **High Memory Usage**: > 90% capacity
- **Circuit Breaker Open**: Steam API degraded
- **High Eviction Rate**: Cache too small for workload

This caching strategy ensures reliable, fast access to Dead by Daylight data while protecting the Steam API from overload and providing graceful degradation during outages.
