<script lang="ts">
	import type { PageData } from './$types';
	
	export let data: PageData;
	
	$: player = data.data;
	
	// Tab state
	let activeTab: 'overview' | 'adepts' | 'stats' | 'achievements' = 'overview';
	
	// Dead by Daylight character mappings (restored from your original work)
	const adeptMapping = {
		// Base Game Survivors
		"ACH_UNLOCK_DWIGHT_PERKS": { name: "dwight", type: "survivor", displayName: "Dwight Fairfield" },
		"ACH_UNLOCK_MEG_PERKS": { name: "meg", type: "survivor", displayName: "Meg Thomas" },
		"ACH_UNLOCK_CLAUDETTE_PERKS": { name: "claudette", type: "survivor", displayName: "Claudette Morel" },
		"ACH_USE_JAKE_PERKS": { name: "jake", type: "survivor", displayName: "Jake Park" },

		// DLC Survivors
		"ACH_USE_NEA_PERKS": { name: "nea", type: "survivor", displayName: "Nea Karlsson" },
		"ACH_DLC2_SURVIVOR_1": { name: "laurie", type: "survivor", displayName: "Laurie Strode" },
		"ACH_DLC3_SURVIVOR_3": { name: "ace", type: "survivor", displayName: "Ace Visconti" },
		"SURVIVOR7_ACHIEVEMENT_3": { name: "bill", type: "survivor", displayName: "Bill Overbeck" },
		"ACH_DLC4_SURVIVOR_3": { name: "feng", type: "survivor", displayName: "Feng Min" },
		"ACH_DLC5_SURVIVOR_3": { name: "david", type: "survivor", displayName: "David King" },
		"ACH_DLC7_SURVIVOR_3": { name: "quentin", type: "survivor", displayName: "Quentin Smith" },
		"ACH_DLC8_SURVIVOR_3": { name: "tapp", type: "survivor", displayName: "Detective Tapp" },
		"ACH_DLC9_SURVIVOR_3": { name: "kate", type: "survivor", displayName: "Kate Denson" },
		"ACH_CHAPTER9_SURVIVOR_3": { name: "adam", type: "survivor", displayName: "Adam Francis" },
		"ACH_CHAPTER10_SURVIVOR_3": { name: "jeff", type: "survivor", displayName: "Jeff Johansen" },
		"ACH_CHAPTER11_SURVIVOR_3": { name: "jane", type: "survivor", displayName: "Jane Romero" },
		"ACH_CHAPTER12_SURVIVOR_3": { name: "ash", type: "survivor", displayName: "Ash Williams" },
		"ACH_CHAPTER14_SURVIVOR_3": { name: "yui", type: "survivor", displayName: "Yui Kimura" },
		"NEW_ACHIEVEMENT_146_31": { name: "zarina", type: "survivor", displayName: "Zarina Kassir" },
		"ACH_CHAPTER16_SURVIVOR_3": { name: "cheryl", type: "survivor", displayName: "Cheryl Mason" },
		"ACH_CHAPTER17_SURVIVOR_3": { name: "felix", type: "survivor", displayName: "Felix Richter" },
		"ACH_CHAPTER18_SURVIVOR_3": { name: "elodie", type: "survivor", displayName: "Élodie Rakoto" },
		"ACH_CHAPTER19_SURVIVOR_3": { name: "yun-jin", type: "survivor", displayName: "Yun-Jin Lee" },
		"ACH_CHAPTER20_SURVIVOR_3": { name: "jill", type: "survivor", displayName: "Jill Valentine" },
		"ACH_CHAPTER20_SURVIVOR_2": { name: "leon", type: "survivor", displayName: "Leon S. Kennedy" },
		"NEW_ACHIEVEMENT_211_3": { name: "mikaela", type: "survivor", displayName: "Mikaela Reid" },
		"ACH_CHAPTER22_SURVIVOR_3": { name: "jonah", type: "survivor", displayName: "Jonah Vasquez" },
		"NEW_ACHIEVEMENT_211_15": { name: "yoichi", type: "survivor", displayName: "Yoichi Asakawa" },
		"NEW_ACHIEVEMENT_211_21": { name: "haddie", type: "survivor", displayName: "Haddie Kaur" },
		"NEW_ACHIEVEMENT_211_26_NAME": { name: "ada", type: "survivor", displayName: "Ada Wong" },
		"NEW_ACHIEVEMENT_211_27_NAME": { name: "rebecca", type: "survivor", displayName: "Rebecca Chambers" },
		"NEW_ACHIEVEMENT_245_1": { name: "vittorio", type: "survivor", displayName: "Vittorio Toscano" },
		"NEW_ACHIEVEMENT_245_6": { name: "thalita", type: "survivor", displayName: "Thalita Lyra" },
		"NEW_ACHIEVEMENT_245_7": { name: "renato", type: "survivor", displayName: "Renato Lyra" },
		"NEW_ACHIEVEMENT_245_13": { name: "gabriel", type: "survivor", displayName: "Gabriel Soma" },
		"NEW_ACHIEVEMENT_245_17": { name: "nicolas", type: "survivor", displayName: "Nicolas Cage" },
		"NEW_ACHIEVEMENT_245_23": { name: "ellen", type: "survivor", displayName: "Ellen Ripley" },
		"NEW_ACHIEVEMENT_245_29": { name: "alan", type: "survivor", displayName: "Alan Wake" },
		"NEW_ACHIEVEMENT_280_3": { name: "sable", type: "survivor", displayName: "Sable Ward" },
		"NEW_ACHIEVEMENT_280_10": { name: "troupe", type: "survivor", displayName: "The Troupe" },
		"NEW_ACHIEVEMENT_280_13": { name: "lara", type: "survivor", displayName: "Lara Croft" },
		"NEW_ACHIEVEMENT_280_19": { name: "trevor", type: "survivor", displayName: "Trevor Belmont" },
		"NEW_ACHIEVEMENT_280_25": { name: "taurie", type: "survivor", displayName: "Taurie Cain" },
		"NEW_ACHIEVEMENT_280_31": { name: "orela", type: "survivor", displayName: "Orela" },
		"NEW_ACHIEVEMENT_312_4": { name: "rick", type: "survivor", displayName: "Rick Grimes" },
		"NEW_ACHIEVEMENT_312_5": { name: "michonne", type: "survivor", displayName: "Michonne" },

		// Base Game Killers
		"ACH_UNLOCK_CHUCKLES_PERKS": { name: "trapper", type: "killer", displayName: "The Trapper" },
		"ACH_UNLOCKBANSHEE_PERKS": { name: "wraith", type: "killer", displayName: "The Wraith" },
		"ACH_UNLOCKHILLBILY_PERKS": { name: "hillbilly", type: "killer", displayName: "The Hillbilly" },
		"ACH_DLC1_KILLER_3": { name: "nurse", type: "killer", displayName: "The Nurse" },

		// DLC Killers
		"ACH_DLC2_KILLER_1": { name: "shape", type: "killer", displayName: "The Shape" },
		"ACH_DLC3_KILLER_3": { name: "hag", type: "killer", displayName: "The Hag" },
		"ACH_DLC4_KILLER_3": { name: "doctor", type: "killer", displayName: "The Doctor" },
		"ACH_DLC5_KILLER_3": { name: "huntress", type: "killer", displayName: "The Huntress" },
		"ACH_DLC6_KILLER_3": { name: "cannibal", type: "killer", displayName: "The Cannibal" },
		"ACH_DLC7_KILLER_3": { name: "nightmare", type: "killer", displayName: "The Nightmare" },
		"ACH_DLC8_KILLER_3": { name: "pig", type: "killer", displayName: "The Pig" },
		"ACH_DLC9_KILLER_3": { name: "clown", type: "killer", displayName: "The Clown" },
		"ACH_CHAPTER9_KILLER_3": { name: "spirit", type: "killer", displayName: "The Spirit" },
		"ACH_CHAPTER10_KILLER_3": { name: "legion", type: "killer", displayName: "The Legion" },
		"ACH_CHAPTER11_KILLER_3": { name: "plague", type: "killer", displayName: "The Plague" },
		"ACH_CHAPTER12_KILLER_3": { name: "ghostface", type: "killer", displayName: "The Ghost Face" },
		"ACH_CHAPTER14_KILLER_3": { name: "oni", type: "killer", displayName: "The Oni" },
		"NEW_ACHIEVEMENT_146_28": { name: "deathslinger", type: "killer", displayName: "The Deathslinger" },
		"ACH_CHAPTER16_KILLER_3": { name: "executioner", type: "killer", displayName: "The Executioner" },
		"ACH_CHAPTER17_KILLER_3": { name: "blight", type: "killer", displayName: "The Blight" },
		"ACH_CHAPTER18_KILLER_3": { name: "twins", type: "killer", displayName: "The Twins" },
		"ACH_CHAPTER19_KILLER_3": { name: "trickster", type: "killer", displayName: "The Trickster" },
		"ACH_CHAPTER20_KILLER_3": { name: "nemesis", type: "killer", displayName: "The Nemesis" },
		"ACH_CHAPTER21_KILLER_3": { name: "cenobite", type: "killer", displayName: "The Cenobite" },
		"ACH_CHAPTER22_KILLER_3": { name: "artist", type: "killer", displayName: "The Artist" },
		"NEW_ACHIEVEMENT_211_12": { name: "onryo", type: "killer", displayName: "The Onryō" },
		"NEW_ACHIEVEMENT_211_18": { name: "dredge", type: "killer", displayName: "The Dredge" },
		"NEW_ACHIEVEMENT_211_24_NAME": { name: "mastermind", type: "killer", displayName: "The Mastermind" },
		"NEW_ACHIEVEMENT_211_30": { name: "knight", type: "killer", displayName: "The Knight" },
		"NEW_ACHIEVEMENT_245_4": { name: "skull-merchant", type: "killer", displayName: "The Skull Merchant" },
		"NEW_ACHIEVEMENT_245_10": { name: "singularity", type: "killer", displayName: "The Singularity" },
		"NEW_ACHIEVEMENT_245_20": { name: "xenomorph", type: "killer", displayName: "The Xenomorph" },
		"NEW_ACHIEVEMENT_245_26": { name: "chucky", type: "killer", displayName: "The Good Guy" },
		"NEW_ACHIEVEMENT_280_0": { name: "unknown", type: "killer", displayName: "The Unknown" },
		"NEW_ACHIEVEMENT_280_7": { name: "vecna", type: "killer", displayName: "Vecna" },
		"NEW_ACHIEVEMENT_280_16": { name: "dark-lord", type: "killer", displayName: "The Dark Lord" },
		"NEW_ACHIEVEMENT_280_22": { name: "houndmaster", type: "killer", displayName: "The Houndmaster" },
		"NEW_ACHIEVEMENT_312_1": { name: "lich", type: "killer", displayName: "The Lich" },
		"NEW_ACHIEVEMENT_312_2": { name: "animatronic", type: "killer", displayName: "The Animatronic" },
		"NEW_ACHIEVEMENT_312_8": { name: "ghoul", type: "killer", displayName: "The Ghoul" }
	};
	
	// Tab navigation
	function setActiveTab(tab: 'overview' | 'adepts' | 'stats' | 'achievements') {
		activeTab = tab;
	}

	// Keyboard navigation for tabs
	function handleTabKeydown(event: KeyboardEvent, tab: 'overview' | 'adepts' | 'stats' | 'achievements') {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			setActiveTab(tab);
		}
	}

	// Categorize achievements by type using the original Player data structure
	// Note: Adept achievements are excluded here since they have their own dedicated tab
	$: achievementsByCategory = (() => {
		if (!player?.achievements?.mapped) return { killer: [], survivor: [], general: [] };

		const killer = [];
		const survivor = [];
		const general = [];

		for (const achievement of player.achievements.mapped) {
			// SKIP adept achievements - they have their own dedicated tab
			// Check both the mapping and the achievement type field for comprehensive exclusion
			if (Object.keys(adeptMapping).includes(achievement.id) || 
			    achievement.type === 'adept_survivor' || 
			    achievement.type === 'adept_killer') {
				continue; // Skip adept achievements entirely
			}
			// Categorize by achievement name patterns (non-adept only)
			else if (achievement.id.includes('KILLER') || achievement.id.includes('SACRIFICE') || achievement.id.includes('HOOK') || achievement.id.includes('MORI')) {
				killer.push(achievement);
			}
			else if (achievement.id.includes('SURVIVOR') || achievement.id.includes('ESCAPE') || achievement.id.includes('HEAL') || achievement.id.includes('UNHOOK')) {
				survivor.push(achievement);
			}
			else {
				general.push(achievement);
			}
		}

		return { killer, survivor, general };
	})();

	// Process adept achievements from the API response (46 survivors, 39 killers)
	$: adeptData = (() => {
		if (!player?.achievements?.adepts) return { survivors: [], killers: [] };
		
		const survivors = Object.entries(player.achievements.adepts.survivors || {}).map(([name, unlocked]) => ({
			name,
			displayName: name,
			unlocked: Boolean(unlocked)
		}));
		
		const killers = Object.entries(player.achievements.adepts.killers || {}).map(([name, unlocked]) => ({
			name,
			displayName: name,
			unlocked: Boolean(unlocked)
		}));
		
		return { survivors, killers };
	})();

	// Format unlock time
	function formatUnlockTime(unlockTime?: number): string {
		if (!unlockTime) return '';
		return new Date(unlockTime * 1000).toLocaleDateString();
	}

	function getGradeColor(value: number): string {
		if (value >= 17) return 'text-red-600'; // Iri grades
		if (value >= 13) return 'text-purple-600'; // Gold grades  
		if (value >= 9) return 'text-yellow-600'; // Silver grades
		if (value >= 5) return 'text-gray-600'; // Bronze grades
		return 'text-amber-700'; // Ash grades
	}
