<script lang="ts">
	import { goto } from '$app/navigation';
	import { navigating } from '$app/stores';
	
	let input = '';
	let error = '';

	function parseSteamInput(v: string): string {
		const trimmed = v.trim();
		
		// https://steamcommunity.com/profiles/<steamID64>
		const profileMatch = trimmed.match(/steamcommunity\.com\/profiles\/(\d+)/);
		if (profileMatch) {
			return profileMatch[1];
		}
		
		// https://steamcommunity.com/id/<vanity>
		const vanityMatch = trimmed.match(/steamcommunity\.com\/id\/([^\/]+)/);
		if (vanityMatch) {
			return vanityMatch[1];
		}
		
		// raw vanity or raw steamID64
		return trimmed;
	}

	function handleSubmit() {
		const rawInput = input.trim();
		if (!rawInput) {
			error = 'Please enter a Steam ID or vanity URL';
			return;
		}
		
		const steamId = parseSteamInput(rawInput);
		error = '';
		goto('/player/' + steamId);
	}
</script>

<form 
	class="mx-auto max-w-xl space-y-3" 
	on:submit|preventDefault={handleSubmit}
	aria-busy={$navigating ? 'true' : 'false'}
>
	<label for="steamId" class="block text-sm text-neutral-300">Steam64 ID or Vanity</label>
	<input
		id="steamId"
		class="w-full rounded-xl border border-neutral-700 bg-neutral-900 px-3 py-2 outline-none focus:border-neutral-400"
		bind:value={input}
		placeholder="7656119… or your vanity"
		disabled={!!$navigating}
	/>
	{#if error}
		<div class="text-sm text-red-400" aria-live="polite">{error}</div>
	{/if}
	<button 
		class="rounded-xl bg-white/90 px-3 py-2 text-neutral-900 hover:bg-white disabled:opacity-50" 
		type="submit"
		disabled={!!$navigating}
	>
		{$navigating ? 'Searching...' : 'Search'}
	</button>
</form>