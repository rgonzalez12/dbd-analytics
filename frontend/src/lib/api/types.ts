export interface PlayerStats {
	steam_id: string;
	display_name: string;
	killer_pips: number;
	survivor_pips: number;
	killed_campers: number;
	sacrificed_campers: number;
	mori_kills: number;
	hooks_performed: number;
	uncloak_attacks: number;
	generator_pct: number;
	heal_pct: number;
	escapes_ko: number;
	escapes: number;
	skill_check_success: number;
	hooked_and_escape: number;
	unhook_or_heal: number;
	heals_performed: number;
	unhook_or_heal_post_exit: number;
	post_exit_actions: number;
	escape_through_hatch: number;
	bloodweb_points: number;
	camper_perfect_games: number;
	killer_perfect_games: number;
	camper_full_loadout: number;
	killer_full_loadout: number;
	camper_new_item: number;
	total_matches: number;
	time_played_hours: number;
	last_updated: string;
}

export interface MappedAchievement {
	id: string;
	name: string;
	display_name: string;
	description: string;
	character?: string;
	type: string;
	unlocked: boolean;
	unlock_time?: number;
}

export interface AchievementSummary {
	total_achievements: number;
	unlocked_count: number;
	survivor_count: number;
	killer_count: number;
	general_count: number;
	adept_survivors: string[];
	adept_killers: string[];
	completion_rate: number;
}

export interface AchievementData {
	adept_survivors: Record<string, boolean>;
	adept_killers: Record<string, boolean>;
	mapped_achievements?: MappedAchievement[];
	summary?: AchievementSummary;
	last_updated: string;
}

export interface DataSourceInfo {
	success: boolean;
	source: string;
	error?: string;
	fetched_at: string;
}

export interface DataSourceStatus {
	stats: DataSourceInfo;
	achievements: DataSourceInfo;
}

export interface PlayerStatsWithAchievements {
	steam_id: string;
	display_name: string;
	killer_pips: number;
	survivor_pips: number;
	killed_campers: number;
	sacrificed_campers: number;
	mori_kills: number;
	hooks_performed: number;
	uncloak_attacks: number;
	generator_pct: number;
	heal_pct: number;
	escapes_ko: number;
	escapes: number;
	skill_check_success: number;
	hooked_and_escape: number;
	unhook_or_heal: number;
	heals_performed: number;
	unhook_or_heal_post_exit: number;
	post_exit_actions: number;
	escape_through_hatch: number;
	bloodweb_points: number;
	camper_perfect_games: number;
	killer_perfect_games: number;
	camper_full_loadout: number;
	killer_full_loadout: number;
	camper_new_item: number;
	total_matches: number;
	time_played_hours: number;
	last_updated: string;
	achievements?: AchievementData;
	data_sources: DataSourceStatus;
}

export interface SteamPlayer {
	steamid: string;
	personaname: string;
	avatar: string;
	avatarfull: string;
}
