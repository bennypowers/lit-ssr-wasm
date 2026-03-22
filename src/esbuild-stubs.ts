/**
 * esbuild plugin that stubs Node.js built-in modules for QuickJS.
 *
 * The Buffer.from(s, 'binary').toString('base64') path delegates to
 * globalThis.btoa, which the lit-ssr-wasm runtime provides.
 */

import type { Plugin } from 'esbuild';

export const stubNodeBuiltins: Plugin = {
  name: 'stub-node-builtins',
  setup(build) {
    const builtins = /^(?:node:)?(?:buffer|fs|path|stream|util|events|crypto|os|url|http|https|net|tls|child_process|module|vm|zlib)(?:\/|$)/;
    build.onResolve({ filter: builtins }, args => ({
      path: args.path,
      namespace: 'stub',
    }));
    build.onLoad({ filter: /.*/, namespace: 'stub' }, () => ({
      contents: [
        'export default {};',
        'export const Buffer = {',
        '  from(x, encoding) {',
        '    if (typeof x === "string") {',
        '      if (encoding === "binary") {',
        '        return {',
        '          toString(enc) {',
        '            if (enc === "base64") return globalThis.btoa(x);',
        '            return x;',
        '          },',
        '        };',
        '      }',
        '      return new TextEncoder().encode(x);',
        '    }',
        '    return new Uint8Array(x);',
        '  },',
        '  isBuffer: () => false,',
        '  alloc: n => new Uint8Array(n),',
        '};',
        'export const readFileSync = () => "";',
      ].join('\n'),
    }));
  },
};
