/**
 * Core Lit SSR rendering harness.
 *
 * Takes HTML containing custom elements and returns the same HTML
 * with Declarative Shadow DOM (DSD) injected via Lit SSR.
 *
 * Component definitions must be imported before calling processHTML
 * so they are registered with the customElements registry.
 * Registered elements are rendered with DSD; unregistered elements
 * pass through unchanged.
 *
 * SECURITY: Input is rendered via Lit's unsafeHTML directive without
 * sanitization. Callers must ensure that input HTML is trusted or
 * sanitized upstream to prevent XSS in the rendered output.
 */

import { render } from '@lit-labs/ssr';
import { html } from 'lit';
import { unsafeHTML } from 'lit/directives/unsafe-html.js';

/** Process HTML, rendering all registered custom elements with DSD. */
export function processHTML(input: string): string {
  const template = html`${unsafeHTML(input)}`;
  const chunks: string[] = [];
  for (const chunk of render(template)) {
    chunks.push(String(chunk));
  }
  return chunks.join('');
}
