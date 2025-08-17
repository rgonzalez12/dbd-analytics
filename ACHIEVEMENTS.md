# Achievement System

Complete tracking system for Dead by Daylight's 86+ adept achievements with character name normalization.

## Problem Description

Dead by Daylight has 86 adept achievements (46 survivors + 40 killers), but Steam provides them with inconsistent naming conventions:

```json
// Raw Steam achievement data
{
  "ACH_UNLOCK_DWIGHT_PERKS": "Achieved with a level...",
  "ACH_UNLOCK_CHUCKLES_PERKS": "Adept Chuckles",  
  "ACH_UNLOCK_CANNIBAL_PERKS": "Adept cannibal"
}
```

Issues with the raw data:
- Inconsistent capitalization (`cannibal` vs `Chuckles`)
- Cryptic internal names (`CHUCKLES_PERKS` refers to The Trapper)
- Lack of character context in achievement names

## Normalization Solution

The system transforms raw Steam data into consistent, readable format:

```json
// Processed achievement data
{
  "adept_dwight": {
    "unlocked": true,
    "character": "Dwight Fairfield",
    "display_name": "Adept Dwight",
    "type": "adept_survivor"
  },
  "adept_trapper": {
    "unlocked": false, 
    "character": "The Trapper",
    "display_name": "Adept Trapper",
    "type": "adept_killer"
  }
}
```

## Character Mapping Implementation

### Survivor Character Mapping
```go
var survivorMap = map[string]string{
    "dwight":     "Dwight Fairfield",
    "meg":        "Meg Thomas", 
    "claudette":  "Claudette Morel",
    "jake":       "Jake Park",
    "nea":        "Nea Karlsson",
    "laurie":     "Laurie Strode",
    "ace":        "Ace Visconti",
    "feng":       "Feng Min",
    // ... 38 more mappings
}
```

### Killer Mapping (40 Characters)  
```go
var killerMap = map[string]string{
    "chuckles":   "The Trapper",
    "wraith":     "The Wraith",
    "hillbilly":  "The Hillbilly", 
    "nurse":      "The Nurse",
    "shape":      "The Shape",
    "cannibal":   "The Cannibal",
    "nightmare":  "The Nightmare",
    "pig":        "The Pig",
    // ... 32 more mappings
}
```

## Processing Pipeline

### 1. Achievement Discovery
```go
func processAchievements(playerAchievements []SteamAchievement) {
    // Get all possible achievements from Steam schema
    allAchievements := getCompleteAchievementCatalog()
    
    // Merge with player's unlocked achievements
    return mergePlayerProgress(allAchievements, playerAchievements)
}
```

### 2. Adept Filtering & Classification
```go
func classifyAchievement(apiName, displayName string) AchievementType {
    // Check for adept patterns
    if isAdeptPattern(apiName, displayName) {
        if isKillerCharacter(extractCharacter(apiName)) {
            return "adept_killer"
        }
        return "adept_survivor" 
    }
    return "general"
}
```

### 3. Character Name Normalization
```go
func normalizeCharacterName(rawName string) string {
    // Clean up the name
    cleaned := strings.ToLower(strings.TrimSpace(rawName))
    
    // Remove common prefixes/suffixes
    cleaned = strings.TrimPrefix(cleaned, "the ")
    cleaned = strings.TrimSuffix(cleaned, " perks")
    
    // Apply character mapping
    if proper, exists := characterMap[cleaned]; exists {
        return proper
    }
    
    // Fallback to title case
    return strings.Title(cleaned)
}
```

## Complete Achievement Catalog

### Complete Coverage: Always 86 Adepts

Even with an empty Steam account, our system returns the complete catalog:

```json
{
  "summary": {
    "total_achievements": 86,
    "unlocked_count": 0,
    "survivor_count": 46, 
    "killer_count": 40,
    "completion_rate": 0.0
  },
  "achievements": [
    // All 86 adepts with unlocked: false
  ]
}
```

### Benefits
- **Complete Picture**: See what you haven't unlocked yet
- **Progress Tracking**: Clear completion percentage  
- **Character Discovery**: Learn about all available characters

## 🎭 Character Categories

### Survivors (46 Total)
- **Original Four**: Dwight, Meg, Claudette, Jake
- **Licensed**: Laurie Strode, Bill Overbeck, Ash Williams, etc.
- **Recent Additions**: All new survivors automatically mapped

### Killers (40 Total)  
- **Original Trio**: Trapper, Wraith, Hillbilly
- **Licensed**: The Shape, Pig, Ghost Face, Pyramid Head, etc.
- **Recent Additions**: Automated detection for new killers

## 🔧 Error Handling & Fallbacks

### Unknown Characters
```go
func handleUnknownCharacter(apiName string) Achievement {
    log.Warn("Unknown character detected", 
        "api_name", apiName,
        "action", "using_fallback_display")
        
    return Achievement{
        Character: "Unknown Character",
        DisplayName: generateFallbackName(apiName),
        Type: "adept_unknown"
    }
}
```

### Steam API Failures
```go
func getAchievements(playerData *PlayerAchievements) map[string]interface{} {
    // Try Steam schema first
    schema := fetchAchievementSchema()
    if schema == nil {
        log.Warn("Schema unavailable, using hardcoded catalog")
        schema = getHardcodedCatalog()
    }
    
    // Process with available data
    return processWithSchema(schema, playerData)
}
```

## Performance Features

### Efficient Processing
- **Parallel Fetching**: Schema + player data fetched concurrently
- **Smart Caching**: Character mappings cached for 1 hour
- **Batch Processing**: All achievements processed in single pass

### Memory Optimization
- **Lazy Loading**: Only load mappings when needed
- **Compact Storage**: Minimal memory footprint per achievement
- **Garbage Collection**: Automatic cleanup of unused mappings

## 📈 Real-World Usage

### API Response Example
```bash
curl http://localhost:8080/api/player/counteredspell/achievements
```

```json
{
  "steam_id": "76561198215615835",
  "display_name": "counteredspell", 
  "summary": {
    "total_achievements": 86,
    "unlocked_count": 23,
    "completion_rate": 26.7,
    "survivor_count": 46,
    "killer_count": 40
  },
  "achievements": {
    "adept_dwight": {
      "unlocked": true,
      "character": "Dwight Fairfield",
      "unlock_time": "2024-10-15T14:30:00Z"
    },
    "adept_trapper": {
      "unlocked": false,
      "character": "The Trapper"
    }
  }
}
```

This achievement system transforms Steam's raw data into a comprehensive, user-friendly catalog that enhances the Dead by Daylight player experience.
