// Test components for Go tests.
// Bundled by esbuild with litSsrWasmPlugin, which resolves
// @lit-labs/ssr-dom-shim to globalThis re-exports.

import { LitElement, html, css } from 'lit';

class TestCard extends LitElement {
  static styles = css`
    :host { display: block; border: 1px solid #ccc; padding: 1rem; }
    ::slotted([slot="header"]) { font-weight: bold; }
  `;

  override render() {
    return html`
      <div class="card">
        <slot name="header"></slot>
        <slot></slot>
        <slot name="footer"></slot>
      </div>
    `;
  }
}
customElements.define('test-card', TestCard);

class TestBadge extends LitElement {
  static properties = {
    state: { type: String },
  };

  declare state: string;

  constructor() {
    super();
    this.state = 'neutral';
  }

  static styles = css`
    :host { display: inline-block; padding: 0.25em 0.5em; border-radius: 4px; }
  `;

  override render() {
    return html`<span class="${this.state}"><slot></slot></span>`;
  }
}
customElements.define('test-badge', TestBadge);

// Test component using CSSStyleSheet (CSS module import pattern)
// instead of Lit's css tag, to verify the ssr-css-fix works.
const sheet = new CSSStyleSheet();
sheet.replaceSync(':host { display: block; color: green; }');

class TestSheet extends LitElement {
  static styles = sheet;

  override render() {
    return html`<p><slot></slot></p>`;
  }
}
customElements.define('test-sheet', TestSheet);
