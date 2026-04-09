import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],

	server: {
		port: 5173,
		host: '0.0.0.0',
		proxy: {
			// In development, proxy API requests to the Go backend
			'/api': {
				target: process.env.INTERNAL_API_BASE_URL?.replace('/api/v1', '') || 'http://localhost:8080',
				changeOrigin: true
			}
		}
	},

	// Prevent vite from trying to resolve pdfjs worker as an ES module
	optimizeDeps: {
		exclude: ['pdfjs-dist']
	},

	build: {
		sourcemap: false,
		rollupOptions: {
			output: {
				// Split large vendor chunks (pdfjs excluded — handled as external)
				manualChunks: {
					epubjs: ['epubjs'],
					zxing: ['@zxing/library']
				}
			}
		}
	},

	// Required for pdfjs-dist worker
	worker: {
		format: 'es'
	}
});
