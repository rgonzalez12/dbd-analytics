import type { ApiError } from './types';

const DEFAULT_TIMEOUT_MS = 12_000;

function withTimeout(signal: AbortSignal | undefined, ms: number) {
	const ctrl = new AbortController();
	const id = setTimeout(() => ctrl.abort(), ms);
	const combo = new AbortController();
	const onAbort = () => combo.abort();
	ctrl.signal.addEventListener('abort', onAbort);
	if (signal) signal.addEventListener('abort', onAbort);
	return { signal: combo.signal, cancel: () => { clearTimeout(id); ctrl.signal.removeEventListener('abort', onAbort); } };
}

async function request<T>(path: string, init?: RequestInit & { timeoutMs?: number }): Promise<T> {
	const { timeoutMs = DEFAULT_TIMEOUT_MS, ...rest } = init ?? {};
	const { signal, cancel } = withTimeout(rest.signal || undefined, timeoutMs);
	try {
		const res = await fetch(path, { ...rest, signal, headers: { 'accept': 'application/json', ...(rest?.headers ?? {}) } });
		if (!res.ok) {
			const text = await res.text();
			const err: ApiError = { status: res.status, message: text || res.statusText };
			throw err;
		}
		return (await res.json()) as T;
	} finally {
		cancel();
	}
}

export const api = {
	player: {
		summary: (steamId: string) => request(`/api/player/${steamId}/summary`),
		stats: (steamId: string) => request(`/api/player/${steamId}/stats`),
		combined: (steamId: string) => request(`/api/player/${steamId}`)
	}
};
