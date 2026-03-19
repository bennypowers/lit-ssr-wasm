// lit-ssr-runtime is a CLI for server-rendering Lit web components via
// the runtime WASM module. Component definitions are loaded from JS files
// at startup, not baked into the WASM.
//
// Usage:
//
//	lit-ssr-runtime --components ./js/my-card.js --components ./js/my-alert.js
//
// Then pipe NUL-terminated HTML on stdin, get NUL-terminated rendered HTML
// on stdout -- same protocol as lit-ssr (builtin mode).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	litssr "bennypowers.dev/lit-ssr-go"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// stringSlice implements flag.Value for repeated --components flags.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// runtimeRequest is the JSON payload sent to the runtime WASM module.
type runtimeRequest struct {
	Source   string   `json:"source"`
	HTML     string   `json:"html"`
	Elements []string `json:"elements"`
}

var defineRe = regexp.MustCompile(`customElements\.define\(\s*['"]([^'"]+)['"]`)

func main() {
	var components stringSlice
	var componentDir string
	flag.Var(&components, "components", "path to a component JS file (repeatable)")
	flag.StringVar(&componentDir, "dir", "", "directory of component JS files")
	flag.Parse()

	// Collect JS files from --dir and --components
	var jsFiles []string
	if componentDir != "" {
		matches, err := filepath.Glob(filepath.Join(componentDir, "*.js"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "lit-ssr-runtime: glob error: %v\n", err)
			os.Exit(1)
		}
		jsFiles = append(jsFiles, matches...)
	}
	jsFiles = append(jsFiles, components...)

	if len(jsFiles) == 0 {
		fmt.Fprintln(os.Stderr, "lit-ssr-runtime: no component files specified. Use --dir or --components.")
		os.Exit(1)
	}

	// Read and concatenate all JS sources, extract element names
	var sourceBuilder strings.Builder
	var elements []string
	for _, f := range jsFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "lit-ssr-runtime: read %s: %v\n", f, err)
			os.Exit(1)
		}
		sourceBuilder.Write(data)
		sourceBuilder.WriteByte('\n')

		for _, m := range defineRe.FindAllSubmatch(data, -1) {
			elements = append(elements, string(m[1]))
		}
	}
	source := sourceBuilder.String()

	if len(elements) == 0 {
		fmt.Fprintln(os.Stderr, "lit-ssr-runtime: no customElements.define() calls found in component files")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "lit-ssr-runtime: loaded %d components (%s) from %d files\n",
		len(elements), strings.Join(elements, ", "), len(jsFiles))

	// Start the WASM runtime
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr-runtime: WASI init: %v\n", err)
		os.Exit(1)
	}

	compiled, err := rt.CompileModule(ctx, litssr.RuntimeWasm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr-runtime: compile WASM: %v\n", err)
		os.Exit(1)
	}

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go func() {
		_, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
			WithName("").
			WithStdin(stdinR).
			WithStdout(stdoutW).
			WithStderr(os.Stderr))
		_ = err
		stdoutW.Close()
	}()

	wasmReader := bufio.NewReader(stdoutR)

	// Read NUL-terminated HTML from our stdin, construct JSON payloads
	// for the WASM, read NUL-terminated responses, write to our stdout.
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(splitNul)

	for scanner.Scan() {
		input := scanner.Text()
		if input == "" {
			continue
		}

		req := runtimeRequest{
			Source:   source,
			HTML:     input,
			Elements: elements,
		}
		payload, _ := json.Marshal(req)
		payload = append(payload, '\n')

		if _, err := stdinW.Write(payload); err != nil {
			fmt.Fprintf(os.Stderr, "lit-ssr-runtime: write to WASM: %v\n", err)
			os.Stdout.Write([]byte{0})
			continue
		}

		result, err := wasmReader.ReadString('\x00')
		if err != nil {
			fmt.Fprintf(os.Stderr, "lit-ssr-runtime: read from WASM: %v\n", err)
			os.Stdout.Write([]byte{0})
			continue
		}

		// Pass through the NUL-terminated response
		os.Stdout.WriteString(result)
	}

	stdinW.Close()

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr-runtime: read error: %v\n", err)
		os.Exit(1)
	}
}

func splitNul(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i, b := range data {
		if b == 0 {
			return i + 1, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}
