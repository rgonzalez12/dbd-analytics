<script lang="ts">
	export let data: { steamId: string; stats: Record<string, unknown>; source: 'stats' | 'combined' };
	
	const { steamId, stats, source } = data;
	
	function get<T>(obj: Record<string, unknown>, key: string): T | undefined {
		return obj?.[key] as T | undefined;
	}
</script>

<section class="space-y-6">
	<div class="rounded-2xl border border-neutral-800 p-4">
		<div class="flex items-center justify-between gap-4">
			<h2 class="text-xl font-semibold">Player {steamId}</h2>
			<span class="text-xs rounded-full border border-neutral-700 px-2 py-1 text-neutral-400">
				source: {source}
			</span>
		</div>

		<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
			<div class="rounded-xl border border-neutral-800 p-4">
				<div class="text-xs text-neutral-400">Persona</div>
				<div class="text-lg">{get<string>(stats, 'persona_name') ?? '—'}</div>
			</div>
			<div class="rounded-xl border border-neutral-800 p-4">
				<div class="text-xs text-neutral-400">Steam ID</div>
				<div class="text-lg">{steamId}</div>
			</div>
			<div class="rounded-xl border border-neutral-800 p-4">
				<div class="text-xs text-neutral-400">Matches</div>
				<div class="text-lg">{get<number>(stats, 'matches') ?? '—'}</div>
			</div>
		</div>
	</div>

	<details class="rounded-2xl border border-neutral-800 p-4">
		<summary class="cursor-pointer text-sm text-neutral-400 hover:text-neutral-300">
			Show raw data
		</summary>
		<pre class="mt-4 overflow-auto rounded-xl bg-black/40 p-4 text-xs">{JSON.stringify(stats, null, 2)}</pre>
	</details>
</section>
