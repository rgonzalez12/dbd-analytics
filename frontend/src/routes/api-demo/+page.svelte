<script lang="ts">
	import { onMount } from 'svelte';
	import { apiGet, withQuery, ApiError } from '$lib/api/client';
	import type { PlayerStatsWithAchievements, PlayerStats } from '$lib/api/types';

	let playerData: PlayerStatsWithAchievements | null = null;
	let loading = false;
	let error: string | null = null;
	let searchId = '';

	async function loadPlayerData(steamId: string) {
		loading = true;
		error = null;
		
		try {
			const url = withQuery('/api/player', { 
				steamId,
				include_achievements: true,
				format: 'detailed'
			});
			
			playerData = await apiGet<PlayerStatsWithAchievements>(url);
			
			console.log('Player data loaded:', playerData);
		} catch (err) {
			if (err instanceof ApiError) {
				error = `API Error (${err.status}): ${err.message}`;
				console.error('API Error Body:', err.body);
			} else {
				error = err instanceof Error ? err.message : 'Unknown error';
			}
		} finally {
			loading = false;
		}
	}

	function handleSearch() {
		if (searchId.trim()) {
			loadPlayerData(searchId.trim());
		}
	}

	onMount(() => {
		const mockPlayerData: PlayerStatsWithAchievements = {
			steam_id: '76561198000000000',
			display_name: 'Demo Player',
			killer_pips: 150,
			survivor_pips: 89,
			killed_campers: 1250,
			sacrificed_campers: 890,
			mori_kills: 45,
			hooks_performed: 2800,
			uncloak_attacks: 120,
			generator_pct: 78.5,
			heal_pct: 85.2,
			escapes_ko: 15,
			escapes: 245,
			skill_check_success: 9800,
			hooked_and_escape: 180,
			unhook_or_heal: 340,
			heals_performed: 220,
			unhook_or_heal_post_exit: 45,
			post_exit_actions: 45,
			escape_through_hatch: 32,
			bloodweb_points: 45000000,
			camper_perfect_games: 8,
			killer_perfect_games: 12,
			camper_full_loadout: 150,
			killer_full_loadout: 98,
			camper_new_item: 85,
			total_matches: 892,
			time_played_hours: 245,
			last_updated: new Date().toISOString(),
			achievements: {
				adept_survivors: {
					'Meg Thomas': true,
					'Dwight Fairfield': true,
					'Claudette Morel': false
				},
				adept_killers: {
					'Trapper': true,
					'Wraith': false,
					'Hillbilly': true
				},
				mapped_achievements: [
					{
						id: 'ACH_ADEPT_TRAPPER',
						name: 'Adept Trapper',
						display_name: 'Adept Trapper',
						description: 'Achieve a merciless victory with The Trapper using only his unique perks.',
						character: 'Trapper',
						type: 'killer',
						unlocked: true,
						unlock_time: 1640995200
					}
				],
				summary: {
					total_achievements: 156,
					unlocked_count: 89,
					survivor_count: 45,
					killer_count: 38,
					general_count: 73,
					adept_survivors: ['Meg Thomas', 'Dwight Fairfield'],
					adept_killers: ['Trapper', 'Hillbilly'],
					completion_rate: 57.05
				},
				last_updated: new Date().toISOString()
			},
			data_sources: {
				stats: {
					success: true,
					source: 'cache',
					fetched_at: new Date().toISOString()
				},
				achievements: {
					success: true,
					source: 'api',
					fetched_at: new Date().toISOString()
				}
			}
		};

		playerData = mockPlayerData;
	});
</script>

