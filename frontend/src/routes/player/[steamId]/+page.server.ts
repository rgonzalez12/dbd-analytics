import type { PageServerLoad } from './$types';
import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { ApiError, PlayerStatsWithAchievements } from '$lib/api/types';

export const load: PageServerLoad<{ data: PlayerStatsWithAchievements }> = async ({ params, fetch }) => {
	const { steamId } = params;
	
	try {
		const data = await api.player.combined(steamId, fetch);
		return { data };
	} catch (e) {
		const apiError = e as ApiError;
		
		if (apiError?.status === 404) {
			throw error(404, 'Player not found');
		}
		
		if (apiError?.status === 429) {
			const retryMessage = apiError.retryAfter 
				? `Rate limited. Try again in ${apiError.retryAfter} seconds.`
				: 'Rate limited';
			throw error(429, retryMessage);
		}
		
		throw error(502, 'Upstream error');
	}
};
