package litssr

import (
	"context"
	_ "embed"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/unbundled-component.ts
var unbundledFixture string

func TestNeedsBundleWithImports(t *testing.T) {
	if !needsBundle(unbundledFixture) {
		t.Error("expected source with imports to need bundling")
	}
}

func TestNeedsBundleAlreadyBundled(t *testing.T) {
	// The pre-built test-components.js is already bundled
	if needsBundle(testSource) {
		t.Error("expected bundled source to not need bundling")
	}
}

func TestBundleSource(t *testing.T) {
	resolveDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	bundled, err := bundleSource(unbundledFixture, resolveDir)
	if err != nil {
		t.Fatalf("bundleSource: %v", err)
	}

	if strings.Contains(bundled, "\nimport ") {
		t.Error("bundled output should not contain import statements")
	}
	if !strings.Contains(bundled, "unbundled-el") {
		t.Error("missing element tag name")
	}
	if !strings.Contains(bundled, "customElements.define") {
		t.Error("missing customElements.define")
	}
}

func TestBundleFiles(t *testing.T) {
	bundled, err := bundleFiles([]string{"testdata/unbundled-component.ts"})
	if err != nil {
		t.Fatalf("bundleFiles: %v", err)
	}

	if !strings.Contains(bundled, "unbundled-el") {
		t.Error("missing element tag name")
	}
	if !strings.Contains(bundled, "customElements.define") {
		t.Error("missing customElements.define")
	}
}

func TestBundleFilesCSS(t *testing.T) {
	bundled, err := bundleFiles([]string{"testdata/css-component/css-el.ts"})
	if err != nil {
		t.Fatalf("bundleFiles: %v", err)
	}

	if !strings.Contains(bundled, "css-el") {
		t.Error("missing element tag name")
	}
	if !strings.Contains(bundled, "replaceSync") {
		t.Error("CSS should be bundled via replaceSync")
	}
	if !strings.Contains(bundled, "color: green") {
		t.Error("CSS content should be present")
	}
}

func TestNewFromUnbundledSource(t *testing.T) {
	r, err := New(context.Background(), unbundledFixture, 1)
	if err != nil {
		t.Fatalf("New with unbundled source: %v", err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })

	out, err := r.RenderHTML(context.Background(), `<unbundled-el></unbundled-el>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(out, `shadowrootmode="open"`) {
		t.Error("missing DSD in output")
	}
	if !strings.Contains(out, "unbundled") {
		t.Error("missing rendered content")
	}
}

func TestNewFromFiles(t *testing.T) {
	r, err := NewFromFiles(context.Background(), []string{"testdata/unbundled-component.ts"}, 1)
	if err != nil {
		t.Fatalf("NewFromFiles: %v", err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })

	out, err := r.RenderHTML(context.Background(), `<unbundled-el></unbundled-el>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(out, `shadowrootmode="open"`) {
		t.Error("missing DSD in output")
	}
}

func TestNewFromFilesExported(t *testing.T) {
	r, err := NewFromFiles(context.Background(), []string{"testdata/exported-component.ts"}, 1)
	if err != nil {
		t.Fatalf("NewFromFiles: %v", err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })

	out, err := r.RenderHTML(context.Background(), `<exported-el></exported-el>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(out, `shadowrootmode="open"`) {
		t.Error("missing DSD in output")
	}
}

func TestNewFromFilesCSS(t *testing.T) {
	r, err := NewFromFiles(context.Background(), []string{"testdata/css-component/css-el.ts"}, 1)
	if err != nil {
		t.Fatalf("NewFromFiles: %v", err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })

	out, err := r.RenderHTML(context.Background(), `<css-el></css-el>`)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(out, `shadowrootmode="open"`) {
		t.Error("missing DSD in output")
	}
	if !strings.Contains(out, "color: green") {
		t.Error("CSS should be in rendered output")
	}
}

func TestBundleSourceExportStripped(t *testing.T) {
	// Verify that exports in raw source are stripped by esbuild,
	// since the output is eval'd (not imported as a module).
	resolveDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	src := `import { LitElement, html } from 'lit';
export class Foo extends LitElement {
  render() { return html` + "`<p>hi</p>`" + `; }
}
customElements.define('foo-el', Foo);`

	bundled, err := bundleSource(src, resolveDir)
	if err != nil {
		t.Fatalf("bundleSource: %v", err)
	}
	if strings.Contains(bundled, "\nexport ") {
		t.Error("bundled output should not contain export statements (eval can't handle them)")
	}
}
