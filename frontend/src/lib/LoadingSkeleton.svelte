<div class="space-y-6">
	<div class="animate-pulse bg-card rounded-lg p-6">
		<div class="h-8 w-48 rounded bg-neutral-800/50"></div>
		<div class="mt-2 h-4 w-32 rounded bg-neutral-800/50"></div>
	</div>
	
	<div class="grid gap-4 md:grid-cols-2">
		{#each Array(3) as _}
			<div class="animate-pulse bg-card rounded-lg p-4 space-y-3">
				<div class="h-5 w-24 rounded bg-neutral-800/50"></div>
				<div class="h-4 w-16 rounded bg-neutral-800/50"></div>
			</div>
		{/each}
	</div>
</div>