<div class="container mx-auto p-6 max-w-4xl">
	<h1 class="text-3xl font-bold text-dbd-red mb-6">API Client Demo</h1>
	
	<div class="mb-6 bg-dbd-gray p-4 rounded-lg">
		<h2 class="text-xl font-semibold mb-4">Test API Client</h2>
		<div class="flex gap-2">
			<input 
				bind:value={searchId}
				placeholder="Enter Steam ID"
				class="flex-1 px-3 py-2 bg-gray-800 border border-gray-600 rounded text-white placeholder-gray-400"
			/>
			<button 
				on:click={handleSearch}
				disabled={loading}
				class="px-4 py-2 bg-dbd-red text-white rounded hover:bg-red-700 disabled:opacity-50"
			>
				{loading ? 'Loading...' : 'Search'}
			</button>
		</div>
		{#if error}
			<div class="mt-2 p-2 bg-red-600 text-white rounded text-sm">
				{error}
			</div>
		{/if}
	</div>

	{#if playerData}
		<div class="bg-gray-800 p-6 rounded-lg">
			<h2 class="text-2xl font-semibold mb-4">Player Data (TypeScript Typed)</h2>
			
			<!-- Player Profile -->
			<div class="mb-6">
				<h3 class="text-lg font-semibold text-red-600 mb-2">Profile</h3>
				<div class="grid grid-cols-2 gap-4 text-sm">
					<div><span class="text-gray-400">Steam ID:</span> {playerData.steam_id}</div>
					<div><span class="text-gray-400">Name:</span> {playerData.display_name}</div>
					<div><span class="text-gray-400">Last Updated:</span> {new Date(playerData.last_updated).toLocaleString()}</div>
				</div>
			</div>

			<!-- DBD Stats -->
			<div class="mb-6">
				<h3 class="text-lg font-semibold text-red-600 mb-2">DBD Stats</h3>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-2 text-sm">
					<div class="flex justify-between">
						<span class="text-gray-400">Total Matches:</span>
						<span>{playerData.total_matches.toLocaleString()}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-400">Hours Played:</span>
						<span>{playerData.time_played_hours.toLocaleString()}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-400">Bloodweb Points:</span>
						<span>{playerData.bloodweb_points.toLocaleString()}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-400">Killer Pips:</span>
						<span>{playerData.killer_pips.toLocaleString()}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-400">Survivor Pips:</span>
						<span>{playerData.survivor_pips.toLocaleString()}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-400">Kills:</span>
						<span>{playerData.killed_campers.toLocaleString()}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-400">Escapes:</span>
						<span>{playerData.escapes.toLocaleString()}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-400">Generator Progress:</span>
						<span>{playerData.generator_pct.toFixed(1)}%</span>
					</div>
				</div>
			</div>

			<!-- Achievements -->
			{#if playerData.achievements}
				<div class="mb-6">
					<h3 class="text-lg font-semibold text-red-600 mb-2">Achievements</h3>
					
					<!-- Summary -->
					{#if playerData.achievements.summary}
						<div class="mb-4 text-sm">
							<div class="grid grid-cols-2 md:grid-cols-4 gap-2">
								<div><span class="text-gray-400">Total:</span> {playerData.achievements.summary.total_achievements}</div>
								<div><span class="text-gray-400">Unlocked:</span> {playerData.achievements.summary.unlocked_count}</div>
								<div><span class="text-gray-400">Rate:</span> {playerData.achievements.summary.completion_rate.toFixed(1)}%</div>
								<div><span class="text-gray-400">Survivors:</span> {playerData.achievements.summary.survivor_count}</div>
							</div>
						</div>
					{/if}

					<!-- Adept Survivors -->
					<div class="mb-3">
						<h4 class="font-medium mb-2">Adept Survivors</h4>
						<div class="grid grid-cols-2 md:grid-cols-3 gap-1 text-xs">
							{#each Object.entries(playerData.achievements.adept_survivors) as [character, unlocked]}
								<div class="flex items-center gap-2">
									<span class={unlocked ? 'text-green-400' : 'text-red-400'}>
										{unlocked ? '✓' : '✗'}
									</span>
									<span>{character}</span>
								</div>
							{/each}
						</div>
					</div>

					<!-- Adept Killers -->
					<div class="mb-3">
						<h4 class="font-medium mb-2">Adept Killers</h4>
						<div class="grid grid-cols-2 md:grid-cols-3 gap-1 text-xs">
							{#each Object.entries(playerData.achievements.adept_killers) as [character, unlocked]}
								<div class="flex items-center gap-2">
									<span class={unlocked ? 'text-green-400' : 'text-red-400'}>
										{unlocked ? '✓' : '✗'}
									</span>
									<span>{character}</span>
								</div>
							{/each}
						</div>
					</div>
				</div>
			{/if}

			<!-- Data Sources -->
			<div>
				<h3 class="text-lg font-semibold text-red-600 mb-2">Data Sources</h3>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
					<div class="p-3 bg-gray-700 rounded">
						<h4 class="font-medium mb-1">Stats</h4>
						<div class="text-xs space-y-1">
							<div><span class="text-gray-400">Success:</span> {playerData.data_sources.stats.success ? '✓' : '✗'}</div>
							<div><span class="text-gray-400">Source:</span> {playerData.data_sources.stats.source}</div>
							<div><span class="text-gray-400">Fetched:</span> {new Date(playerData.data_sources.stats.fetched_at).toLocaleTimeString()}</div>
						</div>
					</div>
					<div class="p-3 bg-gray-700 rounded">
						<h4 class="font-medium mb-1">Achievements</h4>
						<div class="text-xs space-y-1">
							<div><span class="text-gray-400">Success:</span> {playerData.data_sources.achievements.success ? '✓' : '✗'}</div>
							<div><span class="text-gray-400">Source:</span> {playerData.data_sources.achievements.source}</div>
							<div><span class="text-gray-400">Fetched:</span> {new Date(playerData.data_sources.achievements.fetched_at).toLocaleTimeString()}</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- API Usage Documentation -->
	<div class="mt-8 bg-dbd-gray p-6 rounded-lg">
		<h2 class="text-xl font-semibold mb-4">API Client Usage</h2>
		<pre class="text-sm text-gray-300 overflow-x-auto"><code>{`// Import the API client and types
import { apiGet, withQuery, ApiError } from '$lib/api/client';
import type { PlayerPageData } from '$lib/api/types';

// Build URL with query parameters
const url = withQuery('/api/player/stats', {
  steamId: '76561198000000000',
  include_achievements: true,
  format: 'detailed'
});

// Make typed API request
try {
  const data = await apiGet<PlayerPageData>(url, fetch);
  console.log('Player data:', data.profile.name);
} catch (error) {
  if (error instanceof ApiError) {
    console.error('API Error:', error.status, error.body);
  }
}`}</code></pre>
	</div>
</div>
