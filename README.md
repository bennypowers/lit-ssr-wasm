# lit-ssr-wasm

Server-side render [Lit][lit] web components from **any language** via
WebAssembly.

Standard LitElement components are bundled with [`@lit-labs/ssr`][lit-ssr],
compiled to WebAssembly via [Javy][javy] (QuickJS), and executed from Go using
[wazero][wazero] -- a pure-Go WASM runtime with no CGo dependency.

The result: **Declarative Shadow DOM** injected into your HTML on the server,
giving users instant first paint with zero layout shift. No Node.js sidecar
required.

```
LitElement (TS) --> esbuild --> Javy (QuickJS) --> .wasm (2 MB) --> Go / wazero
```

## Live Demo

**[Try it in your browser][demo]**

The demo runs Lit SSR directly in the browser (the same rendering engine used
in the WASM module), so you can experiment with both modes interactively.

---

## Two Modes

### Compiled Mode

Component definitions are baked into the WASM module at build time. Fast,
predictable, and ideal for:

- Static site generators (like `cem serve`)
- Build pipelines where components are known ahead of time
- Applications where render latency matters (~350ms cold start)

### Runtime Mode

Component definitions are provided as JavaScript source at render time.
Flexible and dynamic, ideal for:

- Design tools and component playgrounds
- Scenarios where components change without rebuilding the WASM module
- Development servers that pick up component changes on the fly

---

## Quick Start

### Prerequisites

- [Node.js][nodejs] >= 18
- [Javy][javy] >= 8.0
- [Go][golang] >= 1.21 (for the Go library)

### Install and Build

```sh
npm install
npm run build          # Produces dist/lit-ssr-bundle.mjs and dist/lit-ssr-javy.mjs
npm run build:wasm     # Compiles to dist/lit-ssr.wasm (requires javy in PATH)
```

### Test with Node.js

```sh
echo '<x-card><h2 slot="header">Hello</h2><p>World</p></x-card>' \
  | node dist/lit-ssr-bundle.mjs
```

### Test with Go

```sh
cp dist/lit-ssr.wasm go/lit-ssr.wasm
cd go && go test -v
```

---

## Project Structure

```
lit-ssr-wasm/
+-- src/
|   +-- harness/
|   |   +-- render.ts         # Core SSR pipeline (processHTML, renderElementToString)
|   +-- io/
|   |   +-- javy.ts           # Javy.IO stdin/stdout for WASM
|   |   +-- node.ts           # node:fs stdin/stdout for Node.js
|   +-- components/            # Example Lit elements (x-card, x-tabs, etc.)
|   +-- demo/
|   |   +-- compiled.ts       # Browser bundle: components baked in
|   |   +-- runtime.ts        # Browser bundle: dynamic component loading
|   +-- entry.ts              # WASM/CLI entry point
+-- scripts/
|   +-- build.ts              # esbuild: ESM for Node + ESM for Javy
|   +-- build-demo.ts         # Builds GH Pages demo bundles
+-- go/
|   +-- litssr.go             # Go package: Renderer using wazero
|   +-- litssr_test.go        # Go tests
|   +-- go.mod
+-- docs/                      # GitHub Pages demo site
|   +-- index.html            # Side-by-side compiled and runtime demos
|   +-- compiled.html         # Compiled mode standalone demo
|   +-- runtime.html          # Runtime mode standalone demo
+-- demo/fixtures/             # Example HTML inputs
```

---

## How It Works

### The Rendering Pipeline

1. **Author components** as standard LitElement classes in TypeScript
2. **Bundle** with [esbuild][esbuild] into a single ESM file, with Node.js
   built-in modules stubbed out (QuickJS does not need them)
3. **Compile** to WebAssembly with [Javy][javy], which embeds QuickJS as the JS
   runtime
4. **Execute** from Go via [wazero][wazero]: pipe HTML into stdin, get
   DSD-enhanced HTML from stdout

### Why This Works

Lit SSR deliberately avoids full DOM emulation. It intercepts Lit's template
system at the string level, using a minimal DOM shim
(`@lit-labs/ssr-dom-shim`). This makes it feasible to run in QuickJS via WASM
-- you do not need a full browser engine, just enough JS runtime to execute
Lit's template rendering logic.

The WASM module communicates via WASI stdin/stdout, making it callable from any
language with a WASI-compatible WASM runtime.

### Stubbing Node.js APIs

