import { describe, it, expect } from 'vitest';

// Extract the parseSteamInput function for testing
function parseSteamInput(v: string): string {
	const trimmed = v.trim();
	
	// https://steamcommunity.com/profiles/<steamID64>
	const profileMatch = trimmed.match(/steamcommunity\.com\/profiles\/(\d+)/);
	if (profileMatch && profileMatch[1]) {
		return profileMatch[1];
	}
	
	// https://steamcommunity.com/id/<vanity>
	const vanityMatch = trimmed.match(/steamcommunity\.com\/id\/([^\/]+)/);
	if (vanityMatch && vanityMatch[1]) {
		return vanityMatch[1];
	}
	
	// raw vanity or raw steamID64
	return trimmed;
}

describe('Steam input normalization', () => {
	it('extracts steamID64 from profile URL', () => {
		const input = 'https://steamcommunity.com/profiles/76561198123456789';
		const result = parseSteamInput(input);
		expect(result).toBe('76561198123456789');
	});

	it('extracts vanity from vanity URL', () => {
		const input = 'https://steamcommunity.com/id/myvanityname';
		const result = parseSteamInput(input);
		expect(result).toBe('myvanityname');
	});

	it('returns raw steamID64 as-is', () => {
		const input = '76561198123456789';
		const result = parseSteamInput(input);
		expect(result).toBe('76561198123456789');
	});

	it('returns raw vanity name as-is', () => {
		const input = 'myvanityname';
		const result = parseSteamInput(input);
		expect(result).toBe('myvanityname');
	});

	it('trims whitespace', () => {
		const input = '  76561198123456789  ';
		const result = parseSteamInput(input);
		expect(result).toBe('76561198123456789');
	});
});
