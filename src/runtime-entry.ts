/**
 * Entry point for the Lit SSR WASM module (runtime mode).
 *
 * Two-phase protocol:
 *   1. Init:   JSON line {"source":"...","elements":[...]}\n -> ack \0
 *   2. Render: NUL-terminated HTML -> NUL-terminated rendered HTML
 * Errors go to stderr. Exits cleanly at EOF.
 */

// SSR shims must be installed before Lit loads. These side-effect
// imports are evaluated first in the bundled output.
import './ssr-css-fix.js';
import './ssr-shims.js';

import { processHTML } from './harness/render.js';
import { readLine, readUntilNul, writeStdout, writeStderr } from './io.js';

// Phase 1: read init message
const initLine = readLine();
if (initLine === null) throw new Error('unexpected EOF before init');

let known: Set<string>;
try {
  const init = JSON.parse(initLine) as { source: string; elements: string[] };
  known = new Set(init.elements);
  if (init.source) {
    (0, eval)(init.source);
  }
  writeStdout('\0'); // ack
} catch (e: unknown) {
  const msg = e instanceof Error ? e.message : String(e);
  writeStderr(msg + '\n');
  writeStdout('\0');
  throw e;
}

// Phase 2: render loop -- NUL-terminated HTML in, NUL-terminated HTML out.
for (;;) {
  const html = readUntilNul();
  if (html === null) break;
  if (html.trim() === '') continue;

  try {
    const output = processHTML(html, known);
    writeStdout(output + '\0');
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    writeStderr(msg + '\n');
    writeStdout('\0');
  }
}
