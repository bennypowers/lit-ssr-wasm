# lit-ssr-wasm

Server-side render [Lit][lit] web components from **any language** via
WebAssembly.

The [`@lit-labs/ssr`][lit-ssr] rendering engine is compiled to WebAssembly via
[Javy][javy] (QuickJS) and executed from Go using [wazero][wazero] -- a pure-Go
WASM runtime with no CGo dependency.

The result: **Declarative Shadow DOM** injected into your HTML on the server,
giving users instant first paint with zero layout shift. No Node.js sidecar
required.

## Live Demo

**[Try it in your browser][demo]**

The demo runs the actual WASM module directly in the browser using a minimal
WASI shim.

---

## How It Works

1. **Bundle** your LitElement components with [esbuild][esbuild], using the
   provided `litSsrWasmPlugin` to bridge `@lit-labs/ssr-dom-shim` to the
   WASM runtime's globals
2. **Pass** the bundled JS to the Go library or CLI
3. **Pipe** HTML to stdin, get DSD-enhanced HTML from stdout

The WASM module provides the SSR environment (DOM shims, `@lit-labs/ssr` render
engine, `btoa`/`atob`, `URL`, `CSS`, etc.). Your bundle provides Lit and your
component definitions.

### Declarative Shadow DOM

The output includes `<template shadowrootmode="open">` elements containing the
component's styles and rendered shadow DOM. The browser attaches these as shadow
roots during HTML parsing, before any JavaScript runs. Users see styled, laid-out
content on first paint with zero layout shift.

---

## Quick Start

### Prerequisites

- [Node.js][nodejs] >= 22
- [Javy][javy] >= 8.0
- [Go][golang] >= 1.25 (for the Go library)

### Install and Build

```sh
npm install
npm run build     # Builds WASM module + demo site
```

### Test with Go

```sh
cp dist/lit-ssr-runtime.wasm go/lit-ssr-runtime.wasm
cd go && go test -v
```

---

## Go Library

The `go/` directory contains a self-contained Go package with the runtime WASM
module embedded via `//go:embed`. The renderer manages a pool of warm WASM
instances for concurrent rendering.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    litssr "github.com/bennypowers/lit-ssr-wasm/go"
)

