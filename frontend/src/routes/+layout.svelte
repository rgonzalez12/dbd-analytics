<script lang="ts">
	import '../app.css';
	import { navigating, page } from '$app/stores';
	import LoadingSkeleton from '$lib/LoadingSkeleton.svelte';
	
	// Determine which loading skeleton to show based on route
	$: isPlayerRoute = $page.route.id?.includes('/player/[steamId]');
</script>

<svelte:head>
	<title>DBD Analytics</title>
	<meta name="description" content="Dead by Daylight analytics and stats" />
</svelte:head>

<div class="min-h-screen">
	<div class="mx-auto max-w-7xl relative">
		<!-- Only show header on player pages -->
		{#if isPlayerRoute}
			<header class="p-6 bg-gradient-to-r from-black/60 via-red-950/10 to-black/60 backdrop-blur-sm">
				<div class="flex items-center justify-between">
					<a href="/" class="flex items-center space-x-3 text-red-400">
						<div class="p-2 rounded-lg bg-red-950/20 border border-red-900/30">
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path>
							</svg>
						</div>
						<span class="font-medium horror-subtitle text-gray-400">Back to Search</span>
					</a>
					<h1 class="text-xl font-medium text-gray-300" style="font-family: 'Inter', -apple-system, BlinkMacSystemFont, system-ui, sans-serif; font-weight: 500; letter-spacing: -0.015em;">DBD Analytics</h1>
				</div>
			</header>
		{/if}
		
		{#if $navigating}
			<LoadingSkeleton />
		{:else}
			<slot />
		{/if}
	</div>
</div>
