package models

import (
	"time"
	steamSchema "github.com/rgonzalez12/dbd-analytics/internal/steam/schema"
)

// SchemaPlayerSummary represents a schema-driven player summary response
type SchemaPlayerSummary struct {
	PlayerID       string         `json:"playerId"`
	DisplayName    string         `json:"displayName,omitempty"`
	SurvivorGrade  *steamSchema.Grade  `json:"survivorGrade,omitempty"`
	KillerGrade    *steamSchema.Grade  `json:"killerGrade,omitempty"`
	Stats          []steamSchema.UIStat      `json:"stats"`
	Achievements   []steamSchema.UIAchievement `json:"achievements"`
	LastUpdated    time.Time      `json:"lastUpdated"`
	DataSources    DataSources    `json:"dataSources"`
}

// DataSources tracks where data was fetched from for debugging
type DataSources struct {
	Schema       DataSource `json:"schema"`
	Stats        DataSource `json:"stats"`
	Achievements DataSource `json:"achievements"`
}

// DataSource represents the origin of a piece of data
type DataSource struct {
	Success   bool      `json:"success"`
	Source    string    `json:"source"` // "cache", "api", "fallback"
	FetchedAt time.Time `json:"fetched_at"`
	Error     string    `json:"error,omitempty"`
}
