/**
 * Core Lit SSR rendering harness.
 *
 * Takes HTML containing custom elements and returns the same HTML
 * with Declarative Shadow DOM (DSD) injected via Lit SSR.
 *
 * Component definitions must be imported before calling processHTML
 * so they are registered with the customElements registry.
 */

import { render } from '@lit-labs/ssr';
import { html, unsafeStatic } from 'lit/static-html.js';

interface Replacement {
  start: number;
  end: number;
  replacement: string;
}

/** Render a single custom element to an HTML string with DSD. */
export function renderElementToString(
  tag: string,
  attrs: Record<string, string>,
  slotContent: string,
): string {
  const tagLiteral = unsafeStatic(tag);
  const attrStr = Object.entries(attrs)
    .map(([k, v]) => v === '' ? k : `${k}="${v}"`)
    .join(' ');
  const attrLiteral = attrStr ? unsafeStatic(` ${attrStr}`) : unsafeStatic('');
  const contentLiteral = unsafeStatic(slotContent);

  const template = html`<${tagLiteral}${attrLiteral}>${contentLiteral}</${tagLiteral}>`;
  const chunks: string[] = [];
  for (const chunk of render(template)) {
    chunks.push(String(chunk));
  }
  return chunks.join('');
}

/**
 * Find the matching close tag, handling nesting of the same tag name.
 * Returns the index of the `<` in `</tagName>`, or -1 if not found.
 */
function findCloseTag(input: string, tagName: string, searchFrom: number): number {
  const openPattern = new RegExp(`<${tagName}[\\s>]`, 'gi');
  const closeStr = `</${tagName}>`;
  let depth = 1;
  let pos = searchFrom;

  while (depth > 0 && pos < input.length) {
    const nextClose = input.indexOf(closeStr, pos);
    if (nextClose === -1) return -1;

    openPattern.lastIndex = pos;
    let openMatch: RegExpExecArray | null;
    while ((openMatch = openPattern.exec(input)) !== null && openMatch.index < nextClose) {
      depth++;
    }

    depth--;
    if (depth === 0) return nextClose;
    pos = nextClose + closeStr.length;
  }
  return -1;
}

/** Parse HTML attributes from a raw attribute string. */
function parseAttrs(attrString: string): Record<string, string> {
  const attrs: Record<string, string> = {};
  const re = /([a-zA-Z_][a-zA-Z0-9_-]*)(?:="([^"]*)")?/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(attrString)) !== null) {
    attrs[m[1]] = m[2] ?? '';
  }
  return attrs;
}

/**
 * Process an HTML string, rendering all known custom elements with DSD.
 *
 * Lit SSR handles nested custom elements automatically, so this function
 * only matches top-level known elements; children are rendered recursively
 * by the Lit SSR engine itself.
 */
export function processHTML(input: string, knownElements: Set<string>): string {
  const tagRe = /<([a-z][a-z0-9]*-[a-z0-9-]*)((?:\s+[a-zA-Z_][a-zA-Z0-9_-]*(?:="[^"]*")?)*)\s*>/gi;

  let result = input;
  const replacements: Replacement[] = [];

  let match: RegExpExecArray | null;
  while ((match = tagRe.exec(input)) !== null) {
    const fullMatch = match[0];
    const tagName = match[1].toLowerCase();
    const attrString = match[2] ?? '';

    if (!knownElements.has(tagName)) continue;

    // Skip elements nested inside an already-matched parent.
    if (replacements.some(r => match!.index > r.start && match!.index < r.end)) continue;

    const attrs = parseAttrs(attrString);
    const contentStart = match.index + fullMatch.length;
    const closeIdx = findCloseTag(input, tagName, contentStart);
    if (closeIdx === -1) continue;

    const closeTag = `</${tagName}>`;
    const slotContent = input.substring(contentStart, closeIdx);

    try {
      replacements.push({
        start: match.index,
        end: closeIdx + closeTag.length,
        replacement: renderElementToString(tagName, attrs, slotContent),
      });
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      console.error(`Failed to render <${tagName}>: ${msg}`);
    }
  }

  // Apply in reverse to preserve earlier indices.
  for (let i = replacements.length - 1; i >= 0; i--) {
    const { start, end, replacement } = replacements[i];
    result = result.substring(0, start) + replacement + result.substring(end);
  }
  return result;
}
