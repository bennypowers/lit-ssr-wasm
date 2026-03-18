import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';

type BadgeState = 'danger' | 'warning' | 'neutral' | 'success' | 'info';

@customElement('x-badge')
export class XBadge extends LitElement {
  @property({ reflect: true })
  accessor state: BadgeState = 'neutral';

  @property({ type: Number })
  accessor number: number | undefined;

  @property({ type: Number })
  accessor threshold: number | undefined;

  static styles = css`
    :host { display: inline-block; }

    span {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 1.5em;
      padding: 0.125em 0.5em;
      border-radius: 64px;
      font-size: 0.75rem;
      font-weight: 700;
    }

    :host([state="info"])    span { background: #06c;    color: #fff; }
    :host([state="success"]) span { background: #3e8635; color: #fff; }
    :host([state="warning"]) span { background: #f0ab00; color: #000; }
    :host([state="danger"])  span { background: #c9190b; color: #fff; }
    :host([state="neutral"]) span { background: #6a6e73; color: #fff; }
  `;

  get #displayText(): string {
    if (this.number == null) return '';
    if (this.threshold != null && this.number > this.threshold) return `${this.threshold}+`;
    return String(this.number);
  }

  override render() {
    return html`<span>${this.#displayText}</span><slot></slot>`;
  }
}
