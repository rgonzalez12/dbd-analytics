package models

import "time"

type AchievementData struct {
	AdeptSurvivors map[string]bool `json:"adept_survivors"` // character name -> unlocked status
	AdeptKillers   map[string]bool `json:"adept_killers"`   // character name -> unlocked status

	// Enhanced achievement data with mapping
	MappedAchievements []MappedAchievement `json:"mapped_achievements,omitempty"`
	Summary            AchievementSummary  `json:"summary,omitempty"`

	LastUpdated time.Time `json:"last_updated"`
}

type MappedAchievement struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`         // displayName from schema
	DisplayName string  `json:"display_name"`
	Description string  `json:"description"`
	Icon        string  `json:"icon,omitempty"`
	IconGray    string  `json:"icon_gray,omitempty"`
	Hidden      bool    `json:"hidden,omitempty"`
	Character   string  `json:"character,omitempty"`
	Type        string  `json:"type"` // "survivor", "killer", "general", "adept"
	Unlocked    bool    `json:"unlocked"`
	UnlockTime  int64   `json:"unlock_time,omitempty"`
	Rarity      float64 `json:"rarity,omitempty"` // 0-100 global completion percentage
}

// AchievementSummary provides statistical overview of achievements
type AchievementSummary struct {
	TotalAchievements int      `json:"total_achievements"`
	UnlockedCount     int      `json:"unlocked_count"`
	SurvivorCount     int      `json:"survivor_count"`
	KillerCount       int      `json:"killer_count"`
	GeneralCount      int      `json:"general_count"`
	AdeptSurvivors    []string `json:"adept_survivors"`
	AdeptKillers      []string `json:"adept_killers"`
	CompletionRate    float64  `json:"completion_rate"`
}

// PlayerStatsWithAchievements represents the enhanced response with both stats and achievements
type PlayerStatsWithAchievements struct {
	PlayerStats

	Achievements *AchievementData `json:"achievements,omitempty"`

	// Structured stats data using schema as source of truth
	Stats *StatsData `json:"stats,omitempty"`

	// Data source tracking
	DataSources DataSourceStatus `json:"data_sources"`

	APIProvider   string    `json:"api_provider"`
	SchemaVersion string    `json:"schema_version"`
	CacheHit      bool      `json:"cache_hit"`
	LastUpdated   time.Time `json:"last_updated"`
}

// StatsData represents structured player statistics
type StatsData struct {
	Stats   []interface{} `json:"stats"`   // Will be populated with steam.Stat objects
	Summary interface{}   `json:"summary"` // Will be populated with summary data
}

// DataSourceStatus tracks the success/failure status of different data sources
type DataSourceStatus struct {
	Stats           DataSourceInfo `json:"stats"`
	Achievements    DataSourceInfo `json:"achievements"`
	StructuredStats DataSourceInfo `json:"structured_stats"` // New field for our schema-based stats
}

// DataSourceInfo provides detailed information about data source fetch results
type DataSourceInfo struct {
	Success   bool      `json:"success"`
	Source    string    `json:"source"` // "cache" | "api" | "fallback"
	Error     string    `json:"error,omitempty"`
	FetchedAt time.Time `json:"fetched_at"`
}
