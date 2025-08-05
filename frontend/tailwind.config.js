/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			colors: {
				'dbd-red': '#b71c1c',
				'dbd-dark': '#1a1a1a',
				'dbd-gray': '#2a2a2a'
			}
		}
	},
	plugins: []
};
