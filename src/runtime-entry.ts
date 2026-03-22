/**
 * Entry point for the Lit SSR WASM module (runtime mode).
 *
 * No component definitions baked in. Read loop:
 *   request (stdin):  JSON line {"source":"...","html":"...","elements":[...]}\n
 *   response (stdout): rendered HTML, NUL-terminated
 *   errors go to stderr
 * Exits cleanly at EOF.
 */

// -- SSR environment shims (must run before any Lit imports) --

// CSSStyleSheet.prototype.cssText fix for CSS module imports.
import './ssr-css-fix.js';

// btoa / atob: needed by @lit-labs/ssr-client digestForTemplateResult()
// for base64-encoded template digests (hydration matching).
// QuickJS doesn't provide these.
const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';

function btoa(s: string): string {
  let out = '';
  for (let i = 0; i < s.length; i += 3) {
    const a = s.charCodeAt(i);
    const b = i + 1 < s.length ? s.charCodeAt(i + 1) : 0;
    const c = i + 2 < s.length ? s.charCodeAt(i + 2) : 0;
    out += chars[(a >> 2) & 63];
    out += chars[((a << 4) | (b >> 4)) & 63];
    out += i + 1 < s.length ? chars[((b << 2) | (c >> 6)) & 63] : '=';
    out += i + 2 < s.length ? chars[c & 63] : '=';
  }
  return out;
}

function atob(s: string): string {
  let out = '';
  const clean = s.replace(/=+$/, '');
  for (let i = 0; i < clean.length; i += 4) {
    const a = chars.indexOf(clean[i]);
    const b = chars.indexOf(clean[i + 1]);
    const c = chars.indexOf(clean[i + 2]);
    const d = chars.indexOf(clean[i + 3]);
    out += String.fromCharCode(((a << 2) | (b >> 4)) & 255);
    if (c >= 0) out += String.fromCharCode(((b << 4) | (c >> 2)) & 255);
    if (d >= 0) out += String.fromCharCode(((c << 6) | d) & 255);
  }
  return out;
}

// Minimal URL / URLSearchParams: needed by Lit's dom-shim
// (location: new URL('http://localhost')).
class URLShim {
  href: string;
  protocol: string;
  hostname: string;
  pathname: string;
  search: string;
  hash: string;
  host: string;
  origin: string;
  port: string;
  searchParams: URLSearchParamsShim;

  constructor(url: string) {
    this.href = url;
    const m = url.match(/^(\w+):\/\/([^/:]+)(?::(\d+))?(\/[^?#]*)?(\?[^#]*)?(#.*)?$/) ?? [];
    this.protocol = (m[1] ?? 'http') + ':';
    this.hostname = m[2] ?? 'localhost';
    this.port = m[3] ?? '';
    this.pathname = m[4] ?? '/';
    this.search = m[5] ?? '';
    this.hash = m[6] ?? '';
    this.host = this.port ? `${this.hostname}:${this.port}` : this.hostname;
    this.origin = `${this.protocol}//${this.host}`;
    this.searchParams = new URLSearchParamsShim(this.search);
  }

  toString() { return this.href; }
}

class URLSearchParamsShim {
  private _entries: [string, string][] = [];

  constructor(init?: string) {
    if (init?.startsWith('?')) init = init.slice(1);
    if (init) {
      for (const pair of init.split('&')) {
        const eq = pair.indexOf('=');
        const k = eq < 0 ? pair : pair.slice(0, eq);
        const v = eq < 0 ? '' : pair.slice(eq + 1);
        this._entries.push([decodeURIComponent(k), decodeURIComponent(v)]);
      }
    }
  }

  get(key: string) {
    const found = this._entries.find(([k]) => k === key);
    return found ? found[1] : null;
  }

  has(key: string) { return this._entries.some(([k]) => k === key); }
  toString() { return this._entries.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`).join('&'); }
}

// CSS object: components may reference CSS.supports() at module scope
// for feature detection.
const CSSShim = {
  supports() { return false; },
  escape(s: string) { return s; },
};

import { processHTML } from './harness/render.js';
import { readLine, writeStdout, writeStderr } from './io.js';

// Install SSR DOM shims as globals. Consumer bundles bring their own
// Lit copy which references these via the litSsrWasmPlugin esbuild
// plugin (resolves @lit-labs/ssr-dom-shim to globalThis re-exports).
import {
  HTMLElement,
  Element,
  CustomElementRegistry,
  customElements,
  CSSStyleSheet,
  Event,
  CustomEvent,
  EventTarget,
  ariaMixinAttributes,
  HYDRATE_INTERNALS_ATTR_PREFIX,
} from '@lit-labs/ssr-dom-shim';

Object.assign(globalThis, {
  // DOM shims from @lit-labs/ssr-dom-shim
  HTMLElement,
  Element,
  CustomElementRegistry,
  customElements,
  CSSStyleSheet,
  Event,
  CustomEvent,
  EventTarget,
  ariaMixinAttributes,
  HYDRATE_INTERNALS_ATTR_PREFIX,
  // Web API shims missing from QuickJS
  btoa,
  atob,
  URL: URLShim,
  URLSearchParams: URLSearchParamsShim,
  CSS: CSSShim,
  // Buffer shim: Lit internals (e.g. @lit-labs/ssr-client digest computation)
  // reference Buffer globally. The binary/base64 path delegates to btoa.
  Buffer: {
    from(x: unknown, encoding?: string) {
      if (typeof x === 'string') {
        if (encoding === 'binary') {
          return {
            toString(enc?: string) {
              if (enc === 'base64') return btoa(x);
              return x;
            },
          };
        }
        return new TextEncoder().encode(x);
      }
      return new Uint8Array(x as ArrayBuffer);
    },
    isBuffer() { return false; },
    alloc(n: number) { return new Uint8Array(n); },
  },
  // Minimal Document shim needed by Lit's supportsAdoptingStyleSheets check
  Document: class Document {
    get adoptedStyleSheets() { return []; }
    createTreeWalker() { return {}; }
    createTextNode() { return {}; }
    createElement() { return {}; }
  },
  ShadowRoot: class ShadowRoot {},
  MutationObserver: class MutationObserver { observe() {} },
  requestAnimationFrame() {},
});

// @ts-expect-error -- minimal Document instance for Lit
globalThis.document = new globalThis.Document();

// Two-phase protocol:
//   1. Init: JSON line {"source":"...","elements":[...]} -> eval, ack \0
//   2. Render: raw HTML line -> rendered HTML + \0
// Source is sent once at init, not on every render.

// Phase 1: read init message
const initLine = readLine();
if (initLine === null) throw new Error('unexpected EOF before init');

let known: Set<string>;
try {
  const init = JSON.parse(initLine) as { source: string; elements: string[] };
  known = new Set(init.elements);
  if (init.source) {
    (0, eval)(init.source);
  }
  writeStdout('\0'); // ack
} catch (e: unknown) {
  const msg = e instanceof Error ? e.message : String(e);
  writeStderr('init: ' + msg + '\n');
  writeStdout('\0');
  throw e;
}

// Phase 2: render loop -- raw HTML lines in, NUL-terminated HTML out
for (;;) {
  const html = readLine();
  if (html === null) break;
  if (html.trim() === '') continue;

  try {
    const output = processHTML(html, known);
    writeStdout(output + '\0');
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    writeStderr(msg + '\n');
    writeStdout('\0');
  }
}
