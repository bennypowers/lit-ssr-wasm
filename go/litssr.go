// Package litssr provides Go bindings to a Lit SSR WASM module.
//
// It runs LitElement server-side rendering via a Javy-compiled QuickJS
// WASM module, using wazero as a pure-Go WASM runtime. No CGo, no
// Node.js sidecar -- just a single embedded .wasm blob.
//
// The renderer accepts bundled JavaScript component definitions at
// construction time. Internally it uses the runtime WASM module, which
// evaluates the JS source in QuickJS, registers custom elements, and
// renders HTML with Declarative Shadow DOM.
//
// Usage:
//
//	source, _ := os.ReadFile("components.js")
//	renderer, err := litssr.New(ctx, string(source), 0)
//	if err != nil { ... }
//	defer renderer.Close(ctx)
//
//	html, err := renderer.RenderHTML(ctx, `<my-card>hello</my-card>`)
package litssr

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed lit-ssr-runtime.wasm
var runtimeWasm []byte

// Matches both direct define calls and Lit's @customElement decorator:
//   customElements.define('my-el', ...)
//   customElement("my-el")
//   [customElement("my-el")]
var defineRe = regexp.MustCompile(`(?:customElements\.define|customElement)\(\s*['"]([^'"]+)['"]`)

// runtimeRequest is the JSON payload sent to the runtime WASM module.
type runtimeRequest struct {
	Source   string   `json:"source"`
	HTML     string   `json:"html"`
	Elements []string `json:"elements"`
}

// request is sent to a worker via its channel.
type request struct {
	html string
	resp chan<- response
}

// response is returned from a worker.
type response struct {
	html string
	err  error
}

// worker is a long-running WASM instance that processes requests.
type worker struct {
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr *stderrCollector
}

// stderrCollector accumulates stderr output from the WASM module.
type stderrCollector struct {
	mu  sync.Mutex
	buf []byte
}

func (s *stderrCollector) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf = append(s.buf, p...)
	return len(p), nil
}

func (s *stderrCollector) drain() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := string(s.buf)
	s.buf = s.buf[:0]
	return out
}

// Renderer manages a pool of warm WASM instances for rendering Lit
// elements to HTML with Declarative Shadow DOM.
type Renderer struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	source   string
	elements []string
	workers  []*worker
	work     chan request
	wg       sync.WaitGroup
}

// New creates a renderer pool with the given concurrency.
// componentSource is bundled JavaScript containing component
// definitions (customElements.define calls). Element tag names are
// extracted automatically via regex.
// If workers is 0, defaults to runtime.NumCPU().
func New(ctx context.Context, componentSource string, workers int) (*Renderer, error) {
	elements := extractElements(componentSource)
	if len(elements) == 0 {
		return nil, fmt.Errorf("litssr: no customElements.define() calls found in component source")
	}
	return NewWithElements(ctx, componentSource, elements, workers)
}

// NewWithElements creates a renderer pool with an explicit element list.
// Use this when element tag names can't be extracted from the source
// (e.g., when using decorator-based registration).
// If workers is 0, defaults to runtime.NumCPU().
func NewWithElements(ctx context.Context, componentSource string, elements []string, workers int) (*Renderer, error) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	if len(elements) == 0 {
		return nil, fmt.Errorf("litssr: no elements provided")
	}

	rt := wazero.NewRuntime(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("litssr: instantiate WASI: %w", err)
	}

	compiled, err := rt.CompileModule(ctx, runtimeWasm)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("litssr: compile WASM: %w", err)
	}

	r := &Renderer{
		runtime:  rt,
		compiled: compiled,
		source:   componentSource,
		elements: elements,
		work:     make(chan request),
	}

	for i := range workers {
		w, err := r.startWorker(ctx, i)
		if err != nil {
			r.Close(ctx)
			return nil, fmt.Errorf("litssr: start worker %d: %w", i, err)
		}
		r.workers = append(r.workers, w)
		r.wg.Add(1)
		go r.runWorker(w)
	}

	return r, nil
}

// extractElements finds all customElements.define('tag-name', ...) calls.
func extractElements(source string) []string {
	matches := defineRe.FindAllStringSubmatch(source, -1)
	elements := make([]string, 0, len(matches))
	for _, m := range matches {
		elements = append(elements, m[1])
	}
	return elements
}

func (r *Renderer) startWorker(ctx context.Context, _ int) (*worker, error) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderr := &stderrCollector{}

	cfg := wazero.NewModuleConfig().
		WithName("").
		WithStdin(stdinR).
		WithStdout(stdoutW).
		WithStderr(stderr)

	go func() {
		_, err := r.runtime.InstantiateModule(ctx, r.compiled, cfg)
		_ = err // Module exits when stdin is closed (EOF)
		stdoutW.Close()
	}()

	return &worker{
		stdin:  stdinW,
		stdout: bufio.NewReader(stdoutR),
		stderr: stderr,
	}, nil
}

func (r *Renderer) runWorker(w *worker) {
	defer r.wg.Done()
	for req := range r.work {
		html, err := w.render(r.source, r.elements, req.html)
		req.resp <- response{html: html, err: err}
	}
}

func (w *worker) render(source string, elements []string, inputHTML string) (string, error) {
	req := runtimeRequest{
		Source:   source,
		HTML:     inputHTML,
		Elements: elements,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("litssr: marshal request: %w", err)
	}
	payload = append(payload, '\n')

	if _, err := w.stdin.Write(payload); err != nil {
		return "", fmt.Errorf("litssr: write to worker: %w", err)
	}

	result, err := w.stdout.ReadString('\x00')
	if err != nil {
		return "", fmt.Errorf("litssr: read from worker: %w", err)
	}

	// Strip the trailing NUL
	result = result[:len(result)-1]

	if errMsg := w.stderr.drain(); errMsg != "" {
		if result == "" {
			return "", fmt.Errorf("litssr: %s", errMsg)
		}
		return result, fmt.Errorf("litssr warnings: %s", errMsg)
	}

	return result, nil
}

// RenderHTML sends HTML to a worker and returns the rendered result.
// Safe for concurrent use.
func (r *Renderer) RenderHTML(ctx context.Context, inputHTML string) (string, error) {
	resp := make(chan response, 1)
	select {
	case r.work <- request{html: inputHTML, resp: resp}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	select {
	case res := <-resp:
		return res.html, res.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// RenderBatch renders multiple HTML strings, distributing across workers.
// Returns results in the same order as inputs.
func (r *Renderer) RenderBatch(ctx context.Context, inputs []string) ([]string, error) {
	results := make([]string, len(inputs))
	resps := make([]chan response, len(inputs))

	for i, html := range inputs {
		resp := make(chan response, 1)
		resps[i] = resp
		select {
		case r.work <- request{html: html, resp: resp}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	var firstErr error
	for i, resp := range resps {
		select {
		case res := <-resp:
			results[i] = res.html
			if res.err != nil && firstErr == nil {
				firstErr = res.err
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return results, firstErr
}

// Close shuts down all workers and releases resources.
func (r *Renderer) Close(ctx context.Context) error {
	for _, w := range r.workers {
		w.stdin.Close()
	}
	close(r.work)
	r.wg.Wait()
	return r.runtime.Close(ctx)
}
