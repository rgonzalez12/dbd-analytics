<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	$: status = $page.status;
	$: message = $page.error?.message || '';
	$: isPlayerContext = $page.url.pathname.includes('/player/');
	
	function getErrorTitle(status: number): string {
		if (status === 404) return 'Player Not Found';
		if (status === 429) return 'Rate Limited';
		if (status === 503) return 'Service Unavailable';
		return 'Something Went Wrong';
	}
	
	function getErrorMessage(status: number, message: string, isPlayerContext: boolean): string {
		if (status === 404 && isPlayerContext) {
			return 'This Steam profile might be private, the ID might be incorrect, or the player might not have Dead by Daylight data.';
		}
		if (status === 429) {
			const retryMatch = message?.match(/Retry after (\d+) seconds/);
			const retrySeconds = retryMatch?.[1] ? parseInt(retryMatch[1]) : null;
			return retrySeconds 
				? `Try again in ${retrySeconds} seconds.`
				: 'Try again in a few seconds.';
		}
		if (status === 503) {
			return message || 'Steam API is currently unavailable. Please try again later.';
		}
		if (status >= 500) {
			return 'Server error. Please try again later.';
		}
		return message || 'An unexpected error occurred.';
	}
</script>

<div class="mx-auto max-w-2xl p-6 text-center">
	<div class="space-y-4">
		<h1 class="text-2xl font-bold">{getErrorTitle(status)}</h1>
		<p class="text-neutral-300">
			{getErrorMessage(status, message, isPlayerContext)}
		</p>
		<button 
			class="rounded-xl bg-white/90 px-4 py-2 text-neutral-900 hover:bg-white"
			on:click={() => goto('/')}
		>
			{status === 404 && isPlayerContext ? 'Try Another ID' : 'Go Home'}
		</button>
	</div>
	
	<details class="mt-8 text-left">
		<summary class="cursor-pointer text-sm text-neutral-500 hover:text-neutral-400">Show details</summary>
		<pre class="mt-2 overflow-auto rounded bg-neutral-900 p-4 text-xs text-neutral-300">{JSON.stringify($page.error, null, 2)}</pre>
	</details>
</div>
