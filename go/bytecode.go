package litssr

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed javy-plugin.wasm
var javyPluginWasm []byte

//go:embed lit-ssr-bytecode-entry.js
var bytecodeEntryTemplate string

const userSourcePlaceholder = `globalThis.__LITSSR_USER_SOURCE__ = true;`

// CompileSource pre-compiles bundled JavaScript component source to QuickJS
// bytecode. The source must be ready to eval (no import/export, no TypeScript).
// The returned bytecode can be cached and reused with NewFromBytecode.
//
// Bytecode is tied to the QuickJS engine version embedded in the Javy plugin.
// Cache invalidation is the caller's responsibility when upgrading Javy.
func CompileSource(ctx context.Context, componentSource string) ([]byte, error) {
	// Wrap user source in an IIFE to isolate its scope from the template.
	// Both the template and user source bundle Lit independently; without
	// isolation their mangled variable names collide.
	wrapped := "(function(){" + componentSource + "})();\n"
	combined := strings.Replace(bytecodeEntryTemplate, userSourcePlaceholder, wrapped, 1)
	if combined == bytecodeEntryTemplate {
		return nil, fmt.Errorf("litssr: compile: user source placeholder not found in template")
	}

	rt := wazero.NewRuntime(ctx)
	defer func() { _ = rt.Close(ctx) }()

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		return nil, fmt.Errorf("litssr: compile: instantiate WASI: %w", err)
	}

	compiled, err := rt.CompileModule(ctx, javyPluginWasm)
	if err != nil {
		return nil, fmt.Errorf("litssr: compile: compile plugin: %w", err)
	}

	cfg := wazero.NewModuleConfig().
		WithName("compiler").
		WithStartFunctions() // don't call _start/_initialize

	mod, err := rt.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		return nil, fmt.Errorf("litssr: compile: instantiate plugin: %w", err)
	}
	defer func() { _ = mod.Close(ctx) }()

	return compileBytecode(ctx, mod, []byte(combined))
}

// CompileFiles bundles JS/TS source files with esbuild and compiles to
// QuickJS bytecode. Handles import/export, TypeScript, CSS modules, and
// lit-ssr-wasm shim bridging automatically.
func CompileFiles(ctx context.Context, files []string) ([]byte, error) {
	source, err := bundleFiles(files)
	if err != nil {
		return nil, err
	}
	return CompileSource(ctx, source)
}

// compileBytecode calls the Javy plugin's compile-src export.
func compileBytecode(ctx context.Context, mod api.Module, source []byte) ([]byte, error) {
	memory := mod.Memory()
	realloc := mod.ExportedFunction("cabi_realloc")
	compileSrc := mod.ExportedFunction("compile-src")

	if memory == nil || realloc == nil || compileSrc == nil {
		return nil, fmt.Errorf("litssr: compile: plugin missing required exports")
	}

	// Allocate memory for source
	srcLen := uint64(len(source))
	results, err := realloc.Call(ctx, 0, 0, 1, srcLen)
	if err != nil {
		return nil, fmt.Errorf("litssr: compile: alloc source: %w", err)
	}
	srcPtr := uint32(results[0])

	if !memory.Write(srcPtr, source) {
		return nil, fmt.Errorf("litssr: compile: write source: out of bounds")
	}

	// Call compile-src
	results, err = compileSrc.Call(ctx, uint64(srcPtr), srcLen)
	if err != nil {
		return nil, fmt.Errorf("litssr: compile: compile-src: %w", err)
	}
	retPtr := uint32(results[0])

	// Read result: [status:u32, data_ptr:u32, data_len:u32]
	retBuf, ok := memory.Read(retPtr, 12)
	if !ok {
		return nil, fmt.Errorf("litssr: compile: read result: out of bounds")
	}

	status := binary.LittleEndian.Uint32(retBuf[0:4])
	dataPtr := binary.LittleEndian.Uint32(retBuf[4:8])
	dataLen := binary.LittleEndian.Uint32(retBuf[8:12])

	data, ok := memory.Read(dataPtr, dataLen)
	if !ok {
		return nil, fmt.Errorf("litssr: compile: read data: out of bounds")
	}

	if status != 0 {
		return nil, fmt.Errorf("litssr: compile: %s", string(data))
	}

	// Copy bytecode out of WASM memory (it will be freed when module closes)
	bytecode := make([]byte, len(data))
	copy(bytecode, data)
	return bytecode, nil
}

