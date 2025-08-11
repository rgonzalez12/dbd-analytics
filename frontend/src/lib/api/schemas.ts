import { z } from 'zod';

export const DataSourceInfoSchema = z.object({
  success: z.boolean(),
  source: z.enum(['cache','api','fallback']),
  error: z.string().optional(),
  fetched_at: z.string().optional()
}).passthrough();

const MappedAchievementSchema = z.object({
  id: z.string(),
  type: z.enum(['survivor','killer']),
  character: z.string(),
  unlocked: z.boolean()
}).passthrough();

const AchievementDataSchema = z.object({
  summary: z.object({
    total_achievements: z.coerce.number().int().nonnegative().optional(),
    unlocked_count: z.coerce.number().int().nonnegative().optional(),
    survivor_count: z.coerce.number().int().nonnegative().optional(),
    killer_count: z.coerce.number().int().nonnegative().optional(),
    general_count: z.coerce.number().int().nonnegative().optional(),
    completion_rate: z.coerce.number().nonnegative().optional(),
    adept_survivors: z.array(z.unknown()).optional(),
    adept_killers: z.array(z.unknown()).optional(),
    last_updated: z.string().optional()
  }).partial().optional(),
  mapped_achievements: z.array(z.unknown()).optional(), // Make this more permissive
  adept_survivors: z.record(z.string(), z.boolean()).optional(),
  adept_killers: z.record(z.string(), z.boolean()).optional(),
  last_updated: z.string().optional()
}).passthrough();

export const ApiPlayerStatsSchema = z.object({
  steam_id: z.string(),
  display_name: z.string(),
  total_matches: z.coerce.number().int().nonnegative().nullable().optional(),
  last_updated: z.string().nullable().optional(),
  killer_pips: z.coerce.number().int().nonnegative().nullable().optional(),
  survivor_pips: z.coerce.number().int().nonnegative().nullable().optional(),
  killed_campers: z.coerce.number().int().nonnegative().nullable().optional(),
  sacrificed_campers: z.coerce.number().int().nonnegative().nullable().optional(),
  mori_kills: z.coerce.number().int().nonnegative().nullable().optional(),
  hooks_performed: z.coerce.number().int().nonnegative().nullable().optional(),
  uncloak_attacks: z.coerce.number().int().nonnegative().nullable().optional(),
  generator_pct: z.coerce.number().nonnegative().nullable().optional(),
  heal_pct: z.coerce.number().nonnegative().nullable().optional(),
  escapes_ko: z.coerce.number().int().nonnegative().nullable().optional(),
  escapes: z.coerce.number().int().nonnegative().nullable().optional(),
  skill_check_success: z.coerce.number().int().nonnegative().nullable().optional(),
  hooked_and_escape: z.coerce.number().int().nonnegative().nullable().optional(),
  unhook_or_heal: z.coerce.number().int().nonnegative().nullable().optional(),
  heals_performed: z.coerce.number().int().nonnegative().nullable().optional(),
  unhook_or_heal_post_exit: z.coerce.number().int().nonnegative().nullable().optional(),
  post_exit_actions: z.coerce.number().int().nonnegative().nullable().optional(),
  escape_through_hatch: z.coerce.number().int().nonnegative().nullable().optional(),
  bloodweb_points: z.coerce.number().int().nonnegative().nullable().optional(),
  camper_perfect_games: z.coerce.number().int().nonnegative().nullable().optional(),
  killer_perfect_games: z.coerce.number().int().nonnegative().nullable().optional(),
  camper_full_loadout: z.coerce.number().int().nonnegative().nullable().optional(),
  killer_full_loadout: z.coerce.number().int().nonnegative().nullable().optional(),
  camper_new_item: z.coerce.number().int().nonnegative().nullable().optional(),
  time_played_hours: z.coerce.number().int().nonnegative().nullable().optional(),
  achievements: AchievementDataSchema.optional(),
  data_sources: z.object({
    stats: DataSourceInfoSchema.optional(),
    achievements: DataSourceInfoSchema.optional()
  }).partial().optional()
}).passthrough();
