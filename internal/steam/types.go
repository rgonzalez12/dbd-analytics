package steam

import "time"
// Steam API Response Types

type SteamPlayerResponse struct {
	Response SteamResponse `json:"response"`
}

type SteamResponse struct {
	Players []SteamPlayer `json:"players"`
}

type SteamPlayer struct {
	SteamID     string `json:"steamid"`
	PersonaName string `json:"personaname"`
	Avatar      string `json:"avatar"`
	AvatarFull  string `json:"avatarfull"`
}

type SteamStatsResponse struct {
	Playerstats SteamPlayerstats `json:"playerstats"`
}

type SteamPlayerstats struct {
	SteamID  string     `json:"steamID"`
	GameName string     `json:"gameName"`
	Stats    []SteamStat `json:"stats"`
}

type SteamStat struct {
	Name  string `json:"name"`
	Value float64 `json:"value"`
}

type VanityURLResponse struct {
	Response VanityResponse `json:"response"`
}

type VanityResponse struct {
	SteamID string `json:"steamid"`
	Success int    `json:"success"`
}

// Dead by Daylight Player Statistics

type DBDPlayerStats struct {
	SteamID     string `json:"steam_id"`
	DisplayName string `json:"display_name"`
	
	// Killer Statistics
	Killer KillerStats `json:"killer"`
	
	// Survivor Statistics
	Survivor SurvivorStats `json:"survivor"`
	
	// General Game Stats
	General GeneralStats `json:"general"`
}

type KillerStats struct {
	TotalKills        int     `json:"total_kills"`
	SacrificedVictims int     `json:"sacrificed_victims"`
	MoriKills         int     `json:"mori_kills"`
	HooksPerformed    int     `json:"hooks_performed"`
	PerfectGames      int     `json:"perfect_games"`
	FullLoadoutGames  int     `json:"full_loadout_games"`
	KillerPips        int     `json:"killer_pips"`
	UncloakAttacks    int     `json:"uncloak_attacks"`
}

type SurvivorStats struct {
	TotalEscapes         int     `json:"total_escapes"`
	EscapesThroughHatch  int     `json:"escapes_through_hatch"`
	EscapesKnockedOut    int     `json:"escapes_knocked_out"`
	HookedAndEscaped     int     `json:"hooked_and_escaped"`
	GeneratorsCompleted  float64 `json:"generators_completed_pct"`
	HealingCompleted     float64 `json:"healing_completed_pct"`
	SkillChecksHit       int     `json:"skill_checks_hit"`
	UnhooksPerformed     int     `json:"unhooks_performed"`
	HealsPerformed       int     `json:"heals_performed"`
	PostExitActions      int     `json:"post_exit_actions"`
	PerfectGames         int     `json:"perfect_games"`
	FullLoadoutGames     int     `json:"full_loadout_games"`
	NewItemsFound        int     `json:"new_items_found"`
	SurvivorPips         int     `json:"survivor_pips"`
}

type GeneralStats struct {
	BloodwebPoints   int `json:"bloodweb_points"`
	TotalMatches     int `json:"total_matches"`
	TimePlayed       int `json:"time_played_hours"`
	LastUpdated      time.Time `json:"last_updated"`
}
