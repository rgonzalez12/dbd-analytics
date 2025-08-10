import type { PageServerLoad } from './$types';
import { api } from '$lib';
import { error } from '@sveltejs/kit';
import type { ApiError } from '$lib/api/types';

export const load: PageServerLoad = async ({ params, fetch }) => {
	const { steamId } = params;
	const [summary, stats] = await Promise.allSettled([
		api.player.summary(steamId, fetch),
		api.player.stats(steamId, fetch)
	]);
	const okSummary = summary.status === 'fulfilled' ? summary.value : null;
	const okStats = stats.status === 'fulfilled' ? stats.value : null;

	if (okSummary || okStats) {
		return { steamId, stats: { ...(okSummary ?? {}), ...(okStats ?? {}) }, source: 'merged' as const };
	}
	const err = (summary.status === 'rejected' ? summary.reason : stats.status === 'rejected' ? stats.reason : null) as ApiError | null;
	if (err?.status === 404) throw error(404, 'Player not found or profile is private');
	throw error(500, 'Unable to load player data');
};