// NewFromBytecode creates a renderer pool from pre-compiled QuickJS bytecode.
// Use CompileSource or CompileFiles to produce the bytecode.
// If workers is 0, defaults to runtime.NumCPU().
func NewFromBytecode(ctx context.Context, bytecode []byte, workers int) (*Renderer, error) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	rt := wazero.NewRuntime(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("litssr: instantiate WASI: %w", err)
	}

	compiled, err := rt.CompileModule(ctx, javyPluginWasm)
	if err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("litssr: compile plugin: %w", err)
	}

	r := &Renderer{
		runtime:  rt,
		compiled: compiled,
		work:     make(chan request),
	}

	for i := range workers {
		w, err := r.startBytecodeWorker(ctx, bytecode, i)
		if err != nil {
			_ = r.Close(ctx)
			return nil, fmt.Errorf("litssr: start bytecode worker %d: %w", i, err)
		}
		r.workers = append(r.workers, w)
		r.wg.Add(1)
		go r.runWorker(w)
	}

	return r, nil
}

func (r *Renderer) startBytecodeWorker(ctx context.Context, bytecode []byte, _ int) (*worker, error) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderr := &stderrCollector{}

	cfg := wazero.NewModuleConfig().
		WithName("").
		WithStartFunctions(). // don't call _start/_initialize
		WithStdin(stdinR).
		WithStdout(stdoutW).
		WithStderr(stderr)

	r.wasmWg.Add(1)
	go func() {
		defer r.wasmWg.Done()
		mod, err := r.runtime.InstantiateModule(ctx, r.compiled, cfg)
		if err != nil {
			_ = stdinR.CloseWithError(err)
			_ = stdoutW.Close()
			return
		}
		defer func() { _ = mod.Close(ctx) }()

		// Allocate memory for bytecode and call invoke
		memory := mod.Memory()
		realloc := mod.ExportedFunction("cabi_realloc")
		invoke := mod.ExportedFunction("invoke")

		if memory == nil || realloc == nil || invoke == nil {
			_ = stdinR.CloseWithError(fmt.Errorf("plugin missing required exports"))
			_ = stdoutW.Close()
			return
		}

		bcLen := uint64(len(bytecode))
		results, err := realloc.Call(ctx, 0, 0, 1, bcLen)
		if err != nil {
			_ = stdinR.CloseWithError(err)
			_ = stdoutW.Close()
			return
		}
		bcPtr := uint32(results[0])

		if !memory.Write(bcPtr, bytecode) {
			_ = stdinR.CloseWithError(fmt.Errorf("bytecode write out of bounds"))
			_ = stdoutW.Close()
			return
		}

		// invoke(bytecode_ptr, bytecode_len, fn_name_discriminator=0, fn_name_ptr=0, fn_name_len=0)
		_, err = invoke.Call(ctx, uint64(bcPtr), bcLen, 0, 0, 0)
		if err != nil {
			_ = stdinR.CloseWithError(err)
		}
		_ = stdoutW.Close()
	}()

	w := &worker{
		stdin:  stdinW,
		stdout: bufio.NewReader(stdoutR),
		stderr: stderr,
	}

	if err := w.initBytecode(); err != nil {
		_ = stdinW.Close()
		return nil, err
	}

	return w, nil
}

// initBytecode sends an empty init message to a bytecode worker.
// The WASM loads the pre-compiled bytecode (which includes the component
// source) and acks with \0.
func (w *worker) initBytecode() error {
	if _, err := w.stdin.Write([]byte("{}\n")); err != nil {
		return fmt.Errorf("litssr: write init: %w", err)
	}

	// Wait for ack (\0)
	_, err := w.stdout.ReadString('\x00')
	if err != nil {
		return fmt.Errorf("litssr: read init ack: %w", err)
	}

	if errMsg := strings.TrimSpace(w.stderr.drain()); errMsg != "" {
		return fmt.Errorf("litssr: init: %s", errMsg)
	}

	return nil
}
