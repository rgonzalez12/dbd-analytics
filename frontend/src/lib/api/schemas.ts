import { z } from 'zod';

export const DataSourceInfoSchema = z.object({
	success: z.boolean(),
	source: z.enum(['cache', 'api', 'fallback']),
	error: z.string().optional(),
	fetched_at: z.string()
});

export const PlayerStatsSchema = z.object({
	steam_id: z.string(),
	display_name: z.string(),
	total_matches: z.number(),
	last_updated: z.string(),
	time_played_hours: z.number().optional(),
	bloodweb_points: z.number().optional(),
	killer_pips: z.number().optional(),
	survivor_pips: z.number().optional(),
	killed_campers: z.number().optional(),
	escapes: z.number().optional(),
	generator_pct: z.number().optional()
}).catchall(z.unknown());

export const MappedAchievementSchema = z.object({
	id: z.string(),
	name: z.string(),
	display_name: z.string(),
	description: z.string(),
	character: z.string().optional(),
	type: z.enum(['survivor', 'killer', 'general']),
	unlocked: z.boolean(),
	unlock_time: z.number().optional()
});

export const AchievementSummarySchema = z.object({
	total_achievements: z.number(),
	unlocked_count: z.number(),
	survivor_count: z.number(),
	killer_count: z.number(),
	general_count: z.number(),
	adept_survivors: z.array(z.string()),
	adept_killers: z.array(z.string()),
	completion_rate: z.number()
});

export const AchievementDataSchema = z.object({
	adept_survivors: z.record(z.string(), z.boolean()),
	adept_killers: z.record(z.string(), z.boolean()),
	mapped_achievements: z.array(MappedAchievementSchema).optional(),
	summary: AchievementSummarySchema.optional(),
	last_updated: z.string()
});

export const PlayerStatsWithAchievementsSchema = PlayerStatsSchema.extend({
	achievements: AchievementDataSchema.optional(),
	data_sources: z.object({
		stats: DataSourceInfoSchema,
		achievements: DataSourceInfoSchema
	})
});
