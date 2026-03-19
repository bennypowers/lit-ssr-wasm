import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import { classMap } from 'lit/directives/class-map.js';

@customElement('my-alert')
export class MyAlert extends LitElement {

  static styles = css`
    #container {
      display: block;
      padding: 1rem;
      border-radius: 8px;
      margin: 0.5rem;

      &.success {
        background: light-dark(#d4edda, #1a3a2a);
        color: light-dark(#155724, #a3d9a5);
        border: 1px solid light-dark(#c3e6cb, #2d5a3d);
      }

      &.error {
        background: light-dark(#f8d7da, #3a1a1a);
        color: light-dark(#721c24, #e5a3a3);
        border: 1px solid light-dark(#f5c6cb, #5a2d2d);
      }

      &.info {
        background: light-dark(#cce5ff, #1a2a3a);
        color: light-dark(#004085, #a3c4e5);
        border: 1px solid light-dark(#b8daff, #2d3d5a);
      }
    }

    #icon { margin-inline-end: 0.5rem; }
  `;

  /** Sets the visual style of the alert */
  @property({ type: String, reflect: true })
  accessor type: 'success' | 'error' | 'info' = 'info';

  get #icon() {
    switch (this.type) {
      case 'success': return '\u2705';
      case 'error':   return '\u274c';
      case 'info':    return '\u2139\ufe0f';
      default:        return '\u2139\ufe0f';
    }
  }

  override render() {
    const { type } = this;
    return html`
      <div id="container" class="${classMap({[type]: !!type})}">
        <span id="icon">${this.#icon}</span>
        <slot></slot>
      </div>
    `;
  }
}
