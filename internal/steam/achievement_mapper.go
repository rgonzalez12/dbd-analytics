package steam

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/cache"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

type AchievementMapping struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`         // displayName from schema
	DisplayName string  `json:"display_name"` // kept for backwards compatibility  
	Description string  `json:"description"`
	Icon        string  `json:"icon,omitempty"`
	IconGray    string  `json:"icon_gray,omitempty"`
	Hidden      bool    `json:"hidden,omitempty"`
	Character   string  `json:"character"`
	Type        string  `json:"type"`    // "adept_survivor", "adept_killer", "general"
	Unlocked    bool    `json:"unlocked"`
	UnlockTime  int64   `json:"unlock_time,omitempty"`
	Rarity      float64 `json:"rarity,omitempty"` // 0-100 global completion percentage
}

type UnknownAchievement struct {
	APIName     string    `json:"api_name"`
	FirstSeen   time.Time `json:"first_seen"`
	Occurrences int       `json:"occurrences"`
}

type AchievementMapper struct {
	unknownAchievs    map[string]*UnknownAchievement
	unknownsMutex     sync.RWMutex
	client            *Client
	adeptRegex        *regexp.Regexp
	adeptsByAPI       map[string]string // apiName -> "killer"|"survivor"
}

func NewAchievementMapper() *AchievementMapper {
	// Build adepts lookup from existing mapping
	adeptsByAPI := make(map[string]string)
	for apiName, adept := range AdeptAchievementMapping {
		adeptsByAPI[apiName] = adept.Type
	}

	client := NewClient()
	log.Info("Created achievement mapper", "steam_client_exists", client != nil)

	return &AchievementMapper{
		unknownAchievs: make(map[string]*UnknownAchievement),
		client:         client,
		adeptRegex:     regexp.MustCompile(`^Adept\s+(?:The\s+)?(.+)$`),
		adeptsByAPI:    adeptsByAPI,
	}
}

func (am *AchievementMapper) MapPlayerAchievements(achievements *PlayerAchievements) []AchievementMapping {
	return am.MapPlayerAchievementsWithCache(achievements, nil)
}

func (am *AchievementMapper) MapPlayerAchievementsWithCache(achievements *PlayerAchievements, cacheManager cache.Cache) []AchievementMapping {
	ctx := context.Background()
	
	// 1) Build map from player data
	unlockedMap := make(map[string]SteamAchievement)
	for _, achievement := range achievements.Achievements {
		unlockedMap[achievement.APIName] = achievement
	}

	// 2) Fetch schema (only direct call available)
	var fullSchema *SchemaGame
	if am.client != nil {
		log.Info("Attempting to fetch achievement schema from Steam API", "app_id", DBDAppID, "client_exists", true)
		schema, err := am.client.GetSchemaForGame(DBDAppID)
		if err != nil {
			log.Error("Failed to get achievement schema, falling back to hardcoded", "error", err, "error_type", fmt.Sprintf("%T", err))
		} else if schema == nil {
			log.Error("Schema is nil from Steam API")
		} else if schema.AvailableGameStats.Achievements == nil {
			log.Error("Schema achievements field is nil")
		} else {
			fullSchema = schema
			log.Info("Successfully fetched achievement schema", "count", len(fullSchema.AvailableGameStats.Achievements))
		}
	} else {
		log.Error("Steam client is nil, cannot fetch schema")
	}

	// If schema missing/empty, fall back to processing all player achievements
	if fullSchema == nil || len(fullSchema.AvailableGameStats.Achievements) == 0 {
		log.Warn("Schema unavailable or empty, processing all player achievements with fallback classification")
		return am.buildAllAchievementMappings(unlockedMap, nil, cacheManager, ctx)
	}

	// 3) Fetch global percentages (try cached first; else direct)
	var globalPercentages map[string]float64
	if cacheManager != nil && am.client != nil {
		if percentages, err := am.client.GetGlobalAchievementPercentagesCached(ctx, cacheManager); err == nil {
			globalPercentages = percentages
			log.Debug("Using cached global achievement percentages", "count", len(globalPercentages))
		}
	}
	
	if globalPercentages == nil && am.client != nil {
		if percentages, err := am.client.FetchGlobalAchievementPercentages(ctx); err == nil {
			globalPercentages = percentages
			log.Debug("Using direct global achievement percentages", "count", len(globalPercentages))
		} else {
			log.Warn("Failed to get global achievement percentages", "error", err)
		}
	}

	// 4) For each schema achievement, build mapping
	var mapped []AchievementMapping
	for _, schemaAch := range fullSchema.AvailableGameStats.Achievements {
		// id := schema.Name
		id := schemaAch.Name
		
		// title := schema.DisplayName (render this, no synthesis)
		title := schemaAch.DisplayName
		
		// desc := schema.Description; if schema.Hidden==1 && !unlocked -> desc=""
		description := schemaAch.Description
		unlocked := false
		var unlockTime int64
		if unlockedAch, exists := unlockedMap[id]; exists {
			unlocked = unlockedAch.Achieved == 1
			if unlockedAch.UnlockTime > 0 {
				unlockTime = int64(unlockedAch.UnlockTime)
			}
		}
		if schemaAch.Hidden == 1 && !unlocked {
			description = ""
		}
		
		// rarity := globals[id] if present
		rarity := float64(0)
		if globalPercentages != nil {
			if percentage, exists := globalPercentages[id]; exists {
				rarity = percentage
			}
		}
		
		// type/character classification
		achType := "general"
		character := ""
		
		if strings.HasPrefix(title, "Adept ") {
			// Use adeptsByAPI to decide killer|survivor
			if roleType, exists := am.adeptsByAPI[id]; exists {
				if roleType == "killer" {
					achType = "adept_killer"
				} else {
					achType = "adept_survivor"
				}
			}
			
			// Extract character with regex (do not add/remove "The" in title)
			if matches := am.adeptRegex.FindStringSubmatch(title); len(matches) == 2 {
				character = strings.ToLower(strings.TrimSpace(matches[1]))
			}
		}
		
		mapping := AchievementMapping{
			ID:          id,
			Name:        title,        // Both reflect schema DisplayName
			DisplayName: title,        // for legacy consumers
			Description: description,
			Icon:        schemaAch.Icon,
			IconGray:    schemaAch.IconGray,
			Hidden:      schemaAch.Hidden == 1,
			Character:   character,
			Type:        achType,
			Unlocked:    unlocked,
			UnlockTime:  unlockTime,
			Rarity:      rarity,
		}
		
		mapped = append(mapped, mapping)
	}

	// 5) Sort by DisplayName, then ID for stability
	sort.Slice(mapped, func(i, j int) bool {
		if mapped[i].DisplayName == mapped[j].DisplayName {
			return mapped[i].ID < mapped[j].ID
		}
		return mapped[i].DisplayName < mapped[j].DisplayName
	})

	// 6) If len(mapped) < 200, log a WARN
	if len(mapped) < 200 {
		log.Warn("Achievement count unexpectedly low - schema likely incomplete",
			"mapped_count", len(mapped),
			"expected_minimum", 200,
			"suggestion", "Check Steam API availability or schema completeness")
	}

	log.Info("Schema-based achievement mapping completed",
		"total_achievements", len(mapped),
		"schema_source", "steam_api")

	return mapped
}

// buildAllAchievementMappings processes all player achievements when schema is unavailable
func (am *AchievementMapper) buildAllAchievementMappings(unlockedMap map[string]SteamAchievement, globalPercentages map[string]float64, _ cache.Cache, _ context.Context) []AchievementMapping {
	var mapped []AchievementMapping
	
	// Process each achievement from player data
	for apiName, steamAch := range unlockedMap {
		unlocked := steamAch.Achieved == 1
		var unlockTime int64
		if steamAch.UnlockTime > 0 {
			unlockTime = int64(steamAch.UnlockTime)
		}
		
		rarity := float64(0)
		if globalPercentages != nil {
			if percentage, exists := globalPercentages[apiName]; exists {
				rarity = percentage
			}
		}
		
		// Use existing mapping logic or fallback
		mapping := am.getAchievementMapping(apiName)
		mapping.Unlocked = unlocked
		mapping.UnlockTime = unlockTime
		mapping.Rarity = rarity
		
		mapped = append(mapped, mapping)
	}
	
	// Sort by DisplayName, then ID for stability
	sort.Slice(mapped, func(i, j int) bool {
		if mapped[i].DisplayName == mapped[j].DisplayName {
			return mapped[i].ID < mapped[j].ID
		}
		return mapped[i].DisplayName < mapped[j].DisplayName
	})
	
	log.Info("All-achievement mapping completed",
		"total_achievements", len(mapped),
		"source", "player_data_with_fallback_classification")
	
	return mapped
}

// GetAchievementSummary returns a summary of achievements by type with enhanced adept tracking
func (am *AchievementMapper) GetAchievementSummary(mapped []AchievementMapping) map[string]interface{} {
	summary := map[string]interface{}{
		"total_achievements":     len(mapped),
		"unlocked_count":         0,
		"general_count":          0,
		"adept_survivor_count":   0,
		"adept_killer_count":     0,
		"adept_survivors":        []string{},
		"adept_killers":          []string{},
		"completion_rate":        0.0,
	}

	var adeptSurvivors, adeptKillers []string

	for _, achievement := range mapped {
		if achievement.Unlocked {
			summary["unlocked_count"] = summary["unlocked_count"].(int) + 1
		}

		switch achievement.Type {
		case "adept_survivor":
			summary["adept_survivor_count"] = summary["adept_survivor_count"].(int) + 1
			if achievement.Character != "" {
				adeptSurvivors = append(adeptSurvivors, achievement.Character)
			}
		case "adept_killer":
			summary["adept_killer_count"] = summary["adept_killer_count"].(int) + 1
			if achievement.Character != "" {
				adeptKillers = append(adeptKillers, achievement.Character)
			}
		case "survivor":
			// Legacy compatibility for old tests
			summary["adept_survivor_count"] = summary["adept_survivor_count"].(int) + 1
			if achievement.Character != "" {
				adeptSurvivors = append(adeptSurvivors, achievement.Character)
			}
		case "killer":
			// Legacy compatibility for old tests  
			summary["adept_killer_count"] = summary["adept_killer_count"].(int) + 1
			if achievement.Character != "" {
				adeptKillers = append(adeptKillers, achievement.Character)
			}
		case "general":
			summary["general_count"] = summary["general_count"].(int) + 1
		default:
			// Treat unknown types as general
			summary["general_count"] = summary["general_count"].(int) + 1
		}
	}

	summary["adept_survivors"] = adeptSurvivors
	summary["adept_killers"] = adeptKillers
	
	if len(mapped) > 0 {
		summary["completion_rate"] = float64(summary["unlocked_count"].(int)) / float64(len(mapped)) * 100
	}

	return summary
}

// GetUnknownAchievements returns list of unmapped achievements for monitoring
func (am *AchievementMapper) GetUnknownAchievements() []*UnknownAchievement {
	am.unknownsMutex.RLock()
	defer am.unknownsMutex.RUnlock()

	unknowns := make([]*UnknownAchievement, 0, len(am.unknownAchievs))
	for _, unknown := range am.unknownAchievs {
		unknowns = append(unknowns, unknown)
	}
	
	return unknowns
}

// ValidateMappingCoverage returns a summary of achievement mapping coverage
func (am *AchievementMapper) ValidateMappingCoverage() map[string]interface{} {
	survivorCount := 0
	killerCount := 0
	
	for _, adept := range AdeptAchievementMapping {
		if adept.Type == "survivor" {
			survivorCount++
		} else if adept.Type == "killer" {
			killerCount++
		}
	}
	
	return map[string]interface{}{
		"survivor_adepts_mapped": survivorCount,
		"killer_adepts_mapped":   killerCount,
		"total_characters":       survivorCount + killerCount,
	}
}

// getAchievementMapping provides fallback achievement mapping for unknown achievements
func (am *AchievementMapper) getAchievementMapping(apiName string) AchievementMapping {
	// Check if it's in our known mapping first
	if adept, exists := AdeptAchievementMapping[apiName]; exists {
		achType := "adept_survivor"
		if adept.Type == "killer" {
			achType = "adept_killer"
		}
		
		return AchievementMapping{
			ID:          apiName,
			Name:        adept.Name,
			DisplayName: adept.Name,
			Description: "Character adept achievement",
			Character:   adept.Name,
			Type:        achType,
			Unlocked:    false,
		}
	}
	
	// Fallback detection for unknown achievements
	achType := "general"
	character := ""
	displayName := apiName
	
	// Simple pattern matching for type detection
	apiLower := strings.ToLower(apiName)
	if strings.Contains(apiLower, "survivor") {
		achType = "survivor"
	} else if strings.Contains(apiLower, "killer") {
		achType = "killer"
	}
	
	// Generate a more readable display name
	displayName = strings.ReplaceAll(apiName, "_", " ")
	displayName = strings.Title(strings.ToLower(displayName))
	
	return AchievementMapping{
		ID:          apiName,
		Name:        displayName,
		DisplayName: displayName,
		Description: "Achievement detected but not in catalog",
		Character:   character,
		Type:        achType,
		Unlocked:    false,
	}
}

// Global mapper instance for caching (lazy initialization)
var (
	globalAchievementMapper *AchievementMapper
	globalMapperOnce        sync.Once
)

func getGlobalMapper() *AchievementMapper {
	globalMapperOnce.Do(func() {
		globalAchievementMapper = NewAchievementMapper()
	})
	return globalAchievementMapper
}

// MapAchievements is a convenience function using the global mapper
func MapAchievements(achievements *PlayerAchievements) []AchievementMapping {
	return getGlobalMapper().MapPlayerAchievements(achievements)
}

// GetMappedAchievements returns mapped achievements with summary and monitoring
func GetMappedAchievements(achievements *PlayerAchievements) map[string]interface{} {
	return GetAchievements(achievements, nil)
}

// GetAchievements returns mapped achievements with schema-based mapping when cache is available
func GetAchievements(achievements *PlayerAchievements, cacheManager cache.Cache) map[string]interface{} {
	mapper := getGlobalMapper()
	mapped := mapper.MapPlayerAchievementsWithCache(achievements, cacheManager)
	summary := mapper.GetAchievementSummary(mapped)
	unknowns := mapper.GetUnknownAchievements()

	log.Info("Achievement mapping completed",
		"total_achievements", len(mapped),
		"unlocked_count", summary["unlocked_count"],
		"survivor_adepts", len(summary["adept_survivors"].([]string)),
		"killer_adepts", len(summary["adept_killers"].([]string)),
		"unknown_achievements", len(unknowns))

	// Log unknown achievements if any found
	if len(unknowns) > 0 {
		log.Warn("Unknown achievements detected - may need mapping updates",
			"unknown_count", len(unknowns),
			"suggestion", "Check Steam API or new game content for updates")
		
		for _, unknown := range unknowns {
			if unknown.Occurrences > 5 { // Only log frequently seen unknowns
				log.Debug("Frequent unknown achievement",
					"api_name", unknown.APIName,
					"first_seen", unknown.FirstSeen.Format(time.RFC3339),
					"occurrences", unknown.Occurrences)
			}
		}
	}

	result := map[string]interface{}{
		"achievements": mapped,
		"summary":      summary,
	}

	// Include unknown achievements in response for monitoring
	if len(unknowns) > 0 {
		result["unknown_achievements"] = unknowns
	}

	return result
}
