package steam

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/cache"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

type AchievementMapping struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Character   string `json:"character"`
	Type        string `json:"type"` // "survivor" or "killer"
	Unlocked    bool   `json:"unlocked"`
	UnlockTime  int64  `json:"unlock_time,omitempty"`
}

// UnknownAchievement tracks achievements not in our mapping
type UnknownAchievement struct {
	APIName     string    `json:"api_name"`
	FirstSeen   time.Time `json:"first_seen"`
	Occurrences int       `json:"occurrences"`
}

type AchievementMapper struct {
	config          *AchievementConfig             // Configurable achievement mapping
	mapping         map[string]AchievementMapping
	unknownAchievs  map[string]*UnknownAchievement
	cacheDuration   time.Duration
	mutex           sync.RWMutex
	unknownsMutex   sync.RWMutex
	client          *Client // Steam client for schema-based mapping
}

func NewAchievementMapper() *AchievementMapper {
	config, err := LoadAchievementConfig()
	if err != nil {
		log.Error("Failed to load achievement config, using hardcoded fallback",
			"error", err)
		config = buildHardcodedConfig()
	}

	log.Info("Achievement mapper initialized",
		"survivors_mapped", len(config.Survivors),
		"killers_mapped", len(config.Killers),
		"general_mapped", len(config.General),
		"total_mapped", config.Metadata.TotalCount,
		"config_source", config.Metadata.Source)

	return &AchievementMapper{
		config:         config,
		mapping:        make(map[string]AchievementMapping),
		unknownAchievs: make(map[string]*UnknownAchievement),
		cacheDuration:  24 * time.Hour, // Cache for 24 hours
		client:         NewClient(),    // Create Steam client for schema access
	}
}

func (am *AchievementMapper) MapPlayerAchievements(achievements *PlayerAchievements) []AchievementMapping {
	return am.MapPlayerAchievementsWithCache(achievements, nil)
}

