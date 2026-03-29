// lit-ssr is a CLI for server-rendering Lit web components via WASM.
//
// Component definitions are loaded from JS/TS files at startup and
// automatically bundled with esbuild. In the default (stdin) mode it reads
// NUL-terminated HTML from stdin, renders each with Declarative Shadow DOM,
// and writes NUL-terminated results to stdout.
//
// The render subcommand processes HTML files in-place, useful for SSG
// post-processing workflows.
//
// Usage:
//
//	lit-ssr src/my-card.ts src/my-alert.ts          # stdin/stdout NUL protocol
//	lit-ssr --dir ./components/                     # stdin/stdout NUL protocol
//	lit-ssr --skip-bundle dist/components.js        # stdin/stdout NUL protocol
//	lit-ssr render --dir ./components/ out/**/*.html # render files in-place
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	litssr "bennypowers.dev/lit-ssr-wasm/go"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "render" {
		os.Exit(renderMain(os.Args[2:]))
	}
	os.Exit(stdinMain())
}

func stdinMain() int {
	var skipBundle string
	var dir string
	flag.StringVar(&skipBundle, "skip-bundle", "", "path to a pre-bundled JS file (skips esbuild)")
	flag.StringVar(&dir, "dir", "", "directory of component source files (*.ts, *.js)")
	flag.Parse()

	ctx := context.Background()

	renderer, err := createRenderer(ctx, skipBundle, dir, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr: %v\n", err)
		return 1
	}
	defer func() { _ = renderer.Close(ctx) }()

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
		_, _ = os.Stdout.WriteString(output)
		_, _ = os.Stdout.Write([]byte{0})
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr: read error: %v\n", err)
		return 1
	}
	return 0
}

func renderMain(args []string) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var skipBundle string
	var dir string
	fs.StringVar(&skipBundle, "skip-bundle", "", "path to a pre-bundled JS file (skips esbuild)")
	fs.StringVar(&dir, "dir", "", "directory of component source files (*.ts, *.js)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	htmlFiles := fs.Args()
	if len(htmlFiles) == 0 {
		fmt.Fprintf(os.Stderr, "lit-ssr render: no HTML files specified\n")
		return 1
	}

	if skipBundle == "" && dir == "" {
		fmt.Fprintf(os.Stderr, "lit-ssr render: --dir or --skip-bundle is required\n")
		return 1
	}

	ctx := context.Background()

	renderer, err := createRenderer(ctx, skipBundle, dir, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr render: %v\n", err)
		return 1
	}
	defer func() { _ = renderer.Close(ctx) }()

	// Read all files
	inputs := make([]string, len(htmlFiles))
	perms := make([]os.FileMode, len(htmlFiles))
	for i, path := range htmlFiles {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "lit-ssr render: %v\n", err)
			return 1
		}
		perms[i] = info.Mode().Perm()
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "lit-ssr render: %v\n", err)
			return 1
		}
		inputs[i] = string(data)
	}

	results, err := renderer.RenderBatch(ctx, inputs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lit-ssr render: %v\n", err)
		return 1
	}

	failed := false
	for i, result := range results {
		if err := writeFileAtomic(htmlFiles[i], []byte(result), perms[i]); err != nil {
			fmt.Fprintf(os.Stderr, "lit-ssr render: write %s: %v\n", htmlFiles[i], err)
			failed = true
		}
	}
	if failed {
		return 1
	}
	return 0
}

// writeFileAtomic writes data to a temp file in the same directory, syncs it,
// then renames it over dst to avoid truncating the original on failure.
func writeFileAtomic(dst string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".lit-ssr-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
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
		return litssr.New(ctx, string(data), 0)
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

	return litssr.NewFromFiles(ctx, deduped, 0)
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
