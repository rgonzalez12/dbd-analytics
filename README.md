# dbd-analytics

Analytics and Stats tool for Dead by Daylight

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/rgonzalez12/dbd-analytics.git
cd dbd-analytics

# Build the application
go build -o bin/dbd-analytics.exe ./cmd/app

# Run the server
./bin/dbd-analytics.exe
```

The server will start on `http://localhost:8080` by default.

## ⚙️ Configuration

### Environment Variables

The application supports configuration via environment variables for production deployments:

#### Cache Configuration
```bash
# Cache TTL Settings (Time-To-Live)
CACHE_PLAYER_STATS_TTL=5m      # Player statistics cache duration (default: 5 minutes)
CACHE_PLAYER_SUMMARY_TTL=10m   # Player summary cache duration (default: 10 minutes)  
CACHE_STEAM_API_TTL=3m         # Steam API response cache duration (default: 3 minutes)
CACHE_DEFAULT_TTL=3m           # Default cache duration (default: 3 minutes)

# Examples of valid duration formats:
# 30s, 5m, 1h, 2h30m, 1h30m45s
```

#### Circuit Breaker Configuration
```bash
# Circuit breaker settings for Steam API reliability
CIRCUIT_BREAKER_MAX_FAILURES=5           # Failures before opening circuit (default: 5)
CIRCUIT_BREAKER_RESET_TIMEOUT=30s        # Time before retry attempt (default: 30s)
CIRCUIT_BREAKER_SUCCESS_RESET=3          # Successes needed to close circuit (default: 3)
CIRCUIT_BREAKER_FAILURE_THRESHOLD=0.5    # Failure rate threshold 0.0-1.0 (default: 0.5)
```

#### Cache Warm-up (Optional)
```bash
CACHE_WARMUP_ENABLED=true                # Enable cache pre-loading on startup
CACHE_WARMUP_TIMEOUT=30s                 # Maximum time for warm-up process
CACHE_WARMUP_CONCURRENT_JOBS=3           # Parallel warm-up workers
```

### Configuration Priority

The application uses the following configuration priority:
1. **Environment Variables** (highest priority)
2. **Deprecated Constants** (backward compatibility)
3. **Hardcoded Defaults** (fallback)

## 🏗️ Architecture

### Cache System
- **In-memory cache** with LRU eviction and TTL support
- **Circuit breaker** for Steam API protection with graceful degradation
- **Stale data fallback** - serves expired cache during outages
- **Corruption detection** and automatic recovery
- **Comprehensive metrics** for monitoring and debugging

### API Endpoints
- `GET /api/player/{id}/stats` - Player statistics with caching
- `GET /api/player/{id}/summary` - Player summary with circuit breaker protection
- `GET /api/cache/status` - Cache and circuit breaker health metrics

### Production Features
- ✅ Thread-safe concurrent access
- ✅ Graceful shutdown and cleanup
- ✅ Structured logging with observability
- ✅ Jitter-based recovery (prevents thundering herd)
- ✅ State persistence for circuit breaker
- ✅ Comprehensive error handling

## 📊 Monitoring

### Cache Metrics
The application exposes detailed metrics for production monitoring:

```json
{
  "cache_stats": {
    "hits": 1234,
    "misses": 56,
    "hit_rate": 95.6,
    "entries": 500,
    "memory_usage": 1048576,
    "evictions": 12,
    "corruption_events": 0,
    "uptime_seconds": 3600
  },
  "circuit_breaker": {
    "state": "closed",
    "failures": 0,
    "failure_rate": 0.0,
    "last_success": "2025-08-03T20:35:30Z"
  }
}
```

### Log Examples
```log
INFO  Cache TTL configuration loaded player_stats_ttl=5m source_priority="env_vars > deprecated_constants > defaults"
WARN  Circuit breaker triggered for key="player:123" error="timeout" circuit_state="open" failure_count=3
INFO  Circuit breaker recovered and closed recovery_successes=3 downtime_duration=45s
INFO  Serving stale data from fallback cache key="player:123" circuit_state="open"
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test suites
go test ./internal/cache -v                    # Cache system tests
go test ./internal/api -v                      # API handler tests  
go test ./internal/steam -v                    # Steam API client tests
```

## 🚀 Production Deployment

### Recommended Environment Variables
```bash
# Production cache settings
CACHE_PLAYER_STATS_TTL=10m
CACHE_PLAYER_SUMMARY_TTL=30m
CACHE_STEAM_API_TTL=5m

# Robust circuit breaker for production load
CIRCUIT_BREAKER_MAX_FAILURES=10
CIRCUIT_BREAKER_RESET_TIMEOUT=60s
CIRCUIT_BREAKER_SUCCESS_RESET=5

# Enable warm-up for better UX
CACHE_WARMUP_ENABLED=true
CACHE_WARMUP_TIMEOUT=60s
```

### Performance Characteristics
- **Memory Cache**: Efficiently handles up to 100K entries
- **Concurrent Access**: Tested with multiple goroutines
- **Circuit Breaker**: 60-second sliding window with configurable thresholds
- **Recovery**: Jitter-based to prevent thundering herd issues

## 📁 Project Structure

```
dbd-analytics/
├── cmd/app/                 # Application entry point
├── internal/
│   ├── api/                 # HTTP handlers and routes
│   ├── cache/               # Cache system with circuit breaker
│   │   ├── circuit_breaker.go
│   │   ├── manager.go
│   │   ├── memory.go
│   │   ├── warmup.go
│   │   ├── jitter.go
│   │   └── persistence.go
│   ├── steam/               # Steam API client
│   ├── models/              # Data models
│   └── log/                 # Structured logging
├── static/                  # Frontend assets
├── templates/               # HTML templates
└── bin/                     # Compiled binaries
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.