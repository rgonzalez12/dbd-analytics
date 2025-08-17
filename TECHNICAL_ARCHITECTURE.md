# Technical Architecture & Design Decisions

## Overview

This document explains the technical approach, design decisions, and implementation details for the DBD Analytics system. It covers why specific approaches were taken and how the various components work together.

## 🎯 Core Problem & Solution

### The Challenge
Dead by Daylight statistics from Steam API presented several challenges:
1. **Grade Detection**: Steam provides raw numeric values without context for killer vs survivor grades
2. **Data Reliability**: Steam API can be unreliable, requiring robust caching and fallback strategies
3. **Achievement Mapping**: 86+ adept achievements need intelligent character name mapping
4. **Performance**: Real-time data fetching needs caching without stale data issues

### Our Solution
We implemented a **field-aware, multi-layer caching system** with intelligent grade detection and graceful degradation.

## 🏗️ System Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   SvelteKit     │    │   Go API Server  │    │   Steam API     │
│   Frontend      │◄──►│                  │◄──►│                 │
│                 │    │  ┌─────────────┐ │    │                 │
│ - TypeScript    │    │  │ Circuit     │ │    │ - Player Stats  │
│ - API Client    │    │  │ Breaker     │ │    │ - Achievements  │
│ - Responsive    │    │  └─────────────┘ │    │ - Schema Data   │
└─────────────────┘    │  ┌─────────────┐ │    └─────────────────┘
                       │  │ Memory      │ │
                       │  │ Cache       │ │
                       │  └─────────────┘ │
                       │  ┌─────────────┐ │
                       │  │ Grade       │ │
                       │  │ Mapper      │ │
                       │  └─────────────┘ │
                       └──────────────────┘
```

## 🧠 Field-Aware Grade Detection

### The Problem
Steam API returns grades as raw numbers (e.g., `16`, `65`, `73`) without indicating whether they represent killer or survivor grades. The same number could mean different grades for different roles.

### Our Approach
We implemented **field-aware grade detection** that uses the Steam schema field names to determine the context:

```go
func decodeGrade(gradeCode int, fieldID string) Grade {
    // Use fieldID to determine if this is killer or survivor grade
    isKillerGrade := strings.Contains(fieldID, "DBD_SlasherTierIncrement")
    isSurvivorGrade := strings.Contains(fieldID, "DBD_UnlockRanking")
    
    // Apply appropriate grade mapping based on field type
    return gradeMapping[gradeCode]
}
```

### Why This Approach
1. **Accuracy**: Each grade value is interpreted in the correct context
2. **Future-Proof**: New grade fields can be easily accommodated
3. **Debugging**: Field IDs provide clear traceability in logs
4. **Maintainability**: Single mapping table with context-aware application

## 💾 Multi-Layer Caching Strategy

### Cache Hierarchy
1. **Memory Cache**: Fast in-memory LRU cache with TTL
2. **Circuit Breaker**: Protects Steam API from overload
3. **Stale Data Fallback**: Serves expired data during outages
4. **Corruption Detection**: Validates and recovers corrupted cache entries

### Implementation Details

#### Circuit Breaker Pattern
```go
type CircuitBreaker struct {
    state       State           // closed, open, half-open
    failures    int            // consecutive failure count
    lastFailure time.Time      // timestamp of last failure
    timeout     time.Duration  // how long to stay open
}
```

**Why Circuit Breaker?**
- **Steam API Protection**: Prevents overwhelming Steam with requests during outages
- **Fast Failure**: Immediately returns errors when Steam is down instead of waiting for timeouts
- **Automatic Recovery**: Gradually reopens when Steam API recovers

#### Memory Cache with Corruption Detection
```go
type CacheEntry struct {
    Data       interface{}
    ExpiredAt  time.Time
    AccessedAt time.Time
    Size       int64
}
```

**Features:**
- **LRU Eviction**: Removes least recently used entries when memory limit reached
- **TTL Management**: Automatic expiration with configurable timeouts
- **Corruption Detection**: Validates entries and quarantines corrupted data
- **Graceful Degradation**: Serves stale data when fresh data unavailable

### Cache TTL Strategy
Different data types have different volatility and importance:

```go
PlayerStats:     5 minutes   // Stats change frequently during gameplay
PlayerSummary:   10 minutes  // Profile info changes less often
Achievements:    2 minutes   // Achievement unlocks are time-sensitive
SteamAPI:        3 minutes   // General Steam API responses
```

## 🏆 Achievement System Architecture

### The Challenge
Dead by Daylight has 86+ adept achievements with complex character names:
- Special characters: "The Onryō", "The Ghoul"
- Multiple formats: "Ghost Face" vs "Ghostface"
- Legacy names: Achievement names may not match current character names

### Our Solution: Three-Layer Mapping

#### 1. Hardcoded Achievement Mapping
```go
var achievementMapping = map[string]CharacterInfo{
    "ACH_UNLOCK_DWIGHT_PERKS":     {Character: "dwight", Type: "survivor"},
    "ACH_UNLOCK_CHUCKLES_PERKS":   {Character: "trapper", Type: "killer"},
    "ACH_UNLOCK_ONRYO_PERKS":      {Character: "onryo", Type: "killer"},
}
```

#### 2. Character Name Normalization
```go
func normalizeCharacterName(name string) string {
    // Remove "The " prefix
    name = strings.TrimPrefix(name, "The ")
    // Convert special characters
    name = strings.ReplaceAll(name, "ō", "o")
    // Convert to lowercase and remove spaces
    return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}
