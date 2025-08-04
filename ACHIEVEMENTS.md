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

## Achievement API Name Discovery

### Problem Statement

Initially, the achievement mapping only worked for base game characters because DLC character achievement names were guessed rather than using actual Steam API identifiers. Steam's achievement naming conventions are inconsistent across different DLC releases, making manual guessing unreliable.

### Solution: Steam GetSchemaForGame API

We used Steam's `GetSchemaForGame` endpoint to fetch the official achievement schema and extract the real API names:

**Endpoint:**
```
https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?appid=381210&key={STEAM_API_KEY}
```

**Discovery Process:**

1. **Fetch Full Schema**: Query the GetSchemaForGame endpoint for Dead by Daylight (AppID: 381210)
2. **Filter Adept Achievements**: Extract achievements with names matching "Adept {CharacterName}"
3. **Map API Names**: Record the `name` field (API identifier) for each Adept achievement
4. **Verify Coverage**: Ensure all 84 characters (46 survivors + 38 killers) have mappings

**Key Findings:**

- **Base Game**: Consistent naming like `ACH_UNLOCK_DWIGHT_PERKS`
- **Early DLC**: Pattern like `ACH_DLC2_SURVIVOR_1`, `ACH_DLC3_KILLER_3`
- **Chapter System**: Pattern like `ACH_CHAPTER9_SURVIVOR_3`, `ACH_CHAPTER16_KILLER_3`
- **Modern DLC**: Pattern like `NEW_ACHIEVEMENT_245_23`, `NEW_ACHIEVEMENT_280_10`

**Example Schema Response:**
```json
{
  "game": {
    "availableGameStats": {
      "achievements": [
        {
          "name": "ACH_DLC2_SURVIVOR_1",
          "displayName": "Adept Laurie Strode",
          "description": "...",
          "icon": "...",
          "hidden": 0
        }
      ]
    }
  }
}
```

### Applying to Other Games

This methodology works for any Steam game with achievements:

1. **Find the AppID**: Look up the game's Steam AppID
2. **Query Schema**: Use `GetSchemaForGame` with your Steam API key
3. **Parse Response**: Extract achievement names and API identifiers
4. **Map Achievements**: Create mappings between display names and API names
5. **Verify**: Test with known achievement data

**Required Steam API Key**: You'll need a Steam Web API key from https://steamcommunity.com/dev/apikey

This approach ensures 100% accuracy versus guessing achievement names, and scales to any Steam game with achievements.