</script>

<svelte:head>
	<title>{player.name} - DBD Analytics</title>
</svelte:head>

{#if player}
	<div class="min-h-screen bg-gray-50">
		<!-- Header -->
		<div class="bg-white shadow-sm">
			<div class="max-w-7xl mx-auto px-4 py-6">
				<div class="flex items-center gap-4">
					{#if player.avatar}
						<img src={player.avatar} alt="{player.name}'s avatar" class="w-16 h-16 rounded-lg" />
					{/if}
					<div>
						<h1 class="text-3xl font-bold text-gray-900">{player.name}</h1>
						<p class="text-gray-600">Steam ID: {player.steamId || player.id}</p>
					</div>
				</div>
			</div>
		</div>

		<!-- Tab Navigation -->
		<div class="max-w-7xl mx-auto px-4">
			<nav class="flex space-x-8 mt-6" aria-label="Tabs">
				{#each [
					{ id: 'overview' as const, label: 'Overview' },
					{ id: 'adepts' as const, label: 'Adept Achievements' },
					{ id: 'stats' as const, label: 'Statistics' },
					{ id: 'achievements' as const, label: 'All Achievements' }
				] as tab}
					<button
						class="py-3 px-1 border-b-2 font-medium text-sm transition-colors {
							activeTab === tab.id
								? 'border-blue-500 text-blue-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}"
						on:click={() => setActiveTab(tab.id)}
						on:keydown={(e) => handleTabKeydown(e, tab.id)}
						tabindex="0"
					>
						{tab.label}
					</button>
				{/each}
			</nav>
		</div>

		<!-- Tab Content -->
		<div class="max-w-7xl mx-auto px-4 py-8">
			{#if activeTab === 'overview'}
				<div class="space-y-8">
					<!-- Quick Stats -->
					<div class="grid grid-cols-1 md:grid-cols-5 gap-6">
						<div class="bg-white p-6 rounded-lg shadow">
							<h3 class="text-lg font-medium text-gray-900">Total Achievements</h3>
							<p class="text-3xl font-bold text-blue-600">{player.achievements?.mapped?.length || 0}</p>
						</div>
						<div class="bg-white p-6 rounded-lg shadow">
							<h3 class="text-lg font-medium text-gray-900">Completed</h3>
							<p class="text-3xl font-bold text-green-600">
								{player.achievements?.mapped?.filter(a => a.achieved).length || 0}
							</p>
						</div>
						<div class="bg-white p-6 rounded-lg shadow">
							<h3 class="text-lg font-medium text-gray-900">Adepts Earned</h3>
							<p class="text-3xl font-bold text-purple-600">
								{adeptData.survivors.filter(a => a.unlocked).length + adeptData.killers.filter(a => a.unlocked).length}
							</p>
							<p class="text-sm text-gray-500">
								{adeptData.survivors.filter(a => a.unlocked).length}/{adeptData.survivors.length} Survivors, 
								{adeptData.killers.filter(a => a.unlocked).length}/{adeptData.killers.length} Killers
							</p>
						</div>
						<div class="bg-white p-6 rounded-lg shadow">
							<h3 class="text-lg font-medium text-gray-900">Highest Prestige</h3>
							<p class="text-3xl font-bold text-orange-600">
								{player.stats?.header?.highestPrestige || 0}
							</p>
						</div>
						<div class="bg-white p-6 rounded-lg shadow">
							<h3 class="text-lg font-medium text-gray-900">Profile Status</h3>
							<p class="text-sm font-medium {(player.public ?? true) ? 'text-green-600' : 'text-red-600'}">
								{(player.public ?? true) ? 'Public' : 'Private'}
							</p>
						</div>
					</div>

					<!-- Recent Achievements -->
					{#if player.achievements?.mapped}
						<div class="bg-white rounded-lg shadow">
							<div class="px-6 py-4 border-b">
								<h2 class="text-xl font-bold text-gray-900">Recent Achievements</h2>
							</div>
							<div class="p-6">
								{#if player.achievements.mapped.filter(a => a.achieved && a.unlockTime).sort((a, b) => (b.unlockTime || 0) - (a.unlockTime || 0)).slice(0, 5).length > 0}
									<div class="space-y-4">
										{#each player.achievements.mapped.filter(a => a.achieved && a.unlockTime).sort((a, b) => (b.unlockTime || 0) - (a.unlockTime || 0)).slice(0, 5) as achievement}
											<div class="flex items-center justify-between">
												<div>
													<h3 class="font-medium text-gray-900">{achievement.displayName || achievement.name}</h3>
													<p class="text-sm text-gray-600">{achievement.description}</p>
												</div>
												<span class="text-sm text-gray-500">{formatUnlockTime(achievement.unlockTime)}</span>
											</div>
										{/each}
									</div>
								{:else}
									<p class="text-gray-500 text-center py-8">No recent achievements to display.</p>
								{/if}
							</div>
						</div>
					{/if}
				</div>

			{:else if activeTab === 'adepts'}
				<div class="space-y-8">
					<!-- Survivor Adepts -->
					<div class="bg-white rounded-lg shadow">
						<div class="px-6 py-4 border-b">
							<h2 class="text-xl font-bold text-gray-900">Survivor Adepts</h2>
							<p class="text-sm text-gray-600">
								{adeptData.survivors.filter(a => a.unlocked).length} of {adeptData.survivors.length} unlocked
							</p>
						</div>
						<div class="p-6">
							<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
								{#each adeptData.survivors as adept}
									<div class="flex items-center gap-3 p-3 rounded-lg {adept.unlocked ? 'bg-green-50' : 'bg-gray-50'}">
										<div class="text-lg">
											{adept.unlocked ? '✅' : '❌'}
										</div>
										<div class="flex-1">
											<p class="font-medium {adept.unlocked ? 'text-green-800' : 'text-gray-600'}">
												Adept {adept.displayName}
											</p>
										</div>
									</div>
								{/each}
							</div>
						</div>
					</div>

					<!-- Killer Adepts -->
					<div class="bg-white rounded-lg shadow">
						<div class="px-6 py-4 border-b">
							<h2 class="text-xl font-bold text-gray-900">Killer Adepts</h2>
							<p class="text-sm text-gray-600">
								{adeptData.killers.filter(a => a.unlocked).length} of {adeptData.killers.length} unlocked
							</p>
						</div>
						<div class="p-6">
							<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
								{#each adeptData.killers as adept}
									<div class="flex items-center gap-3 p-3 rounded-lg {adept.unlocked ? 'bg-red-50' : 'bg-gray-50'}">
										<div class="text-lg">
											{adept.unlocked ? '✅' : '❌'}
										</div>
										<div class="flex-1">
											<p class="font-medium {adept.unlocked ? 'text-red-800' : 'text-gray-600'}">
												Adept {adept.displayName}
											</p>
										</div>
									</div>
								{/each}
							</div>
						</div>
					</div>
				</div>

			{:else if activeTab === 'stats'}
				<div class="space-y-8">
					{#if player.stats}
						<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
							{#each Object.entries(player.stats) as [key, value]}
								<div class="bg-white p-6 rounded-lg shadow">
									<h3 class="text-lg font-medium text-gray-900 capitalize">{key.replace(/_/g, ' ')}</h3>
									<p class="text-3xl font-bold {getGradeColor(Number(value))}">
										{typeof value === 'number' ? value.toLocaleString() : value}
									</p>
								</div>
							{/each}
						</div>
					{:else}
						<div class="bg-white rounded-lg shadow p-8 text-center">
							<p class="text-gray-500">No statistics available</p>
						</div>
					{/if}
				</div>

			{:else if activeTab === 'achievements'}
				<div class="space-y-8">
					{#if player.achievements?.mapped}
						<!-- Achievement Categories -->
						{#each [
							{ title: 'General Achievements', achievements: achievementsByCategory.general },
							{ title: 'Killer Achievements', achievements: achievementsByCategory.killer },
							{ title: 'Survivor Achievements', achievements: achievementsByCategory.survivor }
						] as category}
							{#if category.achievements.length > 0}
								<div class="bg-white rounded-lg shadow">
									<div class="px-6 py-4 border-b">
										<h2 class="text-xl font-bold text-gray-900">{category.title}</h2>
										<p class="text-sm text-gray-600">
											{category.achievements.filter(a => a.achieved).length} of {category.achievements.length} unlocked
										</p>
									</div>
									<div class="p-6">
										<div class="space-y-4">
											{#each category.achievements as achievement}
												<div class="flex items-start gap-4 p-4 rounded-lg {achievement.achieved ? 'bg-green-50' : 'bg-gray-50'}">
													<div class="w-6 h-6 rounded {achievement.achieved ? 'bg-green-500' : 'bg-gray-300'} flex-shrink-0 mt-1"></div>
													<div class="flex-1">
														<h3 class="font-medium {achievement.achieved ? 'text-green-800' : 'text-gray-700'}">
															{achievement.displayName || achievement.name}
														</h3>
														{#if achievement.description}
															<p class="text-sm text-gray-600 mt-1">{achievement.description}</p>
														{/if}
														{#if achievement.achieved && achievement.unlockTime}
															<p class="text-sm text-gray-500 mt-2">Unlocked: {formatUnlockTime(achievement.unlockTime)}</p>
														{/if}
													</div>
												</div>
											{/each}
										</div>
									</div>
								</div>
							{/if}
						{/each}
					{:else}
						<div class="bg-white rounded-lg shadow p-8 text-center">
							<p class="text-gray-500">No achievements available</p>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>
{:else}
	<div class="min-h-screen bg-gray-50 flex items-center justify-center">
		<div class="text-center">
			<h1 class="text-2xl font-bold text-gray-900 mb-4">Player Not Found</h1>
			<p class="text-gray-600">The requested player could not be found or their profile is private.</p>
		</div>
	</div>
{/if}
