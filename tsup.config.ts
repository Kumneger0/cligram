import { fixImportsPlugin } from 'esbuild-fix-imports-plugin'
import { defineConfig } from 'tsup'
import { wasmLoader } from 'esbuild-plugin-wasm';

export default defineConfig({
  entry: ['src/index.ts'],
  splitting: false,
  sourcemap: true,
  clean: true,
  bundle: true,
  // noExternal: [/(.*)/],
})




const trackImports = {
  name: 'track-imports',
  setup(build) {
    build.onResolve({ filter: /.*/ }, args => {
      console.log(`[track-imports] Resolving: ${args.path} imported from ${args.importer || '<entry>'}`);
      return { path: args.path };
    });
  }
};
