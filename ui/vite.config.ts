import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: 'http://localhost:8090',
				changeOrigin: true
			},
			'/graphql': {
				target: 'http://localhost:8090',
				changeOrigin: true
			}
		}
	},
	test: {
		include: ['src/**/*.test.ts'],
		environment: 'node',
		alias: {
			'$lib': '/Users/coby/Code/c360/semsage/ui/src/lib'
		}
	}
});
