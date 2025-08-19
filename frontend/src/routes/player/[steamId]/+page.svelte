<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import type { Player, Achievement, Adept } from '../../../lib/api/types';
	import type { PageData } from './$types';

	export let data: PageData;

	let player: Player | null = data.data;
	let activeTab: 'overview' | 'adepts' | 'stats' | 'achievements' = 'overview';

	// Process adept data
	$: adeptData = processAdeptData(player?.achievements?.mapped || []);

	// Process achievements by category
	$: achievementsByCategory = categorizeAchievements(player?.achievements?.mapped || []);

	function setActiveTab(tab: 'overview' | 'adepts' | 'stats' | 'achievements') {
		activeTab = tab;
	}

	function processAdeptData(achievements: Achievement[]): { survivors: Adept[], killers: Adept[] } {
		// Use the correct nested structure: achievements.adepts.survivors/killers
		if (player?.achievements?.adepts?.survivors && player?.achievements?.adepts?.killers) {
			const survivors: Adept[] = [];
			const killers: Adept[] = [];

			// Process survivors from the nested structure
			for (const [character, unlocked] of Object.entries(player.achievements.adepts.survivors)) {
				survivors.push({
					name: `Adept ${character}`,
					displayName: character,
					unlocked: unlocked,
					category: 'survivor' as const
				});
			}

			// Process killers from the nested structure
			for (const [character, unlocked] of Object.entries(player.achievements.adepts.killers)) {
				killers.push({
					name: `Adept ${character}`,
					displayName: character,
					unlocked: unlocked,
					category: 'killer' as const
				});
			}

			return {
				survivors: survivors.sort((a, b) => a.displayName.localeCompare(b.displayName)),
				killers: killers.sort((a, b) => a.displayName.localeCompare(b.displayName))
			};
		}

		// If the API structure isn't available, return empty arrays
		return { survivors: [], killers: [] };
	}

	function extractAdeptName(fullName: string): string {
		return fullName.replace(/^Adept\s*/i, '').trim();
	}

	function determineAdeptCategory(name: string): 'survivor' | 'killer' {
		const survivorNames = [
			'dwight', 'meg', 'claudette', 'jake', 'nea', 'laurie', 'ace', 'feng', 'david', 'quentin',
			'tapp', 'kate', 'adam', 'jeff', 'jane', 'ash', 'nancy', 'steve', 'yui', 'zarina',
			'cheryl', 'felix', 'elodie', 'yun-jin', 'jill', 'leon', 'mikaela', 'jonah', 'yoichi',
			'haddie', 'ada', 'rebecca', 'vittorio', 'thalita', 'renato', 'gabriel', 'nicolas',
			'ellen', 'alan', 'lara', 'sable', 'bill'
		];
		
		const killerNames = [
			'trapper', 'wraith', 'hillbilly', 'nurse', 'shape', 'hag', 'doctor', 'huntress', 
			'cannibal', 'nightmare', 'pig', 'clown', 'spirit', 'legion', 'plague', 'ghost face',
			'demogorgon', 'oni', 'deathslinger', 'executioner', 'blight', 'twins', 'trickster',
			'nemesis', 'cenobite', 'artist', 'onryo', 'dredge', 'mastermind', 'knight', 'skull merchant',
			'singularity', 'xenomorph', 'good guy', 'unknown', 'lich', 'dark lord', 'animatronic',
			'cage', 'ghoul', 'houndmaster', 'min', 'michonne', 'orela', 'rick', 'ripley', 'taurie',
			'trevor', 'troupe'
		];
		
		const nameToCheck = name.toLowerCase();
		
		// Check survivors first
		if (survivorNames.some(survivor => nameToCheck.includes(survivor))) {
			return 'survivor';
		}
		
		// Check killers
		if (killerNames.some(killer => nameToCheck.includes(killer))) {
			return 'killer';
		}
		
		// Default to killer if we can't determine (most adepts tend to be killer-focused)
		return 'killer';
	}

	function categorizeAchievements(achievements: Achievement[]): { general: Achievement[], killer: Achievement[], survivor: Achievement[] } {
		return achievements.reduce((categories, achievement) => {
			const name = (achievement.displayName || achievement.display_name || achievement.name || '').toLowerCase();
			
			if (name.includes('killer') || name.includes('sacrifice') || name.includes('hook') || name.includes('mori')) {
				categories.killer.push(achievement);
			} else if (name.includes('survivor') || name.includes('escape') || name.includes('repair') || name.includes('heal')) {
				categories.survivor.push(achievement);
			} else {
				categories.general.push(achievement);
			}
			
			return categories;
		}, { general: [] as Achievement[], killer: [] as Achievement[], survivor: [] as Achievement[] });
	}

	function formatUnlockTime(timestamp: number | undefined): string {
		if (!timestamp) return 'Unknown';
		return new Date(timestamp * 1000).toLocaleDateString();
	}

	function getGradeColor(value: number): string {
		if (value >= 1000000) return 'text-purple-400';
		if (value >= 100000) return 'text-blue-400';
		if (value >= 10000) return 'text-green-400';
		if (value >= 1000) return 'text-yellow-400';
		return 'text-gray-300';
	}
