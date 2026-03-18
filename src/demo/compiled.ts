/**
 * Compiled mode: all component definitions are baked into this bundle.
 * Exports processHTML for use in the demo page.
 */

import { processHTML } from '../harness/render.js';

import '../components/x-card.js';
import '../components/x-cta.js';
import '../components/x-tabs.js';
import '../components/x-tab.js';
import '../components/x-tab-panel.js';
import '../components/x-badge.js';

const KNOWN = new Set([
  'x-card', 'x-cta', 'x-tabs', 'x-tab', 'x-tab-panel', 'x-badge',
]);

export function render(html: string): string {
  return processHTML(html, KNOWN);
}
