package models

import "time"

type PlayerStats struct {
	// Core player identification
	SteamID     string `json:"steam_id" validate:"required"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=32"`
	
	// Progression metrics
	KillerPips   int `json:"killer_pips" validate:"min=0"`
	SurvivorPips int `json:"survivor_pips" validate:"min=0"`
	
	// Killer statistics
	KilledCampers     int `json:"killed_campers" validate:"min=0"`
	SacrificedCampers int `json:"sacrificed_campers" validate:"min=0"`
	MoriKills         int `json:"mori_kills" validate:"min=0"`          
	HooksPerformed    int `json:"hooks_performed" validate:"min=0"`     
	UncloakAttacks    int `json:"uncloak_attacks" validate:"min=0"`
	
	// Survivor statistics  
	GeneratorPct         float64 `json:"generator_pct" validate:"min=0,max=100"`
	HealPct              float64 `json:"heal_pct" validate:"min=0,max=100"`
	EscapesKO            int     `json:"escapes_ko" validate:"min=0"`
	Escapes              int     `json:"escapes" validate:"min=0"`
	SkillCheckSuccess    int     `json:"skill_check_success" validate:"min=0"`
	HookedAndEscape      int     `json:"hooked_and_escape" validate:"min=0"`
	UnhookOrHeal         int     `json:"unhook_or_heal" validate:"min=0"`
	HealsPerformed       int     `json:"heals_performed" validate:"min=0"`      
	UnhookOrHealPostExit int     `json:"unhook_or_heal_post_exit" validate:"min=0"`
	PostExitActions      int     `json:"post_exit_actions" validate:"min=0"`    
	EscapeThroughHatch   int     `json:"escape_through_hatch" validate:"min=0"`
	
	// Game progression
	BloodwebPoints int `json:"bloodweb_points" validate:"min=0"`
	
	// Achievement counters
	CamperPerfectGames int `json:"camper_perfect_games" validate:"min=0"`
	KillerPerfectGames int `json:"killer_perfect_games" validate:"min=0"`
	
	// Equipment tracking
	CamperFullLoadout int `json:"camper_full_loadout" validate:"min=0"`
	KillerFullLoadout int `json:"killer_full_loadout" validate:"min=0"`
	CamperNewItem     int `json:"camper_new_item" validate:"min=0"`
	
	// General game statistics
	TotalMatches int `json:"total_matches" validate:"min=0"`           
	TimePlayed   int `json:"time_played_hours" validate:"min=0"`       
	
	// Metadata
	LastUpdated time.Time `json:"last_updated"`                        // When stats were last updated
}
