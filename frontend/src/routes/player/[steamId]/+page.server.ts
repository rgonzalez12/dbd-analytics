import type { PageServerLoad } from './$types';
import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { ApiError, PlayerStatsWithAchievements } from '$lib/api/types';

export const load: PageServerLoad<{ data: PlayerStatsWithAchievements }> = async ({ params, fetch }) => {
	const { steamId } = params;
	
	try {
		const data = await api.player.combined(steamId, fetch);
		return { data };
	} catch (err) {
		const apiError = err as ApiError;
		console.error('Player data fetch error:', apiError);
		
		if (apiError?.status === 404) {
			throw error(404, 'Player not found');
		}
		
		if (apiError?.status === 429) {
			const message = apiError.retryAfter 
				? `Rate limited. Retry after ${apiError.retryAfter} seconds.`
				: 'Rate limited';
			throw error(429, message);
		}
		
		// Check if it's a Steam API authentication issue
		if (apiError?.status === 500 || apiError?.status === 502) {
			const errorMessage = apiError.message?.toLowerCase() || '';
			if (errorMessage.includes('403') && errorMessage.includes('steam')) {
				throw error(503, 'Steam API access issue. The player profile might be private or the Steam ID might be invalid.');
			}
		}
		
		throw error(502, `API error: ${apiError?.message || 'Unknown upstream error'}`);
	}
};
