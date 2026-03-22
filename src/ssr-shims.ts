/**
 * SSR environment shims for QuickJS.
 *
 * Sets up globals that @lit-labs/ssr and Lit expect from a Node.js
 * environment. Must be imported (and thus evaluated) before any
 * Lit modules load.
 */

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

// btoa / atob: needed by @lit-labs/ssr-client digestForTemplateResult()
// for base64-encoded template digests (hydration matching).
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

// Minimal URL / URLSearchParams
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
  CSS: { supports() { return false; }, escape(s: string) { return s; } },
  // Buffer shim: Lit internals reference Buffer globally.
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
  // Minimal Document shim for Lit's import-time initialization.
  Document: class Document {
    get adoptedStyleSheets() { return []; }
    createTreeWalker() {
      return { currentNode: null as unknown, nextNode() { return null; } };
    }
    createTextNode(data = '') { return { nodeType: 3, data, textContent: data }; }
    createComment(data = '') { return { nodeType: 8, data }; }
    createElement(tag: string) {
      if (tag.toLowerCase() === 'template') return { tagName: 'TEMPLATE', content: {} };
      return { tagName: tag.toUpperCase() };
    }
  },
  ShadowRoot: class ShadowRoot {},
  MutationObserver: class MutationObserver { observe() {} },
  requestAnimationFrame() {},
});

// @ts-expect-error -- minimal Document instance for Lit
globalThis.document = new globalThis.Document();
