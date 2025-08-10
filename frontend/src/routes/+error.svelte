<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	$: isPlayerNotFound = $page.status === 404 && $page.url.pathname.includes('/player/');
</script>

<div class="mx-auto max-w-2xl p-6 text-center">
	{#if isPlayerNotFound}
		<div class="space-y-4">
			<h1 class="text-2xl font-bold">Player Not Found</h1>
			<p class="text-neutral-300">
				This Steam profile might be private, the ID might be incorrect, or the player might not have Dead by Daylight data.
			</p>
			<button 
				class="rounded-xl bg-white/90 px-4 py-2 text-neutral-900 hover:bg-white"
				on:click={() => goto('/')}
			>
				Try Another ID
			</button>
		</div>
	{:else}
		<div class="space-y-4">
			<h1 class="text-2xl font-bold">Something went wrong</h1>
			<p class="text-neutral-300">Status: {$page.status}</p>
			<button 
				class="rounded-xl bg-white/90 px-4 py-2 text-neutral-900 hover:bg-white"
				on:click={() => goto('/')}
			>
				Go Home
			</button>
		</div>
	{/if}
	
	<details class="mt-8 text-left">
		<summary class="cursor-pointer text-sm text-neutral-500 hover:text-neutral-400">Show details</summary>
		<pre class="mt-2 overflow-auto rounded bg-neutral-900 p-4 text-xs text-neutral-300">{JSON.stringify($page.error, null, 2)}</pre>
	</details>
</div>
