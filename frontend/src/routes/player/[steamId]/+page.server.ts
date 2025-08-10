import { api } from '$lib';
import { error } from '@sveltejs/kit';
import type { ApiError } from '$lib/api/types';

export async function load({ params, fetch }: { params: { steamId: string }, fetch: typeof globalThis.fetch }) {
	const { steamId } = params;

	try {
		const stats = await api.player.stats(steamId, fetch);
		return { steamId, stats, source: 'stats' as const };
	} catch {
		try {
			const combined = await api.player.combined(steamId, fetch);
			return { steamId, stats: combined, source: 'combined' as const };
		} catch (fallbackErr) {
			const apiError = fallbackErr as ApiError;
			if (apiError.status === 404) {
				throw error(404, 'Player not found or profile is private');
			}
			throw error(500, 'Unable to load player data');
		}
	}
}