// MapPlayerAchievementsWithCache converts raw achievements using schema-based mapping when cache is available
func (am *AchievementMapper) MapPlayerAchievementsWithCache(achievements *PlayerAchievements, cacheManager cache.Cache) []AchievementMapping {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var mapped []AchievementMapping
	
	// Create a map of unlocked achievements for quick lookup
	unlockedMap := make(map[string]SteamAchievement)
	for _, achievement := range achievements.Achievements {
		unlockedMap[achievement.APIName] = achievement
	}

	// Try to get schema-based adept mapping if cache is available
	var schemaAdeptMap map[string]AdeptEntry
	if cacheManager != nil && am.client != nil {
		ctx := context.Background()
		if adeptMap, err := am.client.GetAdeptMapCached(ctx, cacheManager); err == nil {
			schemaAdeptMap = adeptMap
			log.Info("Using schema-based adept mapping", "adept_count", len(schemaAdeptMap))
		} else {
			log.Warn("Failed to get schema-based adepts, using hardcoded fallback", "error", err)
		}
	}

	// First, add all possible adept achievements (whether unlocked or not)
	// Use schema-based mapping if available, otherwise fall back to hardcoded
	if schemaAdeptMap != nil {
		// Use schema-based adept mapping
		for apiName, entry := range schemaAdeptMap {
			mapping := AchievementMapping{
				ID:          apiName,
				Name:        entry.Character,
				DisplayName: fmt.Sprintf("Adept %s", capitalizeFirst(entry.Character)),
				Description: fmt.Sprintf("Reach Player Level 10 with %s using only their unique perks", capitalizeFirst(entry.Character)),
				Character:   entry.Character,
				Type:        entry.Kind,
				Unlocked:    false, // Default to false
			}
			
			// Check if this achievement is actually unlocked
			if unlockedAchievement, exists := unlockedMap[apiName]; exists {
				mapping.Unlocked = unlockedAchievement.Achieved == 1
				if unlockedAchievement.UnlockTime > 0 {
					mapping.UnlockTime = int64(unlockedAchievement.UnlockTime)
				}
			}
			
			mapped = append(mapped, mapping)
		}
	} else {
		// Fall back to hardcoded adept mapping
		for apiName, adept := range AdeptAchievementMapping {
			mapping := AchievementMapping{
				ID:          apiName,
				Name:        adept.Name,
				DisplayName: fmt.Sprintf("Adept %s", capitalizeFirst(adept.Name)),
				Description: fmt.Sprintf("Reach Player Level 10 with %s using only their unique perks", capitalizeFirst(adept.Name)),
				Character:   adept.Name,
				Type:        adept.Type,
				Unlocked:    false, // Default to false
			}
			
			// Check if this achievement is actually unlocked
			if unlockedAchievement, exists := unlockedMap[apiName]; exists {
				mapping.Unlocked = unlockedAchievement.Achieved == 1
				if unlockedAchievement.UnlockTime > 0 {
					mapping.UnlockTime = int64(unlockedAchievement.UnlockTime)
				}
			}
			
			mapped = append(mapped, mapping)
		}
	}

	// Next, add all possible general achievements (whether unlocked or not)
	// This ensures complete catalog guarantee like we do for adepts
	generalAchievements := am.getAllGeneralAchievements()
	for _, general := range generalAchievements {
		mapping := AchievementMapping{
			ID:          general.APIName,
			Name:        general.Name,
			DisplayName: general.DisplayName,
			Description: general.Description,
			Type:        general.Type,
			Unlocked:    false, // Default to false
		}
		
		// Check if this achievement is actually unlocked
		if unlockedAchievement, exists := unlockedMap[general.APIName]; exists {
			mapping.Unlocked = unlockedAchievement.Achieved == 1
			if unlockedAchievement.UnlockTime > 0 {
				mapping.UnlockTime = int64(unlockedAchievement.UnlockTime)
			}
		}
		
		mapped = append(mapped, mapping)
	}

	// Finally, add any unknown achievements that were unlocked but aren't in our catalogs
	knownAPINames := make(map[string]bool)
	if schemaAdeptMap != nil {
		for apiName := range schemaAdeptMap {
			knownAPINames[apiName] = true
		}
	} else {
		for apiName := range AdeptAchievementMapping {
			knownAPINames[apiName] = true
		}
	}
	for _, general := range generalAchievements {
		knownAPINames[general.APIName] = true
	}

	for _, achievement := range achievements.Achievements {
		// Skip if this is a known achievement (already processed above)
		if knownAPINames[achievement.APIName] {
			continue
		}
		
		mapping := am.getAchievementMapping(achievement.APIName)
		mapping.Unlocked = achievement.Achieved == 1
		if achievement.UnlockTime > 0 {
			mapping.UnlockTime = int64(achievement.UnlockTime)
		}
		mapped = append(mapped, mapping)
	}

	return mapped
}