</script>

{#if player}
	<div class="container-custom py-8 section-spacing">
		<!-- Player Header -->
		<div class="bg-card player-header">
			{#if player.avatar}
				<img 
					src={player.avatar} 
					alt="{player.name}'s avatar" 
					class="player-avatar"
				/>
			{/if}
			<div class="player-info">
				<h1>{player.name}</h1>
				<div class="mt-2">
					<span class="steam-id-only">Steam ID: {player.steamId || player.id}</span>
				</div>
			</div>
		</div>

		<!-- Tab Navigation -->
		<div class="tab-navigation">
			{#each [
				{ id: 'overview' as const, label: 'Overview' },
				{ id: 'adepts' as const, label: 'Adepts' },
				{ id: 'stats' as const, label: 'Statistics' },
				{ id: 'achievements' as const, label: 'Achievements' }
			] as tab}
				<button
					class="tab-button {activeTab === tab.id ? 'active' : 'inactive'}"
					on:click={() => setActiveTab(tab.id)}
				>
					{tab.label}
				</button>
			{/each}
		</div>
		
		<!-- Tab Content -->
		<div>
			{#if activeTab === 'overview'}
				<div class="section-spacing">
					<!-- Stats Cards -->
					<div class="stats-grid">
						<div class="bg-card stats-card">
							<h3>Total Achievements</h3>
							<p class="text-white">{player.achievements?.mapped?.length || 0}</p>
						</div>
						
						<div class="bg-card stats-card">
							<h3>Completed</h3>
							<p class="text-success-glow">
								{player.achievements?.mapped?.filter(a => a.unlocked || a.achieved).length || 0}
							</p>
						</div>
						
						<div class="bg-card stats-card">
							<h3>Adepts Earned</h3>
							<p class="text-horror-primary">
								{(adeptData.survivors.filter(s => s.unlocked).length + adeptData.killers.filter(k => k.unlocked).length)}
							</p>
						</div>
						
						<div class="bg-card stats-card">
							<h3>Highest Prestige</h3>
							<p class="text-warning-glow">
								{player.stats?.header?.highestPrestige || 'N/A'}
							</p>
						</div>
					</div>

					<!-- Additional Stats Row -->
					<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
						<div class="bg-card stats-card">
							<h3>Completion Rate</h3>
							<p class="text-info-glow">
								{Math.round(((player.achievements?.mapped?.filter(a => a.unlocked || a.achieved).length || 0) / (player.achievements?.mapped?.length || 1)) * 100)}%
							</p>
						</div>
						
						<div class="bg-card stats-card">
							<h3>Grades</h3>
							<div class="flex gap-4 text-lg">
								{#if player.stats?.header?.killerGrade}
									<span class="text-horror-primary">Killer: {player.stats.header.killerGrade}</span>
								{/if}
								{#if player.stats?.header?.survivorGrade}
									<span class="text-success-glow">Survivor: {player.stats.header.survivorGrade}</span>
								{/if}
								{#if !player.stats?.header?.killerGrade && !player.stats?.header?.survivorGrade}
									<span class="text-gray-400">Not Available</span>
								{/if}
							</div>
						</div>
					</div>

					<!-- Recent Achievements -->
					{#if player.achievements?.mapped}
						<div class="bg-card p-8">
							<h2 class="text-2xl font-bold text-white mb-6">Recent Achievements</h2>
							{#if player.achievements.mapped.filter(a => (a.unlocked || a.achieved) && (a.unlock_time || a.unlockTime)).sort((a, b) => ((b.unlock_time || b.unlockTime) || 0) - ((a.unlock_time || a.unlockTime) || 0)).slice(0, 5).length > 0}
								<div class="achievement-grid">
									{#each player.achievements.mapped.filter(a => (a.unlocked || a.achieved) && (a.unlock_time || a.unlockTime)).sort((a, b) => ((b.unlock_time || b.unlockTime) || 0) - ((a.unlock_time || a.unlockTime) || 0)).slice(0, 5) as achievement}
										<div class="achievement-item achievement-unlocked">
											<span class="text-3xl">🏆</span>
											<div class="flex-1">
												<h3 class="font-semibold text-success-glow text-lg">{achievement.displayName || achievement.display_name || achievement.name}</h3>
												<p class="text-gray-300 mt-1">{achievement.description}</p>
												<p class="text-gray-500 text-sm mt-2">Unlocked: {formatUnlockTime(achievement.unlock_time || achievement.unlockTime)}</p>
											</div>
										</div>
									{/each}
								</div>
							{:else}
								<p class="text-slate-400 text-center py-8">No recent achievements found</p>
							{/if}
						</div>
					{/if}
				</div>

			{:else if activeTab === 'adepts'}
				<div class="section-spacing">
					<!-- Survivor Adepts -->
					<div class="bg-card p-8">
						<h2 class="text-2xl font-bold text-white mb-4">Survivor Adepts</h2>
						<p class="text-slate-400 mb-6 text-lg">
							{adeptData.survivors.filter(a => a.unlocked).length} of {adeptData.survivors.length} unlocked
						</p>
						<div class="adept-grid">
							{#each adeptData.survivors as adept}
								<div class="adept-item {adept.unlocked ? 'adept-unlocked' : 'adept-locked'}">
									<span class="text-2xl">{adept.unlocked ? '✅' : '💀'}</span>
									<span class="font-semibold text-lg {adept.unlocked ? 'text-success-glow' : 'text-gray-400'}">
										Adept {adept.displayName}
									</span>
								</div>
							{/each}
						</div>
					</div>

					<!-- Killer Adepts -->
					<div class="bg-card p-8">
						<h2 class="text-2xl font-bold text-white mb-4">Killer Adepts</h2>
						<p class="text-gray-400 mb-6 text-lg">
							{adeptData.killers.filter(a => a.unlocked).length} of {adeptData.killers.length} unlocked
						</p>
						<div class="adept-grid">
							{#each adeptData.killers as adept}
								<div class="adept-item {adept.unlocked ? 'adept-unlocked' : 'adept-locked'}">
									<span class="text-2xl">{adept.unlocked ? '🔪' : '💀'}</span>
									<span class="font-semibold text-lg {adept.unlocked ? 'text-horror-primary' : 'text-gray-400'}">
										Adept {adept.displayName}
									</span>
								</div>
							{/each}
						</div>
					</div>
				</div>

			{:else if activeTab === 'stats'}
				<div class="bg-card p-8">
					<h2 class="text-2xl font-bold text-white mb-6">Statistics</h2>
					{#if player.stats}
						<div class="stats-grid">
							{#each Object.entries(player.stats) as [key, value]}
								<div class="bg-slate-800/50 p-6 rounded-xl border border-slate-700/30">
									<h3 class="text-slate-400 capitalize text-sm font-medium mb-2">{key.replace(/_/g, ' ')}</h3>
									<p class="text-2xl font-bold {getGradeColor(Number(value))}">
										{typeof value === 'number' ? value.toLocaleString() : value}
									</p>
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-slate-400 text-center py-8">No statistics available</p>
					{/if}
				</div>

			{:else if activeTab === 'achievements'}
				<div class="section-spacing">
					{#if player.achievements?.mapped}
						{#each [
							{ title: 'General Achievements', achievements: achievementsByCategory.general },
							{ title: 'Killer Achievements', achievements: achievementsByCategory.killer },
							{ title: 'Survivor Achievements', achievements: achievementsByCategory.survivor }
						] as category}
							{#if category.achievements.length > 0}
								<div class="bg-card p-8">
									<h2 class="text-2xl font-bold text-white mb-4">{category.title}</h2>
									<p class="text-slate-400 mb-6 text-lg">
										{category.achievements.filter(a => a.unlocked || a.achieved).length} of {category.achievements.length} unlocked
									</p>
									<div class="achievement-grid">
										{#each category.achievements as achievement}
											<div class="achievement-item {(achievement.unlocked || achievement.achieved) ? 'achievement-unlocked' : 'achievement-locked'}">
												<span class="text-3xl">{(achievement.unlocked || achievement.achieved) ? '🏆' : '�'}</span>
												<div class="flex-1">
													<h3 class="font-semibold text-lg {(achievement.unlocked || achievement.achieved) ? 'text-success-glow' : 'text-gray-400'}">
														{achievement.displayName || achievement.display_name || achievement.name}
													</h3>
													{#if achievement.description}
														<p class="text-gray-400 mt-2">{achievement.description}</p>
													{/if}
													{#if (achievement.unlocked || achievement.achieved) && (achievement.unlock_time || achievement.unlockTime)}
														<p class="text-gray-500 text-sm mt-2">Unlocked: {formatUnlockTime(achievement.unlock_time || achievement.unlockTime)}</p>
													{/if}
												</div>
											</div>
										{/each}
									</div>
								</div>
							{/if}
						{/each}
					{:else}
						<div class="bg-card p-8 text-center">
							<p class="text-gray-400">No achievements found in the Entity's realm</p>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>
{:else}
	<div class="min-h-screen flex items-center justify-center">
		<div class="text-center bg-card p-12">
			<h1 class="text-3xl font-bold text-horror-primary mb-4 horror-title">Player Lost in the Fog</h1>
			<p class="text-gray-300 text-lg horror-subtitle">The Entity has consumed this player's data or their profile remains hidden in darkness.</p>
		</div>
	</div>
{/if}
