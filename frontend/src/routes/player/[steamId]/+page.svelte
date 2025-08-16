<script lang="ts">
	import type { PageData } from './$types';
	
	export let data: PageData;
	
	$: player = data.data;
	
	// Tab state
	let activeTab: 'stats' | 'adepts' | 'achievements' = 'stats';
	let achievementTab: 'all' | 'survivor' | 'killer' = 'all';
	
	// Tab navigation
	function setActiveTab(tab: 'stats' | 'adepts' | 'achievements') {
		activeTab = tab;
	}

	// Achievement tab navigation
	function setAchievementTab(tab: 'all' | 'survivor' | 'killer') {
		achievementTab = tab;
	}

	// Keyboard navigation for achievement tabs
	function handleAchievementTabKeydown(event: KeyboardEvent, tab: 'all' | 'survivor' | 'killer') {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			setAchievementTab(tab);
		}
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

	// Convert pips to rank/grade (Dead by Daylight uses grades now)
	function convertPipsToGrade(pips: number): { grade: string; roman: string; color: string } {
		if (pips >= 85) return { grade: 'Iridescent I', roman: 'Iri I', color: 'text-purple-400' };
		if (pips >= 80) return { grade: 'Iridescent II', roman: 'Iri II', color: 'text-purple-400' };
		if (pips >= 75) return { grade: 'Iridescent III', roman: 'Iri III', color: 'text-purple-400' };
		if (pips >= 70) return { grade: 'Iridescent IV', roman: 'Iri IV', color: 'text-purple-400' };
		if (pips >= 65) return { grade: 'Gold I', roman: 'Gold I', color: 'text-yellow-400' };
		if (pips >= 60) return { grade: 'Gold II', roman: 'Gold II', color: 'text-yellow-400' };
		if (pips >= 55) return { grade: 'Gold III', roman: 'Gold III', color: 'text-yellow-400' };
		if (pips >= 50) return { grade: 'Gold IV', roman: 'Gold IV', color: 'text-yellow-400' };
		if (pips >= 45) return { grade: 'Silver I', roman: 'Silver I', color: 'text-gray-300' };
		if (pips >= 40) return { grade: 'Silver II', roman: 'Silver II', color: 'text-gray-300' };
		if (pips >= 35) return { grade: 'Silver III', roman: 'Silver III', color: 'text-gray-300' };
		if (pips >= 30) return { grade: 'Silver IV', roman: 'Silver IV', color: 'text-gray-300' };
		if (pips >= 25) return { grade: 'Bronze I', roman: 'Bronze I', color: 'text-orange-600' };
		if (pips >= 20) return { grade: 'Bronze II', roman: 'Bronze II', color: 'text-orange-600' };
		if (pips >= 15) return { grade: 'Bronze III', roman: 'Bronze III', color: 'text-orange-600' };
		if (pips >= 10) return { grade: 'Bronze IV', roman: 'Bronze IV', color: 'text-orange-600' };
		if (pips >= 5) return { grade: 'Ash I', roman: 'Ash I', color: 'text-neutral-500' };
		if (pips >= 4) return { grade: 'Ash II', roman: 'Ash II', color: 'text-neutral-500' };
		if (pips >= 3) return { grade: 'Ash III', roman: 'Ash III', color: 'text-neutral-500' };
		if (pips >= 2) return { grade: 'Ash IV', roman: 'Ash IV', color: 'text-neutral-500' };
		return { grade: 'Unranked', roman: 'Unranked', color: 'text-neutral-600' };
	}

	// Format rarity percentage
	function formatRarity(rarity?: number): string {
		if (rarity === undefined || rarity === 0) return '';
		return `${rarity.toFixed(1)}%`;
	}

	// Get rarity color class
	function getRarityColor(rarity?: number): string {
		if (!rarity || rarity === 0) return 'text-neutral-500';
		if (rarity >= 50) return 'text-green-400'; // Common
		if (rarity >= 25) return 'text-yellow-400'; // Uncommon  
		if (rarity >= 10) return 'text-orange-400'; // Rare
		if (rarity >= 5) return 'text-red-400';     // Epic
		return 'text-purple-400';                   // Legendary
	}

	// Filter achievements by type
	let achievementSort: 'name' | 'rarity' | 'unlocked' | 'type' = 'name';
	
	// Separate achievements by category (excluding adepts from general achievements)
	$: achievementsByType = (() => {
		const all = player.achievements.mapped.filter(a => a.type !== 'adept');
		const survivor = all.filter(a => a.type === 'survivor');
		const killer = all.filter(a => a.type === 'killer');
		
		return { all, survivor, killer };
	})();
	
	$: currentAchievements = (() => {
		let achievements = achievementTab === 'all' 
			? achievementsByType.all
			: achievementsByType[achievementTab];
		
		// Sort achievements
		return achievements.sort((a, b) => {
			switch (achievementSort) {
				case 'unlocked':
					if (a.unlocked !== b.unlocked) return b.unlocked ? 1 : -1;
					break;
				case 'rarity':
					const aRarity = a.rarity || 100;
					const bRarity = b.rarity || 100;
					if (aRarity !== bRarity) return aRarity - bRarity; // Lower rarity = rarer
					break;
				case 'type':
					if (a.type !== b.type) return (a.type || '').localeCompare(b.type || '');
					break;
			}
			// Default to name sort
			return (a.name || a.character || a.id).localeCompare(b.name || b.character || b.id);
		});
	})();

	$: achievementStats = (() => {
		const byType = player.achievements.mapped.reduce((acc, a) => {
			const type = a.type || 'unknown';
			if (!acc[type]) acc[type] = { total: 0, unlocked: 0 };
			acc[type].total++;
			if (a.unlocked) acc[type].unlocked++;
			return acc;
		}, {} as Record<string, { total: number; unlocked: number }>);
		
		return byType;
	})();	// Calculate adept completion
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
								<span class="text-neutral-300">Grade</span>
								<span class="font-mono {convertPipsToGrade(player.stats.killerPips).color}">
									{convertPipsToGrade(player.stats.killerPips).roman}
								</span>
							</div>
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
								<span class="text-neutral-300">Grade</span>
								<span class="font-mono {convertPipsToGrade(player.stats.survivorPips).color}">
									{convertPipsToGrade(player.stats.survivorPips).roman}
								</span>
							</div>
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
				<!-- Achievement Tab Header -->
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<h3 class="text-lg font-semibold">Achievements</h3>
						<div class="text-sm text-neutral-400">
							{player.achievements.unlocked} of {player.achievements.total} total unlocked
						</div>
					</div>

					<!-- Achievement Sub-tabs -->
					<div class="border-b border-neutral-700">
						<nav class="flex space-x-8" aria-label="Achievement tabs">
							{#each [
								{ id: 'all', label: 'All Non-Adept', count: achievementsByType.all.length },
								{ id: 'survivor', label: 'Survivor', count: achievementsByType.survivor.length },
								{ id: 'killer', label: 'Killer', count: achievementsByType.killer.length }
							] as tab}
								<button
									role="tab"
									tabindex={achievementTab === tab.id ? 0 : -1}
									aria-selected={achievementTab === tab.id}
									class="border-b-2 px-1 py-2 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-neutral-900 {achievementTab === tab.id 
										? 'border-blue-500 text-blue-400' 
										: 'border-transparent text-neutral-400 hover:border-neutral-300 hover:text-neutral-200'}"
									on:click={() => setAchievementTab(tab.id as 'all' | 'survivor' | 'killer')}
									on:keydown={(e) => handleAchievementTabKeydown(e, tab.id as 'all' | 'survivor' | 'killer')}
								>
									{tab.label} ({tab.count})
								</button>
							{/each}
						</nav>
					</div>

					<!-- Achievement stats for current tab -->
					<div class="flex flex-wrap gap-2 text-xs">
						{#if achievementTab === 'all'}
							{#each Object.entries(achievementStats) as [type, stats]}
								{#if type !== 'adept'}
									<span class="inline-flex items-center rounded-full px-2 py-1 {type === 'survivor' 
										? 'bg-blue-900/50 text-blue-300' 
										: type === 'killer' 
										? 'bg-red-900/50 text-red-300'
										: 'bg-neutral-700/50 text-neutral-300'}">
										{type}: {stats.unlocked}/{stats.total}
									</span>
								{/if}
							{/each}
						{:else}
							{@const stats = achievementStats[achievementTab]}
							{#if stats}
								<span class="inline-flex items-center rounded-full px-2 py-1 {achievementTab === 'survivor' 
									? 'bg-blue-900/50 text-blue-300' 
									: 'bg-red-900/50 text-red-300'}">
									{achievementTab}: {stats.unlocked}/{stats.total}
								</span>
							{/if}
						{/if}
					</div>

					<!-- Sort controls -->
					<div class="flex items-center justify-between">
						<div class="text-sm text-neutral-400">
							Showing {currentAchievements.length} achievements
						</div>
						
						<div class="flex items-center gap-2 text-sm">
							<label for="achievement-sort" class="text-neutral-400">Sort by:</label>
							<select 
								id="achievement-sort"
								bind:value={achievementSort}
								class="rounded border border-neutral-600 bg-neutral-800 px-2 py-1 text-neutral-200 focus:border-blue-500 focus:outline-none"
							>
								<option value="name">Name</option>
								<option value="unlocked">Unlocked First</option>
								<option value="rarity">Rarity (Rarest First)</option>
								<option value="type">Type</option>
							</select>
						</div>
					</div>
				</div>

				{#if currentAchievements.length > 0}
					<div class="grid gap-4">
						{#each currentAchievements as achievement}
							<div class="group rounded-lg border border-neutral-700 bg-neutral-900/30 p-4 transition-all duration-200 hover:border-neutral-600 hover:bg-neutral-800/50 hover:shadow-lg">
								<div class="flex gap-4">
									<!-- Achievement Icon -->
									<div class="flex-shrink-0">
										{#if achievement.icon}
											<div class="relative">
												<img 
													src={achievement.icon} 
													alt=""
													class="h-16 w-16 rounded-lg border border-neutral-600 transition-all {achievement.unlocked ? 'border-green-500/30 shadow-md' : 'grayscale opacity-50'}"
													loading="lazy"
												/>
												{#if achievement.unlocked}
													<div class="absolute -bottom-1 -right-1 h-4 w-4 rounded-full bg-green-500 border-2 border-neutral-800"></div>
												{/if}
											</div>
										{:else}
											<div class="flex h-16 w-16 items-center justify-center rounded-lg border border-neutral-600 bg-neutral-700/50">
												<span class="text-2xl {achievement.unlocked ? '' : 'grayscale'}">
													{achievement.type === 'killer' ? '🔪' : achievement.type === 'survivor' ? '🏃' : achievement.type === 'adept' ? '🏆' : '⭐'}
												</span>
											</div>
										{/if}
									</div>

									<!-- Achievement Details -->
									<div class="min-w-0 flex-1 space-y-2">
										<!-- Name and status -->
										<div class="flex items-start justify-between gap-2">
											<div class="min-w-0 flex-1">
												<h4 class="text-base font-medium {achievement.unlocked ? 'text-neutral-200' : 'text-neutral-400'}">
													{achievement.name || achievement.character || achievement.id}
													{#if !achievement.name && achievement.character}
														<span class="text-xs text-neutral-500">(fallback)</span>
													{/if}
												</h4>
												{#if achievement.description}
													<p class="text-sm text-neutral-400 {achievement.hidden && !achievement.unlocked ? 'italic' : ''}">
														{achievement.hidden && !achievement.unlocked ? 'Hidden achievement' : achievement.description}
													</p>
												{:else if achievement.character}
													<p class="text-sm text-neutral-500 italic">
														{achievement.type} achievement for {achievement.character}
													</p>
												{/if}
											</div>
											<div class="flex items-center gap-2">
												<!-- Unlock status -->
												<div class="h-3 w-3 rounded-full {achievement.unlocked ? 'bg-green-500' : 'bg-neutral-600'}"></div>
												<!-- Type badge -->
												<span class="inline-flex items-center rounded-full px-2 py-1 text-xs font-medium {achievement.type === 'killer' 
													? 'bg-red-900/50 text-red-300' 
													: achievement.type === 'survivor'
													? 'bg-blue-900/50 text-blue-300'
													: achievement.type === 'adept'
													? 'bg-purple-900/50 text-purple-300'
													: 'bg-neutral-700/50 text-neutral-300'}">
													{achievement.type || 'unknown'}
												</span>
											</div>
										</div>

										<!-- Metadata row -->
										<div class="flex items-center justify-between text-xs text-neutral-500">
											<div class="flex items-center gap-4">
												{#if achievement.character}
													<span>Character: {achievement.character}</span>
												{/if}
												{#if achievement.unlock_time}
													<span>Unlocked: {new Date(achievement.unlock_time * 1000).toLocaleDateString()}</span>
												{/if}
												{#if achievement.hidden}
													<span class="text-yellow-400">Hidden</span>
												{/if}
											</div>
											{#if achievement.rarity && achievement.rarity > 0}
												<div class="flex items-center gap-1">
													<span class="text-neutral-400">Global:</span>
													<span class="{getRarityColor(achievement.rarity)} font-medium">
														{formatRarity(achievement.rarity)}
													</span>
												</div>
											{/if}
										</div>
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="flex h-32 items-center justify-center text-neutral-400">
						<span>No achievements found in this category</span>
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
