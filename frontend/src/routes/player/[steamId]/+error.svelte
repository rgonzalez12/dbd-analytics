<script lang="ts">
	import { page } from '$app/stores';
	import type { ApiError } from '$lib/api/types';
	
	export let error: ApiError;
	
	$: steamId = $page.params.steamId;
	
	function getErrorTitle(status: number): string {
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
	
	function getErrorDescription(error: ApiError): string {
		switch (error.status) {
			case 404:
				return 'This Steam profile could not be found or may be private.';
			case 429:
				return error.retryAfter 
					? `Too many requests. Please try again in ${error.retryAfter} seconds.`
					: 'Too many requests. Please try again later.';
			case 502:
				return 'Steam API is currently unavailable. Please try again later.';
			default:
				return error.message || 'An unexpected error occurred while loading player data.';
		}
	}
	
	function getErrorIcon(status: number): string {
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
			{getErrorIcon(error.status)}
		</div>
		
		<div class="space-y-2">
			<h1 class="text-2xl font-bold text-neutral-100">
				{getErrorTitle(error.status)}
			</h1>
			<p class="text-neutral-400">
				{getErrorDescription(error)}
			</p>
		</div>
		
		<div class="space-y-3">
			<div class="rounded-lg border border-neutral-700 bg-neutral-900/50 p-4 text-left">
				<div class="text-sm text-neutral-400">Steam ID</div>
				<div class="font-mono text-neutral-200">{steamId}</div>
			</div>
			
			{#if error.status === 429 && error.retryAfter}
				<div class="rounded-lg border border-orange-600 bg-orange-900/20 p-4">
					<div class="text-sm text-orange-300">
						Retry after {error.retryAfter} seconds
					</div>
				</div>
			{/if}
		</div>
		
		<div class="flex flex-col gap-3 sm:flex-row sm:justify-center">
			<a 
				href="/"
				class="inline-flex items-center justify-center rounded-lg border border-neutral-600 bg-neutral-800 px-4 py-2 text-sm font-medium text-neutral-200 transition-colors hover:bg-neutral-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-neutral-900"
			>
				← Back to Home
			</a>
			
			{#if error.status !== 404}
				<button 
					on:click={() => window.location.reload()}
					class="inline-flex items-center justify-center rounded-lg border border-blue-600 bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-neutral-900"
				>
					Try Again
				</button>
			{/if}
		</div>
		
		{#if error.details}
			<details class="text-left text-sm text-neutral-400">
				<summary class="cursor-pointer hover:text-neutral-300">Technical Details</summary>
				<pre class="mt-2 overflow-auto rounded-lg bg-black/40 p-3 text-xs">{JSON.stringify(error.details, null, 2)}</pre>
			</details>
		{/if}
	</div>
</article>
