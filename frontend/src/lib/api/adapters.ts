import type { Player, SchemaPlayer } from '$lib/api/types';
import type { PlayerBundle, DbdAdept, DbdAchievement, DbdStats } from '$lib/types';
import type { ApiPlayerStats, ApiSchemaPlayerSummary } from '$lib/api/types';
import { normalizePlayerPayload, toUIStats, sortStats, selectHeader, groupStats, type WirePlayerResponse } from '$lib/api/player-adapter';

function toNum(v: unknown, d = 0): number {
	if (typeof v === 'number') return isNaN(v) ? d : v;
	if (typeof v === 'string') {
		const parsed = parseFloat(v);
		return isNaN(parsed) ? d : parsed;
	}
	return d;
}

export function toDomainPlayer(raw: ApiPlayerStats): Player {
	// Helper function to convert strings/nulls to numbers
	const toNum = (val: number | string | null | undefined): number => {
		if (typeof val === 'number') return val;
		if (typeof val === 'string') {
			const parsed = parseFloat(val);
			return isNaN(parsed) ? 0 : parsed;
		}
		return 0;
	};

	// Direct access to achievements from the API response
	const achievementsData = raw.achievements?.mapped_achievements || [];
	const achievementSummary = raw.achievements?.summary;
	
	// Process achievements directly from API response
	const mapped = achievementsData.map((achievement: any) => ({
		id: achievement.id,
		name: achievement.display_name || achievement.name || achievement.id,
		display_name: achievement.display_name || achievement.name || achievement.id,
		displayName: achievement.display_name || achievement.name || achievement.id, // For backward compatibility
		description: achievement.description || "",
		...(achievement.icon !== undefined && { icon: achievement.icon }),
		...(achievement.icon_gray !== undefined && { icon_gray: achievement.icon_gray }),
		...(achievement.hidden !== undefined && { hidden: achievement.hidden }),
		...(achievement.character !== undefined && { character: achievement.character }),
		type: achievement.type || "general",
		unlocked: Boolean(achievement.unlocked),
		achieved: Boolean(achievement.unlocked), // For backward compatibility with existing UI
		...(achievement.unlock_time !== undefined && { unlock_time: achievement.unlock_time }),
		...(achievement.unlock_time !== undefined && { unlockTime: achievement.unlock_time }), // For backward compatibility
		...(achievement.rarity !== undefined && { rarity: achievement.rarity })
	}));

	const total = achievementSummary?.total_achievements || mapped.length;
	const unlocked = achievementSummary?.unlocked_count || mapped.filter((a: any) => a.unlocked).length;

	// Process stats data using the player adapter
	const wireStats = (raw.stats?.stats || []).map(apiStat => {
		const wireStat: any = {
			id: apiStat.id,
			display_name: apiStat.display_name,
			value: apiStat.value,
			category: apiStat.category,
			value_type: apiStat.value_type,
			sort_weight: apiStat.sort_weight
		};
		if (apiStat.formatted !== undefined) wireStat.formatted = apiStat.formatted;
		if (apiStat.icon !== undefined) wireStat.icon = apiStat.icon;
		if (apiStat.alias !== undefined) wireStat.alias = apiStat.alias;
		if (apiStat.matched_by !== undefined) wireStat.matched_by = apiStat.matched_by;
		return wireStat;
	});
	
	const statsSummary = raw.stats?.summary || {};
	
	// Convert to UI stats and organize
	const uiStats = toUIStats(wireStats);
	const sortedStats = sortStats(uiStats);
	const groupedStats = groupStats(sortedStats);
	const header = selectHeader(uiStats, statsSummary);
	
	const statsSummaryData = {
		total_stats: uiStats.length,
		killer_count: groupedStats.killer.length,
		survivor_count: groupedStats.survivor.length,
		general_count: groupedStats.general.length
	};

	return {
		id: raw.steam_id,
		steamId: raw.steam_id,
		name: raw.display_name || raw.steam_id,
		avatar: raw.avatar || `/favicon.png`, // Use API avatar or fallback to favicon
		public: true, // Assume public if we got data
		matches: toNum(raw.total_matches),
		lastUpdated: raw.last_updated ?? null,
		stats: {
			// New structured stats (properly processed)
			all: sortedStats,
			killer: groupedStats.killer,
			survivor: groupedStats.survivor,
			general: groupedStats.general,
			header: header,
			summary: statsSummaryData,
			// Legacy individual fields for backward compatibility
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
		name: achievement.display_name || achievement.name || achievement.id,
		description: achievement.description || `${achievement.type} achievement${achievement.character ? ` for ${achievement.character}` : ''}`,
		unlocked: achievement.unlocked,
		unlockTime: achievement.unlock_time ? new Date(achievement.unlock_time * 1000).toISOString() : null,
		...(achievement.icon && { iconUrl: achievement.icon }),
		...(achievement.icon_gray && { iconGrayUrl: achievement.icon_gray }),
		...(achievement.hidden !== undefined && { hidden: achievement.hidden }),
		...(achievement.character && { character: achievement.character }),
		...(achievement.type && { type: achievement.type }),
		...(achievement.rarity !== undefined && { rarity: achievement.rarity })
	})) ?? [];

	return {
		player,
		stats,
		adepts,
		achievements
	};
}

function steamIdToName(steamId: string): string {
	return `Player ${steamId}`;
}

export function toSchemaPlayer(api: ApiSchemaPlayerSummary): SchemaPlayer {
	const result: SchemaPlayer = {
		id: api.playerId,
		name: api.displayName || steamIdToName(api.playerId),
		stats: api.stats,
		achievements: api.achievements,
		lastUpdated: api.lastUpdated,
		dataSources: api.dataSources
	};

	if (api.survivorGrade) {
		result.survivorGrade = api.survivorGrade;
	}

	if (api.killerGrade) {
		result.killerGrade = api.killerGrade;
	}

	return result;
}
