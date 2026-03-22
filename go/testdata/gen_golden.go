//go:build ignore

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	litssr "github.com/bennypowers/lit-ssr-wasm/go"
)

//go:embed test-components.js
var testSource string

var fixtures = map[string]string{
	"card":           `<test-card><h2 slot="header">Hi</h2><p>Body</p></test-card>`,
	"badge":          `<test-badge state="success">up</test-badge>`,
	"sheet":          `<test-sheet>styled</test-sheet>`,
	"passthrough":    `<unknown-el>hello</unknown-el>`,
}

func main() {
	ctx := context.Background()
	r, err := litssr.New(ctx, testSource, 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init: %v\n", err)
		os.Exit(1)
	}
	defer r.Close(ctx)

	dir := filepath.Dir(os.Args[0])
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	for name, input := range fixtures {
		out, err := r.RenderHTML(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
			os.Exit(1)
		}
		path := filepath.Join(dir, name+".golden")
		if err := os.WriteFile(path, []byte(out), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", path)
	}
}
