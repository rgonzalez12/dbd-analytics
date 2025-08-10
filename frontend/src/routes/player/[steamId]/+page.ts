import { api } from '$lib';

export const load = async ({ params, fetch }: { params: { steamId: string }, fetch: typeof globalThis.fetch }) => {
	const { steamId } = params;

	try {
		const stats = await api.player.stats(steamId, fetch);
		return { steamId, stats, source: 'stats' as const };
	} catch {
		const combined = await api.player.combined(steamId, fetch);
		return { steamId, stats: combined, source: 'combined' as const };
	}
};
