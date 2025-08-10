<div class="space-y-6">
	<div class="animate-pulse">
		<div class="h-8 w-48 rounded bg-neutral-800"></div>
		<div class="mt-2 h-4 w-32 rounded bg-neutral-800"></div>
	</div>
	
	<div class="grid gap-4 md:grid-cols-2">
		{#each Array(3) as _}
			<div class="animate-pulse space-y-3 rounded-xl border border-neutral-700 bg-neutral-900/50 p-4">
				<div class="h-5 w-24 rounded bg-neutral-800"></div>
				<div class="h-4 w-16 rounded bg-neutral-800"></div>
			</div>
		{/each}
	</div>
</div>
