// Package litssr provides Go bindings to a Lit SSR WASM module.
//
// It runs LitElement server-side rendering via a Javy-compiled QuickJS
// WASM module, using wazero as a pure-Go WASM runtime. No CGo, no
// Node.js sidecar -- just a single embedded .wasm blob.
//
// Usage:
//
//	renderer, err := litssr.New(ctx)
//	if err != nil { ... }
//	defer renderer.Close(ctx)
//
//	html, err := renderer.RenderHTML(ctx, `<x-card>hello</x-card>`)
package litssr

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed lit-ssr.wasm
var litSSRWasm []byte

// Renderer holds a pre-compiled WASM module for rendering Lit
// elements to HTML with Declarative Shadow DOM.
type Renderer struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	mu       sync.Mutex
}

// New creates a new Lit SSR renderer. Call Close when done.
func New(ctx context.Context) (*Renderer, error) {
	rt := wazero.NewRuntime(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("litssr: instantiate WASI: %w", err)
	}

	compiled, err := rt.CompileModule(ctx, litSSRWasm)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("litssr: compile WASM: %w", err)
	}

	return &Renderer{runtime: rt, compiled: compiled}, nil
}

// RenderHTML takes an HTML string containing custom elements and
// returns the same HTML with Declarative Shadow DOM injected.
func (r *Renderer) RenderHTML(ctx context.Context, inputHTML string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stdin := bytes.NewBufferString(inputHTML)
	var stdout, stderr bytes.Buffer

	cfg := wazero.NewModuleConfig().
		WithName("").
		WithStdin(stdin).
		WithStdout(&stdout).
		WithStderr(&stderr)

	_, err := r.runtime.InstantiateModule(ctx, r.compiled, cfg)
	if err != nil {
		// Javy modules call proc_exit(0) after running; wazero
		// surfaces this as sys.ExitError. If we got stdout output,
		// the render succeeded.
		if stdout.Len() > 0 {
			if stderr.Len() > 0 {
				return stdout.String(), fmt.Errorf("litssr warnings: %s", stderr.String())
			}
			return stdout.String(), nil
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("litssr: %s", stderr.String())
		}
		return "", fmt.Errorf("litssr: WASM execution failed: %w", err)
	}

	if stderr.Len() > 0 {
		return stdout.String(), fmt.Errorf("litssr warnings: %s", stderr.String())
	}
	return stdout.String(), nil
}

// Close releases all resources held by the renderer.
func (r *Renderer) Close(ctx context.Context) error {
	return r.runtime.Close(ctx)
}
