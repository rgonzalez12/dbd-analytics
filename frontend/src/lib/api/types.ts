import type { UIStat } from './player-adapter';

export type ApiError = { status: number; message: string; details?: unknown; retryAfter?: number };

// New stats structure from the API
export type ApiStat = {
  id: string;
  display_name: string;
  value: number;
  formatted?: string;
  category: 'killer' | 'survivor' | 'general';
  value_type: 'count' | 'float' | 'grade' | 'level' | 'duration';
  sort_weight: number;
  icon?: string;
  alias?: string;
  matched_by?: 'schema' | 'alias' | 'fallback';
};

export type ApiStatsSummary = {
  killer_grade?: string;
  killer_pips?: number;
  prestige_max?: number;
  survivor_grade?: string;
  survivor_pips?: number;
};

// Raw API types - match backend exactly with optional/nullable fields
export type ApiPlayerStats = {
  steam_id: string;
  display_name: string;
  total_matches?: number | string | null;
  last_updated?: string | null;
  // Legacy fields for backward compatibility
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
  // New structured stats
  stats?: {
    stats?: ApiStat[];
    summary?: ApiStatsSummary;
  };
  achievements?: {
    summary?: { 
      total?: number | string | null; 
      unlocked?: number | string | null; 
      last_updated?: string | null;
      total_achievements?: number;
      unlocked_count?: number;
      survivor_count?: number;
      killer_count?: number;
      general_count?: number;
      adept_survivors?: string;
      adept_killers?: string;
      completion_rate?: number;
    };
    mapped_achievements?: Array<{
      id: string;
      name: string;
      display_name: string;
      description: string;
      icon?: string;
      icon_gray?: string;
      hidden?: boolean;
      character?: string;
      type: string;
      unlocked: boolean;
      unlock_time?: number;
      rarity?: number;
    }>;
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
    // New structured stats
    all: UIStat[];
    killer: UIStat[];
    survivor: UIStat[];
    general: UIStat[];
    header: {
      killerGrade: string;
      survivorGrade: string;
      highestPrestige: string;
    };
    summary: {
      total_stats: number;
      killer_count: number;
      survivor_count: number;
      general_count: number;
    };
    // Legacy individual fields for backward compatibility
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
    mapped: Array<{
      id: string;
      name: string;
      display_name: string;
      description: string;
      icon?: string;
      icon_gray?: string;
      hidden?: boolean;
      character?: string;
      type: string;
      unlocked: boolean;
      unlock_time?: number;
      rarity?: number;
    }>;
    adepts: { survivors: Record<string, boolean>; killers: Record<string, boolean> };
  };
  sources: {
    stats?: { success: boolean; source: 'cache'|'api'|'fallback'; error?: string; fetched_at?: string };
    achievements?: { success: boolean; source: 'cache'|'api'|'fallback'; error?: string; fetched_at?: string };
  };
};

export type LoadState<T> = { ok: true; data: T } | { ok: false; error: ApiError };