func main() {
    ctx := context.Background()

    // Read your pre-bundled component JS
    source, err := os.ReadFile("components.js")
    if err != nil {
        log.Fatal(err)
    }

    // Start a renderer pool (0 = one worker per CPU core)
    renderer, err := litssr.New(ctx, string(source), 0)
    if err != nil {
        log.Fatal(err)
    }
    defer renderer.Close(ctx)

    // Single render
    html, err := renderer.RenderHTML(ctx, `<my-card>Hello</my-card>`)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(html)

    // Batch render (distributed across workers)
    results, err := renderer.RenderBatch(ctx, []string{
        `<my-card>One</my-card>`,
        `<my-card>Two</my-card>`,
    })
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
| `New(ctx, source, workers)` | Create a renderer pool. Extracts element names from source via regex. 0 workers = `runtime.NumCPU()`. |
| `NewWithElements(ctx, source, elements, workers)` | Create a renderer pool with an explicit element list. |
| `RenderHTML(ctx, html)` | Render HTML with DSD. Concurrent-safe. |
| `RenderBatch(ctx, inputs)` | Batch render across workers. Ordered results. |
| `Close(ctx)` | Shut down workers and release resources. |

### Performance

| Metric | Value |
|---|---|
| WASM module size | ~2 MB |
| Cold start (pool init) | ~350ms |
| Warm render (single) | ~0.32ms |
| Batch of 50 | ~1ms total |
| Go dependency | [wazero][wazero] only (pure Go, no CGo) |

---

## CLI

A single `lit-ssr` binary accepts component JS via `--bundle`, `--dir`, or
`--components` flags. It reads NUL-terminated HTML from stdin and writes
NUL-terminated rendered HTML to stdout.

```sh
lit-ssr --bundle components.js
lit-ssr --dir ./components/
lit-ssr --components ./js/my-card.js --components ./js/my-alert.js
```

The NUL-delimited protocol is transparent to callers (e.g. PHP, Ruby).

---

## Bundling Components

Consumers bundle their components with [esbuild][esbuild] using two provided
plugins:

### `litSsrWasmPlugin()`

Resolves `@lit-labs/ssr-dom-shim` imports to `globalThis` re-exports, so the
consumer's Lit copy shares the same `customElements` registry and DOM shims
that the WASM runtime provides.

### `stubNodeBuiltins`

Stubs Node.js built-in modules (`fs`, `path`, `buffer`, etc.) for the QuickJS
environment. The `Buffer.from(s, 'binary').toString('base64')` path delegates
to `globalThis.btoa` for correct hydration digest computation.

### Example build script

The plugins are in `src/esbuild-plugin.ts` and `src/esbuild-stubs.ts`. Copy
them into your project or reference by path:

```typescript
import * as esbuild from 'esbuild';
import { litSsrWasmPlugin } from './path/to/esbuild-plugin.ts';
import { stubNodeBuiltins } from './path/to/esbuild-stubs.ts';

// Entry point should NOT export anything -- just define and register
// components. esbuild's ESM format is fine as long as the entry has
// no top-level exports (eval does not support import/export syntax).
await esbuild.build({
  entryPoints: ['src/components/index.ts'],
  outfile: 'dist/components.js',
  bundle: true,
  format: 'esm',
  target: 'es2022',
  platform: 'node',
  conditions: ['node'],
  plugins: [litSsrWasmPlugin(), stubNodeBuiltins],
});
```

### SSR environment provided by the runtime

The WASM module sets up these globals before evaluating consumer code:

| Global | Source |
|---|---|
| `customElements`, `HTMLElement`, `Element` | `@lit-labs/ssr-dom-shim` |
| `CSSStyleSheet` (with `.cssText` getter) | `@lit-labs/ssr-dom-shim` + fix |
| `Event`, `CustomEvent`, `EventTarget` | `@lit-labs/ssr-dom-shim` |
| `Document`, `document`, `ShadowRoot` | Minimal shims |
| `btoa`, `atob` | Base64 encode/decode |
| `URL`, `URLSearchParams` | Minimal shims |
| `CSS.supports()`, `CSS.escape()` | No-op / identity |
| `MutationObserver`, `requestAnimationFrame` | No-op |

---

## Wire Protocol

The WASM module uses a two-phase protocol that keeps instances warm across
multiple renders, amortizing the ~350ms cold start.

### Phase 1: Init (once per worker)

```
stdin:  {"source":"...","elements":["my-el","my-other"]}\n
stdout: \0  (ack)
```

The WASM evaluates the component source and registers custom elements.

### Phase 2: Render (per request)

```
stdin:  <raw HTML>\0
stdout: <rendered HTML>\0
```

NUL-delimited on both sides so multi-line HTML is handled correctly.
Errors are written to stderr; on error, stdout gets an empty response (`\0`).
The module exits cleanly when stdin reaches EOF.

The Go library and CLI handle this protocol internally. External callers
use NUL-delimited HTML on both sides (the CLI translates).

---

## Project Structure

```
lit-ssr-wasm/
+-- src/
|   +-- harness/
|   |   +-- render.ts             # Core SSR pipeline
|   +-- io/
|   |   +-- javy.ts               # Javy.IO for WASM
|   |   +-- node.ts               # node:fs for Node.js
|   +-- components/               # Demo components (x-card, my-alert, etc.)
|   +-- runtime-entry.ts          # WASM entry: shims + read loop
|   +-- demo-entry.ts             # Demo site entry (components baked in)
|   +-- ssr-css-fix.ts            # CSSStyleSheet.prototype.cssText patch
|   +-- esbuild-plugin.ts         # litSsrWasmPlugin for consumers
|   +-- esbuild-stubs.ts          # stubNodeBuiltins for consumers
|   +-- ssr-rhds.ts               # SSR entry for demo site chrome
+-- scripts/
|   +-- build.ts                  # esbuild: JS bundles for Node + Javy
|   +-- build-demo.ts             # Assembles demo site into _site/
|   +-- dev-server.ts             # Live-reloading dev server
+-- go/
|   +-- litssr.go                 # Go package: worker pool using wazero
|   +-- litssr_test.go            # Tests + benchmarks
|   +-- cmd/lit-ssr/              # CLI binary
|   +-- testdata/                 # Test components + golden fixtures
|   +-- go.mod
+-- docs/                         # Demo site source
+-- _site/                        # Build output (gitignored)
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
[nodejs]: https://nodejs.org
[golang]: https://go.dev
[demo]: https://bennypowers.github.io/lit-ssr-wasm/compiled.html
[demo-runtime]: https://bennypowers.github.io/lit-ssr-wasm/runtime.html
