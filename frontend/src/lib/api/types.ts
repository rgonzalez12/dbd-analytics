export type ApiError = { status: number; message: string; details?: unknown; retryAfter?: number };
export type DataSourceInfo = { success: boolean; source: 'cache'|'api'|'fallback'; error?: string; fetched_at: string };

export type AchievementSummary = { total: number; unlocked: number; last_updated?: string };
export type MappedAchievement = { id: string; type: 'survivor'|'killer'; character: string; unlocked: boolean };
export type AchievementData = {
  summary?: AchievementSummary;
  mapped_achievements?: MappedAchievement[];
  adept_survivors?: Record<string, boolean>;
  adept_killers?: Record<string, boolean>;
  last_updated?: string;
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
};

export type PlayerStatsWithAchievements = PlayerStats & {
  achievements?: AchievementData;
  data_sources: { stats: DataSourceInfo; achievements: DataSourceInfo };
};

export type LoadState<T> = { ok: true; data: T } | { ok: false; error: ApiError };
