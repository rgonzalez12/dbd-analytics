export class ApiError extends Error {
	readonly status: number;
	readonly body?: unknown;

	constructor(message: string, status: number, body?: unknown) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.body = body;
	}
}

export async function apiGet<T>(path: string, fetcher: typeof fetch = fetch): Promise<T> {
	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), 8000);

	try {
		const response = await fetcher(path, {
			signal: controller.signal,
			headers: {
				'Accept': 'application/json'
			}
		});

		clearTimeout(timeoutId);

		if (!response.ok) {
			let body: unknown;
			try {
				body = await response.json();
			} catch {
				body = await response.text();
			}
			throw new ApiError(
				`HTTP ${response.status}: ${response.statusText}`,
				response.status,
				body
			);
		}

		return await response.json();
	} catch (error) {
		clearTimeout(timeoutId);
		if (error instanceof ApiError) {
			throw error;
		}
		throw new ApiError(
			error instanceof Error ? error.message : 'Request failed',
			0
		);
	}
}

export function withQuery(
	path: string,
	params: Record<string, string | number | boolean | null | undefined>
): string {
	const searchParams = new URLSearchParams();

	for (const [key, value] of Object.entries(params)) {
		if (value != null) {
			searchParams.append(key, String(value));
		}
	}

	const queryString = searchParams.toString();
	return queryString ? `${path}?${queryString}` : path;
}
