/**
 * Entry point for the Lit SSR WASM module (runtime mode).
 *
 * No component definitions baked in. Read loop:
 *   request (stdin):  JSON line {"source":"...","html":"...","elements":[...]}\n
 *   response (stdout): rendered HTML, NUL-terminated
 *   errors go to stderr
 * Exits cleanly at EOF.
 */

import { processHTML } from './harness/render.js';
import { readLine, writeStdout, writeStderr } from './io.js';

import { LitElement, html, css, nothing, noChange } from 'lit';
import { classMap } from 'lit/directives/class-map.js';
import { styleMap } from 'lit/directives/style-map.js';
import { repeat } from 'lit/directives/repeat.js';
import { unsafeHTML } from 'lit/directives/unsafe-html.js';

Object.assign(globalThis, {
  LitElement, html, css, nothing, noChange,
  classMap, styleMap, repeat, unsafeHTML,
});

for (;;) {
  const line = readLine();
  if (line === null) break;
  if (line.trim() === '') continue;

  try {
    const req = JSON.parse(line) as {
      source: string;
      html: string;
      elements: string[];
    };

    // Only eval source if it contains elements we haven't registered yet.
    // QuickJS keeps state across the read loop, so once components are
    // defined they stay registered for all subsequent renders.
    const known = new Set(req.elements);
    const needsEval = req.elements.some(name => !customElements.get(name));
    if (needsEval && req.source) {
      (0, eval)(req.source);
    }

    const output = processHTML(req.html, known);
    writeStdout(output + '\0');
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    writeStderr(msg + '\n');
    writeStdout('\0');
  }
}
