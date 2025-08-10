// Simple runtime tests for API client functionality
// Run this in the browser console to test the API client

import { ApiError, apiGet, withQuery } from './client.js';

// Test withQuery function
console.group('Testing withQuery');

const url1 = withQuery('/api/test', {
	param1: 'value1',
	param2: 42,
	param3: true,
	param4: false
});
console.log('✓ Basic params:', url1); // Should be: /api/test?param1=value1&param2=42&param3=true&param4=false

const url2 = withQuery('/api/test', {
	param1: 'value1',
	param2: null,
	param3: undefined,
	param4: 'value4'
});
console.log('✓ Null/undefined filtered:', url2); // Should be: /api/test?param1=value1&param4=value4

const url3 = withQuery('/api/test', {
	param1: null,
	param2: undefined
});
console.log('✓ Empty params:', url3); // Should be: /api/test

console.groupEnd();

// Test ApiError class
console.group('Testing ApiError');

const error = new ApiError('Test error', 404, { message: 'Not found' });
console.log('✓ ApiError created:', {
	name: error.name,
	message: error.message,
	status: error.status,
	body: error.body
});

console.groupEnd();

// Example usage tests (these won't run without a real API)
console.group('API Client Usage Examples');

console.log(`
// Example 1: Basic GET request
try {
  const data = await apiGet('/api/player/76561198000000000');
  console.log('Player data:', data);
} catch (error) {
  if (error instanceof ApiError) {
    console.error('API Error:', error.status, error.body);
  }
}

// Example 2: With custom fetch (SvelteKit load function)
export async function load({ fetch }) {
  try {
    const playerData = await apiGet('/api/player/stats', fetch);
    return { playerData };
  } catch (error) {
    if (error instanceof ApiError) {
      return { error: error.message, status: error.status };
    }
    throw error;
  }
}

// Example 3: Query parameters
const searchUrl = withQuery('/api/search', {
  q: 'player name',
  limit: 10,
  include_achievements: true
});
const searchResults = await apiGet(searchUrl);
`);

console.groupEnd();

console.log('🧪 API Client tests completed. Check console output above.');
