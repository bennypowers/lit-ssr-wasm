import { LitElement, html, css } from 'lit';

class UnbundledEl extends LitElement {
  static styles = css`:host { display: block; color: blue; }`;
  render() { return html`<p>unbundled</p>`; }
}
customElements.define('unbundled-el', UnbundledEl);