```

#### 3. Steam Schema Integration
Uses Steam's achievement schema as primary source with hardcoded fallback:
- **Schema First**: Attempts to fetch current achievement data from Steam
- **Fallback Protection**: Uses hardcoded mapping when schema unavailable
- **Validation**: Cross-references both sources for accuracy

### Why This Approach
1. **Reliability**: Always works even when Steam schema is unavailable
2. **Accuracy**: Handles all character name variations and special characters
3. **Maintainability**: New characters can be added to hardcoded mapping
4. **Performance**: In-memory mapping provides instant lookups

## 🔍 Input Validation & Security

### Steam ID and Vanity URL Validation
The system accepts both Steam IDs and vanity URLs with robust validation:

```go
func validateSteamIDOrVanity(input string) *steam.APIError {
    // Check for Steam ID format (17 digits starting with 7656119)
    if len(input) >= 7 && input[:7] == "7656119" {
        return validateSteamID(input)
    }
    
    // Check for vanity URL format (3-32 alphanumeric with _-)
    if !isValidVanityURL(input) {
        return steam.NewValidationError("Invalid vanity URL format")
    }
    
    return nil
}
```

### Why This Approach
1. **User-Friendly**: Accepts both `76561198215615835` and `counteredspell`
2. **Security**: Prevents injection attacks and invalid requests
3. **Steam Compatibility**: Follows Steam's exact validation rules
4. **Error Clarity**: Provides specific error messages for different validation failures

## 📊 Error Handling & Observability

### Structured Logging
All operations log structured data for monitoring and debugging:

```go
log.Info("Killer grade decoded successfully",
    "raw_value", 16,
    "tier", "Ash",
    "sub", 4,
    "field_id", "DBD_SlasherTierIncrement")
```

### Error Classification
Errors are classified for appropriate handling:

- **ValidationError**: Client-side issues (400 response)
- **NotFoundError**: Resource doesn't exist (404 response)
- **RateLimitError**: Steam API rate limiting (429 response)
- **APIError**: Steam API issues (502 response)
- **InternalError**: Server-side issues (500 response)

### Monitoring Endpoints
- `/api/cache/status` - Cache metrics and circuit breaker state
- `/api/health` - Application health check
- Structured logs provide comprehensive operation tracing

## 🚀 Performance Optimizations

### Parallel Data Fetching
```go
func (h *Handler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
    // Launch parallel goroutines for different data sources
    go fetchPlayerStats(ctx, steamID, resultChan)
    go fetchPlayerAchievements(ctx, steamID, resultChan)
    go fetchPlayerSummary(ctx, steamID, resultChan)
    
    // Combine results as they arrive
    combineResults(results)
}
```

### Memory Management
- **LRU Cache**: Automatic memory management with configurable limits
- **Goroutine Pools**: Reused goroutines for Steam API calls
- **Connection Pooling**: HTTP client connection reuse

### Caching Strategy
- **Cache-First**: Always check cache before making Steam API calls
- **Background Refresh**: Refresh popular entries before expiration
- **Compression**: Efficient serialization of cached data

## 🧪 Testing Strategy

### Test Categories
1. **Unit Tests**: Individual function validation
2. **Integration Tests**: API endpoint testing with real scenarios
3. **Cache Tests**: Memory management and corruption detection
4. **Circuit Breaker Tests**: Failure scenarios and recovery
5. **Validation Tests**: Input validation edge cases

### Test Data Strategy
- **Real Vanity URLs**: Tests use actual Steam vanity URL `counteredspell`
- **Realistic Values**: Grade codes (16, 65, 73) match current DBD system
- **Edge Cases**: Unknown grades, network timeouts, corrupted cache data
- **Concurrent Testing**: Multi-goroutine cache access validation

## 🔧 Configuration Management

### Environment-Based Configuration
```go
type Config struct {
    APITimeout              time.Duration `env:"API_TIMEOUT" default:"30s"`
    CachePlayerStatsTTL     time.Duration `env:"CACHE_PLAYER_STATS_TTL" default:"5m"`
    CircuitBreakerMaxFails  int           `env:"CIRCUIT_BREAKER_MAX_FAILURES" default:"5"`
    CircuitBreakerTimeout   time.Duration `env:"CIRCUIT_BREAKER_RESET_TIMEOUT" default:"60s"`
}
```

### Why Environment-Based
1. **Security**: Sensitive values not in source code
2. **Flexibility**: Different settings for dev/staging/production
3. **Monitoring**: Easy to adjust timeouts and thresholds
4. **Deployment**: Container-friendly configuration

## 📈 Scalability Considerations

### Current Implementation
- **Memory Cache**: Handles up to 100K entries efficiently
- **Concurrent Requests**: Thread-safe operations with proper synchronization
- **Circuit Breaker**: Protects against Steam API overload

### Future Scaling Options
- **Redis Cache**: Distributed caching for multi-instance deployment
- **Database Layer**: Persistent storage for historical data
- **Load Balancing**: Multiple API server instances
- **CDN Integration**: Static asset distribution

## 🔄 Deployment & Operations

### Build Process
```bash
# Backend: Single binary with embedded assets
go build -ldflags="-s -w" -o dbd-analytics.exe ./cmd/app

# Frontend: Optimized static build
npm run build
```

### Health Monitoring
- **Health Endpoint**: `/api/health` for load balancer checks
- **Metrics Endpoint**: `/api/cache/status` for operational monitoring
- **Structured Logs**: JSON format for log aggregation systems
- **Circuit Breaker Metrics**: Automatic failure detection and alerting

This architecture provides a robust, scalable, and maintainable solution for DBD analytics with comprehensive error handling, intelligent caching, and field-aware data processing.