// getAchievementMapping returns mapping for a specific achievement ID with fallback monitoring
func (am *AchievementMapper) getAchievementMapping(apiName string) AchievementMapping {
	// Check if we have this achievement in our adept mapping
	if adept, exists := AdeptAchievementMapping[apiName]; exists {
		return AchievementMapping{
			ID:          apiName,
			Name:        adept.Name,
			DisplayName: fmt.Sprintf("Adept %s", capitalizeFirst(adept.Name)),
			Description: fmt.Sprintf("Reach Player Level 10 with %s using only their unique perks", capitalizeFirst(adept.Name)),
			Character:   adept.Name,
			Type:        adept.Type,
		}
	}

	// Enhanced mapping for common achievements
	enhancedMappings := map[string]AchievementMapping{
		// General achievements
		"ACH_SAVE_YOURSELF":         {ID: apiName, Name: "self_preservation", DisplayName: "Taking One For The Team", Description: "Protect a Survivor from being hit 25 times", Type: "general"},
		"ACH_PERFECT_KILLER":        {ID: apiName, Name: "perfect_killer", DisplayName: "Perfect Killer", Description: "Get a perfect score as a Killer", Type: "killer"},
		"ACH_PERFECT_SURVIVOR":      {ID: apiName, Name: "perfect_survivor", DisplayName: "Perfect Escape", Description: "Get a perfect score as a Survivor", Type: "survivor"},
		"ACH_ESCAPE_OBSESSION":      {ID: apiName, Name: "obsession_escape", DisplayName: "Escaped!", Description: "Escape as the Obsession", Type: "survivor"},
		"ACH_GENERATOR_SOLO":        {ID: apiName, Name: "generator_solo", DisplayName: "Technician", Description: "Repair a generator in the Killer's Terror Radius", Type: "survivor"},
		"ACH_HEAL_SURVIVOR":         {ID: apiName, Name: "healer", DisplayName: "Medic", Description: "Heal 25 Survivors", Type: "survivor"},
		"ACH_RESCUE_UNHOOK":         {ID: apiName, Name: "rescuer", DisplayName: "Savior", Description: "Rescue 25 Survivors from hooks", Type: "survivor"},
		"ACH_KILL_BY_HAND":          {ID: apiName, Name: "mori_master", DisplayName: "By Your Hand", Description: "Kill 25 Survivors by your hand", Type: "killer"},
		"ACH_BASEMENT_HOOK":         {ID: apiName, Name: "basement_master", DisplayName: "Basement Party", Description: "Hook a Survivor in the basement", Type: "killer"},
		"ACH_HIT_SURVIVORS_EXPOSED": {ID: apiName, Name: "exposed_master", DisplayName: "Exposed", Description: "Hit 25 Survivors suffering from the Exposed Status Effect", Type: "killer"},
	}

	if enhanced, exists := enhancedMappings[apiName]; exists {
		return enhanced
	}

	// Track unknown achievements for monitoring and future updates
	am.trackUnknownAchievement(apiName)

	// Improved fallback for unknown achievements
	return AchievementMapping{
		ID:          apiName,
		Name:        generateFallbackName(apiName),
		DisplayName: formatAchievementName(apiName),
		Description: "Achievement details not yet mapped - may be from new content",
		Type:        inferAchievementType(apiName),
	}
}

// trackUnknownAchievement logs and tracks unknown achievements for monitoring
func (am *AchievementMapper) trackUnknownAchievement(apiName string) {
	am.unknownsMutex.Lock()
	defer am.unknownsMutex.Unlock()

	if unknown, exists := am.unknownAchievs[apiName]; exists {
		unknown.Occurrences++
	} else {
		am.unknownAchievs[apiName] = &UnknownAchievement{
			APIName:     apiName,
			FirstSeen:   time.Now(),
			Occurrences: 1,
		}
		
		// Log new unknown achievement for monitoring
		log.Warn("Unknown achievement encountered - may need mapping update",
			"api_name", apiName,
			"inferred_type", inferAchievementType(apiName),
			"suggested_action", "Check Steam GetSchemaForGame for new content")
	}
}

// GetAchievementSummary returns a summary of achievements by type
func (am *AchievementMapper) GetAchievementSummary(mapped []AchievementMapping) map[string]interface{} {
	summary := map[string]interface{}{
		"total_achievements": len(mapped),
		"unlocked_count":     0,
		"survivor_count":     0,
		"killer_count":       0,
		"general_count":      0,
		"adept_survivors":    []string{},
		"adept_killers":      []string{},
	}

	var adeptSurvivors, adeptKillers []string

	for _, achievement := range mapped {
		if achievement.Unlocked {
			summary["unlocked_count"] = summary["unlocked_count"].(int) + 1
		}

		switch achievement.Type {
		case "survivor":
			summary["survivor_count"] = summary["survivor_count"].(int) + 1
			adeptSurvivors = append(adeptSurvivors, achievement.Character)
		case "killer":
			summary["killer_count"] = summary["killer_count"].(int) + 1
			adeptKillers = append(adeptKillers, achievement.Character)
		default:
			summary["general_count"] = summary["general_count"].(int) + 1
		}
	}

	summary["adept_survivors"] = adeptSurvivors
	summary["adept_killers"] = adeptKillers
	summary["completion_rate"] = float64(summary["unlocked_count"].(int)) / float64(len(mapped)) * 100

	return summary
}

