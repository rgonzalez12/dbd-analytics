# Achievement System Implementation

## Overview

The achievement system provides comprehensive tracking for Dead by Daylight achievements, with specialized focus on **Adept achievements** which require character name mapping for proper display.

## Key Features

### 🎯 Character Name Normalization
- **86 tracked adept achievements** with character name mapping
- **Steam API inconsistencies resolved**: Maps Steam names like "cannibal" → "The Cannibal"
- **Display-ready formatting**: Proper capitalization and "The" prefix handling
- **Missing character handling**: Graceful fallback for unmapped characters

### 📊 Achievement Categories
- **All Steam achievements**: Complete set from Dead by Daylight (381210)
- **Adept filtering**: Easy identification of character-specific achievements  
- **Progress tracking**: Unlocked status and unlock timestamps
- **Data completeness**: Handles achievements with missing descriptions or icons

### ⚡ Performance & Reliability
- **Intelligent caching**: Multi-layer cache with TTL and corruption detection
- **Parallel fetching**: Concurrent API calls for achievements + stats
- **Circuit breaker protection**: Automatic Steam API failure recovery
- **Graceful degradation**: Continues operation even with partial Steam API failures

## Architecture Details

### Character Mapping System

**Problem Solved**: Steam API returns inconsistent character names for adept achievements.

**Solution**: Comprehensive character mapping that normalizes names for proper display:

```go
// Character mapping handles Steam API inconsistencies
var characterMapping = map[string]string{
    "cannibal":     "The Cannibal",     // Steam: "cannibal" → Display: "The Cannibal"
    "shape":        "The Shape",        // Steam: "shape" → Display: "The Shape"  
    "nightmare":    "The Nightmare",    // Steam: "nightmare" → Display: "The Nightmare"
    "pig":          "The Pig",          // Steam: "pig" → Display: "The Pig"
    // ... 86 total mappings
}
```

**Why This Approach:**
- **User Experience**: Proper character names instead of internal Steam IDs
- **Consistency**: Uniform "The" prefix handling across all characters
- **Maintainability**: Single mapping table for all character name transformations
- **Future-Proof**: Easy to add new characters as they're released

### Achievement Processing Pipeline

1. **Fetch**: Get all achievements from Steam API (app 381210)
2. **Filter**: Identify adept achievements using name patterns
3. **Map**: Transform character names using mapping table
4. **Enhance**: Add proper formatting and display names
5. **Cache**: Store processed results with intelligent TTL
6. **Serve**: Return achievement data with proper character names

### Integration with Stats System

The achievement system works alongside the field-aware stats system:

- **Parallel Data Fetching**: Achievements and stats fetched concurrently
- **Unified Response**: Single API endpoint returns both achievements and stats
- **Consistent Caching**: Both use the same multi-layer cache infrastructure
- **Error Isolation**: Achievement failures don't impact stats display (and vice versa)

## Technical Implementation

### Core Files

#### Achievement Processing
- **`internal/steam/achievements.go`** - Core achievement fetching and processing
- **`internal/steam/adepts.go`** - Specialized adept achievement handling with character mapping
- **`internal/steam/achievement_mapper.go`** - Character name transformation logic

#### Character Mapping
- **`internal/steam/achievement_config.go`** - Complete character mapping table (86 entries)
- **Handles edge cases**: Characters with/without "The" prefix, special characters, DLC characters

#### API Integration  
- **`internal/api/handlers.go`** - Parallel achievement + stats fetching
- **`internal/steam/client.go`** - Steam API communication with caching

### Steam API Schema Discovery

**Problem**: Achievement names were initially guessed rather than using official Steam API identifiers.

**Solution**: Used Steam's `GetSchemaForGame` API to discover official achievement structure:

```
https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?appid=381210&key={STEAM_API_KEY}
```

**Discovery Process:**
1. Fetch complete achievement schema for Dead by Daylight (AppID: 381210)
2. Filter achievements matching "Adept {CharacterName}" pattern
3. Map Steam API names to proper character display names
4. Verify coverage for all characters (survivors + killers + DLC)

**Naming Pattern Evolution:**
- **Base Game**: `ACH_UNLOCK_DWIGHT_PERKS`
- **Early DLC**: `ACH_DLC2_SURVIVOR_1`, `ACH_DLC3_KILLER_3` 
- **Chapter System**: `ACH_CHAPTER9_SURVIVOR_3`, `ACH_CHAPTER16_KILLER_3`
- **Modern DLC**: `NEW_ACHIEVEMENT_245_23`, `NEW_ACHIEVEMENT_280_10`

**Why Schema-First Approach:**
- **100% Accuracy**: Uses official Steam identifiers instead of guessing
- **Complete Coverage**: Includes all DLC and chapter characters
- **Future-Proof**: New achievements automatically discovered
- **Maintenance-Free**: No manual updates needed when new characters release

### Testing Coverage

#### Unit Tests (All Passing ✅)
- **Character mapping validation**: All 86 adept achievements properly mapped
- **Name transformation**: Proper "The" prefix handling and capitalization
- **Edge case handling**: Unknown characters, malformed names, missing data
- **API integration**: Mocked Steam API responses and error scenarios

#### Integration Tests
- **Live Steam API**: Tests against actual Steam API with real account data
- **Character accuracy**: Validates character names match actual game names
- **Performance validation**: Confirms caching reduces API calls
- **Error recovery**: Tests graceful handling of Steam API failures

## Usage Examples

### Character Name Display
```
Steam API Returns: "adept_cannibal"
System Transforms To: "Adept The Cannibal"

Steam API Returns: "adept_trapper" 
System Transforms To: "Adept The Trapper"
```

### API Response Structure
```json
{
  "achievements": [
    {
      "name": "adept_cannibal",
      "displayName": "Adept The Cannibal",
      "description": "Achieve a merciless victory...",
      "unlocked": true,
      "unlockTime": "2023-10-15T14:30:00Z",
      "isAdept": true,
      "characterName": "The Cannibal"
    }
  ],
  "adeptCount": 25,
  "totalAchievements": 127
}
```

## API Endpoints

### GET /api/player/{steamID}

Fetches player statistics and achievements in a single endpoint.

**Graceful Degradation:**
- **Both succeed**: Full response with stats and achievements
- **Stats succeed, achievements fail**: Stats-only response with error in `data_sources`
- **Stats fail**: Returns HTTP error (stats are critical)

**Caching Strategy:**
- **Player Stats**: 5 minutes TTL
- **Player Achievements**: 30 minutes TTL  
- **Combined Response**: 10 minutes TTL

Cache keys:
- `player_stats:{steamID}`
- `player_achievements:{steamID}`
- `player_combined:{steamID}`

**Error Handling:**
- Private Steam profiles return achievement errors but continue with stats
- Invalid Steam IDs return validation errors
- API failures are logged with detailed context
- Circuit breaker protects against Steam API outages

## Future Considerations

### Scalability
- **New Character Support**: Easy addition to character mapping table
- **Achievement Expansion**: System handles new achievement types automatically
- **Performance**: Caching strategy scales with achievement count growth

### Maintenance
- **Character Updates**: Simple mapping table updates for character reworks
- **Steam API Changes**: Robust error handling for API modifications
- **Testing**: Comprehensive test suite ensures reliability across updates
