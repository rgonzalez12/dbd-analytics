package models

import "time"

// AchievementData represents processed achievement information for a player
type AchievementData struct {
	AdeptSurvivors map[string]bool `json:"adept_survivors"` // character name -> unlocked status
	AdeptKillers   map[string]bool `json:"adept_killers"`   // character name -> unlocked status
	LastUpdated    time.Time       `json:"last_updated"`
}

// PlayerStatsWithAchievements represents the enhanced response with both stats and achievements
type PlayerStatsWithAchievements struct {
	PlayerStats
	
	// Achievement data (optional - may be nil if achievements failed to load)
	Achievements *AchievementData `json:"achievements,omitempty"`
	
	// Data source information for debugging and monitoring
	DataSources DataSourceStatus `json:"data_sources"`
}

// DataSourceStatus tracks the success/failure status of different data sources
type DataSourceStatus struct {
	Stats        DataSourceInfo `json:"stats"`
	Achievements DataSourceInfo `json:"achievements"`
}

// DataSourceInfo provides detailed information about data source fetch results
type DataSourceInfo struct {
	Success   bool      `json:"success"`
	Source    string    `json:"source"`    // "cache" | "api" | "fallback"
	Error     string    `json:"error,omitempty"`
	FetchedAt time.Time `json:"fetched_at"`
}
