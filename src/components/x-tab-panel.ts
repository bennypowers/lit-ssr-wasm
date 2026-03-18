import { LitElement, html, css } from 'lit';
import { customElement } from 'lit/decorators.js';

@customElement('x-tab-panel')
export class XTabPanel extends LitElement {
  static styles = css`
    :host { display: block; padding: 24px; }
  `;

  override render() {
    return html`<div id="container" role="tabpanel"><slot></slot></div>`;
  }
}
