import type { PageServerLoad } from './$types';
import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { ApiError, PlayerStatsWithAchievements } from '$lib/api/types';

export const load: PageServerLoad = async ({ params, fetch }): Promise<{ data: PlayerStatsWithAchievements }> => {
	const { steamId } = params;
	
	try {
		const data = await api.player.combined(steamId, fetch);
		return { data };
	} catch (err) {
		const apiError = err as ApiError;
		
		if (apiError?.status === 404) {
			throw error(404, 'Player not found');
		}
		
		if (apiError?.status === 429) {
			const message = apiError.retryAfter 
				? `Rate limited. Retry after ${apiError.retryAfter} seconds.`
				: 'Rate limited';
			throw error(429, message);
		}
		
		throw error(502, 'Upstream error');
	}
};
