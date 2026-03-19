// lit-ssr is a CLI for server-rendering Lit web components via WASM.
//
// It reads NUL-terminated HTML from stdin, renders each with Declarative
// Shadow DOM, and writes NUL-terminated results to stdout. The WASM
// instance stays warm across renders.
//
// Usage:
//
//	printf '<x-card><p>hello</p></x-card>\0' | lit-ssr
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	litssr "bennypowers.dev/lit-ssr-go"
)

func main() {
	ctx := context.Background()

	renderer, err := litssr.New(ctx, 1)
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
