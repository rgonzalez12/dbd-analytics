# DBD Analytics

Dead by Daylight player statistics and achievement tracker with Steam API integration.

## Architecture

- **Backend**: Go API server with caching and Steam integration
- **Frontend**: SvelteKit web application with TypeScript
- **Data**: Steam API for player stats and achievements

## Quick Start

### Prerequisites
- Go 1.21+ for backend
- Node.js 18+ for frontend  
- Steam API key (set as `STEAM_API_KEY` environment variable)

### 1. Clone the Repository
```bash
git clone https://github.com/rgonzalez12/dbd-analytics.git
cd dbd-analytics
```

### 2. Set Up Environment
Create a `.env` file in the root directory:
```bash
STEAM_API_KEY=your_steam_api_key_here
LOG_LEVEL=info
PORT=8080
```

### 3. Start the Backend
```bash
# Build and run the Go server
go build -o dbd-analytics.exe ./cmd/app
./dbd-analytics.exe
```

The backend will start on `http://localhost:8080`

### 4. Start the Frontend (New Terminal)
```bash
# Navigate to frontend directory
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend will start on `http://localhost:5173` with proxy to backend

### 5. Access the Application
- **Web Interface**: http://localhost:5173
- **API Endpoints**: http://localhost:8080/api
- **Example**: http://localhost:5173/player/counteredspell

## Configuration

### Environment Variables

Required:
```bash
STEAM_API_KEY=your_steam_api_key_here
```

Optional:
```bash
PORT=8080
LOG_LEVEL=info

# Cache TTL settings
CACHE_PLAYER_STATS_TTL=5m
CACHE_PLAYER_SUMMARY_TTL=10m  
CACHE_STEAM_API_TTL=3m

# Circuit breaker settings
CIRCUIT_BREAKER_MAX_FAILURES=5
CIRCUIT_BREAKER_RESET_TIMEOUT=30s
```

## Features

### Cache System
- In-memory cache with LRU eviction and TTL
- Circuit breaker for Steam API protection
- Stale data fallback during outages
- Automatic corruption detection and recovery

### Production Ready
- Thread-safe concurrent access
- Graceful shutdown and cleanup
- Structured logging with observability
- Comprehensive error handling
- Steam vanity URL resolution
- Player achievement tracking

## Monitoring

Cache metrics available at `/api/cache/status`:

```json
{
  "cache_stats": {
    "hits": 1234,
    "misses": 56,
    "hit_rate": 95.6,
    "entries": 500,
    "memory_usage": 1048576
  },
  "circuit_breaker": {
    "state": "closed",
    "failures": 0,
    "last_success": "2025-08-03T20:35:30Z"
  }
}
```
```

### Log Examples
```log
INFO  Cache TTL configuration loaded player_stats_ttl=5m source_priority="env_vars > deprecated_constants > defaults"
WARN  Circuit breaker triggered for key="player:123" error="timeout" circuit_state="open" failure_count=3
INFO  Circuit breaker recovered and closed recovery_successes=3 downtime_duration=45s
INFO  Serving stale data from fallback cache key="player:123" circuit_state="open"
```

## Testing

```bash
# Backend tests
go test ./...
go test -cover ./...
go test -race ./...

# Frontend tests
cd frontend
npm run check
npm run lint
```

## Production Deployment

### Backend Build
```bash
go build -ldflags="-s -w" -o dbd-analytics.exe ./cmd/app
```

### Frontend Build
```bash
cd frontend
npm run build
```

### Production Environment
```bash
STEAM_API_KEY=your_production_key
PORT=8080
LOG_LEVEL=warn
CACHE_PLAYER_STATS_TTL=10m
CACHE_PLAYER_SUMMARY_TTL=30m
```

## 📁 Project Structure

```
dbd-analytics/
├── cmd/app/                    # Go application entry point
│   └── main.go
├── internal/                   # Go backend source code
│   ├── api/                    # HTTP handlers and API routes
│   ├── cache/                  # Caching system with circuit breaker
│   ├── steam/                  # Steam API client integration
│   ├── models/                 # Data models and types
│   ├── security/               # Validation and security
│   └── log/                    # Structured logging
├── frontend/                   # SvelteKit web application
│   ├── src/
│   │   ├── lib/                # Shared utilities and API client
│   │   │   └── api/            # TypeScript API client
│   │   ├── routes/             # SvelteKit page routes
│   │   │   ├── +layout.svelte  # Root layout
│   │   │   ├── +page.svelte    # Home page
│   │   │   └── player/         # Player pages
│   │   └── app.html            # HTML template
│   └── package.json            # Frontend dependencies
├── tools/                      # Development and testing utilities
├── .env                        # Environment configuration
└── go.mod                      # Go module definition
```

## Development

### Backend
```bash
# Live reload (install: go install github.com/air-verse/air@latest)
air

# Direct run
go run ./cmd/app
```

### Frontend
```bash
cd frontend
npm run dev
```

## API Endpoints

### Player Data
- `GET /api/player/{steamId}` - Combined player stats and achievements
- `GET /api/player/{steamId}/stats` - Player statistics only
- `GET /api/player/{steamId}/summary` - Player summary only

### System Status
- `GET /api/cache/status` - Cache and circuit breaker metrics
- `GET /api/health` - Application health check

### Example Usage
```bash
curl http://localhost:8080/api/player/76561198000000000
curl http://localhost:8080/api/player/counteredspell
curl http://localhost:8080/api/cache/status
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/name`)
3. Make your changes and add tests
4. Ensure tests pass (`go test ./...` and `npm run check`)
5. Commit changes (`git commit -m 'Add feature'`)
6. Push to branch (`git push origin feature/name`)
7. Open a Pull Request

## 📈 Performance Characteristics

- **Memory Cache**: Efficiently handles up to 100K entries
- **Concurrent Access**: Tested with multiple goroutines
- **Circuit Breaker**: 60-second sliding window with configurable thresholds
- **Recovery**: Jitter-based to prevent thundering herd issues
- **Frontend**: Optimized bundle with code splitting
- **API Response**: Typical response times under 100ms (cached)

## Troubleshooting

**Backend not starting:**
- Check `STEAM_API_KEY` is set in `.env` file
- Verify port 8080 is available
- Check logs for configuration errors

**Frontend not loading data:**
- Ensure backend is running on port 8080
- Check browser network tab for API errors
- Verify proxy configuration in `vite.config.ts`

**Steam API errors:**
- Validate Steam API key
- Check rate limiting (Steam has request limits)
- Review circuit breaker status at `/api/cache/status`

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.