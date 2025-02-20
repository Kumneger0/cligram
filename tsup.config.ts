import { fixImportsPlugin } from 'esbuild-fix-imports-plugin'
import { defineConfig } from 'tsup'

export default defineConfig({
    entry: ['src/index.ts'],
    splitting: false,
    sourcemap: true,
    clean: true,
    plugins: [fixImportsPlugin()],
})
