import type { ApiError, PlayerSummary, PlayerStats, PlayerStatsWithAchievements } from './types';
import { browser } from '$app/environment';

const DEFAULT_TIMEOUT_MS = 10000;

function getBaseUrl(): string {
	if (browser && typeof window !== 'undefined') {
		// Check if PUBLIC_API_BASE_URL is defined in window.ENV or similar
		// For now, use relative URLs for dev proxy
		return '';
	}
	return ''; // Use relative URLs for dev proxy
}

function withTimeout(signal: AbortSignal | undefined, ms: number) {
	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), ms);
	
	// Merge with existing signal
	if (signal) {
		signal.addEventListener('abort', () => controller.abort());
	}
	
	return {
		signal: controller.signal,
		cancel: () => {
			clearTimeout(timeoutId);
		}
	};
}

export async function request<T>(
	path: string,
	init?: RequestInit & { timeoutMs?: number },
	customFetch = fetch
): Promise<T> {
	const { timeoutMs = DEFAULT_TIMEOUT_MS, ...rest } = init ?? {};
	const { signal, cancel } = withTimeout(rest.signal || undefined, timeoutMs);
	
	const url = `${getBaseUrl()}${path}`;
	
	try {
		const response = await customFetch(url, {
			...rest,
			signal,
			headers: {
				'Accept': 'application/json',
				'Content-Type': 'application/json',
				...rest.headers
			}
		});
		
		if (!response.ok) {
			const text = await response.text();
			const error: ApiError = {
				status: response.status,
				message: text || response.statusText
			};
			
			// Special handling for 429 rate limit
			if (response.status === 429) {
				const retryAfterHeader = response.headers.get('Retry-After');
				if (retryAfterHeader) {
					error.retryAfter = parseInt(retryAfterHeader, 10);
				}
			}
			
			throw error;
		}
		
		return await response.json();
	} finally {
		cancel();
	}
}

export const api = {
	player: {
		summary: (steamId: string, customFetch?: typeof fetch, init?: RequestInit & { timeoutMs?: number }) => 
			request<PlayerSummary>(`/api/player/${steamId}/summary`, init, customFetch),
		stats: (steamId: string, customFetch?: typeof fetch, init?: RequestInit & { timeoutMs?: number }) => 
			request<PlayerStats>(`/api/player/${steamId}/stats`, init, customFetch),
		combined: (steamId: string, customFetch?: typeof fetch, init?: RequestInit & { timeoutMs?: number }) => 
			request<PlayerStatsWithAchievements>(`/api/player/${steamId}`, init, customFetch)
	}
};
