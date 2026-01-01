import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 3000,
		proxy: {
			'/api': {
				target: 'http://localhost:8025',
				changeOrigin: true
			}
		}
	},
	build: {
		target: 'esnext',
		outDir: 'build'
	},
	optimizeDeps: {
		exclude: ['svelte']
	}
});
