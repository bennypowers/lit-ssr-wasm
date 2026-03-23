// lit-ssr is a CLI for server-rendering Lit web components via WASM.
//
// Component definitions are loaded from JS/TS files at startup and
// automatically bundled with esbuild. It reads NUL-terminated HTML from
// stdin, renders each with Declarative Shadow DOM, and writes
// NUL-terminated results to stdout.
//
// Usage:
//
//	lit-ssr src/my-card.ts src/my-alert.ts
//	lit-ssr --dir ./components/
//	lit-ssr --skip-bundle dist/components.js
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	litssr "github.com/bennypowers/lit-ssr-wasm/go"
)

func main() {
	var skipBundle string
	var dir string
	flag.StringVar(&skipBundle, "skip-bundle", "", "path to a pre-bundled JS file (skips esbuild)")
	flag.StringVar(&dir, "dir", "", "directory of component source files (*.ts, *.js)")
	flag.Parse()

	ctx := context.Background()

	renderer, err := createRenderer(ctx, skipBundle, dir, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr: %v\n", err)
		os.Exit(1)
	}
	defer renderer.Close(ctx)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
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

func createRenderer(ctx context.Context, skipBundle, dir string, args []string) (*litssr.Renderer, error) {
	// --skip-bundle: pre-bundled JS, no esbuild
	if skipBundle != "" {
		if dir != "" || len(args) > 0 {
			return nil, fmt.Errorf("--skip-bundle is mutually exclusive with --dir and positional args")
		}
		data, err := os.ReadFile(skipBundle)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", skipBundle, err)
		}
		return litssr.New(ctx, string(data), 1)
	}

	// Collect files from --dir and/or positional args
	var files []string
	if dir != "" {
		info, err := os.Stat(dir)
		if err != nil {
			return nil, fmt.Errorf("dir %s: %w", dir, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", dir)
		}
		for _, pattern := range []string{"*.ts", "*.js"} {
			matches, err := filepath.Glob(filepath.Join(dir, pattern))
			if err != nil {
				return nil, fmt.Errorf("glob %s: %w", pattern, err)
			}
			for _, m := range matches {
				// Skip declaration files and test files
				if strings.HasSuffix(m, ".d.ts") || strings.HasSuffix(m, ".test.ts") || strings.HasSuffix(m, ".test.js") {
					continue
				}
				files = append(files, m)
			}
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("no .ts or .js files found in %s", dir)
		}
	}
	files = append(files, args...)

	if len(files) == 0 {
		return nil, fmt.Errorf("no component files specified. Usage: lit-ssr [--skip-bundle file.js | --dir ./components/ | file1.ts file2.ts ...]")
	}

	// Deduplicate (--dir and positional args may overlap)
	seen := make(map[string]struct{}, len(files))
	deduped := files[:0]
	for _, f := range files {
		abs, err := filepath.Abs(f)
		if err != nil {
			abs = f
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		deduped = append(deduped, f)
	}

	return litssr.NewFromFiles(ctx, deduped, 1)
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
