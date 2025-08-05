// DBD Player Statistics Types
export interface DBDPlayerStats {
	steamId: string;
	displayName: string;
	killer: KillerStats;
	survivor: SurvivorStats;
	general: GeneralStats;
	achievements?: AchievementData;
	source: 'api' | 'cache' | 'mock';
	lastUpdated: string;
}

export interface KillerStats {
	total_kills: number;
	sacrificed_victims: number;
	mori_kills: number;
	hooks_performed: number;
	perfect_games: number;
	full_loadout_games: number;
	killer_pips: number;
	uncloak_attacks: number;
}

export interface SurvivorStats {
	total_escapes: number;
	escapes_through_hatch: number;
	escapes_knocked_out: number;
	hooked_and_escaped: number;
	generators_completed_pct: number;
	healing_completed_pct: number;
	skill_checks_hit: number;
	unhooks_performed: number;
	heals_performed: number;
	post_exit_actions: number;
	perfect_games: number;
	full_loadout_games: number;
	new_items_found: number;
	survivor_pips: number;
}

export interface GeneralStats {
	bloodweb_points: number;
	total_matches: number;
	time_played_hours: number;
	last_updated: string;
}

export interface AchievementData {
	adept_survivors: Record<string, boolean>;
	adept_killers: Record<string, boolean>;
	last_updated: string;
}

// API Response Types
export interface APIResponse<T> {
	data: T;
	success: boolean;
	message?: string;
	error?: string;
}

export interface APIError {
	error: {
		code: string;
		message: string;
		details?: Record<string, any>;
		request_id?: string;
	};
}
