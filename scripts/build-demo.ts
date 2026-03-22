/**
 * Builds the GitHub Pages demo site.
 *
 * Page fragments in docs/ are assembled with docs/_layout.html,
 * then RHDS elements in the chrome are SSR'd with DSD.
 * Output goes to _site/.
 */

import * as esbuild from 'esbuild';
import ts from 'typescript';
import { idiomaticDecoratorsTransformer } from '@lit/ts-transformers';
import { resolve } from 'node:path';
import { readFile, writeFile, readdir, copyFile, mkdir } from 'node:fs/promises';
import { spawn } from 'node:child_process';

const root = resolve(import.meta.dirname!, '..');
const docsDir = resolve(root, 'docs');
const siteDir = resolve(root, '_site');
const distDir = resolve(root, 'dist');

await mkdir(siteDir, { recursive: true });

// Step 1: Copy WASM files and static assets to _site/
const wasmCopies = ['lit-ssr-runtime.wasm', 'lit-ssr-demo.wasm'].map(file =>
  copyFile(resolve(distDir, file), resolve(siteDir, file)).catch(() =>
    console.warn(`Warning: ${file} not found in dist/. Run npm run build:wasm first.`),
  ),
);

const docs = await readdir(docsDir);
const staticCopies = docs
  .filter(f => !f.endsWith('.html'))
  .map(f => copyFile(resolve(docsDir, f), resolve(siteDir, f)));

await Promise.all([...wasmCopies, ...staticCopies]);

// Step 2: Prepare includes for template substitution.
// Use @lit/ts-transformers to convert decorators to idiomatic JS,
// then strip imports (runtime WASM provides lit APIs as globals).
function litTransform(fileName: string, source: string): string {
  const compilerOptions: ts.CompilerOptions = {
    target: ts.ScriptTarget.ESNext,
    module: ts.ModuleKind.ESNext,
    experimentalDecorators: false,
    useDefineForClassFields: true,
    removeComments: true,
  };

  // Convert to 4-space indentation so TS printer output is consistent with template literal contents
  const source4 = source.replace(/^( {2})+/gm, (m: string) => '    '.repeat(m.length / 2));
  const sourceFile = ts.createSourceFile(fileName, source4, ts.ScriptTarget.ESNext, true);
  const host = ts.createCompilerHost(compilerOptions);
  const originalGetSourceFile = host.getSourceFile.bind(host);
  host.getSourceFile = (name, target) =>
    name === fileName ? sourceFile : originalGetSourceFile(name, target);

  const program = ts.createProgram([fileName], compilerOptions, host);
  const result = ts.transform(sourceFile, [idiomaticDecoratorsTransformer(program)]);
  const printer = ts.createPrinter({ newLine: ts.NewLineKind.LineFeed, removeComments: true });
  const output = printer.printFile(result.transformed[0]);
  result.dispose();

  return output
    .replace(/^import\s+.*;\s*$/gm, '')         // strip imports
    .replace(/^export\s+/gm, '')                // strip export keyword
    .replace(/\boverride\s+/g, '')              // strip override keyword
    .replace(/\n{3,}/g, '\n\n')
    .trim();
}

const myAlertTs = await readFile(resolve(root, 'src/components/my-alert.ts'), 'utf-8');
const myAlertJs = litTransform('my-alert.ts', myAlertTs);

const includes: Record<string, string> = {
  'my-alert': myAlertJs,
};

// Step 3: Assemble pages from layout + content fragments.
const layout = await readFile(resolve(docsDir, '_layout.html'), 'utf-8');

const htmlPages = docs.filter(f => f.endsWith('.html') && !f.startsWith('_'));

await Promise.all(htmlPages.map(async file => {
  let content = await readFile(resolve(docsDir, file), 'utf-8');

  // Replace {{name}} placeholders with includes.
  // HTML-escape the value since it's injected into element light DOM.
  for (const [name, value] of Object.entries(includes)) {
    const escaped = value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    content = content.replaceAll(`{{${name}}}`, escaped);
  }

  const titleMatch = content.match(/<!--\s*title:\s*(.+?)\s*-->/);
  const title = titleMatch?.[1] ?? 'lit-ssr-wasm';

  const assembled = layout
    .replace('{{title}}', title)
    .replace('{{content}}', content)
    .replace(
      new RegExp(`(<a\\s+href="${file}")`),
      '$1\n         active',
    );

  await writeFile(resolve(siteDir, file), assembled);
}));

console.log('Pages assembled.');

// Step 3: SSR RHDS elements in the chrome.
const ssrBundle = resolve(distDir, 'ssr-rhds.js');

await esbuild.build({
  entryPoints: [resolve(root, 'src/ssr-rhds.ts')],
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

function ssrFile(html: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const proc = spawn('node', [ssrBundle], { stdio: ['pipe', 'pipe', 'pipe'], timeout: 10_000 });
    const chunks: Buffer[] = [];
    proc.stdout.on('data', (d: Buffer) => chunks.push(d));
    proc.on('close', code => code === 0
      ? resolve(Buffer.concat(chunks).toString('utf-8'))
      : reject(new Error(`exit ${code}`)));
    proc.on('error', reject);
    proc.stdin.end(html);
  });
}

const siteFiles = await readdir(siteDir);
await Promise.all(
  siteFiles
    .filter(f => f.endsWith('.html'))
    .map(async file => {
      const filePath = resolve(siteDir, file);
      const html = await readFile(filePath, 'utf-8');

      if (!/\b(?:rh-(?:subnav|surface|tag)|highlighted-textarea)\b/.test(html)) return;

      try {
        await writeFile(filePath, await ssrFile(html));
        console.log(`SSR'd: ${file}`);
      } catch {
        console.warn(`SSR failed for ${file}, keeping original.`);
      }
    }),
);

console.log('Demo build complete.');
