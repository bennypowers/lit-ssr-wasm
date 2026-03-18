package litssr

import (
	"context"
	"strings"
	"testing"
)

func newRenderer(t *testing.T) *Renderer {
	t.Helper()
	r, err := New(context.Background())
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
	// tabs + 2 tab + 2 tab-panel = 5 DSD
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
	// card, tabs, tab, tab-panel, badge = 5 minimum
	if n := strings.Count(out, `shadowrootmode="open"`); n < 5 {
		t.Errorf("expected >= 5 DSD templates, got %d", n)
	}
}
