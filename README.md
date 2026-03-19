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

The demo runs the actual WASM modules directly in the browser using a minimal
WASI shim. The compiled mode demo does not load any JavaScript definition of
`<my-alert>` on the page -- the styles you see come entirely from Declarative
Shadow DOM output by the WASM module.

---

## Two WASM Modules

### Builtin Mode (`lit-ssr-builtin.wasm`)

Component definitions are baked into the WASM module at build time. The module
reads HTML from stdin and writes DSD-enhanced HTML to stdout.

```sh
printf '<x-card><h2 slot="header">Hello</h2><p>World</p></x-card>\0' \
  | wasmtime lit-ssr-builtin.wasm
```

Fast and predictable -- ideal for static site generators, build pipelines, or
any scenario where components are known ahead of time.

### Runtime Mode (`lit-ssr-runtime.wasm`)

No component definitions are baked in. The module reads JSON from stdin
containing the component JavaScript source, HTML to render, and element tag
names. Lit APIs (`LitElement`, `html`, `css`, `classMap`, etc.) are exposed as
globals inside the QuickJS context, so user-provided source does not need import
statements. The source is evaluated internally by QuickJS, registering custom
elements, then the HTML is rendered with DSD.

```sh
echo '{"source":"class MyEl extends LitElement { ... }","html":"<my-el></my-el>","elements":["my-el"]}' \
  | wasmtime lit-ssr-runtime.wasm
```

Ideal for design tools, component playgrounds, or development servers that pick
up component changes on the fly.

---

## Quick Start

### Prerequisites

- [Node.js][nodejs] >= 18
- [Javy][javy] >= 8.0
- [Go][golang] >= 1.21 (for the Go library)

### Install and Build

```sh
npm install
npm run build     # Builds everything: JS bundles, WASM modules, demo site
```

### Development

```sh
npm start         # Live-reloading dev server on http://localhost:3000
```

### Test with Node.js

```sh
printf '<x-card><h2 slot="header">Hello</h2><p>World</p></x-card>\0' \
  | node dist/lit-ssr-bundle.js
```

### Test with Go

```sh
cp dist/lit-ssr-builtin.wasm go/lit-ssr.wasm
cd go && go test -v
```

---

## Wire Protocol

Both WASM modules use a read-loop protocol that keeps instances warm across
multiple renders, amortizing the ~350ms cold start.

### Builtin mode

NUL-delimited on both sides:

```
stdin:  <raw HTML>\0
stdout: <rendered HTML>\0
```

### Runtime mode

JSON line in, NUL-delimited HTML out:

```
stdin:  {"source":"...","html":"...","elements":[...]}\n
stdout: <rendered HTML>\0
```

Errors are written to stderr. On error, stdout gets an empty response (`\0`).

The module exits cleanly when stdin reaches EOF. Send multiple requests on
the same stdin pipe to reuse the warm instance.

---

## Project Structure

```
lit-ssr-wasm/
+-- src/
|   +-- harness/
|   |   +-- render.ts             # Core SSR pipeline
|   +-- io/
|   |   +-- javy.ts               # Javy.IO for WASM (readUntilNul, readLine)
|   |   +-- node.ts               # node:fs for Node.js
|   +-- components/               # Lit elements (x-card, x-tabs, my-alert, etc.)
|   +-- entry.ts                  # Builtin mode WASM entry (read loop)
|   +-- runtime-entry.ts          # Runtime mode WASM entry (read loop)
|   +-- ssr-rhds.ts               # SSR entry for demo site chrome
+-- scripts/
|   +-- build.ts                  # esbuild: JS bundles for Node + Javy
|   +-- build-demo.ts             # Assembles demo site into _site/
|   +-- dev-server.ts             # Live-reloading dev server
+-- go/
|   +-- litssr.go                 # Go package: worker pool using wazero
|   +-- litssr_test.go            # Tests + benchmarks
|   +-- go.mod
+-- docs/                         # Demo site source (pages, layout, CSS)
|   +-- _layout.html              # Shared chrome (header, subnav, import map)
|   +-- compiled.html             # Compiled mode demo
|   +-- runtime.html              # Runtime mode demo
|   +-- index.html                # How it Works page
|   +-- wasi-shim.js              # Browser WASI shim for running WASM
|   +-- highlighted-textarea.js   # Prism-highlighted textarea component
+-- _site/                        # Build output (gitignored)
```

---

## How It Works

### The Rendering Pipeline

1. **Author components** as standard LitElement classes in TypeScript
2. **Bundle** with [esbuild][esbuild] into a single ESM file. Node.js built-in
   modules are stubbed with lightweight shims (`Buffer` wraps
   `TextEncoder`/`Uint8Array`)
3. **Compile** to WebAssembly with [Javy][javy]. The JS bundle and QuickJS
   engine are embedded together in one `.wasm` file (~2 MB)
4. **Execute** from any WASI host: pipe HTML to stdin, get DSD-enhanced HTML
   from stdout

### Why This Works

Lit SSR deliberately avoids full DOM emulation. It intercepts Lit's template
system at the string level, using a minimal DOM shim
(`@lit-labs/ssr-dom-shim`) that provides just enough of the `HTMLElement` and
`customElements` API for Lit's rendering logic. This means it runs comfortably
in QuickJS -- no full browser engine needed.

### Declarative Shadow DOM

