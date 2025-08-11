import type { ApiError, Player } from './types';
import { toDomainPlayer } from './adapters';
import { env } from '$env/dynamic/public';

const DEFAULT_TIMEOUT_MS = 10000;

function getBaseUrl(): string {
	return env.PUBLIC_API_BASE_URL || '/api';
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
		
		// Check if response is JSON
		const contentType = response.headers.get('content-type');
		if (!contentType || !contentType.includes('application/json')) {
			throw {
				status: response.status,
				message: 'Non-JSON response'
			} as ApiError;
		}
		
		const jsonData = await response.json();
		return jsonData;
	} finally {
		cancel();
	}
}

export const api = {
	player: {
		combined: async (steamId: string, customFetch?: typeof fetch, init?: RequestInit & { timeoutMs?: number }): Promise<Player> => {
			console.log('API: Fetching player data for Steam ID:', steamId);
			const data = await request<unknown>(`/player/${steamId}`, init, customFetch);
			return toDomainPlayer(data);
		}
	}
};
