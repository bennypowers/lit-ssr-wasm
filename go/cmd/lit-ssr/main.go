// lit-ssr is a CLI for server-rendering Lit web components via WASM.
//
// Component definitions are loaded from JavaScript files at startup.
// It reads NUL-terminated HTML from stdin, renders each with Declarative
// Shadow DOM, and writes NUL-terminated results to stdout. The WASM
// instance stays warm across renders.
//
// Usage:
//
//	lit-ssr --bundle components.js
//	lit-ssr --dir ./components/
//	lit-ssr --components ./js/my-card.js --components ./js/my-alert.js
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	litssr "bennypowers.dev/lit-ssr-go"
)

// stringSlice implements flag.Value for repeated --components flags.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	var bundle string
	var components stringSlice
	var componentDir string
	flag.StringVar(&bundle, "bundle", "", "path to a pre-built JS bundle")
	flag.Var(&components, "components", "path to a component JS file (repeatable)")
	flag.StringVar(&componentDir, "dir", "", "directory of component JS files")
	flag.Parse()

	source, err := loadSource(bundle, componentDir, components)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	renderer, err := litssr.New(ctx, source, 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr: init failed: %v\n", err)
		os.Exit(1)
	}
	defer renderer.Close(ctx)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(splitNul)

	for scanner.Scan() {
		input := scanner.Text()
		if input == "" {
			continue
		}

		output, err := renderer.RenderHTML(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "lit-ssr: %v\n", err)
		}
		os.Stdout.WriteString(output)
		os.Stdout.Write([]byte{0})
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr: read error: %v\n", err)
		os.Exit(1)
	}
}

func loadSource(bundle, dir string, components []string) (string, error) {
	if bundle != "" {
		if dir != "" || len(components) > 0 {
			return "", fmt.Errorf("--bundle is mutually exclusive with --dir and --components")
		}
		data, err := os.ReadFile(bundle)
		if err != nil {
			return "", fmt.Errorf("read bundle %s: %w", bundle, err)
		}
		return string(data), nil
	}

	var jsFiles []string
	if dir != "" {
		if _, err := os.Stat(dir); err != nil {
			return "", fmt.Errorf("dir %s: %w", dir, err)
		}
		matches, err := filepath.Glob(filepath.Join(dir, "*.js"))
		if err != nil {
			return "", fmt.Errorf("glob error: %w", err)
		}
		if len(matches) == 0 {
			return "", fmt.Errorf("no .js files found in %s", dir)
		}
		jsFiles = append(jsFiles, matches...)
	}
	jsFiles = append(jsFiles, components...)

	if len(jsFiles) == 0 {
		return "", fmt.Errorf("no component files specified. Use --bundle, --dir, or --components")
	}

	var sb strings.Builder
	for _, f := range jsFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", f, err)
		}
		sb.Write(data)
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

// splitNul is a bufio.SplitFunc that splits on NUL bytes.
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
