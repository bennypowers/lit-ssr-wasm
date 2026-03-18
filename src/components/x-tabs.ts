import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';

@customElement('x-tabs')
export class XTabs extends LitElement {
  @property({ type: Boolean, reflect: true })
  accessor vertical: boolean = false;

  static styles = css`
    :host { display: block; }

    #tablist {
      display: flex;
      overflow-x: auto;
      border-block-end: 1px solid light-dark(#d2d2d2, #444);
    }

    :host([vertical]) #tablist {
      flex-direction: column;
      border-block-end: none;
      border-inline-end: 1px solid light-dark(#d2d2d2, #444);
    }
  `;

  override render() {
    return html`
      <div id="container">
        <div id="tablist" role="tablist" part="tabs">
          <slot name="tab"></slot>
        </div>
        <slot part="panels"></slot>
      </div>`;
  }
}
