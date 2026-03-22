/**
 * esbuild plugin for bundling Lit components for lit-ssr-wasm.
 *
 * Resolves @lit-labs/ssr-dom-shim imports to re-export from globalThis,
 * so the consumer's Lit copy shares the same customElements registry
 * and DOM shims that the lit-ssr-wasm runtime provides.
 *
 * Usage (copy this file into your project or reference by path):
 *   import { litSsrWasmPlugin } from './esbuild-plugin.ts';
 *
 *   esbuild.build({
 *     entryPoints: ['my-components.ts'],
 *     bundle: true,
 *     format: 'esm',
 *     conditions: ['node'],
 *     plugins: [litSsrWasmPlugin()],
 *   });
 */

import type { Plugin } from 'esbuild';

const SHIM_BRIDGE = `
export const customElements = globalThis.customElements;
export const HTMLElement = globalThis.HTMLElement;
export const Element = globalThis.Element;
export const CSSStyleSheet = globalThis.CSSStyleSheet;
export const CustomElementRegistry = globalThis.CustomElementRegistry;
export const Event = globalThis.Event;
export const CustomEvent = globalThis.CustomEvent;
export const EventTarget = globalThis.EventTarget;
export const ariaMixinAttributes = globalThis.ariaMixinAttributes ?? {};
export const HYDRATE_INTERNALS_ATTR_PREFIX = globalThis.HYDRATE_INTERNALS_ATTR_PREFIX ?? 'internals-';
export const ElementInternals = globalThis.ElementInternals ?? class ElementInternals {};
`.trim();

export function litSsrWasmPlugin(): Plugin {
  return {
    name: 'lit-ssr-wasm',
    setup(build) {
      // Redirect @lit-labs/ssr-dom-shim to a bridge that reads from globalThis.
      build.onResolve({ filter: /^@lit-labs\/ssr-dom-shim/ }, args => ({
        path: args.path,
        namespace: 'lit-ssr-wasm-shim',
      }));
      build.onLoad({ filter: /.*/, namespace: 'lit-ssr-wasm-shim' }, () => ({
        contents: SHIM_BRIDGE,
        loader: 'js',
      }));
    },
  };
}
