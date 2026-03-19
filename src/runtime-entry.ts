/**
 * Runtime mode entry point for the Lit SSR WASM module.
 *
 * No component definitions are baked in. Instead, reads JSON from stdin:
 *   { "source": "...", "html": "...", "elements": ["my-alert"] }
 *
 * The source is eval'd in QuickJS, which registers custom elements
 * via customElements.define(). Lit APIs are exposed as globals so
 * eval'd code can reference them without import statements.
 */

import { processHTML } from './harness/render.js';
import { readStdin, writeStdout, writeStderr } from './io.js';

import { LitElement, html, css, nothing, noChange } from 'lit';
import { classMap } from 'lit/directives/class-map.js';
import { styleMap } from 'lit/directives/style-map.js';
import { repeat } from 'lit/directives/repeat.js';
import { unsafeHTML } from 'lit/directives/unsafe-html.js';

// Expose lit APIs as globals so eval'd component source can use them.
Object.assign(globalThis, {
  LitElement, html, css, nothing, noChange,
  classMap, styleMap, repeat, unsafeHTML,
});

try {
  const input = JSON.parse(readStdin()) as {
    source: string;
    html: string;
    elements: string[];
  };

  // Eval the component source. This runs in QuickJS and registers
  // custom elements via customElements.define().
  (0, eval)(input.source);

  const known = new Set(input.elements);
  const output = processHTML(input.html, known);
  writeStdout(output);
} catch (e: unknown) {
  const msg = e instanceof Error ? `${e.message}\n${e.stack}` : String(e);
  writeStderr(`lit-ssr-wasm runtime error: ${msg}\n`);
}
