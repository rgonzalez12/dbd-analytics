package schema

// UIAchievement represents a humanized achievement for frontend display
type UIAchievement struct {
	APIName     string    `json:"apiName"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	Hidden      bool      `json:"hidden"`
	Achieved    bool      `json:"achieved"`
	UnlockTime  int64     `json:"unlockTime,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	IconGray    string    `json:"iconGray,omitempty"`
	Unknown     bool      `json:"unknown"`
}

// UIStat represents a humanized statistic for frontend display
type UIStat struct {
	Key         string      `json:"key"`
	DisplayName string      `json:"displayName"`
	Value       interface{} `json:"value"`
	Unknown     bool        `json:"unknown"`
}

// PlayerAchievement represents raw achievement data from Steam API
type PlayerAchievement struct {
	APIName    string `json:"apiname"`
	Achieved   int    `json:"achieved"`
	UnlockTime int64  `json:"unlocktime,omitempty"`
}

// UserStat represents raw stat data from Steam API
type UserStat struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// HumanizeAchievements converts raw achievements to UI-friendly format using schema
func HumanizeAchievements(schema *Schema, rawAchievements []PlayerAchievement, lang string) []UIAchievement {
	if schema == nil {
		// Fallback when schema is unavailable
		return humanizeAchievementsWithoutSchema(rawAchievements)
	}

	achievements := make([]UIAchievement, 0, len(rawAchievements))

	for _, raw := range rawAchievements {
		ui := UIAchievement{
			APIName:    raw.APIName,
			Achieved:   raw.Achieved == 1,
			UnlockTime: raw.UnlockTime,
		}

		// Try to get metadata from schema
		if meta, exists := schema.Achievements[raw.APIName]; exists {
			ui.DisplayName = meta.DisplayName
			ui.Description = meta.Description
			ui.Hidden = meta.Hidden
			ui.Icon = meta.Icon
			ui.IconGray = meta.IconGray
			ui.Unknown = false

			// Handle hidden achievements
			if ui.Hidden && ui.Description == "" {
				ui.Description = "Hidden achievement"
			}
		} else {
			// Unknown achievement - use raw key as display name
			ui.DisplayName = raw.APIName
			ui.Description = ""
			ui.Hidden = false
			ui.Unknown = true
		}

		achievements = append(achievements, ui)
	}

	return achievements
}

// HumanizeStats converts raw stats to UI-friendly format using schema
func HumanizeStats(schema *Schema, rawStats []UserStat, lang string) []UIStat {
	if schema == nil {
		// Fallback when schema is unavailable
		return humanizeStatsWithoutSchema(rawStats)
	}

	stats := make([]UIStat, 0, len(rawStats))

	for _, raw := range rawStats {
		ui := UIStat{
			Key:   raw.Name,
			Value: raw.Value,
		}

		// Try to get display name from schema
		if displayName, exists := schema.Stats[raw.Name]; exists && displayName != "" {
			ui.DisplayName = displayName
			ui.Unknown = false
		} else {
			// Unknown stat - use raw key as display name
			ui.DisplayName = raw.Name
			ui.Unknown = true
		}

		stats = append(stats, ui)
	}

	return stats
}

// humanizeAchievementsWithoutSchema provides fallback behavior when schema is unavailable
func humanizeAchievementsWithoutSchema(rawAchievements []PlayerAchievement) []UIAchievement {
	achievements := make([]UIAchievement, 0, len(rawAchievements))

	for _, raw := range rawAchievements {
		ui := UIAchievement{
			APIName:     raw.APIName,
			DisplayName: raw.APIName, // Use API name as fallback
			Description: "",
			Hidden:      false,
			Achieved:    raw.Achieved == 1,
			UnlockTime:  raw.UnlockTime,
			Unknown:     true, // Mark as unknown since we don't have schema
		}

		achievements = append(achievements, ui)
	}

	return achievements
}

// humanizeStatsWithoutSchema provides fallback behavior when schema is unavailable
func humanizeStatsWithoutSchema(rawStats []UserStat) []UIStat {
	stats := make([]UIStat, 0, len(rawStats))

	for _, raw := range rawStats {
		ui := UIStat{
			Key:         raw.Name,
			DisplayName: raw.Name, // Use raw name as fallback
			Value:       raw.Value,
			Unknown:     true, // Mark as unknown since we don't have schema
		}

		stats = append(stats, ui)
	}

	return stats
}
