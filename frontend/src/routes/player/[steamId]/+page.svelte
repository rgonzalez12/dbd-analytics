<script lang="ts">
	import type { Player } from '$lib/api/types';
	import { page } from '$app/stores';
	
	// Reactive access to page data and steamId
	$: steamId = $page.params.steamId;
	$: pdata = $page.data as { data: Player };
	$: player = pdata.data;
	$: displayName = player?.name ?? '—';
	$: matches = player?.matches ?? 0;
	$: achievements = player?.achievements;
	
	// Safe comparator for sorting
	const cmp = (a?: string, b?: string) => (a ?? '').localeCompare(b ?? '');
	
	// Reactive derivations with deduplication by character name
	$: adeptSurvivors = (achievements?.mapped ?? [])
		.filter(a => a.type === 'survivor' && a.character)
		.reduce((unique, achievement) => {
			const normalizedChar = achievement.character?.toLowerCase()?.trim() || '';
			// Only keep the first occurrence of each normalized character name
			if (!unique.some(u => (u.character?.toLowerCase()?.trim() || '') === normalizedChar)) {
				unique.push(achievement);
			}
			return unique;
		}, [] as any[])
		.sort((a, b) => cmp(a.character, b.character));
	$: adeptKillers = (() => {
		const killerAchievements = (achievements?.mapped ?? [])
			.filter(a => a.type === 'killer' && a.character);
		
		// Use robust deduplication
		return killerAchievements
			.reduce((unique, achievement) => {
				const normalizedChar = achievement.character?.toLowerCase()?.trim() || '';
				if (!unique.some(u => (u.character?.toLowerCase()?.trim() || '') === normalizedChar)) {
					unique.push(achievement);
				}
				return unique;
			}, [] as any[])
			.sort((a, b) => cmp(a.character, b.character));
	})();
	
	// Check for empty/private profile state
	$: hasNoData = matches === 0;
	
	// Helper function to properly format character names
	function formatCharacterName(name: string | undefined): string {
		if (!name) return '';
		
		// Handle special cases for proper formatting
		const specialCases: Record<string, string> = {
			'skull-merchant': 'Skull Merchant',
			'dark-lord': 'Dark Lord', 
			'yun-jin': 'Yun-Jin',
		};
		
		// Check for special cases first
		const lowerName = name.toLowerCase();
		if (specialCases[lowerName]) {
			return specialCases[lowerName];
		}
		
		// Capitalize first letter of each word
		return name.toLowerCase()
			.split(/[-\s]/)
			.map(word => word.charAt(0).toUpperCase() + word.slice(1))
			.join(name.includes('-') ? '-' : ' ');
	}
</script>