// Helper functions
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func formatAchievementName(apiName string) string {
	// Basic formatting for unknown achievement names
	name := apiName
	if len(name) > 4 && name[:4] == "ACH_" {
		name = name[4:]
	}
	if len(name) > 15 && name[:15] == "NEW_ACHIEVEMENT" {
		name = name[15:]
	}
	return name
}

// generateFallbackName creates a readable name from API name
func generateFallbackName(apiName string) string {
	name := strings.ToLower(apiName)
	
	// Remove common prefixes
	prefixes := []string{"ach_", "new_achievement_", "dlc", "chapter"}
	for _, prefix := range prefixes {
		name = strings.TrimPrefix(name, prefix)
	}
	
	// Clean up common patterns
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.TrimSpace(name)
	
	if name == "" {
		return "unknown_achievement"
	}
	
	return name
}

// inferAchievementType tries to determine if an unknown achievement is survivor/killer/general
func inferAchievementType(apiName string) string {
	apiLower := strings.ToLower(apiName)
	
	// Check for survivor patterns
	survivorPatterns := []string{"survivor", "_survivor_", "escape", "gen", "heal", "unhook", "repair"}
	for _, pattern := range survivorPatterns {
		if strings.Contains(apiLower, pattern) {
			return "survivor"
		}
	}
	
	// Check for killer patterns  
	killerPatterns := []string{"killer", "_killer_", "hook", "sacrifice", "mori", "basement", "hit"}
	for _, pattern := range killerPatterns {
		if strings.Contains(apiLower, pattern) {
			return "killer"
		}
	}
	
	// Check for chapter/DLC patterns that might be character-specific
	if strings.Contains(apiLower, "chapter") || strings.Contains(apiLower, "dlc") {
		// Look for position indicators (survivor achievements often end with _3)
		if strings.HasSuffix(apiLower, "_3") || strings.Contains(apiLower, "survivor") {
			return "survivor"
		}
		if strings.Contains(apiLower, "killer") {
			return "killer"
		}
	}
	
	return "general"
}

// Global mapper instance for caching
var globalAchievementMapper = NewAchievementMapper()

// MapAchievements is a convenience function using the global mapper
func MapAchievements(achievements *PlayerAchievements) []AchievementMapping {
	return globalAchievementMapper.MapPlayerAchievements(achievements)
}

// GetMappedAchievements returns mapped achievements with summary and monitoring
func GetMappedAchievements(achievements *PlayerAchievements) map[string]interface{} {
	return GetMappedAchievementsWithCache(achievements, nil)
}

