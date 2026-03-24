/**
 * Entry point for the Lit SSR WASM module (builtin/compiled mode).
 *
 * Component definitions are baked in. Read loop:
 *   request (stdin):  raw HTML, NUL-terminated
 *   response (stdout): rendered HTML, NUL-terminated
 *   errors go to stderr
 * Exits cleanly at EOF.
 */

import './ssr-css-fix.js';

import { processHTML } from './harness/render.js';
import { readUntilNul, writeStdout, writeStderr } from './io.js';

import './components/x-card.js';
import './components/x-cta.js';
import './components/x-tabs.js';
import './components/x-tab.js';
import './components/x-tab-panel.js';
import './components/x-badge.js';
import './components/my-alert.js';

for (;;) {
  const input = readUntilNul();
  if (input === null) break;

  try {
    const output = processHTML(input);
    writeStdout(output + '\0');
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    writeStderr(msg + '\n');
    writeStdout('\0');
  }
}