{#key steamId}
<section class="space-y-6">
	<div class="rounded-2xl border border-neutral-800 p-4">
		<div class="flex items-center justify-between gap-4">
			<h2 class="text-xl font-semibold">Player {player?.id ?? steamId}</h2>
			<span class="text-xs rounded-full border border-neutral-700 px-2 py-1 text-neutral-400">
				API Data
			</span>
		</div>

		<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
			<div class="rounded-xl border border-neutral-800 p-4">
				<div class="text-xs text-neutral-400">Display Name</div>
				<div class="text-lg">{displayName}</div>
			</div>
			<div class="rounded-xl border border-neutral-800 p-4">
				<div class="text-xs text-neutral-400">Steam ID</div>
				<div class="text-lg">{player?.id ?? steamId}</div>
			</div>
			<div class="rounded-xl border border-neutral-800 p-4">
				<div class="text-xs text-neutral-400">Total Matches</div>
				<div class="text-lg">{matches}</div>
			</div>
		</div>

		{#if hasNoData}
			<div class="mt-4 p-4 rounded-xl border border-yellow-600 bg-yellow-900/20">
				<p class="text-yellow-300">No stats available. This might be a private profile or the player hasn't played Dead by Daylight yet.</p>
			</div>
		{/if}
	</div>

	<!-- Achievement Data -->
	{#if achievements}
		<div class="rounded-2xl border border-neutral-800 p-4">
			<h3 class="text-xl font-semibold mb-4">Achievements</h3>
			
			<!-- Achievement Summary -->
			{#if (achievements.total ?? 0) > 0 || (achievements.unlocked ?? 0) > 0}
				<div class="mb-6">
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-3 mb-4">
						<div class="rounded-xl border border-neutral-800 p-4">
							<div class="text-xs text-neutral-400">Total</div>
							<div class="text-lg">{achievements.total ?? 0}</div>
						</div>
						<div class="rounded-xl border border-neutral-800 p-4">
							<div class="text-xs text-neutral-400">Unlocked</div>
							<div class="text-lg">{achievements.unlocked ?? 0}</div>
						</div>
						<div class="rounded-xl border border-neutral-800 p-4">
							<div class="text-xs text-neutral-400">Percentage</div>
							<div class="text-lg">{(achievements.total ?? 0) > 0 ? (((achievements.unlocked ?? 0) / (achievements.total ?? 1)) * 100).toFixed(1) : '0.0'}%</div>
						</div>
					</div>
					
					<!-- Adept Survivors -->
					{#if adeptSurvivors.length > 0}
						<div class="mb-6">
							<h4 class="text-lg font-medium mb-3 text-blue-300">
								Adept Survivors ({adeptSurvivors.filter(s => s.unlocked).length}/{adeptSurvivors.length})
							</h4>
							<div class="space-y-1">
								{#each adeptSurvivors as survivor}
									<div class="flex justify-between items-center py-2 px-3 rounded-lg bg-neutral-800/30 hover:bg-neutral-800/50 transition-colors">
										<span class="font-medium text-neutral-200">
											{formatCharacterName(survivor.character)}
										</span>
										<span class="{survivor.unlocked ? 'text-green-400 font-semibold' : 'text-red-400'} font-mono">
											{survivor.unlocked ? 'true' : 'false'}
										</span>
									</div>
								{/each}
							</div>
						</div>
					{:else}
						<div class="mb-6 p-4 border border-red-600 bg-red-900/20 rounded-lg">
							<p class="text-red-300">No survivor achievements found</p>
						</div>
					{/if}
					
					<!-- Adept Killers -->
					{#if adeptKillers.length > 0}
						<div class="mb-6">
							<h4 class="text-lg font-medium mb-3 text-red-300">
								Adept Killers ({adeptKillers.filter(k => k.unlocked).length}/{adeptKillers.length})
							</h4>
							<div class="space-y-1">
								{#each adeptKillers as killer}
									<div class="flex justify-between items-center py-2 px-3 rounded-lg bg-neutral-800/30 hover:bg-neutral-800/50 transition-colors">
										<span class="font-medium text-neutral-200">
											{formatCharacterName(killer.character)}
										</span>
										<span class="{killer.unlocked ? 'text-green-400 font-semibold' : 'text-red-400'} font-mono">
											{killer.unlocked ? 'true' : 'false'}
										</span>
									</div>
								{/each}
							</div>
						</div>
					{:else}
						<div class="mb-6 p-4 border border-red-600 bg-red-900/20 rounded-lg">
							<p class="text-red-300">No killer achievements found</p>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	{:else}
		<div class="rounded-2xl border border-neutral-800 p-4">
			<h3 class="text-xl font-semibold mb-4">Achievements</h3>
			<p class="text-neutral-400">No achievement data available</p>
		</div>
	{/if}

	<div class="rounded-2xl border border-neutral-800 p-4">
		<details>
			<summary class="cursor-pointer text-sm text-neutral-400 hover:text-neutral-300">Show raw JSON (debug)</summary>
			<pre class="mt-4 overflow-auto rounded-xl bg-black/40 p-4 text-xs">{JSON.stringify(pdata, null, 2)}</pre>
		</details>
	</div>
</section>
{/key}
