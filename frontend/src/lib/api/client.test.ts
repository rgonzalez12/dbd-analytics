import { describe, it, expect, vi, beforeEach, type MockedFunction } from 'vitest';
import { request, api } from './client';
import type { ApiError } from './types';

// Mock environment variables
vi.mock('$env/dynamic/public', () => ({
	env: { PUBLIC_API_BASE_URL: undefined }
}));

// Mock fetch
const mockFetch: MockedFunction<typeof fetch> = vi.fn();

beforeEach(() => {
	mockFetch.mockClear();
	vi.clearAllTimers();
	vi.useFakeTimers();
});

describe('API client', () => {
	describe('request function', () => {
		it('returns JSON on 200', async () => {
			const mockData = { test: 'data' };
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: () => Promise.resolve(mockData),
				headers: new Headers()
			} as Response);

			const result = await request<typeof mockData>('/test', {}, mockFetch);

			expect(result).toEqual(mockData);
			expect(mockFetch).toHaveBeenCalledWith('/api/test', expect.objectContaining({
				headers: expect.objectContaining({
					'Accept': 'application/json',
					'Content-Type': 'application/json'
				})
			}));
		});

		it('throws ApiError on 404', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 404,
				statusText: 'Not Found',
				text: () => Promise.resolve('Player not found'),
				headers: new Headers()
			} as Response);

			await expect(request('/test', {}, mockFetch)).rejects.toEqual({
				status: 404,
				message: 'Player not found'
			} satisfies ApiError);
		});

		it('throws ApiError with retryAfter on 429', async () => {
			const headers = new Headers();
			headers.set('Retry-After', '60');

			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 429,
				statusText: 'Too Many Requests',
				text: () => Promise.resolve('Rate limited'),
				headers
			} as Response);

			await expect(request('/test', {}, mockFetch)).rejects.toEqual({
				status: 429,
				message: 'Rate limited',
				retryAfter: 60
			} satisfies ApiError);
		});

		it('merges with existing AbortSignal', async () => {
			const controller = new AbortController();
			const signal = controller.signal;

			mockFetch.mockImplementation((_url: RequestInfo | URL, options?: RequestInit) => {
				// Verify that signal is passed through
				expect(options?.signal).toBeDefined();
				return Promise.resolve({
					ok: true,
					json: () => Promise.resolve({}),
					headers: new Headers()
				} as Response);
			});

			await request('/test', { signal }, mockFetch);
		});
	});

	describe('api convenience methods', () => {
		it('calls correct endpoint for player.combined', () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: () => Promise.resolve({}),
				headers: new Headers()
			} as Response);

			api.player.combined('12345', mockFetch);

			expect(mockFetch).toHaveBeenCalledWith(
				'/api/api/player/12345',
				expect.any(Object)
			);
		});

		it('calls correct endpoint for player.stats', () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: () => Promise.resolve({}),
				headers: new Headers()
			} as Response);

			api.player.stats('12345', mockFetch);

			expect(mockFetch).toHaveBeenCalledWith(
				'/api/api/player/12345/stats',
				expect.any(Object)
			);
		});

		it('calls correct endpoint for player.summary', () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: () => Promise.resolve({}),
				headers: new Headers()
			} as Response);

			api.player.summary('12345', mockFetch);

			expect(mockFetch).toHaveBeenCalledWith(
				'/api/api/player/12345/summary',
				expect.any(Object)
			);
		});

		it('uses relative /api URL when PUBLIC_API_BASE_URL is not set', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: () => Promise.resolve({}),
				headers: new Headers()
			} as Response);

			await request('/test', {}, mockFetch);

			expect(mockFetch).toHaveBeenCalledWith(
				'/api/test',
				expect.any(Object)
			);
		});

		it('throws schema validation error for invalid payload on player.combined', async () => {
			const invalidData = { invalid: 'payload' };
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: () => Promise.resolve(invalidData),
				headers: new Headers()
			} as Response);

			await expect(api.player.combined('12345', mockFetch))
				.rejects.toMatchObject({
					status: 502,
					message: 'Invalid payload'
				});
		});
	});
});
