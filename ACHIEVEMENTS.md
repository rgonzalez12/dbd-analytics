# Steam Achievements Integration

## Overview

The API supports fetching both player statistics and Steam achievements in a single endpoint.

## Endpoint

### GET /api/player/{steamID}

Fetches player statistics and adept achievements for all survivors and killers.

**Response:**
```json
{
  "steam_id": "76561198000000000",
  "display_name": "PlayerName",
  
  "killer_pips": 1250,
  "survivor_pips": 980,
  "killed_campers": 450,
  
  "achievements": {
    "adept_survivors": {
      "dwight": true,
      "meg": false,
      "claudette": true
    },
    "adept_killers": {
      "trapper": true,
      "wraith": false,
      "nurse": true
    },
    "last_updated": "2025-08-03T15:30:45Z"
  },
  
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

- **Both succeed**: Full response with stats and achievements
- **Stats succeed, achievements fail**: Stats-only response with error in `data_sources`
- **Stats fail**: Returns HTTP error (stats are critical)

## Caching Strategy

- **Player Stats**: 5 minutes TTL
- **Player Achievements**: 30 minutes TTL  
- **Combined Response**: 10 minutes TTL

Cache keys:
- `player_stats:{steamID}`
- `player_achievements:{steamID}`
- `player_combined:{steamID}`

## Configuration

```bash
CACHE_PLAYER_ACHIEVEMENTS_TTL=30m
CACHE_PLAYER_COMBINED_TTL=10m
```

## Error Handling

- Private Steam profiles return achievement errors but continue with stats
- Invalid Steam IDs return validation errors
- API failures are logged with detailed context
- Circuit breaker protects against Steam API outages

## Migration

Existing endpoints remain unchanged:
- `/api/player/{steamID}/stats` - Stats only
- `/api/player/{steamID}/summary` - Summary only
- `/api/player/{steamID}` - **NEW** Combined stats + achievements

## Achievement API Discovery

### Problem
Initial achievement mapping only worked for base game characters because DLC achievement names were guessed rather than using actual Steam API identifiers.

### Solution: Steam GetSchemaForGame API

Used Steam's `GetSchemaForGame` endpoint to fetch official achievement schema:

**Endpoint:**
```
https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?appid=381210&key={STEAM_API_KEY}
```

**Process:**
1. Fetch full schema for Dead by Daylight (AppID: 381210)
2. Filter achievements matching "Adept {CharacterName}"
3. Map API names to character names
4. Verify coverage for all 84 characters (46 survivors + 38 killers)

**Naming Patterns:**
- **Base Game**: `ACH_UNLOCK_DWIGHT_PERKS`
- **Early DLC**: `ACH_DLC2_SURVIVOR_1`, `ACH_DLC3_KILLER_3`
- **Chapter System**: `ACH_CHAPTER9_SURVIVOR_3`, `ACH_CHAPTER16_KILLER_3`
- **Modern DLC**: `NEW_ACHIEVEMENT_245_23`, `NEW_ACHIEVEMENT_280_10`

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
          "icon": "..."
        }
      ]
    }
  }
}
```

### Applying to Other Games

This methodology works for any Steam game:
1. Find the game's Steam AppID
2. Query schema with `GetSchemaForGame` and your Steam API key
3. Parse response and extract achievement names
4. Create mappings between display names and API identifiers
5. Test with known achievement data

This ensures 100% accuracy versus guessing names.
