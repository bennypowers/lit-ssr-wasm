/**
 * Entry point for the Lit SSR WASM module (bytecode mode).
 *
 * Same protocol as runtime-entry.ts, but the user's component source
 * is injected at compile time (replacing the placeholder below) instead
 * of being eval'd from the init JSON. This allows the combined JS to be
 * pre-compiled to QuickJS bytecode, skipping parse+compile per worker.
 *
 * Two-phase protocol:
 *   1. Init:   JSON line {"elements":[...]}\n -> ack \0
 *   2. Render: NUL-terminated HTML -> NUL-terminated rendered HTML
 * Errors go to stderr. Exits cleanly at EOF.
 */

// SSR shims must be installed before Lit loads. These side-effect
// imports are evaluated first in the bundled output.
import './ssr-css-fix.js';
import './ssr-shims.js';

import { processHTML } from './harness/render.js';
import { readLine, readUntilNul, writeStdout, writeStderr } from './io.js';

// User component source is injected here at compile time by the Go library.
// The placeholder below is replaced with a bundled IIFE.
globalThis.__LITSSR_USER_SOURCE__ = true;

// Phase 1: read init message (elements only, source already loaded above)
const initLine = readLine();
if (initLine === null) throw new Error('unexpected EOF before init');

let known: Set<string>;
try {
  const init = JSON.parse(initLine) as { elements: string[] };
  known = new Set(init.elements);
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
  if (html.trim() === '') {
    writeStdout('\0');
    continue;
  }

  try {
    const output = processHTML(html, known);
    writeStdout(output + '\0');
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    writeStderr(msg + '\n');
    writeStdout('\0');
  }
}
