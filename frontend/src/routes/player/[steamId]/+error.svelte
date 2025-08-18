<script lang="ts">
	import { page } from '$app/stores';
	
	// Make error completely optional
	export let error: any = null;
	
	$: steamId = $page.params.steamId;
	
	function getErrorTitle(error: any): string {
		const status = error?.status;
		switch (status) {
			case 404:
				return 'Player Not Found';
			case 429:
				return 'Rate Limited';
			case 502:
				return 'Service Unavailable';
			default:
				return 'Error Loading Player';
		}
	}
	
	function getErrorDescription(error: any): string {
		const status = error?.status;
		switch (status) {
			case 404:
				return 'This Steam profile could not be found or may be private.';
			case 429:
				return error?.retryAfter 
					? `Too many requests. Please try again in ${error.retryAfter} seconds.`
					: 'Too many requests. Please try again later.';
			case 502:
				return 'Steam API is currently unavailable. Please try again later.';
			default:
				return error?.message || 'An unexpected error occurred while loading player data.';
		}
	}
	
	function getErrorIcon(error: any): string {
		const status = error?.status;
		switch (status) {
			case 404:
				return '👤';
			case 429:
				return '⏱️';
			case 502:
				return '🔧';
			default:
				return '⚠️';
		}
	}
</script>

<svelte:head>
	<title>Error - DBD Analytics</title>
</svelte:head>

<article class="flex min-h-[400px] items-center justify-center">
	<div class="max-w-md text-center space-y-6">
		<div class="text-6xl" role="img" aria-label="Error icon">
			{getErrorIcon(error)}
		</div>
		
		<div class="space-y-2">
			<h1 class="text-2xl font-bold text-neutral-100">
				{getErrorTitle(error)}
			</h1>
			<p class="text-neutral-400">
				{getErrorDescription(error)}
			</p>
		</div>
		
		<div class="space-y-3">
			<div class="rounded-lg border border-neutral-700 bg-neutral-900/50 p-4 text-left">
				<div class="text-sm text-neutral-400">Steam ID</div>
				<div class="font-mono text-neutral-200">{steamId || 'Unknown'}</div>
			</div>
			
			{#if error?.status === 429 && error?.retryAfter}
				<div class="rounded-lg border border-orange-600 bg-orange-900/20 p-4">
					<div class="text-sm text-orange-300">
						Retry after {error.retryAfter} seconds
					</div>
				</div>
			{/if}
		</div>
		
		<div class="space-y-2">
			<a
				href="/"
				class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
			>
				← Back to Home
			</a>
			
			<button
				on:click={() => window.location.reload()}
				class="block w-full rounded-lg border border-neutral-600 bg-neutral-800 px-4 py-2 text-sm font-medium text-neutral-200 transition-colors hover:bg-neutral-700"
			>
				Try Again
			</button>
		</div>
	</div>
</article>
