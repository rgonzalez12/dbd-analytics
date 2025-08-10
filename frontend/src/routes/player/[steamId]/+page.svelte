<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { apiGet } from '$lib/api/client';
	import type { PlayerStatsWithAchievements } from '$lib/api/types';

	let steamId = '';
	let loading = true;
	let error = '';
	let playerData: PlayerStatsWithAchievements | null = null;

	onMount(() => {
		steamId = $page.params.steamId || '';
		if (steamId) {
			loadPlayerData();
		}
	});

	async function loadPlayerData() {
		try {
			loading = true;
			error = '';

			playerData = await apiGet<PlayerStatsWithAchievements>(`/api/player/${steamId}`);
		} catch (err) {
			if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to load player data';
			}
			console.error('Error loading player data:', err);
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>DBD Analytics - Player {steamId}</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-3xl font-bold text-white">Player Statistics</h1>
		<a href="/" class="text-red-500 hover:text-red-400 transition-colors"> ← Back to Search </a>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-red-500"></div>
			<span class="ml-3 text-gray-300">Loading player data...</span>
		</div>
	{:else if error}
		<div class="bg-red-900 border border-red-700 rounded-lg p-4">
			<h3 class="text-red-300 font-medium">Error Loading Player Data</h3>
			<p class="text-red-200 mt-1">{error}</p>
			{#if error.includes('private')}
				<p class="text-red-200 mt-2 text-sm">
					This Steam profile is set to private. The player needs to make their game details public 
					for us to access their DBD statistics.
				</p>
			{/if}
			<button
				on:click={loadPlayerData}
				class="mt-3 px-4 py-2 bg-red-700 text-white rounded hover:bg-red-600 transition-colors"
			>
				Retry
			</button>
		</div>
	{:else if playerData}
		<div class="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
			<!-- Player Info Card -->
			<div class="bg-gray-800 rounded-lg p-6 border border-gray-700">
				<h3 class="text-xl font-semibold text-white mb-4">Player Info</h3>
				<div class="space-y-2 text-gray-300">
					<p><span class="text-gray-400">Steam ID:</span> {steamId}</p>
					<p>
						<span class="text-gray-400">Display Name:</span>
						{playerData.display_name || 'Unknown'}
					</p>
					<p><span class="text-gray-400">Source:</span> {playerData.source || 'Steam API'}</p>
				</div>
			</div>

			<!-- Achievements Card -->
			<div class="bg-gray-800 rounded-lg p-6 border border-gray-700 md:col-span-2">
				<h3 class="text-xl font-semibold text-yellow-400 mb-4">
					Achievements ({playerData.achievements?.length || 0})
				</h3>
				{#if playerData.achievements && playerData.achievements.length > 0}
					<div class="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-64 overflow-y-auto">
						{#each playerData.achievements.slice(0, 10) as achievement}
							<div class="flex items-center space-x-3 p-2 bg-gray-700 rounded">
								<div class="flex-shrink-0">
									{#if achievement.achieved}
										<div class="w-3 h-3 bg-green-500 rounded-full"></div>
									{:else}
										<div class="w-3 h-3 bg-gray-500 rounded-full"></div>
									{/if}
								</div>
								<div class="flex-1 min-w-0">
									<p class="text-sm font-medium text-white truncate">
										{achievement.display_name}
									</p>
									{#if achievement.achieved && achievement.unlock_time}
										<p class="text-xs text-gray-400">
											Unlocked {new Date(achievement.unlock_time * 1000).toLocaleDateString()}
										</p>
									{/if}
								</div>
							</div>
						{/each}
					</div>
					{#if playerData.achievements.length > 10}
						<p class="text-sm text-gray-400 mt-3">
							Showing 10 of {playerData.achievements.length} achievements
						</p>
					{/if}
				{:else}
					<p class="text-gray-400">No achievements data available</p>
				{/if}
			</div>
		</div>

		<!-- Achievement Progress -->
		<div class="bg-gray-800 rounded-lg p-6 border border-gray-700">
			<h3 class="text-xl font-semibold text-blue-400 mb-4">Achievement Progress</h3>
			{#if playerData.achievements && playerData.achievements.length > 0}
				{@const totalAchievements = playerData.achievements.length}
				{@const unlockedAchievements = playerData.achievements.filter(a => a.achieved).length}
				{@const progressPercent = (unlockedAchievements / totalAchievements) * 100}
				
				<div class="space-y-4">
					<div class="flex justify-between text-sm">
						<span class="text-gray-300">{unlockedAchievements} of {totalAchievements} unlocked</span>
						<span class="text-gray-300">{progressPercent.toFixed(1)}%</span>
					</div>
					<div class="w-full bg-gray-700 rounded-full h-2">
						<div class="bg-blue-500 h-2 rounded-full" style="width: {progressPercent}%"></div>
					</div>
				</div>
			{:else}
				<p class="text-gray-400">No achievement data available</p>
			{/if}
		</div>
	{:else}
		<div class="text-center py-12">
			<p class="text-gray-400">No player data available</p>
		</div>
	{/if}
</div>
