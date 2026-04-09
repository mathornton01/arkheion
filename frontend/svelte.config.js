import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Preprocess: TypeScript + CSS
	preprocess: vitePreprocess(),

	kit: {
		// adapter-node produces a Node.js server suitable for Docker
		adapter: adapter({
			out: 'build',
			precompress: true,
			envPrefix: '' // Use env vars as-is
		}),

		// API proxy: forward /api/* to the backend in dev
		// In production, nginx handles this
	}
};

export default config;
