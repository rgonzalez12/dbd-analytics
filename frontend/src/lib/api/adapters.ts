import type { Player } from '$lib/api/types';
import type { PlayerBundle, DbdAdept, DbdAchievement, DbdStats } from '$lib/types';
import type { ApiPlayerStats } from '$lib/api/types';

function toNum(v: unknown, d = 0): number {
	if (typeof v === 'number') return isNaN(v) ? d : v;
	if (typeof v === 'string') {
		const parsed = parseFloat(v);
		return isNaN(parsed) ? d : parsed;
	}
	return d;
}

export function toDomainPlayer(raw: ApiPlayerStats): Player {
	const mapped = raw.achievements?.mapped_achievements?.map(achievement => ({
		id: achievement.id,
		type: achievement.type,
		character: achievement.character,
		unlocked: achievement.unlocked
	})) ?? [];

	const totalFromSummary = toNum(raw.achievements?.summary?.total);
	const unlockedFromSummary = toNum(raw.achievements?.summary?.unlocked);

	const total = totalFromSummary || mapped.length;
	const unlocked = unlockedFromSummary || mapped.filter(a => a.unlocked).length;

	return {
		id: raw.steam_id,
		name: raw.display_name,
		matches: toNum(raw.total_matches),
		lastUpdated: raw.last_updated ?? null,
		stats: {
			killerPips: toNum(raw.killer_pips),
			survivorPips: toNum(raw.survivor_pips),
			killedCampers: toNum(raw.killed_campers),
			sacrificedCampers: toNum(raw.sacrificed_campers),
			moriKills: toNum(raw.mori_kills),
			hooksPerformed: toNum(raw.hooks_performed),
			uncloakAttacks: toNum(raw.uncloak_attacks),
			generatorPct: toNum(raw.generator_pct),
			healPct: toNum(raw.heal_pct),
			escapesKo: toNum(raw.escapes_ko),
			escapes: toNum(raw.escapes),
			skillCheckSuccess: toNum(raw.skill_check_success),
			hookedAndEscape: toNum(raw.hooked_and_escape),
			unhookOrHeal: toNum(raw.unhook_or_heal),
			healsPerformed: toNum(raw.heals_performed),
			unhookOrHealPostExit: toNum(raw.unhook_or_heal_post_exit),
			postExitActions: toNum(raw.post_exit_actions),
			escapeThroughHatch: toNum(raw.escape_through_hatch),
			bloodwebPoints: toNum(raw.bloodweb_points),
			camperPerfectGames: toNum(raw.camper_perfect_games),
			killerPerfectGames: toNum(raw.killer_perfect_games),
			camperFullLoadout: toNum(raw.camper_full_loadout),
			killerFullLoadout: toNum(raw.killer_full_loadout),
			camperNewItem: toNum(raw.camper_new_item),
			timePlayedHours: toNum(raw.time_played_hours)
		},
		achievements: {
			total,
			unlocked,
			mapped,
			adepts: {
				survivors: raw.achievements?.adept_survivors ?? {},
				killers: raw.achievements?.adept_killers ?? {}
			}
		},
		sources: {
			...(raw.data_sources?.stats && { stats: raw.data_sources.stats }),
			...(raw.data_sources?.achievements && { achievements: raw.data_sources.achievements })
		}
	};
}

export function toPlayerBundle(raw: ApiPlayerStats): PlayerBundle {
	const player = {
		id: raw.steam_id,
		name: raw.display_name
	};

	const stats: DbdStats = {
		matchesPlayed: toNum(raw.total_matches),
		escapes: toNum(raw.escapes),
		kills: toNum(raw.killed_campers),
		killerPips: toNum(raw.killer_pips),
		survivorPips: toNum(raw.survivor_pips),
		sacrificedCampers: toNum(raw.sacrificed_campers),
		moriKills: toNum(raw.mori_kills),
		hooksPerformed: toNum(raw.hooks_performed),
		uncloakAttacks: toNum(raw.uncloak_attacks),
		generatorPct: toNum(raw.generator_pct),
		healPct: toNum(raw.heal_pct),
		escapesKo: toNum(raw.escapes_ko),
		skillCheckSuccess: toNum(raw.skill_check_success),
		hookedAndEscape: toNum(raw.hooked_and_escape),
		unhookOrHeal: toNum(raw.unhook_or_heal),
		healsPerformed: toNum(raw.heals_performed),
		unhookOrHealPostExit: toNum(raw.unhook_or_heal_post_exit),
		postExitActions: toNum(raw.post_exit_actions),
		escapeThroughHatch: toNum(raw.escape_through_hatch),
		bloodwebPoints: toNum(raw.bloodweb_points),
		camperPerfectGames: toNum(raw.camper_perfect_games),
		killerPerfectGames: toNum(raw.killer_perfect_games),
		camperFullLoadout: toNum(raw.camper_full_loadout),
		killerFullLoadout: toNum(raw.killer_full_loadout),
		camperNewItem: toNum(raw.camper_new_item),
		timePlayedHours: toNum(raw.time_played_hours)
	};

	const adepts: DbdAdept[] = [
		...Object.entries(raw.achievements?.adept_survivors ?? {}).map(([character, unlocked]) => ({
			id: `adept_survivor_${character}`,
			name: `Adept ${character}`,
			unlocked: Boolean(unlocked),
			unlockTime: null
		})),
		...Object.entries(raw.achievements?.adept_killers ?? {}).map(([character, unlocked]) => ({
			id: `adept_killer_${character}`,
			name: `Adept ${character}`,
			unlocked: Boolean(unlocked),
			unlockTime: null
		}))
	];

	const achievements: DbdAchievement[] = raw.achievements?.mapped_achievements?.map(achievement => ({
		id: achievement.id,
		name: achievement.character || achievement.id,
		description: `${achievement.type} achievement for ${achievement.character}`,
		unlocked: achievement.unlocked,
		unlockTime: null
	})) ?? [];

	return {
		player,
		stats,
		adepts,
		achievements
	};
}
