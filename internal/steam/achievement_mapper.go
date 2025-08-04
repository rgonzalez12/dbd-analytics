package steam

import (
	"fmt"
	"sync"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// AchievementMapping represents a mapped achievement with human-readable information
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

// AchievementMapper handles achievement mapping with caching
type AchievementMapper struct {
	mapping       map[string]AchievementMapping
	lastFetched   time.Time
	cacheDuration time.Duration
	mutex         sync.RWMutex
}

// NewAchievementMapper creates a new achievement mapper with caching
func NewAchievementMapper() *AchievementMapper {
	return &AchievementMapper{
		mapping:       make(map[string]AchievementMapping),
		cacheDuration: 24 * time.Hour, // Cache for 24 hours
	}
}

// MapPlayerAchievements converts raw achievements to human-readable format
func (am *AchievementMapper) MapPlayerAchievements(achievements *PlayerAchievements) []AchievementMapping {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var mapped []AchievementMapping

	for _, achievement := range achievements.Achievements {
		mapping := am.getAchievementMapping(achievement.APIName)
		mapping.Unlocked = achievement.Achieved == 1
		if achievement.UnlockTime > 0 {
			mapping.UnlockTime = int64(achievement.UnlockTime)
		}
		mapped = append(mapped, mapping)
	}

	return mapped
}

// getAchievementMapping returns mapping for a specific achievement ID
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

	// Default mapping for unknown achievements
	return AchievementMapping{
		ID:          apiName,
		Name:        apiName,
		DisplayName: formatAchievementName(apiName),
		Description: "Achievement description not available",
		Type:        "general",
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
			if achievement.Unlocked {
				adeptSurvivors = append(adeptSurvivors, achievement.Character)
			}
		case "killer":
			summary["killer_count"] = summary["killer_count"].(int) + 1
			if achievement.Unlocked {
				adeptKillers = append(adeptKillers, achievement.Character)
			}
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

// Global mapper instance for caching
var globalAchievementMapper = NewAchievementMapper()

// MapAchievements is a convenience function using the global mapper
func MapAchievements(achievements *PlayerAchievements) []AchievementMapping {
	return globalAchievementMapper.MapPlayerAchievements(achievements)
}

// GetMappedAchievements returns mapped achievements with summary
func GetMappedAchievements(achievements *PlayerAchievements) map[string]interface{} {
	mapped := MapAchievements(achievements)
	summary := globalAchievementMapper.GetAchievementSummary(mapped)

	log.Info("Achievement mapping completed",
		"total_achievements", len(mapped),
		"unlocked_count", summary["unlocked_count"],
		"completion_rate", fmt.Sprintf("%.1f%%", summary["completion_rate"]),
		"survivor_adepts", len(summary["adept_survivors"].([]string)),
		"killer_adepts", len(summary["adept_killers"].([]string)))

	return map[string]interface{}{
		"achievements": mapped,
		"summary":      summary,
	}
}
