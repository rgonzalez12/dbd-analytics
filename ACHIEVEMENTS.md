# Steam Achievements Integration

## Overview

The API now supports fetching both player statistics and Steam achievements in a single endpoint, providing comprehensive Dead by Daylight player data.

## New Endpoint

### GET /api/player/{steamID}

Fetches both player statistics and adept achievements for all survivors and killers.

**Response Format:**
```json
{
  "steam_id": "76561198000000000",
  "display_name": "PlayerName",
  
  // All existing player stats...
  "killer_pips": 1250,
  "survivor_pips": 980,
  "killed_campers": 450,
  
  // New achievements section
  "achievements": {
    "adept_survivors": {
      "dwight": true,
      "meg": false,
      "claudette": true,
      // ... all survivors
    },
    "adept_killers": {
      "trapper": true,
      "wraith": false,
      "nurse": true,
      // ... all killers  
    },
    "last_updated": "2025-08-03T15:30:45Z"
  },
  
  // Data source tracking for debugging
  "data_sources": {
    "stats": {
      "success": true,
      "source": "cache",
      "fetched_at": "2025-08-03T15:30:45Z"
    },
    "achievements": {
      "success": true,
      "source": "api", 
      "fetched_at": "2025-08-03T15:30:45Z"
    }
  }
}
```

## Graceful Degradation

The endpoint implements graceful degradation:

- **Both succeed**: Full response with stats and achievements
- **Stats succeed, achievements fail**: Stats-only response with error details in `data_sources.achievements.error`
- **Stats fail**: Returns HTTP error (stats are critical)

## Caching Strategy

- **Player Stats**: 5 minutes TTL (frequent updates from matches)
- **Player Achievements**: 30 minutes TTL (achievements unlock less frequently)
- **Combined Response**: 10 minutes TTL (balanced freshness)

Cache keys:
- `player_stats:{steamID}` - Individual stats cache
- `player_achievements:{steamID}` - Individual achievements cache  
- `player_combined:{steamID}` - Combined response cache

## Environment Variables

New TTL configuration options:

```bash
CACHE_PLAYER_ACHIEVEMENTS_TTL=30m
CACHE_PLAYER_COMBINED_TTL=10m
```

## Error Handling

- Private Steam profiles return achievement errors but continue with stats
- Invalid Steam IDs return validation errors
- API failures are logged with detailed context for monitoring
- Circuit breaker protects against Steam API outages

## Monitoring

Key metrics to monitor:
- Partial response rate (achievements failing)
- Cache hit rates for different data types
- Steam API error rates for achievements vs stats
- Response times for combined endpoint

## Migration

Existing endpoints remain unchanged:
- `/api/player/{steamID}/stats` - Stats only (unchanged)  
- `/api/player/{steamID}/summary` - Summary only (unchanged)
- `/api/player/{steamID}` - **NEW** Combined stats + achievements

Frontend can migrate to the new combined endpoint incrementally.
