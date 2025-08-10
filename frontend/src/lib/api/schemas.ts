import { z } from 'zod';

export const DataSourceInfoSchema = z.object({
	success: z.boolean(),
	source: z.enum(['cache', 'api', 'fallback']),
	error: z.string().optional(),
	fetched_at: z.string()
});

export const AchievementSummarySchema = z.object({
	total: z.number(),
	unlocked: z.number(),
	last_updated: z.string().optional()
});

export const MappedAchievementSchema = z.object({
	id: z.string(),
	type: z.enum(['survivor', 'killer']),
	character: z.string(),
	unlocked: z.boolean()
});

export const AchievementDataSchema = z.object({
	summary: AchievementSummarySchema.optional(),
	mapped_achievements: z.array(MappedAchievementSchema).optional(),
	adept_survivors: z.record(z.string(), z.boolean()).optional(),
	adept_killers: z.record(z.string(), z.boolean()).optional(),
	last_updated: z.string().optional()
});

export const PlayerStatsSchema = z.object({
	steam_id: z.string(),
	display_name: z.string(),
	total_matches: z.number(),
	last_updated: z.string(),
	time_played_hours: z.number().optional(),
	bloodweb_points: z.number().optional(),
	killer_pips: z.number().optional(),
	survivor_pips: z.number().optional()
});

export const PlayerStatsWithAchievementsSchema = PlayerStatsSchema.extend({
	achievements: AchievementDataSchema.optional(),
	data_sources: z.object({
		stats: DataSourceInfoSchema,
		achievements: DataSourceInfoSchema
	})
});
