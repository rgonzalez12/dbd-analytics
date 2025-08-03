package steam

import "time"

// statMapping maps Steam API stat keys to human-readable field names
// This mapping can be expanded as new Dead by Daylight statistics are discovered
var statMapping = map[string]string{
	// Killer Statistics
	"DBD_KilledCampers":         "killer.total_kills",
	"DBD_SacrificedCampers":     "killer.sacrificed_victims",
	"DBD_MoriKills":             "killer.mori_kills",
	"DBD_HooksPerformed":        "killer.hooks_performed",
	"DBD_KillerPerfectGames":    "killer.perfect_games",
	"DBD_KillerFullLoadout":     "killer.full_loadout_games",
	"DBD_KillerPips":            "killer.killer_pips",
	"DBD_UncloakAttacks":        "killer.uncloak_attacks",

	// Survivor Statistics
	"DBD_Escapes":               "survivor.total_escapes",
	"DBD_EscapeThroughHatch":    "survivor.escapes_through_hatch",
	"DBD_EscapesKO":             "survivor.escapes_knocked_out",
	"DBD_HookedAndEscape":       "survivor.hooked_and_escaped",
	"DBD_GeneratorPct":          "survivor.generators_completed_pct",
	"DBD_HealPct":               "survivor.healing_completed_pct",
	"DBD_SkillCheckSuccess":     "survivor.skill_checks_hit",
	"DBD_UnhookOrHeal":          "survivor.unhooks_performed",
	"DBD_HealsPerformed":        "survivor.heals_performed",
	"DBD_UnhookOrHealPostExit":  "survivor.post_exit_actions",
	"DBD_CamperPerfectGames":    "survivor.perfect_games",
	"DBD_CamperFullLoadout":     "survivor.full_loadout_games",
	"DBD_CamperNewItem":         "survivor.new_items_found",
	"DBD_SurvivorPips":          "survivor.survivor_pips",

	// General Statistics
	"DBD_BloodwebPoints":        "general.bloodweb_points",
	"DBD_TotalMatches":          "general.total_matches",
	"DBD_TimePlayed":            "general.time_played_hours",
	"DBD_LastUpdated":           "general.last_updated",
}

// MapSteamStats converts raw Steam API statistics into organized Dead by Daylight player data
func MapSteamStats(raw []SteamStat, steamID, displayName string) DBDPlayerStats {
	// Initialize player stats structure with default values and basic information
	stats := DBDPlayerStats{
		SteamID:     steamID,
		DisplayName: displayName,
		Killer:      KillerStats{},
		Survivor:    SurvivorStats{},
		General:     GeneralStats{},
	}

	// Create a map for fast lookup of raw stats
	rawStatMap := make(map[string]int)
	for _, stat := range raw {
		rawStatMap[stat.Name] = int(stat.Value) // Convert float64 to int
	}

	// Map each known stat to the appropriate field
	for steamKey, fieldPath := range statMapping {
		value, exists := rawStatMap[steamKey]
		if !exists {
			continue // Skip missing stats, use default zero values
		}

		// Set the value in the appropriate struct field
		setStatValue(&stats, fieldPath, value)
	}

	return stats
}

// setStatValue sets a value in the DBDPlayerStats struct based on the field path
func setStatValue(stats *DBDPlayerStats, fieldPath string, value int) {
	switch fieldPath {
	// Killer stats
	case "killer.total_kills":
		stats.Killer.TotalKills = value
	case "killer.sacrificed_victims":
		stats.Killer.SacrificedVictims = value
	case "killer.mori_kills":
		stats.Killer.MoriKills = value
	case "killer.hooks_performed":
		stats.Killer.HooksPerformed = value
	case "killer.perfect_games":
		stats.Killer.PerfectGames = value
	case "killer.full_loadout_games":
		stats.Killer.FullLoadoutGames = value
	case "killer.killer_pips":
		stats.Killer.KillerPips = value
	case "killer.uncloak_attacks":
		stats.Killer.UncloakAttacks = value

	// Survivor stats
	case "survivor.total_escapes":
		stats.Survivor.TotalEscapes = value
	case "survivor.escapes_through_hatch":
		stats.Survivor.EscapesThroughHatch = value
	case "survivor.escapes_knocked_out":
		stats.Survivor.EscapesKnockedOut = value
	case "survivor.hooked_and_escaped":
		stats.Survivor.HookedAndEscaped = value
	case "survivor.generators_completed_pct":
		stats.Survivor.GeneratorsCompleted = float64(value)
	case "survivor.healing_completed_pct":
		stats.Survivor.HealingCompleted = float64(value)
	case "survivor.skill_checks_hit":
		stats.Survivor.SkillChecksHit = value
	case "survivor.unhooks_performed":
		stats.Survivor.UnhooksPerformed = value
	case "survivor.heals_performed":
		stats.Survivor.HealsPerformed = value
	case "survivor.post_exit_actions":
		stats.Survivor.PostExitActions = value
	case "survivor.perfect_games":
		stats.Survivor.PerfectGames = value
	case "survivor.full_loadout_games":
		stats.Survivor.FullLoadoutGames = value
	case "survivor.new_items_found":
		stats.Survivor.NewItemsFound = value
	case "survivor.survivor_pips":
		stats.Survivor.SurvivorPips = value

	// General stats
	case "general.bloodweb_points":
		stats.General.BloodwebPoints = value
	case "general.total_matches":
		stats.General.TotalMatches = value
	case "general.time_played_hours":
		stats.General.TimePlayed = value
	case "general.last_updated":
		// Convert Unix timestamp to time.Time
		stats.General.LastUpdated = time.Unix(int64(value), 0)
	}
}

// GetMappedStatNames returns all known Steam stat keys for debugging/validation
func GetMappedStatNames() []string {
	keys := make([]string, 0, len(statMapping))
	for key := range statMapping {
		keys = append(keys, key)
	}
	return keys
}

// AddStatMapping allows dynamic addition of new stat mappings
func AddStatMapping(steamKey, fieldPath string) {
	statMapping[steamKey] = fieldPath
}
