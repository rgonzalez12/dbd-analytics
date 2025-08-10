# DBD Analytics

A comprehensive analytics and statistics tool for Dead by Daylight with a modern web interface.

## 🏗️ Architecture

- **Backend**: Go API server with Steam integration, caching, and circuit breaker protection
- **Frontend**: SvelteKit web application with TypeScript and Tailwind CSS
- **Data**: Steam API integration for player stats and achievements

## 🚀 Quick Start

### Prerequisites
- **Go 1.21+** for the backend
- **Node.js 18+** for the frontend
- **Steam API Key** (set as `STEAM_API_KEY` environment variable)

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
go build -o bin/dbd-analytics.exe ./cmd/app
./bin/dbd-analytics.exe
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

## ⚙️ Configuration

### Environment Variables

Create a `.env` file in the root directory with the following variables:

#### Required Configuration
```bash
STEAM_API_KEY=your_steam_api_key_here    # Steam API key (required)
```

#### Optional Configuration
```bash
# Server Configuration
PORT=8080                               # Server port (default: 8080)
LOG_LEVEL=info                         # Log level: debug, info, warn, error

# Cache Configuration - Time-To-Live settings
CACHE_PLAYER_STATS_TTL=5m              # Player statistics cache duration
CACHE_PLAYER_SUMMARY_TTL=10m           # Player summary cache duration  
CACHE_STEAM_API_TTL=3m                 # Steam API response cache duration
CACHE_DEFAULT_TTL=3m                   # Default cache duration

# Circuit Breaker Configuration - Steam API protection
CIRCUIT_BREAKER_MAX_FAILURES=5         # Failures before opening circuit
CIRCUIT_BREAKER_RESET_TIMEOUT=30s      # Time before retry attempt
CIRCUIT_BREAKER_SUCCESS_RESET=3        # Successes needed to close circuit

# Cache Warm-up (Optional)
CACHE_WARMUP_ENABLED=true              # Enable cache pre-loading on startup
CACHE_WARMUP_TIMEOUT=30s               # Maximum time for warm-up process
```

### Frontend Configuration

The frontend automatically proxies API requests to the backend through Vite configuration. No additional setup required for development.

### Configuration Priority

1. **Environment Variables** (`.env` file or system environment)
2. **Application Defaults**

## 🏗️ Architecture

### Cache System
- **In-memory cache** with LRU eviction and TTL support
- **Circuit breaker** for Steam API protection with graceful degradation
- **Stale data fallback** - serves expired cache during outages
- **Corruption detection** and automatic recovery
- **Comprehensive metrics** for monitoring and debugging

### Frontend Features
- **SvelteKit** with server-side rendering and TypeScript
- **Type-safe API client** with error handling and timeouts  
- **Tailwind CSS** for responsive design
- **Development proxy** to backend API
- **Loading states** and error boundaries

### Production Features
- ✅ Thread-safe concurrent access
- ✅ Graceful shutdown and cleanup
- ✅ Structured logging with observability
- ✅ Jitter-based recovery (prevents thundering herd)
- ✅ State persistence for circuit breaker
- ✅ Comprehensive error handling
- ✅ Steam vanity URL resolution
- ✅ Player achievement tracking

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

### Backend Tests
```bash
# Run all backend tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test suites
go test ./internal/cache -v              # Cache system tests
go test ./internal/api -v                # API handler tests  
go test ./internal/steam -v              # Steam API client tests

# Run tests with race detection
go test -race ./...
```

### Frontend Tests
```bash
cd frontend

# Type checking
npm run check

# Linting
npm run lint

# Format check
npm run format:check
```

## 🚀 Production Deployment

### Backend Production Build
```bash
# Build optimized binary
go build -ldflags="-s -w" -o bin/dbd-analytics ./cmd/app

# Or build for specific platform
GOOS=linux GOARCH=amd64 go build -o bin/dbd-analytics-linux ./cmd/app
```

### Frontend Production Build
```bash
cd frontend

# Build for production
npm run build

# Preview production build locally
npm run preview
```

### Recommended Production Environment
```bash
# Production-optimized settings
STEAM_API_KEY=your_production_steam_api_key
PORT=8080
LOG_LEVEL=warn

# Extended cache durations for production
CACHE_PLAYER_STATS_TTL=10m
CACHE_PLAYER_SUMMARY_TTL=30m
CACHE_STEAM_API_TTL=5m

# Robust circuit breaker settings
CIRCUIT_BREAKER_MAX_FAILURES=10
CIRCUIT_BREAKER_RESET_TIMEOUT=60s
CIRCUIT_BREAKER_SUCCESS_RESET=5

# Enable warm-up for better user experience
CACHE_WARMUP_ENABLED=true
CACHE_WARMUP_TIMEOUT=60s
```

### Docker Deployment (Optional)
```dockerfile
# Example Dockerfile for backend
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o dbd-analytics ./cmd/app

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/dbd-analytics .
EXPOSE 8080
CMD ["./dbd-analytics"]
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
│   ├── static/                 # Static assets
│   └── package.json            # Frontend dependencies
├── bin/                        # Compiled Go binaries
├── scripts/                    # Utility scripts
├── .env                        # Environment configuration
└── go.mod                      # Go module definition
```

## 🔧 Development

### Backend Development
```bash
# Run with live reload using air (install: go install github.com/air-verse/air@latest)
air

# Or run directly
go run ./cmd/app

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Frontend Development
```bash
cd frontend

# Development with hot reload
npm run dev

# Type checking
npm run check

# Linting
npm run lint

# Build for production
npm run build

# Preview production build
npm run preview
```

## 🌐 API Endpoints

### Player Data
- `GET /api/player/{steamId}` - Combined player stats and achievements
- `GET /api/player/{steamId}/stats` - Player statistics only
- `GET /api/player/{steamId}/summary` - Player summary only

### System Status
- `GET /api/cache/status` - Cache and circuit breaker metrics
- `GET /api/health` - Application health check

### Example Usage
```bash
# Get player data by Steam ID
curl http://localhost:8080/api/player/76561198000000000

# Get player data by vanity URL
curl http://localhost:8080/api/player/counteredspell

# Check system status
curl http://localhost:8080/api/cache/status
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...` and `npm run check`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Development Guidelines
- Follow Go and TypeScript best practices
- Add tests for new features
- Update documentation as needed
- Use conventional commit messages
- Ensure frontend types match backend responses

## 📈 Performance Characteristics

- **Memory Cache**: Efficiently handles up to 100K entries
- **Concurrent Access**: Tested with multiple goroutines
- **Circuit Breaker**: 60-second sliding window with configurable thresholds
- **Recovery**: Jitter-based to prevent thundering herd issues
- **Frontend**: Optimized bundle with code splitting
- **API Response**: Typical response times under 100ms (cached)

## 🆘 Troubleshooting

### Common Issues

**Backend not starting:**
- Check that `STEAM_API_KEY` is set in your `.env` file
- Verify port 8080 is not in use
- Check logs for configuration errors

**Frontend not loading data:**
- Ensure backend is running on port 8080
- Check browser network tab for API errors
- Verify frontend proxy configuration in `vite.config.ts`

**Steam API errors:**
- Validate your Steam API key
- Check rate limiting (Steam API has request limits)
- Review circuit breaker status at `/api/cache/status`

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.