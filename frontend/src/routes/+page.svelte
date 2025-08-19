<script lang="ts">
	import { goto } from '$app/navigation';
	import { navigating } from '$app/stores';
	import { browser } from '$app/environment';
	
	let input = '';
	let error = '';

	function parseSteamInput(v: string): string {
		const trimmed = v.trim();
		
		// https://steamcommunity.com/profiles/<steamID64>
		const profileMatch = trimmed.match(/steamcommunity\.com\/profiles\/(\d+)/);
		if (profileMatch?.[1]) {
			return profileMatch[1];
		}
		
		// https://steamcommunity.com/id/<vanity>
		const vanityMatch = trimmed.match(/steamcommunity\.com\/id\/([^\/]+)/);
		if (vanityMatch?.[1]) {
			return vanityMatch[1];
		}
		
		// raw vanity or raw steamID64
		return trimmed;
	}

	function handleSubmit(event?: Event) {
		if (event) {
			event.preventDefault();
		}
		
		const rawInput = input.trim();
		if (!rawInput) {
			error = 'Please enter a Steam ID or vanity URL';
			return;
		}

		const steamId = parseSteamInput(rawInput);
		error = '';
		
		if (browser) {
			try {
				goto(`/player/${encodeURIComponent(steamId)}`);
			} catch (err) {
				console.error('Navigation error:', err);
				error = 'Navigation failed. Please try again.';
			}
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center p-4">
	<div class="search-container">
		<!-- Header -->
		<div class="text-center mb-16">
			<h1 class="text-6xl font-bold mb-6 horror-title">
				DBD Analytics
			</h1>
			<p class="text-gray-300 text-xl font-light horror-subtitle">
				Uncover the darkness within your Dead by Daylight statistics
			</p>
		</div>
		
		<!-- Search Form -->
		<form on:submit={handleSubmit} class="search-form">
			<input
				bind:value={input}
				placeholder="Steam ID, Profile URL, or Vanity Name"
				class="search-input"
				disabled={!!$navigating}
			/>
			<button 
				type="submit"
				class="search-button"
				disabled={!!$navigating}
			>
				{#if $navigating}
					<div class="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
				{:else}
					Search
				{/if}
			</button>
		</form>
		
		<!-- Additional Info Card -->
		<div class="bg-card p-8 mt-8">
			
			{#if error}
				<div class="mt-4 text-sm text-red-300 bg-red-900/30 border border-red-700/50 rounded-lg p-4">
					{error}
				</div>
			{/if}
		</div>

		<!-- Quick Info -->
		<div class="mt-12 text-center">
			<p class="text-gray-400 text-sm horror-subtitle">
				Enter your Steam profile URL, Steam ID, or vanity name to delve into the Entity's realm
			</p>
		</div>
	</div>
</div>
