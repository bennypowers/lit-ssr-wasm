import { LitElement, html, css } from 'lit';

export class ExportedEl extends LitElement {
  static styles = css`:host { display: block; }`;
  render() { return html`<p>exported</p>`; }
}
customElements.define('exported-el', ExportedEl);
