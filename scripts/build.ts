/**
 * Build script: produces ESM bundles for Node.js testing and Javy WASM compilation.
 *
 * Usage: node --import tsx/esm scripts/build.ts
 */

import * as esbuild from 'esbuild';
import { resolve, dirname } from 'node:path';

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

/** Replace Node.js built-in imports with stubs for the Javy QuickJS runtime. */
const stubNodeBuiltins: esbuild.Plugin = {
  name: 'stub-node-builtins',
  setup(build) {
    const builtins = /^(node:|buffer|fs|path|stream|util|events|crypto|os|url|http|https|net|tls|child_process|module|vm|zlib)/;
    build.onResolve({ filter: builtins }, args => ({
      path: args.path,
      namespace: 'stub',
    }));
    build.onLoad({ filter: /.*/, namespace: 'stub' }, () => ({
      contents: [
        'export default {};',
        'export const Buffer = {',
        '  from: x => typeof x === "string" ? new TextEncoder().encode(x) : new Uint8Array(x),',
        '  isBuffer: () => false,',
        '  alloc: n => new Uint8Array(n),',
        '};',
        'export const readFileSync = () => "";',
      ].join('\n'),
    }));
  },
};

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
  // Compiled mode: components baked in, reads HTML from stdin
  esbuild.build({
    ...common,
    entryPoints: ['src/entry.ts'],
    outfile: 'dist/lit-ssr-bundle.js',
    platform: 'node',
    external: ['node:fs'],
    plugins: [aliasIO('node.ts')],
  }),
  esbuild.build({
    ...common,
    entryPoints: ['src/entry.ts'],
    outfile: 'dist/lit-ssr-javy.js',
    platform: 'node',
    plugins: [aliasIO('javy.ts'), stubNodeBuiltins],
  }),
  // Runtime mode: no components, evals JS source from JSON stdin
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
]);

console.log('Built: compiled + runtime bundles for Node.js and Javy');
