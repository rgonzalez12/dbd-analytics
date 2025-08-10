// place files you want to import through the `$lib` alias in this folder.

export { apiGet, withQuery, ApiError } from './api/client.js';
export type {
	PlayerStats,
	MappedAchievement,
	AchievementSummary,
	AchievementData,
	DataSourceInfo,
	DataSourceStatus,
	PlayerStatsWithAchievements,
	SteamPlayer
} from './api/types.js';

export type {
	DBDPlayerStats,
	KillerStats,
	SurvivorStats,
	GeneralStats,
	AchievementData as LegacyAchievementData,
	APIResponse,
	APIError
} from './types.js';
