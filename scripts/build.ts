/**
 * Build script: produces ESM bundles for Node.js testing and Javy WASM compilation.
 *
 * Usage: node --import tsx/esm scripts/build.ts
 */

import * as esbuild from 'esbuild';
import { resolve, dirname } from 'node:path';
import { stubNodeBuiltins } from '../src/esbuild-stubs.ts';

/** Swap `./io.js` imports to the platform-specific I/O module. */
function aliasIO(target: string): esbuild.Plugin {
  return {
    name: 'alias-io',
    setup(build) {
      build.onResolve({ filter: /^\.\/io\.js$/ }, () => ({
        path: resolve(import.meta.dirname!, '..', 'src', 'io', target),
      }));
    },
  };
}

const common: esbuild.BuildOptions = {
  bundle: true,
  format: 'esm' as const,
  target: 'es2022',
  define: { 'process.env.NODE_ENV': '"production"' },
  conditions: ['node'],
  minify: false,
  sourcemap: false,
};

await Promise.all([
  // Runtime mode: two-phase protocol (JSON init, then NUL-delimited HTML renders)
  esbuild.build({
    ...common,
    entryPoints: ['src/runtime-entry.ts'],
    outfile: 'dist/lit-ssr-runtime-bundle.js',
    platform: 'node',
    external: ['node:fs'],
    plugins: [aliasIO('node.ts')],
  }),
  esbuild.build({
    ...common,
    entryPoints: ['src/runtime-entry.ts'],
    outfile: 'dist/lit-ssr-runtime-javy.js',
    platform: 'node',
    plugins: [aliasIO('javy.ts'), stubNodeBuiltins],
  }),
  // Bytecode mode: template for pre-compiled bytecode (user source injected at compile time)
  // IIFE format avoids ESM module scoping issues when compiled to QuickJS bytecode.
  esbuild.build({
    ...common,
    format: 'iife',
    entryPoints: ['src/bytecode-entry.ts'],
    outfile: 'dist/lit-ssr-bytecode-entry.js',
    platform: 'node',
    plugins: [aliasIO('javy.ts'), stubNodeBuiltins],
  }),
  // Demo: baked-in components for the demo site (not shipped)
  esbuild.build({
    ...common,
    entryPoints: ['src/demo-entry.ts'],
    outfile: 'dist/lit-ssr-demo-bundle.js',
    platform: 'node',
    external: ['node:fs'],
    plugins: [aliasIO('node.ts')],
  }),
  esbuild.build({
    ...common,
    entryPoints: ['src/demo-entry.ts'],
    outfile: 'dist/lit-ssr-demo-javy.js',
    platform: 'node',
    plugins: [aliasIO('javy.ts'), stubNodeBuiltins],
  }),
]);

console.log('Built: runtime + bytecode + demo bundles for Node.js and Javy');
