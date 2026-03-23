package litssr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// importExportRe detects top-level import/export statements that indicate
// unbundled ES module source needing bundling before eval.
var importExportRe = regexp.MustCompile(`(?m)^(?:import\s|export\s)`)

// needsBundle returns true if the source contains top-level import/export
// statements, indicating it needs to be bundled before eval.
func needsBundle(source string) bool {
	return importExportRe.MatchString(source)
}

// bundleSource bundles JavaScript/TypeScript source into a self-contained
// script suitable for eval in the lit-ssr-wasm runtime. If resolveDir is
// empty, the current working directory is used.
func bundleSource(source, resolveDir string) (string, error) {
	if resolveDir == "" {
		resolveDir, _ = os.Getwd()
	}

	nodePaths := findNodeModules(resolveDir)

	result := api.Build(ssrBuildOptions(api.StdinOptions{
		Contents:   stripImportAttributes(source),
		Sourcefile: "components.ts",
		Loader:     api.LoaderTS,
		ResolveDir: resolveDir,
	}, nodePaths))

	if len(result.Errors) > 0 {
		msgs := api.FormatMessages(result.Errors, api.FormatMessagesOptions{})
		return "", fmt.Errorf("litssr: bundle: %s", strings.Join(msgs, "\n"))
	}

	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("litssr: bundle produced no output")
	}

	return string(result.OutputFiles[0].Contents), nil
}

// bundleFiles reads and bundles multiple JS/TS files into a single
// self-contained script. Files are combined via a generated entry point
// that imports each file.
func bundleFiles(files []string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("litssr: no files to bundle")
	}

	// Use the directory of the first file as resolve root
	resolveDir := filepath.Dir(files[0])

	var entry strings.Builder
	for _, f := range files {
		abs, err := filepath.Abs(f)
		if err != nil {
			return "", fmt.Errorf("litssr: abs path %s: %w", f, err)
		}
		fmt.Fprintf(&entry, "import '%s';\n", abs)
	}

	return bundleSource(entry.String(), resolveDir)
}

// findNodeModules walks up from dir looking for node_modules directories.
func findNodeModules(dir string) []string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}
	var paths []string
	for {
		nm := filepath.Join(abs, "node_modules")
		if info, err := os.Stat(nm); err == nil && info.IsDir() {
			paths = append(paths, nm)
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			break
		}
		abs = parent
	}
	return paths
}

// ssrBuildOptions returns esbuild options configured for lit-ssr-wasm
// runtime compatibility. nodePaths allows specifying additional directories
// for module resolution (e.g., a project's node_modules).
func ssrBuildOptions(stdin api.StdinOptions, nodePaths []string) api.BuildOptions {
	return api.BuildOptions{
		Stdin:      &stdin,
		Bundle:     true,
		Format:     api.FormatESModule,
		Target:     api.ES2022,
		Platform:   api.PlatformNode,
		Conditions: []string{"node"},
		Write:      false,
		Define:     map[string]string{"process.env.NODE_ENV": `"production"`},
		LogLevel:   api.LogLevelWarning,
		NodePaths:  nodePaths,
		Plugins: []api.Plugin{
			litSsrWasmPlugin(),
			litCSSPlugin(),
			stubNodeBuiltins(),
		},
	}
}

// litSsrWasmPlugin resolves @lit-labs/ssr-dom-shim to globalThis
// re-exports so the consumer's Lit shares the WASM runtime's registries.
func litSsrWasmPlugin() api.Plugin {
	shimBridge := `
export const customElements = globalThis.customElements;
export const HTMLElement = globalThis.HTMLElement;
export const Element = globalThis.Element;
export const CSSStyleSheet = globalThis.CSSStyleSheet;
export const CustomElementRegistry = globalThis.CustomElementRegistry;
export const Event = globalThis.Event;
export const CustomEvent = globalThis.CustomEvent;
export const EventTarget = globalThis.EventTarget;
export const ariaMixinAttributes = globalThis.ariaMixinAttributes ?? {};
export const HYDRATE_INTERNALS_ATTR_PREFIX = globalThis.HYDRATE_INTERNALS_ATTR_PREFIX ?? 'internals-';
export const ElementInternals = globalThis.ElementInternals ?? class ElementInternals {};
`
	return api.Plugin{
		Name: "lit-ssr-wasm",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^@lit-labs/ssr-dom-shim`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{Path: args.Path, Namespace: "lit-ssr-wasm-shim"}, nil
			})
			build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "lit-ssr-wasm-shim"}, func(_ api.OnLoadArgs) (api.OnLoadResult, error) {
				return api.OnLoadResult{Contents: &shimBridge, Loader: api.LoaderJS}, nil
			})
		},
	}
}

// litCSSPlugin transforms CSS imports (with { type: 'css' }) into
// CSSStyleSheet modules for the lit-ssr-wasm runtime.
func litCSSPlugin() api.Plugin {
	return api.Plugin{
		Name: "lit-css",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `\.css$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				if strings.Contains(args.Importer, "node_modules") {
					return api.OnResolveResult{}, nil
				}
				absPath := filepath.Join(filepath.Dir(args.Importer), args.Path)
				return api.OnResolveResult{
					Path:      absPath,
					Namespace: "lit-css",
				}, nil
			})
			build.OnLoad(api.OnLoadOptions{Filter: `\.css$`, Namespace: "lit-css"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				css, err := os.ReadFile(args.Path)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				jsonCSS, _ := json.Marshal(string(css))
				jsonWrapped, _ := json.Marshal(string(jsonCSS))
				js := fmt.Sprintf(
					"const s=new CSSStyleSheet();s.replaceSync(JSON.parse(%s));export default s;",
					string(jsonWrapped),
				)
				return api.OnLoadResult{Contents: &js, Loader: api.LoaderJS}, nil
			})
		},
	}
}

// stubNodeBuiltins stubs Node.js built-in modules for QuickJS.
// Buffer.from(s, 'binary').toString('base64') delegates to globalThis.btoa.
func stubNodeBuiltins() api.Plugin {
	bufferStub := `export default {};
export const Buffer = {
  from(x, encoding) {
    if (typeof x === "string") {
      if (encoding === "binary") {
        return { toString(enc) { if (enc === "base64") return globalThis.btoa(x); return x; } };
      }
      return new TextEncoder().encode(x);
    }
    return new Uint8Array(x);
  },
  isBuffer: () => false,
  alloc: n => new Uint8Array(n),
};
export const readFileSync = () => "";`
	return api.Plugin{
		Name: "stub-node-builtins",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^(?:node:)?(?:buffer|fs|path|stream|util|events|crypto|os|url|http|https|net|tls|child_process|module|vm|zlib|node-fetch)(?:/|$)`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{Path: args.Path, Namespace: "node-stub"}, nil
			})
			build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "node-stub"}, func(_ api.OnLoadArgs) (api.OnLoadResult, error) {
				return api.OnLoadResult{Contents: &bufferStub, Loader: api.LoaderJS}, nil
			})
		},
	}
}

// stripImportAttributes removes `with { type: 'css' }` from import
// statements since esbuild doesn't support import attributes natively.
func stripImportAttributes(source string) string {
	result := source
	for _, pattern := range []string{
		` with { type: 'css' }`,
		` with { type: "css" }`,
	} {
		result = strings.ReplaceAll(result, pattern, "")
	}
	return result
}
