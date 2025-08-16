<script lang="ts">
	import type { PageData } from './$types';
	
	export let data: PageData;
	
	$: player = data.data;
	
	// Tab state
	let activeTab: 'stats' | 'adepts' | 'achievements' = 'stats';
	
	// Tab navigation
	function setActiveTab(tab: 'stats' | 'adepts' | 'achievements') {
		activeTab = tab;
	}
	
	// Keyboard navigation for tabs
	function handleTabKeydown(event: KeyboardEvent, tab: 'stats' | 'adepts' | 'achievements') {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			setActiveTab(tab);
		}
	}
	
	// Format numbers for display
	function formatNumber(value: number): string {
		return value.toLocaleString();
	}
	
	// Format time played
	function formatTimePlayedHours(hours: number): string {
		if (hours < 1) return '<1 hour';
		if (hours < 24) return `${Math.round(hours)} hours`;
		const days = Math.floor(hours / 24);
		const remainingHours = Math.round(hours % 24);
		return remainingHours > 0 ? `${days}d ${remainingHours}h` : `${days} days`;
	}
	
	// Calculate adept completion
	$: adeptStats = (() => {
		const survivorTotal = Object.keys(player.achievements.adepts.survivors).length;
		const survivorUnlocked = Object.values(player.achievements.adepts.survivors).filter(Boolean).length;
		const killerTotal = Object.keys(player.achievements.adepts.killers).length;
		const killerUnlocked = Object.values(player.achievements.adepts.killers).filter(Boolean).length;
		
		return {
			survivor: { unlocked: survivorUnlocked, total: survivorTotal },
			killer: { unlocked: killerUnlocked, total: killerTotal }
		};
	})();
</script>

<svelte:head>
	<title>{player.name} - DBD Analytics</title>
	<meta name="description" content="Dead by Daylight statistics for {player.name}" />
</svelte:head>

