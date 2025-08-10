<script lang="ts">
	import { navigating } from '$app/stores';
	import LoadingSkeleton from '$lib/LoadingSkeleton.svelte';
</script>

<svelte:head>
	<title>DBD Analytics</title>
	<meta name="description" content="Dead by Daylight analytics and stats" />
</svelte:head>

<div class="mx-auto max-w-6xl p-6">
	<header class="mb-8 flex items-center justify-between">
		<h1 class="text-2xl font-semibold tracking-tight">DBD Analytics</h1>
		<a href="/" class="text-sm text-neutral-400 hover:text-neutral-200 transition">Home</a>
	</header>
	
	{#if $navigating}
		<LoadingSkeleton />
	{:else}
		<slot />
	{/if}
</div>
