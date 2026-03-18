import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';

@customElement('x-cta')
export class XCta extends LitElement {
  @property({ reflect: true })
  accessor variant: 'primary' | 'secondary' | undefined;

  @property()
  accessor href: string | undefined;

  static styles = css`
    :host { display: inline-flex; }

    #container {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      border-radius: 4px;
      font: inherit;
      text-decoration: none;
      cursor: pointer;
    }

    :host([variant="primary"]) #container {
      background: light-dark(#06c, #73bcf7);
      color: light-dark(#fff, #000);
      padding: 8px 24px;
    }

    :host([variant="secondary"]) #container {
      border: 1px solid currentColor;
      padding: 8px 24px;
    }

    ::slotted(a) { color: inherit; text-decoration: none; }
  `;

  override render() {
    return html`<span id="container" part="container"><slot></slot></span>`;
  }
}
