import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';

@customElement('x-card')
export class XCard extends LitElement {
  @property({ reflect: true })
  accessor variant: 'promo' | undefined;

  static styles = css`
    :host {
      display: block;
      container: card / inline-size;
      border: 1px solid light-dark(#d2d2d2, #444);
      border-radius: 8px;
      background: light-dark(#fff, #1a1a1a);
      color: light-dark(#151515, #e0e0e0);
    }

    #container {
      display: flex;
      flex-direction: column;
      height: 100%;
    }

    #header { padding: 16px 24px 0; font-weight: 600; }
    #body   { flex: 1; padding: 16px 24px; }
    #footer { padding: 0 24px 16px; }

    #image ::slotted(*) { width: 100%; }
  `;

  override render() {
    return html`
      <div id="container" part="container">
        <div id="header" part="header"><slot name="header"></slot></div>
        <div id="image"  part="image"><slot name="image"></slot></div>
        <div id="body"   part="body"><slot></slot></div>
        <div id="footer" part="footer"><slot name="footer"></slot></div>
      </div>`;
  }
}
