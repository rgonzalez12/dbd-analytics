# DBD Analytics

A Dead by Daylight statistics and achievement tracker that uses the Steam API to provide player data in a readable format.

## Overview

This application solves several problems with Steam's Dead by Daylight data:

- Steam returns grade data as numbers (16, 73, 439) without context - this tool converts them to readable grades like "Bronze II" or "Ash IV"
- Dead by Daylight has 86+ adept achievements with inconsistent character naming - this application normalizes them
- Steam API responses are slow and unreliable - a caching layer improves performance and availability

## Features

- **Grade Detection**: Converts numeric grade codes to readable rank names for both killer and survivor grades
- **Achievement Mapping**: Complete catalog of all Dead by Daylight adept achievements with standardized character names
- **Caching System**: Multi-layer cache reduces response times and handles Steam API outages
- **Circuit Breaker**: Graceful degradation when Steam API is unavailable
- **REST API**: Clean JSON endpoints for integration with other tools

## Quick Start

### Requirements
- Go 1.21 or higher
- Node.js 18 or higher  
- Steam API key (get from https://steamcommunity.com/dev/apikey)

### Setup

1. Clone the repository:
```bash
git clone https://github.com/rgonzalez12/dbd-analytics.git
cd dbd-analytics
```

2. Create environment configuration:
```bash
echo "STEAM_API_KEY=your_key_here" > .env
echo "LOG_LEVEL=info" >> .env
echo "PORT=8080" >> .env
```

3. Start the backend server:
```bash
go run ./cmd/app
# Server runs at http://localhost:8080
```

4. Start the frontend (optional):
```bash
cd frontend
npm install
npm run dev
# Frontend runs at http://localhost:5173
```

5. Test the API:
```bash
# Get player stats for any Steam ID
curl http://localhost:8080/api/player/76561198215615835
```

## API Response Example
```json
{
  "steam_id": "76561198215615835",
  "display_name": "PlayerName",
  "stats": {
    "killer": {
      "killer_grade": "Bronze II",
      "total_kills": "1,234",
      "sacrificed_victims": "987"
    },
    "survivor": {
      "survivor_grade": "Gold IV", 
      "total_escapes": "456",
      "generators_completed": "78.5%"
    }
  },
  "achievements": {
    "adept_trapper": {"unlocked": true, "character": "The Trapper"},
    "adept_dwight": {"unlocked": false, "character": "Dwight Fairfield"}
  }
}
```

## System Architecture

```
Frontend (SvelteKit)     API Server (Go)         Steam API
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│                 │    │                  │    │                 │
│ • TypeScript    │◄──►│ • HTTP Handlers  │◄──►│ • Player Stats  │
│ • Responsive UI │    │ • Caching Layer  │    │ • Achievements  │
│ • Real-time     │    │ • Circuit Breaker│    │ • Game Schema   │
│                 │    │ • Grade Detection│    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## How It Works

The application converts Steam's numeric grade codes into readable ranks by using the field context from Steam's game schema.

**Problem**: Steam API returns grades as numbers without context
```json
{"DBD_SlasherTierIncrement": 439}
```

**Solution**: Field-aware decoding determines if this is a killer or survivor grade
```json
{"killer_grade": "Bronze II"}
```

The system identifies grade types by examining Steam schema field names:
- `DBD_SlasherTierIncrement` indicates killer grades
- `DBD_UnlockRanking` indicates survivor grades

## Documentation

- [Technical Architecture](TECHNICAL_ARCHITECTURE.md) - System design details
- [Achievement System](ACHIEVEMENTS.md) - Character name mapping for 86+ achievements
- [Caching Strategy](CACHING.md) - Multi-layer cache and circuit breaker implementation
- [Stats Implementation](STATS_IMPLEMENTATION.md) - Grade detection algorithm details
- [Security](SECURITY.md) - Production security measures
- [Logging](LOGGING.md) - Structured logging and monitoring setup

## Development

### Running Tests
```bash
# Backend tests
go test ./...

# Frontend tests  
cd frontend && npm test
```

### Project Structure
```
cmd/app/           # Application entry point
internal/
  ├── api/         # HTTP handlers and middleware
  ├── cache/       # Caching layer with circuit breaker
  ├── steam/       # Steam API integration
  └── models/      # Data structures
frontend/          # SvelteKit application
```

## Deployment

### Using Docker
```bash
docker-compose build
docker-compose up -d
```

### Manual Deployment
```bash
# Build backend
go build -o dbd-analytics ./cmd/app

# Build frontend
cd frontend && npm run build
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Commit your changes: `git commit -m 'Add new feature'`
4. Push to the branch: `git push origin feature/new-feature`
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
