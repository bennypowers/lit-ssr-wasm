import { LitElement, html, css } from 'lit';
import { Prism } from 'prism-esm';

const prism = new Prism();

const GRAMMARS = {
  html: prism.languages.markup,
  javascript: prism.languages.javascript,
  css: prism.languages.css,
};

class HighlightedTextarea extends LitElement {
  static properties = {
    language: { type: String, reflect: true },
  };

  static styles = css`
    :host {
      display: block;
      position: relative;
    }

    pre, textarea {
      font-family: var(--rh-font-family-code);
      font-size: var(--rh-font-size-code-sm);
      line-height: var(--rh-line-height-code);
      padding: var(--rh-space-lg);
      white-space: pre-wrap;
      overflow-wrap: break-word;
      tab-size: 4;
      margin: 0;
      border: 0;
    }

    pre {
      position: absolute;
      inset: 0;
      overflow: auto;
      pointer-events: none;
      color: light-dark(var(--rh-color-text-primary-on-light), var(--rh-color-text-primary-on-dark));

      code {
        font: inherit;
      }

      .token {
        &:is(.comment, .prolog, .doctype, .cdata) { color: light-dark(#6a737d, #8b949e); }
        &.punctuation { color: light-dark(#a3a3a3, #e0e0e0); }
        &:is(.property, .tag, .boolean, .number, .constant, .symbol, .deleted) { color: light-dark(#5e40be, #dca614); }
        &:is(.selector, .attr-name, .string, .char, .builtin, .inserted) { color: light-dark(#147878, #87bb62); }
        &:is(.operator, .entity, .url) { color: light-dark(#96640f, #4394e5); }
        &:is(.atrule, .attr-value, .keyword) { color: light-dark(#004d99, #b6a6e9); }
        &:is(.function, .class-name) { color: light-dark(#a60000, #f5921b); }
        &:is(.regex, .important, .variable) { color: light-dark(#9e4a06, #87bb62); }
      }
    }

    textarea {
      display: block;
      width: 100%;
      min-height: 16rem;
      resize: vertical;
      background: transparent;
      color: transparent;
      caret-color: light-dark(var(--rh-color-text-primary-on-light), var(--rh-color-text-primary-on-dark));

      &::selection {
        background: light-dark(rgba(0, 102, 204, 0.3), rgba(100, 160, 255, 0.3));
      }

      &:focus-visible {
        outline: var(--rh-border-width-md) solid light-dark(var(--rh-color-interactive-primary-default-on-light), var(--rh-color-interactive-primary-default-on-dark));
        outline-offset: var(--rh-border-width-sm);
      }
    }

    #container {
      position: relative;
      background: light-dark(var(--rh-color-surface-lighter), var(--rh-color-surface-darker));
      border: var(--rh-border-width-sm) solid light-dark(var(--rh-color-border-subtle-on-light), var(--rh-color-border-subtle-on-dark));
      border-radius: var(--rh-border-radius-default);
      overflow: hidden;
    }
  `;

  #initialContent;

  constructor() {
    super();
    this.language = 'html';
    this.#initialContent = this.textContent?.trim() || '';
  }

  get value() {
    return this.renderRoot?.querySelector('textarea')?.value ?? '';
  }

  set value(v) {
    const ta = this.renderRoot?.querySelector('textarea');
    if (ta) { ta.value = v; this.#sync(); }
  }

  render() {
    return html`
      <div id="container">
        <pre aria-hidden="true"><code></code></pre>
        <textarea spellcheck="false"></textarea>
      </div>
      <slot hidden></slot>
    `;
  }

  updated() {
    if (!this.hasUpdated) return;
    // After first hydration/render, wire up events and populate
    const ta = this.renderRoot?.querySelector('textarea');
    if (!ta) return;

    if (!ta._wired) {
      ta._wired = true;
      if (this.#initialContent) {
        ta.value = this.#initialContent;
      }
      ta.addEventListener('input', () => this.#sync());
      ta.addEventListener('scroll', () => {
        const pre = this.renderRoot.querySelector('pre');
        pre.scrollTop = ta.scrollTop;
        pre.scrollLeft = ta.scrollLeft;
      });
      this.#sync();
    }
  }

  #sync() {
    const ta = this.renderRoot?.querySelector('textarea');
    const code = this.renderRoot?.querySelector('code');
    if (!ta || !code) return;
    const grammar = GRAMMARS[this.language];
    if (grammar) {
      code.innerHTML = prism.highlight(ta.value + '\n', grammar, this.language);
    } else {
      code.textContent = ta.value + '\n';
    }
  }
}

customElements.define('highlighted-textarea', HighlightedTextarea);
