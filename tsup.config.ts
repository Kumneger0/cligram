import { defineConfig } from 'tsup';

export default defineConfig({
	entry: ['src/index.ts'],
	splitting: false,
	sourcemap: true,
	external: ['bun'],
	clean: true
});
