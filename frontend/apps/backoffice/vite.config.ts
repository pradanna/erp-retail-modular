import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			'/uploads': {
				target: 'http://localhost:8088',
				changeOrigin: true
			},
			'/api/v1/uploads': {
				target: 'http://localhost:8088',
				changeOrigin: true
			}
		}
	}
});
