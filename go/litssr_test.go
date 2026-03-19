package litssr

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func newRenderer(t *testing.T) *Renderer {
	t.Helper()
	r, err := New(context.Background(), 2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })
	return r
}

func TestCard(t *testing.T) {
	r := newRenderer(t)
	out, err := r.RenderHTML(context.Background(), `<x-card><h2 slot="header">Hi</h2><p>Body</p></x-card>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	t.Log(out)
	for _, want := range []string{`shadowrootmode="open"`, "x-card", "<slot", "Body"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output", want)
		}
	}
}

func TestNestedCardCTA(t *testing.T) {
	r := newRenderer(t)
	in := `<x-card><p>Hi</p><x-cta slot="footer" variant="primary">Go</x-cta></x-card>`
	out, err := r.RenderHTML(context.Background(), in)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	t.Log(out)
	if n := strings.Count(out, `shadowrootmode="open"`); n < 2 {
		t.Errorf("expected >= 2 DSD templates, got %d", n)
	}
}

func TestTabs(t *testing.T) {
	r := newRenderer(t)
	in := `<x-tabs>
  <x-tab slot="tab" active>A</x-tab>
  <x-tab-panel>Panel A</x-tab-panel>
  <x-tab slot="tab">B</x-tab>
  <x-tab-panel>Panel B</x-tab-panel>
</x-tabs>`
	out, err := r.RenderHTML(context.Background(), in)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	t.Log(out)
	if n := strings.Count(out, `shadowrootmode="open"`); n < 5 {
		t.Errorf("expected >= 5 DSD templates, got %d", n)
	}
	if !strings.Contains(out, `role="tablist"`) {
		t.Error("missing tablist role")
	}
}

func TestPassthrough(t *testing.T) {
	r := newRenderer(t)
	in := `<unknown-el>hello</unknown-el>`
	out, err := r.RenderHTML(context.Background(), in)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if out != in {
		t.Errorf("unknown elements should pass through unchanged\ngot:  %s\nwant: %s", out, in)
	}
}

func TestMixed(t *testing.T) {
	r := newRenderer(t)
	in := `<x-card>
  <h2 slot="header">Dashboard</h2>
  <x-tabs>
    <x-tab slot="tab" active>Status</x-tab>
    <x-tab-panel><p>OK <x-badge state="success">up</x-badge></p></x-tab-panel>
  </x-tabs>
</x-card>`
	out, err := r.RenderHTML(context.Background(), in)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	t.Log(out)
	if n := strings.Count(out, `shadowrootmode="open"`); n < 5 {
		t.Errorf("expected >= 5 DSD templates, got %d", n)
	}
}

func TestMultipleRenders(t *testing.T) {
	r := newRenderer(t)
	ctx := context.Background()

	// Same instance handles multiple sequential renders
	for i := range 5 {
		out, err := r.RenderHTML(ctx, `<x-card><p>render</p></x-card>`)
		if err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
		if !strings.Contains(out, `shadowrootmode="open"`) {
			t.Errorf("render %d: missing DSD", i)
		}
	}
}

func TestBatch(t *testing.T) {
	r := newRenderer(t)
	inputs := []string{
		`<x-card><p>One</p></x-card>`,
		`<x-badge state="success">up</x-badge>`,
		`<x-cta variant="primary">Click</x-cta>`,
		`<x-card><p>Two</p></x-card>`,
		`<x-badge state="danger">down</x-badge>`,
		`<unknown-el>pass</unknown-el>`,
		`<x-card><p>Three</p></x-card>`,
		`<x-badge state="info">info</x-badge>`,
		`<x-card><p>Four</p></x-card>`,
		`<x-card><p>Five</p></x-card>`,
	}

	results, err := r.RenderBatch(context.Background(), inputs)
	if err != nil {
		t.Fatalf("RenderBatch: %v", err)
	}

	if len(results) != len(inputs) {
		t.Fatalf("got %d results, want %d", len(results), len(inputs))
	}

	for i, out := range results {
		if i == 5 {
			// Unknown element passes through
			if out != inputs[i] {
				t.Errorf("result %d: expected passthrough, got %s", i, out)
			}
			continue
		}
		if !strings.Contains(out, `shadowrootmode="open"`) {
			t.Errorf("result %d: missing DSD", i)
		}
	}
}

func TestConcurrent(t *testing.T) {
	r := newRenderer(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, 20)

	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out, err := r.RenderHTML(ctx, `<x-card><p>concurrent</p></x-card>`)
			if err != nil {
				errs <- fmt.Errorf("render %d: %w", i, err)
				return
			}
			if !strings.Contains(out, `shadowrootmode="open"`) {
				errs <- fmt.Errorf("render %d: missing DSD", i)
			}
		}(i)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func BenchmarkRenderHTML(b *testing.B) {
	r, err := New(context.Background(), 0)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close(context.Background())

	ctx := context.Background()
	input := `<x-card><h2 slot="header">Bench</h2><p>Body</p></x-card>`

	b.ResetTimer()
	for range b.N {
		_, err := r.RenderHTML(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderBatch(b *testing.B) {
	r, err := New(context.Background(), 0)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close(context.Background())

	ctx := context.Background()
	inputs := make([]string, 50)
	for i := range inputs {
		inputs[i] = `<x-card><h2 slot="header">Bench</h2><p>Body</p></x-card>`
	}

	b.ResetTimer()
	for range b.N {
		_, err := r.RenderBatch(ctx, inputs)
		if err != nil {
			b.Fatal(err)
		}
	}
}
