<script lang="ts">
	import { goto } from '$app/navigation';
	
	let steamId = '';

	function handleSearch() {
		if (steamId.trim()) {
			goto(`/player/${encodeURIComponent(steamId.trim())}`);
		}
	}
</script>

<svelte:head>
	<title>DBD Analytics - Dead by Daylight Player Stats</title>
</svelte:head>

<div class="container mx-auto px-4 py-12">
	<div class="text-center mb-12">
		<h1 class="text-5xl font-bold mb-4">Dead by Daylight Analytics</h1>
		<p class="text-xl text-gray-300 mb-8">Track your DBD stats and achievements</p>
		
		<div class="max-w-md mx-auto">
			<div class="flex gap-2">
				<input
					type="text"
					bind:value={steamId}
					placeholder="Enter Steam ID or username..."
					class="flex-1 px-4 py-3 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:border-red-700 focus:outline-none"
					on:keydown={(e) => e.key === 'Enter' && handleSearch()}
				/>
				<button
					on:click={handleSearch}
					class="px-6 py-3 bg-red-700 hover:bg-red-800 rounded-lg font-medium transition-colors"
				>
					Search
				</button>
			</div>
		</div>
		
		<div class="mt-6 text-gray-400 text-sm">
			<p>Enter a Steam ID, custom URL, or username</p>
		</div>
	</div>

	<div class="grid md:grid-cols-3 gap-8">
		<div class="bg-gray-800 p-6 rounded-lg">
			<h3 class="text-xl font-bold mb-2 text-red-700">Player Stats</h3>
			<p class="text-gray-300">View detailed survivor and killer statistics</p>
		</div>
		
		<div class="bg-gray-800 p-6 rounded-lg">
			<h3 class="text-xl font-bold mb-2 text-red-700">Achievements</h3>
			<p class="text-gray-300">Track your progress on all DBD achievements</p>
		</div>
		
		<div class="bg-gray-800 p-6 rounded-lg">
			<h3 class="text-xl font-bold mb-2 text-red-700">Analytics</h3>
			<p class="text-gray-300">Deep dive into your gameplay patterns</p>
		</div>
	</div>
</div>
