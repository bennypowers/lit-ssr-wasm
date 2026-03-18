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
 * The source is injected as a <script type="module"> so the page's
 * import map applies, allowing bare specifiers like `import ... from 'lit'`.
 */
export async function loadAndRender(
  componentSource: string,
  htmlInput: string,
  knownElements: string[],
): Promise<string> {
  // Inject source as an inline module script. The page's import map
  // lets it resolve bare specifiers like 'lit'.
  const script = document.createElement('script');
  script.type = 'module';
  script.textContent = componentSource;

  // Wait for the module to execute by appending a sentinel.
  const sentinel = `__litssr_ready_${Date.now()}`;
  const ready = new Promise<void>(resolve => {
    (globalThis as any)[sentinel] = resolve;
  });
  script.textContent += `\n;globalThis["${sentinel}"]?.();`;
  document.head.append(script);

  try {
    await ready;
  } finally {
    script.remove();
    delete (globalThis as any)[sentinel];
  }

  const known = new Set(knownElements);
  return processHTML(htmlInput, known);
}

export { processHTML };
