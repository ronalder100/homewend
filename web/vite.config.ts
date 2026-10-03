import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			// The Go binary embeds the build: it serves the files and, for any
			// route it has no file for, the fallback page.
			adapter: adapter({ pages: '../internal/web/dist', fallback: '200.html' })
		})
	]
});
