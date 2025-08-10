<script lang="ts">
	import { goto } from '$app/navigation';
	let input = '';

	function handleSubmit() {
		const id = input.trim();
		if (!id) return;
		goto(`/player/${encodeURIComponent(id)}`);
	}
</script>

<form class="mx-auto max-w-xl space-y-3" on:submit|preventDefault={handleSubmit}>
	<label for="steamId" class="block text-sm text-neutral-300">Steam64 ID or Vanity</label>
	<input
		id="steamId"
		class="w-full rounded-xl border border-neutral-700 bg-neutral-900 px-3 py-2 outline-none focus:border-neutral-400"
		bind:value={input}
		placeholder="7656119… or your vanity"
	/>
	<button class="rounded-xl bg-white/90 px-3 py-2 text-neutral-900 hover:bg-white" type="submit">Search</button>
</form>