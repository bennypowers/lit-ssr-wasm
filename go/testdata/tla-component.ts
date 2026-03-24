import { LitElement, html, css } from 'lit';

const data = await Promise.resolve('async-loaded');

class TlaEl extends LitElement {
  static styles = css`:host { display: block; }`;
  render() { return html`<p>${data}</p>`; }
}
customElements.define('tla-el', TlaEl);
