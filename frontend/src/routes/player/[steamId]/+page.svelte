<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';

	let steamId = '';
	let loading = true;
	let error = '';
	let playerData: any = null;

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

			// This will use our Vite proxy to call the Go backend
			const response = await fetch(`/api/player/${steamId}`);

			if (!response.ok) {
				throw new Error(`Failed to load player data: ${response.status}`);
			}

			playerData = await response.json();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load player data';
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
		<a href="/" class="text-dbd-red hover:text-red-400 transition-colors"> ← Back to Search </a>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-dbd-red"></div>
			<span class="ml-3 text-gray-300">Loading player data...</span>
		</div>
	{:else if error}
		<div class="bg-red-900 border border-red-700 rounded-lg p-4">
			<h3 class="text-red-300 font-medium">Error Loading Player Data</h3>
			<p class="text-red-200 mt-1">{error}</p>
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
			<div class="bg-dbd-gray rounded-lg p-6 border border-gray-700">
				<h3 class="text-xl font-semibold text-white mb-4">Player Info</h3>
				<div class="space-y-2 text-gray-300">
					<p><span class="text-gray-400">Steam ID:</span> {steamId}</p>
					<p>
						<span class="text-gray-400">Display Name:</span>
						{playerData.displayName || 'Unknown'}
					</p>
					<p><span class="text-gray-400">Last Updated:</span> {new Date().toLocaleDateString()}</p>
				</div>
			</div>

			<!-- Killer Stats Card -->
			<div class="bg-dbd-gray rounded-lg p-6 border border-gray-700">
				<h3 class="text-xl font-semibold text-dbd-red mb-4">Killer Stats</h3>
				<div class="space-y-2 text-gray-300">
					<p>
						<span class="text-gray-400">Total Kills:</span>
						{playerData.killer?.total_kills || 0}
					</p>
					<p>
						<span class="text-gray-400">Sacrifices:</span>
						{playerData.killer?.sacrificed_victims || 0}
					</p>
					<p><span class="text-gray-400">Hooks:</span> {playerData.killer?.hooks_performed || 0}</p>
					<p>
						<span class="text-gray-400">Perfect Games:</span>
						{playerData.killer?.perfect_games || 0}
					</p>
				</div>
			</div>

			<!-- Survivor Stats Card -->
			<div class="bg-dbd-gray rounded-lg p-6 border border-gray-700">
				<h3 class="text-xl font-semibold text-green-400 mb-4">Survivor Stats</h3>
				<div class="space-y-2 text-gray-300">
					<p>
						<span class="text-gray-400">Escapes:</span>
						{playerData.survivor?.total_escapes || 0}
					</p>
					<p>
						<span class="text-gray-400">Generators:</span>
						{playerData.survivor?.generators_completed_pct || 0}%
					</p>
					<p>
						<span class="text-gray-400">Skill Checks:</span>
						{playerData.survivor?.skill_checks_hit || 0}
					</p>
					<p>
						<span class="text-gray-400">Perfect Games:</span>
						{playerData.survivor?.perfect_games || 0}
					</p>
				</div>
			</div>

			<!-- General Stats Card -->
			<div class="bg-dbd-gray rounded-lg p-6 border border-gray-700 md:col-span-2 lg:col-span-3">
				<h3 class="text-xl font-semibold text-yellow-400 mb-4">General Stats</h3>
				<div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-gray-300">
					<div>
						<p class="text-gray-400">Total Matches</p>
						<p class="text-2xl font-bold">{playerData.general?.total_matches || 0}</p>
					</div>
					<div>
						<p class="text-gray-400">Bloodweb Points</p>
						<p class="text-2xl font-bold">
							{(playerData.general?.bloodweb_points || 0).toLocaleString()}
						</p>
					</div>
					<div>
						<p class="text-gray-400">Time Played</p>
						<p class="text-2xl font-bold">
							{Math.round(playerData.general?.time_played_hours || 0)}h
						</p>
					</div>
					<div>
						<p class="text-gray-400">Data Source</p>
						<p class="text-2xl font-bold capitalize">{playerData.source || 'Unknown'}</p>
					</div>
				</div>
			</div>
		</div>
	{:else}
		<div class="text-center py-12">
			<p class="text-gray-400">No player data available</p>
		</div>
	{/if}
</div>
