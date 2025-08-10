<script lang="ts">
	import type { PlayerStatsSurface } from '$lib/api/types';
	export let data: { steamId: string; stats: PlayerStatsSurface | Record<string, unknown>; source: 'merged' };
	
	const s = data.stats as PlayerStatsSurface;
	const persona = s.persona_name ?? '—';
	const matches = s.matches ?? undefined;
</script>

<section class="space-y-6">
	<div class="rounded-2xl border border-neutral-800 p-4">
		<div class="flex items-center justify-between gap-4">
			<h2 class="text-xl font-semibold">Player {data.steamId}</h2>
			<span class="text-xs rounded-full border border-neutral-700 px-2 py-1 text-neutral-400">
				source: {data.source}
			</span>
		</div>

		<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
			<div class="rounded-xl border border-neutral-800 p-4">
				<div class="text-xs text-neutral-400">Persona</div>
				<div class="text-lg">{persona}</div>
			</div>
			<div class="rounded-xl border border-neutral-800 p-4">
				<div class="text-xs text-neutral-400">Steam ID</div>
				<div class="text-lg">{data.steamId}</div>
			</div>
			<div class="rounded-xl border border-neutral-800 p-4">
				<div class="text-xs text-neutral-400">Matches</div>
				<div class="text-lg">{matches ?? '—'}</div>
			</div>
		</div>
	</div>

	<div class="rounded-2xl border border-neutral-800 p-4">
		<details>
			<summary class="cursor-pointer text-sm text-neutral-400 hover:text-neutral-300">Show raw JSON (debug)</summary>
			<pre class="mt-4 overflow-auto rounded-xl bg-black/40 p-4 text-xs">{JSON.stringify(data.stats, null, 2)}</pre>
		</details>
	</div>
</section>
