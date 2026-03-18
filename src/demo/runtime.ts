/**
 * Runtime mode: no components baked in. The caller provides component
 * definitions as JavaScript source which is eval'd before rendering.
 *
 * This simulates the "dynamic" use case where component definitions
 * are not known at build time.
 */

import { processHTML } from '../harness/render.js';

/**
 * Load component definitions from JS source, then render HTML.
 *
 * The JS source should call customElements.define() for each component.
 * After eval, we discover registered elements and render with SSR.
 */
export async function loadAndRender(
  componentSource: string,
  html: string,
): Promise<string> {
  // Eval the component source as a module via blob URL.
  // This lets the source use `import { LitElement } from 'lit'` etc.
  // We pre-bundle lit into a helper that the eval'd code can import.
  const blob = new Blob([componentSource], { type: 'text/javascript' });
  const url = URL.createObjectURL(blob);
  try {
    await import(/* @vite-ignore */ url);
  } finally {
    URL.revokeObjectURL(url);
  }

  // Discover all custom elements that contain a hyphen (web component convention).
  // The SSR shim's customElements registry tracks definitions.
  const known = new Set<string>();
  // @ts-expect-error __definitions is a Lit SSR shim internal
  const defs = customElements.__definitions ?? customElements._registry;
  if (defs && typeof defs[Symbol.iterator] === 'function') {
    for (const [name] of defs) {
      known.add(name);
    }
  }

  return processHTML(html, known);
}

// Also export direct processHTML for cases where the caller manages registration.
export { processHTML };
