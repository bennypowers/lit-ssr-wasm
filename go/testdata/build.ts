/**
 * Build test components as a consumer would: esbuild + litSsrWasmPlugin.
 * Output goes to test-components.js (embedded by Go tests).
 */

import * as esbuild from 'esbuild';
import { litSsrWasmPlugin } from '../../src/esbuild-plugin.ts';
import { stubNodeBuiltins } from '../../src/esbuild-stubs.ts';
import { resolve } from 'node:path';

await esbuild.build({
  entryPoints: [resolve(import.meta.dirname!, 'test-components.ts')],
  outfile: resolve(import.meta.dirname!, 'test-components.js'),
  bundle: true,
  format: 'esm',
  target: 'es2022',
  platform: 'node',
  conditions: ['node'],
  define: { 'process.env.NODE_ENV': '"production"' },
  plugins: [litSsrWasmPlugin(), stubNodeBuiltins],
  minify: false,
  sourcemap: false,
});

console.log('Built test-components.js');
