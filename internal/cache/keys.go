package cache

// Cache key prefixes for different data types
const (
	// Player-specific cache keys
	PlayerStatsPrefix        = "player_stats"
	PlayerSummaryPrefix      = "player_summary"
	PlayerAchievementsPrefix = "player_achievements"
	PlayerCombinedPrefix     = "player_combined"

	// Steam API cache keys
	SteamAPIPrefix = "steam_api"

	// Achievement system cache keys
	AdeptMapPrefix = "adept_map_v1" // bump version if format changes
)
