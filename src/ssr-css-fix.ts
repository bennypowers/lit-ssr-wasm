/**
 * Patch CSSStyleSheet.prototype.cssText for SSR compatibility.
 *
 * The @lit-labs/ssr-dom-shim provides a CSSStyleSheet shim that stores
 * CSS text in cssRules[0].cssText (via replaceSync), but Lit's
 * LitElementRenderer reads style.cssText directly. When components use
 * CSS module imports (import styles from './foo.css' with { type: 'css' })
 * instead of Lit's css tag, the styles end up as raw CSSStyleSheet
 * instances with no .cssText property, producing empty <style> tags.
 *
 * This must be imported before any Lit modules.
 */

import { CSSStyleSheet } from '@lit-labs/ssr-dom-shim';

if (!Object.getOwnPropertyDescriptor(CSSStyleSheet.prototype, 'cssText')) {
  Object.defineProperty(CSSStyleSheet.prototype, 'cssText', {
    get(this: { cssRules: ArrayLike<{ cssText: string }> }) {
      return Array.from(this.cssRules).map(r => r.cssText).join('');
    },
    configurable: true,
  });
}