[`@lit-labs/ssr`][lit-ssr] depends on [parse5][parse5] (an HTML parser) which
imports Node's `buffer` module. Since QuickJS does not provide Node.js built-in
modules, the build script replaces them with lightweight stubs. The `Buffer`
stub wraps `TextEncoder`/`Uint8Array`, which is sufficient for the SSR code
path.

This is safe because Lit SSR's rendering pipeline only needs string
manipulation capabilities, not actual filesystem or network access.

---

## Go Library API

The `go/` directory contains a self-contained Go package with the WASM module
embedded via `//go:embed`.

```go
package main

import (
    "context"
    "fmt"
    "log"

    litssr "bennypowers.dev/lit-ssr-go"
)

func main() {
    ctx := context.Background()

    renderer, err := litssr.New(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer renderer.Close(ctx)

    html := `<x-card><h2 slot="header">Hello</h2><p>World</p></x-card>`

    output, err := renderer.RenderHTML(ctx, html)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(output)
}
```

### API

| Function | Description |
|---|---|
| `New(ctx) (*Renderer, error)` | Create a renderer. Pre-compiles the WASM module. |
| `(*Renderer).RenderHTML(ctx, html) (string, error)` | Render HTML with DSD injection. |
| `(*Renderer).Close(ctx) error` | Release all resources. |

### Performance

| Metric | Value |
|---|---|
| WASM module size | ~2 MB |
| Cold start (first render) | ~350ms |
| Subsequent renders | ~320ms |
| Go dependency | [wazero][wazero] only (pure Go, no CGo) |

Each render creates a fresh WASM module instance. For production use, consider:

- Pooling renderer instances across goroutines
- Batching multiple elements into a single render call
- Pre-warming the renderer at application startup

---

## Adding Your Own Components

### Compiled Mode

1. Create a LitElement in `src/components/my-element.ts`
2. Import it in `src/entry.ts`
3. Add the tag name to the `KNOWN_ELEMENTS` set
4. Rebuild: `npm run build && npm run build:wasm`

### Runtime Mode

Provide your component source as JavaScript at render time. The source must
call `customElements.define()` to register the element. See the
[runtime demo][demo-runtime] for a live example.

---

## Example Components

The repo includes six example components to demonstrate the approach:

| Component | Description |
|---|---|
| `<x-card>` | Card with header, body, image, and footer slots |
| `<x-cta>` | Call-to-action (variants: primary, secondary) |
| `<x-tabs>` | Tab container |
| `<x-tab>` | Tab trigger (attributes: active, disabled) |
| `<x-tab-panel>` | Tab panel content |
| `<x-badge>` | Status badge (states: success, warning, danger, info, neutral) |

All components use TypeScript with standard ES2025 decorators and the
`accessor` keyword:

```typescript
import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';

@customElement('x-badge')
export class XBadge extends LitElement {
  @property({ reflect: true })
  accessor state: 'success' | 'warning' | 'danger' | 'info' | 'neutral' = 'neutral';

  static styles = css`
    :host { display: inline-block; }
    span {
      padding: 0.125em 0.5em;
      border-radius: 64px;
      font-size: 0.75rem;
      font-weight: 700;
    }
    :host([state="success"]) span { background: #3e8635; color: #fff; }
    :host([state="danger"])  span { background: #c9190b; color: #fff; }
  `;

  override render() {
    return html`<span><slot></slot></span>`;
  }
}
```

---

## Relationship to Lit SSR WASM Issue

This project is a proof of concept responding to [lit/lit#4611][lit-issue],
which requested a WASM option for `@lit-labs/ssr`. Justin Fagnani (Lit
maintainer) commented:

> "It seems like a lot of it should be trying to run the existing SSR code
> within one of the JS-in-WASM toolchains. It would be great to see someone try
> that and report back."

This project proves it works: real Lit SSR, real LitElements, compiled to WASM
via Javy, callable from Go with zero Node.js dependency.

---

## License

MIT

<!-- Reference links -->
[lit]: https://lit.dev
[lit-ssr]: https://www.npmjs.com/package/@lit-labs/ssr
[lit-issue]: https://github.com/lit/lit/issues/4611
[javy]: https://github.com/bytecodealliance/javy
[wazero]: https://wazero.io
[esbuild]: https://esbuild.github.io
[parse5]: https://www.npmjs.com/package/parse5
[nodejs]: https://nodejs.org
[golang]: https://go.dev
[demo]: https://bennypowers.github.io/lit-ssr-wasm/
[demo-runtime]: https://bennypowers.github.io/lit-ssr-wasm/runtime.html
