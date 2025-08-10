import type { ApiError } from './types';

const TIMEOUT_MS = 12000;

function withTimeout(signal: AbortSignal | undefined, ms: number) {
	const ctrl = new AbortController();
	const id = setTimeout(() => ctrl.abort(), ms);
	const combo = new AbortController();
	const onAbort = () => combo.abort();
	
	ctrl.signal.addEventListener('abort', onAbort);
	signal?.addEventListener('abort', onAbort);
	
	return {
		signal: combo.signal,
		cancel: () => {
			clearTimeout(id);
			ctrl.signal.removeEventListener('abort', onAbort);
		}
	};
}

async function request<T>(path: string, init?: RequestInit & { timeoutMs?: number }, customFetch = fetch): Promise<T> {
	const { timeoutMs = TIMEOUT_MS, ...rest } = init ?? {};
	const { signal, cancel } = withTimeout(rest.signal || undefined, timeoutMs);
	
	try {
		const res = await customFetch(path, {
			...rest,
			signal,
			headers: { accept: 'application/json', ...rest.headers }
		});
		
		if (!res.ok) {
			const text = await res.text();
			throw { status: res.status, message: text || res.statusText } satisfies ApiError;
		}
		
		return res.json();
	} finally {
		cancel();
	}
}

export const api = {
	player: {
		summary: (steamId: string, customFetch?: typeof fetch) => request(`/api/player/${steamId}/summary`, undefined, customFetch),
		stats: (steamId: string, customFetch?: typeof fetch) => request(`/api/player/${steamId}/stats`, undefined, customFetch),
		combined: (steamId: string, customFetch?: typeof fetch) => request(`/api/player/${steamId}`, undefined, customFetch)
	}
};
