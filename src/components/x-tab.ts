import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';

@customElement('x-tab')
export class XTab extends LitElement {
  @property({ type: Boolean, reflect: true })
  accessor active: boolean = false;

  @property({ type: Boolean, reflect: true })
  accessor disabled: boolean = false;

  static styles = css`
    :host { display: flex; flex: none; }

    #button {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 8px 16px;
      cursor: pointer;
      border: none;
      background: transparent;
      font: inherit;
      white-space: nowrap;
      border-block-end: 3px solid transparent;
      color: light-dark(#6a6e73, #c9c9c9);
    }

    :host([active]) #button {
      border-block-end-color: light-dark(#06c, #73bcf7);
      color: light-dark(#151515, #fff);
    }
  `;

  override render() {
    return html`
      <div id="button" role="tab" part="button" tabindex="0">
        <slot name="icon" part="icon"></slot>
        <slot part="text"></slot>
      </div>`;
  }
}
