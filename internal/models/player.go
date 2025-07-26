package models

type PlayerStats struct {
	SteamID              string  `json:"steam_id"`
	DisplayName          string  `json:"display_name"`
	KillerPips           int     `json:"killer_pips"`
	SurvivorPips         int     `json:"survivor_pips"`
	KilledCampers        int     `json:"killed_campers"`
	SacrificedCampers    int     `json:"sacrificed_campers"`
	GeneratorPct         float64 `json:"generator_pct"`
	HealPct              float64 `json:"heal_pct"`
	EscapesKO            int     `json:"escapes_ko"`
	Escapes              int     `json:"escapes"`
	SkillCheckSuccess    int     `json:"skill_check_success"`
	HookedAndEscape      int     `json:"hooked_and_escape"`
	UnhookOrHeal         int     `json:"unhook_or_heal"`
	UnhookOrHealPostExit int     `json:"unhook_or_heal_post_exit"`
	BloodwebPoints       int     `json:"bloodweb_points"`
	CamperPerfectGames   int     `json:"camper_perfect_games"`
	KillerPerfectGames   int     `json:"killer_perfect_games"`
	UncloakAttacks       int     `json:"uncloak_attacks"`
	EscapeThroughHatch   int     `json:"escape_through_hatch"`
	CamperFullLoadout    int     `json:"camper_full_loadout"`
	KillerFullLoadout    int     `json:"killer_full_loadout"`
	CamperNewItem        int     `json:"camper_new_item"`
}
