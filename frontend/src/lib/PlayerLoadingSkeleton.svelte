<script lang="ts">
	// Loading skeleton component for player pages
</script>

<article class="space-y-8">
	<!-- Player Header Loading -->
	<header class="space-y-4">
		<div class="space-y-2">
			<div class="h-9 w-64 animate-pulse rounded bg-neutral-800"></div>
			<div class="flex items-center gap-4">
				<div class="h-4 w-32 animate-pulse rounded bg-neutral-800"></div>
				<div class="h-4 w-2 animate-pulse rounded bg-neutral-800"></div>
				<div class="h-4 w-36 animate-pulse rounded bg-neutral-800"></div>
			</div>
		</div>
		
		<!-- Quick Stats Loading -->
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each Array(4) as _}
				<div class="animate-pulse rounded-lg border border-neutral-700 bg-neutral-900/50 p-4">
					<div class="h-4 w-20 rounded bg-neutral-800"></div>
					<div class="mt-2 h-8 w-16 rounded bg-neutral-800"></div>
				</div>
			{/each}
		</div>
	</header>

	<!-- Tab Navigation Loading -->
	<div class="border-b border-neutral-700">
		<div class="flex space-x-8">
			{#each Array(3) as _}
				<div class="h-12 w-24 animate-pulse rounded bg-neutral-800"></div>
			{/each}
		</div>
	</div>

	<!-- Content Loading -->
	<div class="min-h-[400px] space-y-6">
		<div class="grid gap-6 lg:grid-cols-2">
			{#each Array(2) as _}
				<div class="space-y-4">
					<div class="h-6 w-32 animate-pulse rounded bg-neutral-800"></div>
					<div class="space-y-3">
						{#each Array(6) as _}
							<div class="flex justify-between">
								<div class="h-4 w-24 animate-pulse rounded bg-neutral-800"></div>
								<div class="h-4 w-16 animate-pulse rounded bg-neutral-800"></div>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	</div>
</article>