// GetMappedAchievementsWithCache returns mapped achievements with schema-based mapping when cache is available
func GetMappedAchievementsWithCache(achievements *PlayerAchievements, cacheManager cache.Cache) map[string]interface{} {
	mapped := globalAchievementMapper.MapPlayerAchievementsWithCache(achievements, cacheManager)
	summary := globalAchievementMapper.GetAchievementSummary(mapped)
	unknowns := globalAchievementMapper.GetUnknownAchievements()

	log.Info("Achievement mapping completed",
		"total_achievements", len(mapped),
		"unlocked_count", summary["unlocked_count"],
		"completion_rate", fmt.Sprintf("%.1f%%", summary["completion_rate"]),
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

// getAllGeneralAchievements returns all general achievements from schema/config with complete catalog guarantee
func (am *AchievementMapper) getAllGeneralAchievements() []struct {
	APIName     string
	Name        string
	DisplayName string
	Description string
	Type        string
} {
	var generalAchievements []struct {
		APIName     string
		Name        string
		DisplayName string
		Description string
		Type        string
	}

	// First try schema-based general achievements if available
	if am.config != nil {
		for _, general := range am.config.General {
			generalAchievements = append(generalAchievements, struct {
				APIName     string
				Name        string
				DisplayName string
				Description string
				Type        string
			}{
				APIName:     general.APIName,
				Name:        general.Name,
				DisplayName: general.DisplayName,
				Description: general.Description,
				Type:        "general",
			})
		}
	}

	// Add hardcoded general achievements as fallback/supplement
	hardcodedGeneral := map[string]struct {
		Name        string
		DisplayName string
		Description string
	}{
		"ACH_SAVE_YOURSELF":         {Name: "self_preservation", DisplayName: "Taking One For The Team", Description: "Protect a Survivor from being hit 25 times"},
		"ACH_PERFECT_KILLER":        {Name: "perfect_killer", DisplayName: "Perfect Killer", Description: "Get a perfect score as a Killer"},
		"ACH_PERFECT_SURVIVOR":      {Name: "perfect_survivor", DisplayName: "Perfect Escape", Description: "Get a perfect score as a Survivor"},
		"ACH_ESCAPE_OBSESSION":      {Name: "obsession_escape", DisplayName: "Escaped!", Description: "Escape as the Obsession"},
		"ACH_GENERATOR_SOLO":        {Name: "generator_solo", DisplayName: "Technician", Description: "Repair a generator in the Killer's Terror Radius"},
		"ACH_HEAL_SURVIVOR":         {Name: "healer", DisplayName: "Medic", Description: "Heal 25 Survivors"},
		"ACH_RESCUE_UNHOOK":         {Name: "rescuer", DisplayName: "Savior", Description: "Rescue 25 Survivors from hooks"},
		"ACH_KILL_BY_HAND":          {Name: "mori_master", DisplayName: "By Your Hand", Description: "Kill 25 Survivors by your hand"},
		"ACH_BASEMENT_HOOK":         {Name: "basement_master", DisplayName: "Basement Party", Description: "Hook a Survivor in the basement"},
		"ACH_HIT_SURVIVORS_EXPOSED": {Name: "exposed_master", DisplayName: "Exposed", Description: "Hit 25 Survivors suffering from the Exposed Status Effect"},
	}

	// Track which achievements we already have from schema to avoid duplicates
	existingAPINames := make(map[string]bool)
	for _, general := range generalAchievements {
		existingAPINames[general.APIName] = true
	}

	// Add hardcoded achievements that aren't already in schema
	for apiName, hardcoded := range hardcodedGeneral {
		if !existingAPINames[apiName] {
			generalAchievements = append(generalAchievements, struct {
				APIName     string
				Name        string
				DisplayName string
				Description string
				Type        string
			}{
				APIName:     apiName,
				Name:        hardcoded.Name,
				DisplayName: hardcoded.DisplayName,
				Description: hardcoded.Description,
				Type:        "general",
			})
		}
	}

	return generalAchievements
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

func (am *AchievementMapper) ValidateMappingCoverage() map[string]interface{} {
	survivorCount := 0
	killerCount := 0
	
	for _, character := range AdeptAchievementMapping {
		if character.Type == "survivor" {
			survivorCount++
		} else if character.Type == "killer" {
			killerCount++
		}
	}
	
	validation := map[string]interface{}{
		"survivor_adepts_mapped": survivorCount,
		"killer_adepts_mapped":   killerCount,
		"total_characters":       survivorCount + killerCount,
		"coverage_status":        "complete",
	}

	// DBD has approximately 45+ survivors and 30+ killers as of 2024
	expectedSurvivors := 45
	expectedKillers := 30
	
	if survivorCount < expectedSurvivors {
		validation["coverage_status"] = "partial"
		validation["missing_survivors"] = expectedSurvivors - survivorCount
	}
	
	if killerCount < expectedKillers {
		validation["coverage_status"] = "partial"
		validation["missing_killers"] = expectedKillers - killerCount
	}

	log.Info("Achievement mapping coverage validation",
		"survivors_mapped", survivorCount,
		"killers_mapped", killerCount,
		"total_mapped", survivorCount+killerCount,
		"coverage_status", validation["coverage_status"])

	return validation
}
