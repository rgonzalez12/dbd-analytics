export type StatCategory = 'killer' | 'survivor' | 'general';
export type ValueType = 'count' | 'percent' | 'grade' | 'level' | 'duration';
export type StatAlias = 'killer_grade' | 'survivor_grade' | 'highest_prestige';

export interface WireStat {
  id: string;
  display_name: string;
  value: number;
  formatted?: string;
  category: StatCategory;
  value_type: ValueType;
  sort_weight: number;
  icon?: string;
  alias?: StatAlias;
  matched_by?: string;
}

export interface WirePlayerResponse {
  stats?: WireStat[] | { stats?: WireStat[]; summary?: unknown };
  achievements?: unknown[] | { achievements?: unknown[]; summary?: unknown };
}

function extractArray<T = unknown>(maybe: unknown, key: 'stats' | 'achievements'): T[] {
  if (Array.isArray(maybe)) return maybe as T[];
  if (maybe && typeof maybe === 'object') {
    const arr = (maybe as any)[key];
    if (Array.isArray(arr)) return arr as T[];
  }
  return [];
}

export function normalizePlayerPayload(raw: WirePlayerResponse) {
  const stats = extractArray<WireStat>(raw.stats, 'stats');
  const statsSummary = (raw.stats && typeof raw.stats === 'object' && (raw.stats as any).summary) || null;
  const achievements = extractArray<any>(raw.achievements, 'achievements');
  const achievementSummary = (raw.achievements && typeof raw.achievements === 'object' && (raw.achievements as any).summary) || null;
  return { stats, statsSummary, achievements, achievementSummary };
}

export interface UIStat {
  id: string;
  name: string;
  value: number;
  formatted?: string;
  category: StatCategory;
  valueType: ValueType;
  sortWeight: number;
  icon?: string;
  alias?: StatAlias;
  matchedBy?: string;
}

export function toUIStats(stats: WireStat[]): UIStat[] {
  return stats.map(s => {
    const ui: UIStat = {
      id: s.id,
      name: s.display_name,
      value: s.value,
      category: s.category,
      valueType: s.value_type,
      sortWeight: s.sort_weight,
    };
    if (s.formatted !== undefined) ui.formatted = s.formatted;
    if (s.icon !== undefined) ui.icon = s.icon;
    if (s.alias !== undefined) ui.alias = s.alias;
    if (s.matched_by !== undefined) ui.matchedBy = s.matched_by;
    return ui;
  });
}

const categoryOrder: Record<StatCategory, number> = { killer: 0, survivor: 1, general: 2 };

export function sortStats(stats: UIStat[]): UIStat[] {
  return [...stats].sort((a, b) => {
    const ca = categoryOrder[a.category], cb = categoryOrder[b.category];
    if (ca !== cb) return ca - cb;
    if (a.sortWeight !== b.sortWeight) return a.sortWeight - b.sortWeight;
    return a.name.localeCompare(b.name, undefined, { numeric: true });
  });
}

export function groupStats(stats: UIStat[]) {
  return {
    killer: stats.filter(s => s.category === 'killer'),
    survivor: stats.filter(s => s.category === 'survivor'),
    general: stats.filter(s => s.category === 'general'),
  };
}

export function selectHeader(stats: UIStat[]) {
  const byAlias = (a: StatAlias) => stats.find(s => s.alias === a);
  const killer = byAlias('killer_grade');
  const survivor = byAlias('survivor_grade');
  const prestige = byAlias('highest_prestige');
  return {
    killerGrade: killer?.formatted ?? 'Unranked',
    survivorGrade: survivor?.formatted ?? 'Unranked',
    highestPrestige: prestige?.formatted ?? (prestige ? String(prestige.value) : '0'),
  };
}

// Helper function for displaying stat values according to rendering rules
export function displayStatValue(stat: UIStat): string {
  // Always use formatted if available
  if (stat.formatted) return stat.formatted;
  
  // Apply rendering rules based on value type
  switch (stat.valueType) {
    case 'percent':
      return `${stat.value}%`;
    case 'count':
    case 'level':
      return stat.value.toLocaleString();
    case 'duration':
      return formatDuration(stat.value);
    case 'grade':
      return String(stat.value); // Should have formatted, but fallback
    default:
      return String(stat.value);
  }
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  return `${Math.floor(seconds / 3600)}h`;
}
