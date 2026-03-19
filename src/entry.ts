/**
 * Entry point for the Lit SSR WASM module.
 *
 * Imports component definitions, reads HTML from stdin, renders all
 * known custom elements with Declarative Shadow DOM, and writes the
 * result to stdout.
 *
 * The build script swaps the I/O module depending on the target
 * platform (Javy WASM vs Node.js).
 */

import { processHTML } from './harness/render.js';
import { readStdin, writeStdout, writeStderr } from './io.js';

import './components/x-card.js';
import './components/x-cta.js';
import './components/x-tabs.js';
import './components/x-tab.js';
import './components/x-tab-panel.js';
import './components/x-badge.js';
import './components/my-alert.js';

const KNOWN_ELEMENTS = new Set([
  'x-card', 'x-cta',
  'x-tabs', 'x-tab', 'x-tab-panel',
  'x-badge',
  'my-alert',
]);

try {
  const input = readStdin();
  const output = processHTML(input, KNOWN_ELEMENTS);
  writeStdout(output);
} catch (e: unknown) {
  const msg = e instanceof Error ? `${e.message}\n${e.stack}` : String(e);
  writeStderr(`lit-ssr-wasm error: ${msg}\n`);
}
