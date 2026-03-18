/**
 * Builds the GitHub Pages demo site.
 *
 * Source HTML lives in pages/. The build:
 *   1. Builds browser bundles into docs/
 *   2. Copies all pages/ HTML to docs/
 *   3. SSRs pages that use RHDS elements via our own harness
 */

import * as esbuild from 'esbuild';
import { resolve } from 'node:path';
import { readFileSync, writeFileSync, readdirSync, copyFileSync } from 'node:fs';
import { execSync } from 'node:child_process';

const root = resolve(import.meta.dirname!, '..');
const pagesDir = resolve(root, 'pages');
const docsDir = resolve(root, 'docs');

const common: esbuild.BuildOptions = {
  bundle: true,
  format: 'esm' as const,
  target: 'es2022',
  platform: 'browser',
  define: { 'process.env.NODE_ENV': '"production"' },
  conditions: ['browser'],
  minify: true,
  sourcemap: false,
};

// Step 1: Build browser bundles into docs/
await Promise.all([
  esbuild.build({
    ...common,
    entryPoints: [resolve(root, 'src/demo/compiled.ts')],
    outfile: resolve(docsDir, 'compiled-mode.mjs'),
  }),
  esbuild.build({
    ...common,
    entryPoints: [resolve(root, 'src/demo/runtime.ts')],
    outfile: resolve(docsDir, 'runtime-mode.mjs'),
  }),
  esbuild.build({
    ...common,
    entryPoints: [resolve(root, 'src/demo/lit-reexport.ts')],
    outfile: resolve(docsDir, 'lit.mjs'),
  }),
  esbuild.build({
    ...common,
    entryPoints: [resolve(root, 'src/demo/lit-decorators-reexport.ts')],
    outfile: resolve(docsDir, 'lit-decorators.mjs'),
  }),
]);

console.log('Browser bundles built.');

// Step 2: Copy source pages to docs/
for (const file of readdirSync(pagesDir).filter(f => f.endsWith('.html'))) {
  copyFileSync(resolve(pagesDir, file), resolve(docsDir, file));
}

console.log('Pages copied to docs/.');

// Step 3: SSR pages that use RHDS elements.
const ssrEntry = resolve(root, 'src/demo/ssr-rhds.ts');
const ssrBundle = resolve(root, 'dist/ssr-rhds.mjs');

writeFileSync(ssrEntry, `
import { processHTML } from '../harness/render.js';
import { readStdin, writeStdout, writeStderr } from '../io/node.js';

import '@rhds/elements/rh-card/rh-card.js';
import '@rhds/elements/rh-surface/rh-surface.js';
import '@rhds/elements/rh-tabs/rh-tabs.js';
import '@rhds/elements/rh-tag/rh-tag.js';

const KNOWN = new Set(['rh-card', 'rh-surface', 'rh-tabs', 'rh-tab', 'rh-tab-panel', 'rh-tag']);

try {
  const input = readStdin();
  const output = processHTML(input, KNOWN);
  writeStdout(output);
} catch (e) {
  const msg = e instanceof Error ? e.message : String(e);
  writeStderr('SSR error: ' + msg + '\\n');
}
`);

try {
  await esbuild.build({
    entryPoints: [ssrEntry],
    bundle: true,
    outfile: ssrBundle,
    format: 'esm',
    platform: 'node',
    target: 'es2022',
    external: ['node:fs'],
    define: { 'process.env.NODE_ENV': '"production"' },
    conditions: ['node'],
    minify: false,
    sourcemap: false,
  });

  for (const file of readdirSync(docsDir).filter(f => f.endsWith('.html'))) {
    const filePath = resolve(docsDir, file);
    const html = readFileSync(filePath, 'utf-8');

    if (!/\brh-(?:card|surface|tab|tag)\b/.test(html)) continue;

    try {
      const ssrd = execSync(`node ${ssrBundle}`, {
        input: html,
        encoding: 'utf-8',
        timeout: 10_000,
      });
      writeFileSync(filePath, ssrd);
      console.log(`SSR'd: ${file}`);
    } catch (e) {
      console.warn(`SSR failed for ${file}, keeping original.`);
    }
  }
} finally {
  const { unlinkSync } = await import('node:fs');
  try { unlinkSync(ssrEntry); } catch {}
  try { unlinkSync(ssrBundle); } catch {}
}

console.log('Demo build complete.');
