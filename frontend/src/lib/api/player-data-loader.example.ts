// Example usage of the player adapter in a Svelte component

import { normalizePlayerPayload, toUIStats, sortStats, selectHeader, groupStats, type UIStat } from '$lib/api/player-adapter';

export async function loadPlayerData(steamId: string) {
  // Fetch raw data from backend
  const raw = await fetch(`/api/player/${steamId}`).then(r => r.json());
  
  // Normalize the payload (handles nested/flat structures)
  const { stats, statsSummary, achievements, achievementSummary } = normalizePlayerPayload(raw);
  
  // Convert to UI-friendly format and sort
  const uiStats = sortStats(toUIStats(stats));
  
  // Extract header data using stable aliases
  const header = selectHeader(uiStats);
  
  // Group stats by category for rendering
  const groups = groupStats(uiStats);
  
  return {
    header: {
      killerGrade: header.killerGrade,     // "Bronze II" or "Unranked"
      survivorGrade: header.survivorGrade, // "Gold IV" or "Unranked"
      highestPrestige: header.highestPrestige // "15" or "0"
    },
    stats: {
      killer: groups.killer,     // Array of killer stats
      survivor: groups.survivor, // Array of survivor stats
      general: groups.general    // Array of general stats
    },
    meta: {
      statsSummary,
      achievementSummary,
      totalStats: uiStats.length
    }
  };
}

// Rendering helper for displaying stat values
export function displayStatValue(stat: UIStat): string {
  // Always use formatted if available
  if (stat.formatted) return stat.formatted;
  
  // Apply rendering rules based on value type
  switch (stat.valueType) {
    case 'float':
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

// Example Svelte component usage:
/*
<script>
  import { loadPlayerData, displayStatValue } from './player-data-loader';
  
  export let steamId;
  
  let playerData = null;
  
  $: if (steamId) {
    loadPlayerData(steamId).then(data => playerData = data);
  }
</script>

{#if playerData}
  <div class="player-header">
    <div>Killer: {playerData.header.killerGrade}</div>
    <div>Survivor: {playerData.header.survivorGrade}</div>
    <div>Prestige: {playerData.header.highestPrestige}</div>
  </div>
  
  <div class="stats-sections">
    <section>
      <h3>Killer Stats ({playerData.stats.killer.length})</h3>
      {#each playerData.stats.killer as stat}
        <div class="stat-row">
          <span>{stat.name}</span>
          <span>{displayStatValue(stat)}</span>
        </div>
      {/each}
    </section>
    
    <section>
      <h3>Survivor Stats ({playerData.stats.survivor.length})</h3>
      {#each playerData.stats.survivor as stat}
        <div class="stat-row">
          <span>{stat.name}</span>
          <span>{displayStatValue(stat)}</span>
        </div>
      {/each}
    </section>
    
    <section>
      <h3>General Stats ({playerData.stats.general.length})</h3>
      {#each playerData.stats.general as stat}
        <div class="stat-row">
          <span>{stat.name}</span>
          <span>{displayStatValue(stat)}</span>
        </div>
      {/each}
    </section>
  </div>
{/if}
*/
