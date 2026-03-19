/**
 * SSR entry for RHDS elements used in the demo site chrome.
 * Reads HTML from stdin, renders known RHDS elements with DSD, writes to stdout.
 * Used at build time by build-demo.ts.
 */

import { processHTML } from './harness/render.js';
import { readStdin, writeStdout, writeStderr } from './io/node.js';

import '@rhds/elements/rh-subnav/rh-subnav.js';
import '@rhds/elements/rh-surface/rh-surface.js';
import '@rhds/elements/rh-tag/rh-tag.js';
import './components/highlighted-textarea.js';

const KNOWN = new Set(['rh-subnav', 'rh-surface', 'rh-tag', 'highlighted-textarea']);

try {
  const input = readStdin();
  const output = processHTML(input, KNOWN);
  writeStdout(output);
} catch (e) {
  const msg = e instanceof Error ? e.message : String(e);
  writeStderr('SSR error: ' + msg + '\n');
}
