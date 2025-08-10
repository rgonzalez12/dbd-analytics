export type PlayerSummary = {
	steam_id: string;
	display_name?: string;
	avatar?: string;
};

export type PlayerStats = {
	steam_id: string;
	display_name: string;
	total_matches: number;
	last_updated: string;
	time_played_hours?: number;
	bloodweb_points?: number;
	killer_pips?: number;
	survivor_pips?: number;
	killed_campers?: number;
	escapes?: number;
	generator_pct?: number;
	[k: string]: unknown;
};

export type ApiError = {
	status: number;
	message: string;
	details?: unknown;
	retryAfter?: number;
};

export type DataSourceInfo = {
	success: boolean;
	source: 'cache' | 'api' | 'fallback';
	error?: string;
	fetched_at: string;
};

// Achievement-related types to match backend models
export type MappedAchievement = {
	id: string;
	name: string;
	display_name: string;
	description: string;
	character?: string;
	type: 'survivor' | 'killer' | 'general';
	unlocked: boolean;
	unlock_time?: number;
};

export type AchievementSummary = {
	total_achievements: number;
	unlocked_count: number;
	survivor_count: number;
	killer_count: number;
	general_count: number;
	adept_survivors: string[];
	adept_killers: string[];
	completion_rate: number;
};

export type AchievementData = {
	// Legacy format for backward compatibility
	adept_survivors: Record<string, boolean>;
	adept_killers: Record<string, boolean>;
	
	// Enhanced achievement data
	mapped_achievements?: MappedAchievement[];
	summary?: AchievementSummary;
	
	last_updated: string;
};

export type DataSourceStatus = {
	stats: DataSourceInfo;
	achievements: DataSourceInfo;
};

export type PlayerStatsWithAchievements = PlayerStats & {
	achievements?: AchievementData;
	data_sources: DataSourceStatus;
};

export type LoadState<T> =
	| { ok: true; data: T }
	| { ok: false; error: ApiError };
