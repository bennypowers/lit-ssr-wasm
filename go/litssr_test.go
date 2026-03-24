package litssr

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

//go:embed testdata/test-components.js
var testSource string

func newRenderer(t *testing.T) *Renderer {
	t.Helper()
	r, err := New(context.Background(), testSource, 2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })
	return r
}

func golden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name+".golden"))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(data)
}

func TestNoDefinesPassthrough(t *testing.T) {
	r, err := New(context.Background(), "// no components here", 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	input := `<unknown-el>hello</unknown-el>`
	got, err := r.RenderHTML(context.Background(), input)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(got, input) {
		t.Errorf("expected passthrough content\ngot: %s", got)
	}
}

func TestCard(t *testing.T) {
	r := newRenderer(t)
	want := golden(t, "card")
	got, err := r.RenderHTML(context.Background(), `<test-card><h2 slot="header">Hi</h2><p>Body</p></test-card>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if got != want {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBadge(t *testing.T) {
	r := newRenderer(t)
	want := golden(t, "badge")
	got, err := r.RenderHTML(context.Background(), `<test-badge state="success">up</test-badge>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if got != want {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestCSSStyleSheet(t *testing.T) {
	r := newRenderer(t)
	want := golden(t, "sheet")
	got, err := r.RenderHTML(context.Background(), `<test-sheet>styled</test-sheet>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if got != want {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestPassthrough(t *testing.T) {
	r := newRenderer(t)
	want := golden(t, "passthrough")
	got, err := r.RenderHTML(context.Background(), `<unknown-el>hello</unknown-el>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if got != want {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestMultipleRenders(t *testing.T) {
	r := newRenderer(t)
	ctx := context.Background()
	want := golden(t, "card")

	for i := range 5 {
		got, err := r.RenderHTML(ctx, `<test-card><h2 slot="header">Hi</h2><p>Body</p></test-card>`)
		if err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
		if got != want {
			t.Errorf("render %d: mismatch\ngot:\n%s\nwant:\n%s", i, got, want)
		}
	}
}

func TestBatch(t *testing.T) {
	r := newRenderer(t)
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

func TestConcurrent(t *testing.T) {
	r := newRenderer(t)
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

func TestCloseRace(t *testing.T) {
	for range 10 {
		r, err := New(context.Background(), testSource, 2)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}

func BenchmarkNew(b *testing.B) {
	ctx := context.Background()
	for range b.N {
		r, err := New(ctx, testSource, 0)
		if err != nil {
			b.Fatal(err)
		}
		if err := r.Close(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderHTML(b *testing.B) {
	r, err := New(context.Background(), testSource, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close(context.Background())

	ctx := context.Background()
	input := `<test-card><h2 slot="header">Bench</h2><p>Body</p></test-card>`

	b.ResetTimer()
	for range b.N {
		_, err := r.RenderHTML(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderBatch(b *testing.B) {
	r, err := New(context.Background(), testSource, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close(context.Background())

	ctx := context.Background()
	inputs := make([]string, 50)
	for i := range inputs {
		inputs[i] = `<test-card><h2 slot="header">Bench</h2><p>Body</p></test-card>`
	}

	b.ResetTimer()
	for range b.N {
		_, err := r.RenderBatch(ctx, inputs)
		if err != nil {
			b.Fatal(err)
		}
	}
}
