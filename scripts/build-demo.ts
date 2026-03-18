/**
 * Builds the GitHub Pages demo site.
 *
 * Produces three bundles:
 *   1. compiled-mode.mjs  - Components baked in, exports processHTML
 *   2. runtime-mode.mjs   - No components, exports harness for eval'd definitions
 *   3. components.mjs     - Standalone component bundle for the runtime demo
 */

import * as esbuild from 'esbuild';
import { resolve } from 'node:path';
import { cpSync } from 'node:fs';

const root = resolve(import.meta.dirname!, '..');

const common: esbuild.BuildOptions = {
  bundle: true,
  format: 'esm',
  target: 'es2022',
  platform: 'browser',
  define: { 'process.env.NODE_ENV': '"production"' },
  conditions: ['browser'],
  minify: true,
  sourcemap: false,
};

await Promise.all([
  // Compiled mode: components + harness, ready to use
  esbuild.build({
    ...common,
    entryPoints: [resolve(root, 'src/demo/compiled.ts')],
    outfile: resolve(root, 'docs/compiled-mode.mjs'),
  }),
  // Runtime mode: harness only, components loaded dynamically
  esbuild.build({
    ...common,
    entryPoints: [resolve(root, 'src/demo/runtime.ts')],
    outfile: resolve(root, 'docs/runtime-mode.mjs'),
  }),
]);

console.log('Demo bundles built in docs/');
