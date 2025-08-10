<script lang="ts">
	import { page } from '$app/stores';

	$: error = $page.error;
</script>

<div class="space-y-3 rounded-2xl border border-red-500/30 bg-red-500/10 p-4">
	<h2 class="text-lg font-medium">Something went wrong</h2>
	<pre class="whitespace-pre-wrap text-sm/6 opacity-90">{JSON.stringify(error, null, 2)}</pre>
</div>
