import type { PageServerLoad } from './$types';
import { api } from '$lib';
import { error } from '@sveltejs/kit';
import type { ApiError } from '$lib/api/types';

export const load: PageServerLoad = async ({ params, fetch }) => {
	const { steamId } = params;
	
	try {
		const combinedData = await api.player.combined(steamId, fetch);
		return { steamId, stats: combinedData, source: 'combined' as const };
	} catch (err) {
		const apiError = err as ApiError;
		if (apiError?.status === 404) {
			throw error(404, 'Player not found or profile is private');
		}
		throw error(500, 'Unable to load player data');
	}
};
