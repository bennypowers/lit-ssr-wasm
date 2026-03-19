import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';

@customElement('highlighted-textarea')
export class HighlightedTextarea extends LitElement {
  @property({ type: String, reflect: true })
  accessor language: 'html' | 'javascript' | 'css' = 'html';

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

  override render() {
    return html`
      <div id="container">
        <pre aria-hidden="true"><code></code></pre>
        <textarea spellcheck="false"></textarea>
      </div>
      <slot hidden></slot>
    `;
  }
}
