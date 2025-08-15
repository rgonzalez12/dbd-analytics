package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// AchievementConfig represents the configurable achievement mapping
type AchievementConfig struct {
	Survivors []AchievementEntry `json:"survivors"`
	Killers   []AchievementEntry `json:"killers"`
	General   []AchievementEntry `json:"general"`
	Metadata  ConfigMetadata     `json:"metadata"`
}

// AchievementEntry represents a single achievement mapping
type AchievementEntry struct {
	APIName     string `json:"api_name"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Category    string `json:"category,omitempty"`
}

// ConfigMetadata holds information about the achievement config
type ConfigMetadata struct {
	Version     string `json:"version"`
	LastUpdated string `json:"last_updated"`
	Source      string `json:"source"`
	TotalCount  int    `json:"total_count"`
}

func LoadAchievementConfig() (*AchievementConfig, error) {
	configPath := os.Getenv("ACHIEVEMENT_CONFIG_PATH")
	if configPath == "" {
		configPath = "config/achievements.json"
	}

	if fileExists(configPath) {
		config, err := loadFromFile(configPath)
		if err == nil {
			return config, nil
		}
		// Log warning but continue with fallback
		fmt.Printf("Warning: Failed to load achievement config from %s: %v\n", configPath, err)
	}

	// Fallback to hardcoded mapping
	return buildHardcodedConfig(), nil
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// loadFromFile loads achievement config from JSON file
func loadFromFile(configPath string) (*AchievementConfig, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AchievementConfig
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// validateConfig validates the loaded achievement config
func validateConfig(config *AchievementConfig) error {
	if len(config.Survivors) == 0 && len(config.Killers) == 0 {
		return fmt.Errorf("config must contain at least some achievements")
	}

	// Check for duplicate API names
	seen := make(map[string]bool)
	allEntries := append(config.Survivors, config.Killers...)
	allEntries = append(allEntries, config.General...)

	for _, entry := range allEntries {
		if entry.APIName == "" {
			return fmt.Errorf("achievement entry missing api_name")
		}
		if seen[entry.APIName] {
			return fmt.Errorf("duplicate api_name: %s", entry.APIName)
		}
		seen[entry.APIName] = true
	}

	return nil
}

// buildHardcodedConfig builds the configuration from existing hardcoded mapping
func buildHardcodedConfig() *AchievementConfig {
	config := &AchievementConfig{
		Survivors: []AchievementEntry{},
		Killers:   []AchievementEntry{},
		General:   []AchievementEntry{},
		Metadata: ConfigMetadata{
			Version:     "1.0.0",
			LastUpdated: "2025-08-04",
			Source:      "hardcoded_fallback",
			TotalCount:  0,
		},
	}

	// Convert existing AdeptAchievementMapping to new format
	for apiName, character := range AdeptAchievementMapping {
		entry := AchievementEntry{
			APIName:     apiName,
			Name:        character.Name,
			Type:        character.Type,
			DisplayName: formatDisplayName(character.Name),
			Category:    "adept",
		}

		switch character.Type {
		case "survivor":
			config.Survivors = append(config.Survivors, entry)
		case "killer":
			config.Killers = append(config.Killers, entry)
		}
	}

	config.Metadata.TotalCount = len(config.Survivors) + len(config.Killers) + len(config.General)
	return config
}

// formatDisplayName creates a readable display name from internal name
func formatDisplayName(name string) string {
	// Handle special cases
	replacements := map[string]string{
		"yun-jin":       "Yun-Jin Lee",
		"dark-lord":     "The Dark Lord",
		"skull-merchant": "The Skull Merchant",
	}

	if display, exists := replacements[name]; exists {
		return display
	}

	// Default formatting: capitalize first letter
	if len(name) > 0 {
		return strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

// GetAchievementByAPIName retrieves achievement info by API name
func (config *AchievementConfig) GetAchievementByAPIName(apiName string) (*AchievementEntry, bool) {
	allEntries := append(config.Survivors, config.Killers...)
	allEntries = append(allEntries, config.General...)

	for _, entry := range allEntries {
		if entry.APIName == apiName {
			return &entry, true
		}
	}
	return nil, false
}

// GetAchievementsByType returns all achievements of a specific type
func (config *AchievementConfig) GetAchievementsByType(achievementType string) []AchievementEntry {
	switch achievementType {
	case "survivor":
		return config.Survivors
	case "killer":
		return config.Killers
	case "general":
		return config.General
	default:
		return []AchievementEntry{}
	}
}

// SaveToFile saves the current config to a JSON file
func (config *AchievementConfig) SaveToFile(configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Update metadata
	config.Metadata.TotalCount = len(config.Survivors) + len(config.Killers) + len(config.General)

	bytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
