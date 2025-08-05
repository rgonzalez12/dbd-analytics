<script lang="ts">
	let steamId = '';

	function handleSearch() {
		if (steamId.trim()) {
			// Navigate to player page
			window.location.href = `/player/${steamId}`;
		}
	}
</script>

<svelte:head>
	<title>DBD Analytics - Dead by Daylight Player Stats</title>
</svelte:head>

<div class="text-center">
	<h1 class="text-4xl font-bold text-white mb-8">Dead by Daylight Analytics</h1>
	<p class="text-lg text-gray-300 mb-8">Search for player statistics and achievements</p>

	<div class="max-w-md mx-auto">
		<div class="flex gap-4">
			<input
				type="text"
				bind:value={steamId}
				placeholder="Enter Steam ID or profile URL"
				class="flex-1 px-4 py-2 bg-dbd-gray border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:border-dbd-red"
				on:keydown={(e) => e.key === 'Enter' && handleSearch()}
			/>
			<button
				on:click={handleSearch}
				class="px-6 py-2 bg-dbd-red text-white rounded-lg hover:bg-red-700 transition-colors"
			>
				Search
			</button>
		</div>
	</div>

	<div class="mt-12 text-gray-400">
		<p>Enter a Steam ID, custom URL, or full Steam profile URL</p>
		<p class="text-sm mt-2">Example: 76561198000000000 or customURL</p>
	</div>
</div>
