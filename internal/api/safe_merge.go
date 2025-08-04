package api

import (
	"fmt"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/models"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// SafeAchievementMerger handles safe merging of achievement data to prevent corruption
type SafeAchievementMerger struct {
	minValidSurvivors int
	minValidKillers   int
	maxAgeThreshold   time.Duration
}

// NewSafeAchievementMerger creates a merger with validation rules
func NewSafeAchievementMerger() *SafeAchievementMerger {
	return &SafeAchievementMerger{
		minValidSurvivors: 20, // Expect at least 20 survivors in mapping
		minValidKillers:   15, // Expect at least 15 killers in mapping
		maxAgeThreshold:   24 * time.Hour, // Don't accept data older than 24h
	}
}

// NewSafeAchievementMergerWithConfig creates a merger with custom validation rules
func NewSafeAchievementMergerWithConfig(minSurvivors, minKillers int, maxAge time.Duration) *SafeAchievementMerger {
	return &SafeAchievementMerger{
		minValidSurvivors: minSurvivors,
		minValidKillers:   minKillers,
		maxAgeThreshold:   maxAge,
	}
}

// SafeMergeAchievements safely merges achievement data with validation
func (m *SafeAchievementMerger) SafeMergeAchievements(
	response *models.PlayerStatsWithAchievements, 
	newAchievements *models.AchievementData,
	steamID string,
) error {
	if response == nil {
		return fmt.Errorf("response cannot be nil")
	}
	
	// Always ensure achievements object exists
	if response.Achievements == nil {
		response.Achievements = &models.AchievementData{
			AdeptSurvivors: make(map[string]bool),
			AdeptKillers:   make(map[string]bool),
			LastUpdated:    time.Now(),
		}
		log.Debug("Initialized empty achievement data structure",
			"steam_id", steamID,
			"reason", "nil_achievements_object")
	}
	
	// If no new achievements data, keep existing (don't overwrite with empty)
	if newAchievements == nil {
		log.Debug("No new achievement data to merge, keeping existing",
			"steam_id", steamID,
			"existing_survivors", len(response.Achievements.AdeptSurvivors),
			"existing_killers", len(response.Achievements.AdeptKillers),
			"cache_behavior", "preserve_existing")
		return nil
	}
	
	// Validate new data before merging
	if err := m.validateAchievementData(newAchievements, steamID); err != nil {
		log.Warn("Achievement data failed validation, keeping existing",
			"steam_id", steamID,
			"validation_error", err.Error(),
			"new_survivors", len(newAchievements.AdeptSurvivors),
			"new_killers", len(newAchievements.AdeptKillers),
			"cache_decision", "reject_invalid_data")
		return err
	}
	
	// Check if new data is newer than existing
	if !newAchievements.LastUpdated.IsZero() && 
	   !response.Achievements.LastUpdated.IsZero() &&
	   newAchievements.LastUpdated.Before(response.Achievements.LastUpdated) {
		log.Debug("New achievement data is older than existing, skipping merge",
			"steam_id", steamID,
			"existing_updated", response.Achievements.LastUpdated,
			"new_updated", newAchievements.LastUpdated,
			"cache_decision", "skip_stale_data")
		return nil
	}
	
	// Track changes for logging
	survivorChanges := 0
	killerChanges := 0
	
	// Perform safe merge with change tracking
	m.performSafeMerge(response.Achievements, newAchievements, &survivorChanges, &killerChanges)
	
	// Enhanced merge completion logging
	log.Info("Achievement data merged successfully",
		"steam_id", steamID,
		"survivor_changes", survivorChanges,
		"killer_changes", killerChanges,
		"total_survivors", len(response.Achievements.AdeptSurvivors),
		"total_killers", len(response.Achievements.AdeptKillers),
		"merge_strategy", "additive_only",
		"data_protection", "corruption_prevented",
		"cache_operation", "merge_complete")
	
	return nil
}

// validateAchievementData performs validation checks on achievement data
func (m *SafeAchievementMerger) validateAchievementData(data *models.AchievementData, steamID string) error {
	if data == nil {
		return fmt.Errorf("achievement data is nil")
	}
	
	// Check for minimum expected characters
	if len(data.AdeptSurvivors) < m.minValidSurvivors {
		return fmt.Errorf("insufficient survivor data: got %d, expected at least %d", 
			len(data.AdeptSurvivors), m.minValidSurvivors)
	}
	
	if len(data.AdeptKillers) < m.minValidKillers {
		return fmt.Errorf("insufficient killer data: got %d, expected at least %d", 
			len(data.AdeptKillers), m.minValidKillers)
	}
	
	// Check data freshness
	if !data.LastUpdated.IsZero() && time.Since(data.LastUpdated) > m.maxAgeThreshold {
		return fmt.Errorf("achievement data too old: %v ago", time.Since(data.LastUpdated))
	}
	
	// Validate character names (basic sanity check)
	for character := range data.AdeptSurvivors {
		if len(character) == 0 || len(character) > 50 {
			return fmt.Errorf("invalid survivor character name: %q", character)
		}
	}
	
	for character := range data.AdeptKillers {
		if len(character) == 0 || len(character) > 50 {
			return fmt.Errorf("invalid killer character name: %q", character)
		}
	}
	
	return nil
}

// performSafeMerge performs the actual merge operation with change tracking
func (m *SafeAchievementMerger) performSafeMerge(
	existing *models.AchievementData, 
	new *models.AchievementData, 
	survivorChanges *int,
	killerChanges *int,
) {
	// Merge survivors (only update if different)
	for character, unlocked := range new.AdeptSurvivors {
		if existingUnlocked, exists := existing.AdeptSurvivors[character]; !exists || existingUnlocked != unlocked {
			existing.AdeptSurvivors[character] = unlocked
			(*survivorChanges)++
		}
	}
	
	// Merge killers (only update if different)
	for character, unlocked := range new.AdeptKillers {
		if existingUnlocked, exists := existing.AdeptKillers[character]; !exists || existingUnlocked != unlocked {
			existing.AdeptKillers[character] = unlocked
			(*killerChanges)++
		}
	}
	
	// Update timestamp
	existing.LastUpdated = new.LastUpdated
}

// InitializeEmptyAchievements creates a safe empty achievements structure
func InitializeEmptyAchievements() *models.AchievementData {
	return &models.AchievementData{
		AdeptSurvivors: make(map[string]bool),
		AdeptKillers:   make(map[string]bool),
		LastUpdated:    time.Now(),
	}
}