<article class="space-y-8">
	<!-- Player Header -->
	<header class="space-y-4">
		<div class="space-y-2">
			<h1 class="text-3xl font-bold tracking-tight">{player.name}</h1>
			<div class="flex items-center gap-4 text-sm text-neutral-400">
				<span>Steam ID: {player.id}</span>
				{#if player.lastUpdated}
					<span>•</span>
					<span>Updated: {new Date(player.lastUpdated).toLocaleDateString()}</span>
				{/if}
			</div>
		</div>
		
		<!-- Quick Stats Overview -->
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<div class="rounded-lg border border-neutral-700 bg-neutral-900/50 p-4">
				<div class="text-sm text-neutral-400">Total Matches</div>
				<div class="text-2xl font-semibold">{formatNumber(player.matches)}</div>
			</div>
			<div class="rounded-lg border border-neutral-700 bg-neutral-900/50 p-4">
				<div class="text-sm text-neutral-400">Time Played</div>
				<div class="text-2xl font-semibold">{formatTimePlayedHours(player.stats.timePlayedHours)}</div>
			</div>
			<div class="rounded-lg border border-neutral-700 bg-neutral-900/50 p-4">
				<div class="text-sm text-neutral-400">Achievements</div>
				<div class="text-2xl font-semibold">{player.achievements.unlocked}/{player.achievements.total}</div>
			</div>
			<div class="rounded-lg border border-neutral-700 bg-neutral-900/50 p-4">
				<div class="text-sm text-neutral-400">Adepts Complete</div>
				<div class="text-2xl font-semibold">{adeptStats.survivor.unlocked + adeptStats.killer.unlocked}/{adeptStats.survivor.total + adeptStats.killer.total}</div>
			</div>
		</div>
	</header>

	<!-- Tab Navigation -->
	<div class="border-b border-neutral-700" role="tablist" aria-label="Player statistics sections">
		<div class="flex space-x-8">
			{#each [
				{ id: 'stats' as const, label: 'Statistics' },
				{ id: 'adepts' as const, label: 'Adept Achievements' },
				{ id: 'achievements' as const, label: 'All Achievements' }
			] as tab}
				<button
					role="tab"
					tabindex={activeTab === tab.id ? 0 : -1}
					aria-selected={activeTab === tab.id}
					aria-controls="{tab.id}-panel"
					class="border-b-2 px-1 py-4 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-neutral-900 {activeTab === tab.id 
						? 'border-blue-500 text-blue-400' 
						: 'border-transparent text-neutral-400 hover:border-neutral-300 hover:text-neutral-200'}"
					on:click={() => setActiveTab(tab.id)}
					on:keydown={(e) => handleTabKeydown(e, tab.id)}
				>
					{tab.label}
				</button>
			{/each}
		</div>
	</div>

	<!-- Tab Panels -->
	<div class="min-h-[400px]">
		<!-- Statistics Tab -->
		{#if activeTab === 'stats'}
			<div
				id="stats-panel"
				role="tabpanel"
				aria-labelledby="stats-tab"
				class="space-y-8"
			>
				<div class="grid gap-6 lg:grid-cols-2">
					<!-- Killer Stats -->
					<div class="space-y-4">
						<h3 class="text-lg font-semibold text-red-400">Killer Statistics</h3>
						<div class="space-y-3">
							<div class="flex justify-between">
								<span class="text-neutral-300">Pips Earned</span>
								<span class="font-mono">{formatNumber(player.stats.killerPips)}</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Campers Killed</span>
								<span class="font-mono">{formatNumber(player.stats.killedCampers)}</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Campers Sacrificed</span>
								<span class="font-mono">{formatNumber(player.stats.sacrificedCampers)}</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Mori Kills</span>
								<span class="font-mono">{formatNumber(player.stats.moriKills)}</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Hooks Performed</span>
								<span class="font-mono">{formatNumber(player.stats.hooksPerformed)}</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Perfect Games</span>
								<span class="font-mono">{formatNumber(player.stats.killerPerfectGames)}</span>
							</div>
						</div>
					</div>

					<!-- Survivor Stats -->
					<div class="space-y-4">
						<h3 class="text-lg font-semibold text-blue-400">Survivor Statistics</h3>
						<div class="space-y-3">
							<div class="flex justify-between">
								<span class="text-neutral-300">Pips Earned</span>
								<span class="font-mono">{formatNumber(player.stats.survivorPips)}</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Escapes</span>
								<span class="font-mono">{formatNumber(player.stats.escapes)}</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Generator Progress</span>
								<span class="font-mono">{player.stats.generatorPct.toFixed(1)}%</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Healing Progress</span>
								<span class="font-mono">{player.stats.healPct.toFixed(1)}%</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Skill Check Success</span>
								<span class="font-mono">{formatNumber(player.stats.skillCheckSuccess)}</span>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-300">Perfect Games</span>
								<span class="font-mono">{formatNumber(player.stats.camperPerfectGames)}</span>
							</div>
						</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- Adepts Tab -->
		{#if activeTab === 'adepts'}
			<div
				id="adepts-panel"
				role="tabpanel"
				aria-labelledby="adepts-tab"
				class="space-y-8"
			>
				<div class="grid gap-8 lg:grid-cols-2">
					<!-- Survivor Adepts -->
					<div class="space-y-4">
						<div class="flex items-center justify-between">
							<h3 class="text-lg font-semibold text-blue-400">Survivor Adepts</h3>
							<span class="text-sm text-neutral-400">{adeptStats.survivor.unlocked}/{adeptStats.survivor.total}</span>
						</div>
						<div class="grid gap-2 sm:grid-cols-2">
							{#each Object.entries(player.achievements.adepts.survivors) as [survivor, unlocked]}
								<div class="flex items-center gap-2 rounded-lg border border-neutral-700 bg-neutral-900/30 p-3">
									<div class="h-2 w-2 rounded-full {unlocked ? 'bg-green-500' : 'bg-neutral-600'}"></div>
									<span class="text-sm {unlocked ? 'text-neutral-200' : 'text-neutral-400'}">{survivor}</span>
								</div>
							{/each}
						</div>
					</div>

					<!-- Killer Adepts -->
					<div class="space-y-4">
						<div class="flex items-center justify-between">
							<h3 class="text-lg font-semibold text-red-400">Killer Adepts</h3>
							<span class="text-sm text-neutral-400">{adeptStats.killer.unlocked}/{adeptStats.killer.total}</span>
						</div>
						<div class="grid gap-2 sm:grid-cols-2">
							{#each Object.entries(player.achievements.adepts.killers) as [killer, unlocked]}
								<div class="flex items-center gap-2 rounded-lg border border-neutral-700 bg-neutral-900/30 p-3">
									<div class="h-2 w-2 rounded-full {unlocked ? 'bg-green-500' : 'bg-neutral-600'}"></div>
									<span class="text-sm {unlocked ? 'text-neutral-200' : 'text-neutral-400'}">{killer}</span>
								</div>
							{/each}
						</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- Achievements Tab -->
		{#if activeTab === 'achievements'}
			<div
				id="achievements-panel"
				role="tabpanel"
				aria-labelledby="achievements-tab"
				class="space-y-6"
			>
				<div class="flex items-center justify-between">
					<h3 class="text-lg font-semibold">All Achievements</h3>
					<div class="text-sm text-neutral-400">
						{player.achievements.unlocked} of {player.achievements.total} unlocked
					</div>
				</div>

				{#if player.achievements.mapped.length > 0}
					<div class="grid gap-3">
						{#each player.achievements.mapped as achievement}
							<div class="flex items-center gap-3 rounded-lg border border-neutral-700 bg-neutral-900/30 p-4">
								<div class="h-3 w-3 rounded-full {achievement.unlocked ? 'bg-green-500' : 'bg-neutral-600'}"></div>
								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-2">
										<span class="text-sm font-medium {achievement.unlocked ? 'text-neutral-200' : 'text-neutral-400'}">
											{achievement.character}
										</span>
										<span class="inline-flex items-center rounded-full px-2 py-1 text-xs font-medium {achievement.type === 'killer' 
											? 'bg-red-900/50 text-red-300' 
											: 'bg-blue-900/50 text-blue-300'}">
											{achievement.type}
										</span>
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="flex h-32 items-center justify-center text-neutral-400">
						<span>No achievement data available</span>
					</div>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Data Sources Footer -->
	{#if player.sources.stats || player.sources.achievements}
		<footer class="border-t border-neutral-700 pt-6">
			<details class="text-sm text-neutral-400">
				<summary class="cursor-pointer hover:text-neutral-300">Data Sources</summary>
				<div class="mt-3 space-y-2 pl-4">
					{#if player.sources.stats}
						<div>
							<strong>Statistics:</strong> {player.sources.stats.source}
							{#if player.sources.stats.fetched_at}
								(fetched {new Date(player.sources.stats.fetched_at).toLocaleString()})
							{/if}
							{#if player.sources.stats.error}
								<span class="text-red-400">- {player.sources.stats.error}</span>
							{/if}
						</div>
					{/if}
					{#if player.sources.achievements}
						<div>
							<strong>Achievements:</strong> {player.sources.achievements.source}
							{#if player.sources.achievements.fetched_at}
								(fetched {new Date(player.sources.achievements.fetched_at).toLocaleString()})
							{/if}
							{#if player.sources.achievements.error}
								<span class="text-red-400">- {player.sources.achievements.error}</span>
							{/if}
						</div>
					{/if}
				</div>
			</details>
		</footer>
	{/if}
</article>
