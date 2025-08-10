import { api } from '$lib';

export const load = async ({ params, fetch }: { params: { steamId: string }, fetch: typeof globalThis.fetch }) => {
	const { steamId } = params;

	try {
		// Let our client use the page's fetch so the dev proxy applies.
		const originalFetch = globalThis.fetch;
		globalThis.fetch = fetch;
		const stats = await api.player.stats(steamId);
		globalThis.fetch = originalFetch;
		return { steamId, stats, source: 'stats' as const };
	} catch {
		const originalFetch = globalThis.fetch;
		globalThis.fetch = fetch;
		const combined = await api.player.combined(steamId);
		globalThis.fetch = originalFetch;
		return { steamId, stats: combined, source: 'combined' as const };
	}
};
