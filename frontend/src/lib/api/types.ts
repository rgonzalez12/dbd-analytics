export type ApiError = { status: number; message: string; details?: unknown; retryAfter?: number };

// Raw API types - match backend exactly with optional/nullable fields
export type ApiPlayerStats = {
  steam_id: string;
  display_name: string;
  total_matches?: number | string | null;
  last_updated?: string | null;
  killer_pips?: number | string | null;
  survivor_pips?: number | string | null;
  killed_campers?: number | string | null;
  sacrificed_campers?: number | string | null;
  mori_kills?: number | string | null;
  hooks_performed?: number | string | null;
  uncloak_attacks?: number | string | null;
  generator_pct?: number | string | null;
  heal_pct?: number | string | null;
  escapes_ko?: number | string | null;
  escapes?: number | string | null;
  skill_check_success?: number | string | null;
  hooked_and_escape?: number | string | null;
  unhook_or_heal?: number | string | null;
  heals_performed?: number | string | null;
  unhook_or_heal_post_exit?: number | string | null;
  post_exit_actions?: number | string | null;
  escape_through_hatch?: number | string | null;
  bloodweb_points?: number | string | null;
  camper_perfect_games?: number | string | null;
  killer_perfect_games?: number | string | null;
  camper_full_loadout?: number | string | null;
  killer_full_loadout?: number | string | null;
  camper_new_item?: number | string | null;
  time_played_hours?: number | string | null;
  achievements?: {
    summary?: { total?: number | string | null; unlocked?: number | string | null; last_updated?: string | null };
    mapped_achievements?: Array<{ id: string; type: 'survivor'|'killer'; character: string; unlocked: boolean }>;
    adept_survivors?: Record<string, boolean>;
    adept_killers?: Record<string, boolean>;
  };
  data_sources?: {
    stats?: { success: boolean; source: 'cache'|'api'|'fallback'; error?: string; fetched_at?: string };
    achievements?: { success: boolean; source: 'cache'|'api'|'fallback'; error?: string; fetched_at?: string };
  };
};

// Domain types - strict, UI-friendly with defaults
export type Player = {
  id: string;
  name: string;
  matches: number;                  // default to 0
  lastUpdated: string | null;       // normalize null/missing
  stats: {
    killerPips: number;
    survivorPips: number;
    killedCampers: number;
    sacrificedCampers: number;
    moriKills: number;
    hooksPerformed: number;
    uncloakAttacks: number;
    generatorPct: number;
    healPct: number;
    escapesKo: number;
    escapes: number;
    skillCheckSuccess: number;
    hookedAndEscape: number;
    unhookOrHeal: number;
    healsPerformed: number;
    unhookOrHealPostExit: number;
    postExitActions: number;
    escapeThroughHatch: number;
    bloodwebPoints: number;
    camperPerfectGames: number;
    killerPerfectGames: number;
    camperFullLoadout: number;
    killerFullLoadout: number;
    camperNewItem: number;
    timePlayedHours: number;
  };
  achievements: {
    total: number;                  // default to 0
    unlocked: number;               // default to 0
    mapped: Array<{ id: string; type: 'survivor'|'killer'; character: string; unlocked: boolean }>;
    adepts: { survivors: Record<string, boolean>; killers: Record<string, boolean> };
  };
  sources: {
    stats?: { success: boolean; source: 'cache'|'api'|'fallback'; error?: string; fetched_at?: string };
    achievements?: { success: boolean; source: 'cache'|'api'|'fallback'; error?: string; fetched_at?: string };
  };
};

export type LoadState<T> = { ok: true; data: T } | { ok: false; error: ApiError };
