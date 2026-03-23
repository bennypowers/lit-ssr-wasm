import { LitElement, html } from 'lit';
import styles from './styles.css' with { type: 'css' };

class CssEl extends LitElement {
  static styles = styles;
  render() { return html`<p>css import</p>`; }
}
customElements.define('css-el', CssEl);