The WASM module's output includes `<template shadowrootmode="open">` elements
containing the component's styles and rendered shadow DOM. The browser attaches
these as shadow roots during HTML parsing, before any JavaScript runs. Users see
styled, laid-out content on first paint with zero layout shift.

The component's JavaScript definition is not needed on the page for DSD styles
to apply.

---

## Go Library API

The `go/` directory contains a self-contained Go package with the WASM module
embedded via `//go:embed`. The renderer manages a pool of warm WASM instances
for concurrent rendering.

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

    // Start a renderer pool (0 = one worker per CPU core)
    renderer, err := litssr.New(ctx, 0)
    if err != nil {
        log.Fatal(err)
    }
    defer renderer.Close(ctx)

    // Single render
    html, err := renderer.RenderHTML(ctx, `<x-card><h2 slot="header">Hello</h2><p>World</p></x-card>`)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(html)

    // Batch render (distributed across workers)
    inputs := []string{
        `<x-card><p>One</p></x-card>`,
        `<x-card><p>Two</p></x-card>`,
        `<x-card><p>Three</p></x-card>`,
    }
    results, err := renderer.RenderBatch(ctx, inputs)
    if err != nil {
        log.Fatal(err)
    }
    for _, r := range results {
        fmt.Println(r)
    }
}
```

### API

| Function | Description |
|---|---|
| `New(ctx, workers) (*Renderer, error)` | Create a renderer pool. 0 workers = `runtime.NumCPU()`. |
| `(*Renderer).RenderHTML(ctx, html) (string, error)` | Render HTML with DSD. Concurrent-safe. |
| `(*Renderer).RenderBatch(ctx, inputs) ([]string, error)` | Batch render across workers. Ordered results. |
| `(*Renderer).Close(ctx) error` | Shut down workers and release resources. |

### Performance

| Metric | Value |
|---|---|
| WASM module size | ~2 MB |
| Cold start (pool init) | ~350ms |
| Warm render (single) | ~0.32ms |
| Batch of 50 | ~1ms total |
| Go dependency | [wazero][wazero] only (pure Go, no CGo) |

WASM instances stay warm across renders via the read-loop protocol. The ~350ms
cold start is paid once at pool initialization. Subsequent renders run at
~0.32ms each, or ~20us/render when batched across multiple CPU cores.

---

## Building Your Own WASM Module

The builtin WASM module ships with the example components. To build a module
with your own components:

### 1. Write your components

Standard LitElement classes in TypeScript. Nothing special needed.

```typescript
// src/components/my-widget.ts
import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';

@customElement('my-widget')
export class MyWidget extends LitElement {
  @property() accessor label = '';

  static styles = css`:host { display: block; }`;

  override render() {
    return html`<div>${this.label}</div>`;
  }
}
```

### 2. Create an entry point

Import your components and list their tag names:

```typescript
// src/entry.ts
import { processHTML } from './harness/render.js';
import { readUntilNul, writeStdout, writeStderr } from './io.js';

import './components/my-widget.js';

const KNOWN = new Set(['my-widget']);

for (;;) {
  const input = readUntilNul();
  if (input === null) break;
  try {
    writeStdout(processHTML(input, KNOWN) + '\0');
  } catch (e) {
    writeStderr((e instanceof Error ? e.message : String(e)) + '\n');
    writeStdout('\0');
  }
}
```

### 3. Build

```sh
npm run build    # Produces dist/lit-ssr-builtin.wasm
```

### 4. Use from Go

Copy the WASM into your Go module and embed it:

```go
package myssr

import (
    "context"
    _ "embed"
    // same pool/worker structure as go/litssr.go
)

//go:embed my-components.wasm
var wasmBytes []byte
```

Or use the `go/litssr.go` package directly by replacing `go/lit-ssr.wasm` with
your custom-built module.

### Runtime Mode

Alternatively, use `lit-ssr-runtime.wasm` to render any component without
rebuilding. Provide the component source as JavaScript at render time. Note
that QuickJS does not support TC39 decorators -- use `static properties` and
`customElements.define()` instead. See the [runtime demo][demo-runtime] for a
live example.

---

## Example Components

The repo includes example components to demonstrate the approach:

| Component | Description |
|---|---|
| `<x-card>` | Card with header, body, image, and footer slots |
| `<x-cta>` | Call-to-action (variants: primary, secondary) |
| `<x-tabs>` | Tab container |
| `<x-tab>` | Tab trigger (attributes: active, disabled) |
| `<x-tab-panel>` | Tab panel content |
| `<x-badge>` | Status badge (states: success, warning, danger, info, neutral) |
| `<my-alert>` | Alert banner (states: success, error, info) with light-dark() |

---

## Relationship to Lit SSR WASM Issue

This project is a proof of concept responding to [lit/lit#4611][lit-issue],
which requested a WASM option for `@lit-labs/ssr`. Justin Fagnani (Lit
maintainer) commented:

> "It seems like a lot of it should be trying to run the existing SSR code
> within one of the JS-in-WASM toolchains. It would be great to see someone try
> that and report back."

This project proves it works: real Lit SSR, real LitElements, compiled to WASM
via Javy, callable from Go with zero Node.js dependency. Warm renders at 0.32ms
with a worker pool.

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
[demo]: https://bennypowers.github.io/lit-ssr-wasm/compiled.html
[demo-runtime]: https://bennypowers.github.io/lit-ssr-wasm/runtime.html
