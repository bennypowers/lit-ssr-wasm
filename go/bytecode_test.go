package litssr

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestCompileSource(t *testing.T) {
	ctx := context.Background()
	bytecode, err := CompileSource(ctx, testSource)
	if err != nil {
		t.Fatalf("CompileSource: %v", err)
	}
	if len(bytecode) == 0 {
		t.Fatal("CompileSource returned empty bytecode")
	}
}

func TestCompileFilesTLA(t *testing.T) {
	// The Javy plugin's invoke drains the module's top-level Promise
	// after the synchronous module body has already run, so await in
	// the bytecode entry doesn't actually pause execution. TLA
	// components must use the runtime path (New/NewFromFiles) for now.
	t.Skip("bytecode path does not support TLA (plugin evaluates TLA Promise after module body)")
}

func TestCompileFiles(t *testing.T) {
	ctx := context.Background()
	bytecode, err := CompileFiles(ctx, []string{"testdata/unbundled-component.ts"})
	if err != nil {
		t.Fatalf("CompileFiles: %v", err)
	}
	if len(bytecode) == 0 {
		t.Fatal("CompileFiles returned empty bytecode")
	}

	r, err := NewFromBytecode(ctx, bytecode, 1)
	if err != nil {
		t.Fatalf("NewFromBytecode: %v", err)
	}
	defer r.Close(ctx)

	out, err := r.RenderHTML(ctx, `<unbundled-el></unbundled-el>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(out, `shadowrootmode="open"`) {
		t.Error("missing DSD in output")
	}
}

func bytecodeRenderer(t *testing.T) *Renderer {
	t.Helper()
	ctx := context.Background()
	bytecode, err := CompileSource(ctx, testSource)
	if err != nil {
		t.Fatalf("CompileSource: %v", err)
	}
	r, err := NewFromBytecode(ctx, bytecode, 2)
	if err != nil {
		t.Fatalf("NewFromBytecode: %v", err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })
	return r
}

func TestBytecodeCard(t *testing.T) {
	r := bytecodeRenderer(t)
	want := golden(t, "card")
	got, err := r.RenderHTML(context.Background(), `<test-card><h2 slot="header">Hi</h2><p>Body</p></test-card>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if got != want {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBytecodeBadge(t *testing.T) {
	r := bytecodeRenderer(t)
	want := golden(t, "badge")
	got, err := r.RenderHTML(context.Background(), `<test-badge state="success">up</test-badge>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if got != want {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBytecodeCSSStyleSheet(t *testing.T) {
	r := bytecodeRenderer(t)
	want := golden(t, "sheet")
	got, err := r.RenderHTML(context.Background(), `<test-sheet>styled</test-sheet>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if got != want {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBytecodePassthrough(t *testing.T) {
	r := bytecodeRenderer(t)
	want := golden(t, "passthrough")
	got, err := r.RenderHTML(context.Background(), `<unknown-el>hello</unknown-el>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if got != want {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBytecodeBatch(t *testing.T) {
	r := bytecodeRenderer(t)
	inputs := []string{
		`<test-card><h2 slot="header">Hi</h2><p>Body</p></test-card>`,
		`<test-badge state="success">up</test-badge>`,
		`<test-sheet>styled</test-sheet>`,
		`<unknown-el>hello</unknown-el>`,
	}
	goldens := []string{"card", "badge", "sheet", "passthrough"}

	results, err := r.RenderBatch(context.Background(), inputs)
	if err != nil {
		t.Fatalf("RenderBatch: %v", err)
	}

	if len(results) != len(inputs) {
		t.Fatalf("got %d results, want %d", len(results), len(inputs))
	}

	for i, got := range results {
		want := golden(t, goldens[i])
		if got != want {
			t.Errorf("result %d (%s): mismatch\ngot:\n%s\nwant:\n%s", i, goldens[i], got, want)
		}
	}
}

func TestBytecodeConcurrent(t *testing.T) {
	r := bytecodeRenderer(t)
	ctx := context.Background()
	want := golden(t, "card")

	var wg sync.WaitGroup
	errs := make(chan error, 20)

	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got, err := r.RenderHTML(ctx, `<test-card><h2 slot="header">Hi</h2><p>Body</p></test-card>`)
			if err != nil {
				errs <- fmt.Errorf("render %d: %w", i, err)
				return
			}
			if got != want {
				errs <- fmt.Errorf("render %d: mismatch", i)
			}
		}(i)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func BenchmarkCompileSource(b *testing.B) {
	ctx := context.Background()
	for range b.N {
		_, err := CompileSource(ctx, testSource)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewFromBytecode(b *testing.B) {
	ctx := context.Background()
	bytecode, err := CompileSource(ctx, testSource)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		r, err := NewFromBytecode(ctx, bytecode, 0)
		if err != nil {
			b.Fatal(err)
		}
		if err := r.Close(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderHTMLBytecode(b *testing.B) {
	ctx := context.Background()
	bytecode, err := CompileSource(ctx, testSource)
	if err != nil {
		b.Fatal(err)
	}
	r, err := NewFromBytecode(ctx, bytecode, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close(ctx)

	input := `<test-card><h2 slot="header">Bench</h2><p>Body</p></test-card>`

	b.ResetTimer()
	for range b.N {
		_, err := r.RenderHTML(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
