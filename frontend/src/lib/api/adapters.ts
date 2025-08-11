import type { ApiError, ApiPlayerStats, Player } from './types';
import { ApiPlayerStatsSchema } from './schemas';

function safeNumber(value: number | string | null | undefined, defaultValue: number = 0): number {
  if (value === null || value === undefined) return defaultValue;
  if (typeof value === 'number') return value;
  const parsed = Number(value);
  return isNaN(parsed) ? defaultValue : parsed;
}

export function toDomainPlayer(raw: unknown): Player {
  const result = ApiPlayerStatsSchema.safeParse(raw);
  
  if (!result.success) {
    const error: ApiError = {
      status: 502,
      message: 'Invalid payload',
      details: result.error
    };
    throw error;
  }

  const data = result.data;

  return {
    id: data.steam_id,
    name: data.display_name,
    matches: safeNumber(data.total_matches),
    lastUpdated: data.last_updated || null,
    stats: {
      killerPips: safeNumber(data.killer_pips),
      survivorPips: safeNumber(data.survivor_pips),
      killedCampers: safeNumber(data.killed_campers),
      sacrificedCampers: safeNumber(data.sacrificed_campers),
      moriKills: safeNumber(data.mori_kills),
      hooksPerformed: safeNumber(data.hooks_performed),
      uncloakAttacks: safeNumber(data.uncloak_attacks),
      generatorPct: safeNumber(data.generator_pct),
      healPct: safeNumber(data.heal_pct),
      escapesKo: safeNumber(data.escapes_ko),
      escapes: safeNumber(data.escapes),
      skillCheckSuccess: safeNumber(data.skill_check_success),
      hookedAndEscape: safeNumber(data.hooked_and_escape),
      unhookOrHeal: safeNumber(data.unhook_or_heal),
      healsPerformed: safeNumber(data.heals_performed),
      unhookOrHealPostExit: safeNumber(data.unhook_or_heal_post_exit),
      postExitActions: safeNumber(data.post_exit_actions),
      escapeThroughHatch: safeNumber(data.escape_through_hatch),
      bloodwebPoints: safeNumber(data.bloodweb_points),
      camperPerfectGames: safeNumber(data.camper_perfect_games),
      killerPerfectGames: safeNumber(data.killer_perfect_games),
      camperFullLoadout: safeNumber(data.camper_full_loadout),
      killerFullLoadout: safeNumber(data.killer_full_loadout),
      camperNewItem: safeNumber(data.camper_new_item),
      timePlayedHours: safeNumber(data.time_played_hours)
    },
    achievements: {
      total: safeNumber(data.achievements?.summary?.total),
      unlocked: safeNumber(data.achievements?.summary?.unlocked),
      mapped: data.achievements?.mapped_achievements || [],
      adepts: {
        survivors: data.achievements?.adept_survivors || {},
        killers: data.achievements?.adept_killers || {}
      }
    },
    sources: {
      stats: data.data_sources?.stats as any,
      achievements: data.data_sources?.achievements as any
    }
  };
}
