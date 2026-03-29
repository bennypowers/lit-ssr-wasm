package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "lit-ssr-test-*")
	if err != nil {
		panic(err)
	}
	binaryPath = filepath.Join(tmp, "lit-ssr")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("go build failed: " + err.Error())
	}

	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestStdinMode(t *testing.T) {
	input := "<test-card><h2 slot=\"header\">Hi</h2><p>Body</p></test-card>\x00"

	cmd := exec.Command(binaryPath, "--skip-bundle", testdataPath("test-components.js"))
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("lit-ssr exited with error: %v\nstderr: %s", err, stderr.String())
	}

	// Output should be NUL-terminated
	out := stdout.String()
	if !strings.HasSuffix(out, "\x00") {
		t.Fatalf("expected NUL-terminated output, got: %q", out)
	}

	// Strip trailing NUL and compare with golden
	got := strings.TrimSuffix(out, "\x00")
	want, err := os.ReadFile(testdataPath("card.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestStdinMultiple(t *testing.T) {
	input := "<test-badge state=\"success\">up</test-badge>\x00<test-sheet>styled</test-sheet>\x00"

	cmd := exec.Command(binaryPath, "--skip-bundle", testdataPath("test-components.js"))
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("lit-ssr exited with error: %v\nstderr: %s", err, stderr.String())
	}

	parts := strings.Split(stdout.String(), "\x00")
	// Last element after trailing NUL is empty
	if len(parts) < 3 {
		t.Fatalf("expected 2 NUL-delimited results, got output: %q", stdout.String())
	}

	goldens := []string{"badge", "sheet"}
	for i, name := range goldens {
		want, err := os.ReadFile(testdataPath(name + ".golden"))
		if err != nil {
			t.Fatalf("read golden %s: %v", name, err)
		}
		if parts[i] != string(want) {
			t.Errorf("result %d (%s): mismatch\ngot:\n%s\nwant:\n%s", i, name, parts[i], string(want))
		}
	}
}

func TestRenderSubcommand(t *testing.T) {
	// Set up temp dir with HTML files to render
	tmp := t.TempDir()

	cardHTML := `<test-card><h2 slot="header">Hi</h2><p>Body</p></test-card>`
	badgeHTML := `<test-badge state="success">up</test-badge>`

	cardPath := filepath.Join(tmp, "card.html")
	badgePath := filepath.Join(tmp, "badge.html")

	if err := os.WriteFile(cardPath, []byte(cardHTML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badgePath, []byte(badgeHTML), 0o640); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "render",
		"--skip-bundle", testdataPath("test-components.js"),
		cardPath, badgePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("lit-ssr render exited with error: %v\nstderr: %s", err, stderr.String())
	}

	// Check card output matches golden
	gotCard, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatal(err)
	}
	wantCard, err := os.ReadFile(testdataPath("card.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotCard) != string(wantCard) {
		t.Errorf("card.html mismatch\ngot:\n%s\nwant:\n%s", gotCard, wantCard)
	}

	// Check badge output matches golden
	gotBadge, err := os.ReadFile(badgePath)
	if err != nil {
		t.Fatal(err)
	}
	wantBadge, err := os.ReadFile(testdataPath("badge.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBadge) != string(wantBadge) {
		t.Errorf("badge.html mismatch\ngot:\n%s\nwant:\n%s", gotBadge, wantBadge)
	}

	// Verify file permissions preserved (Unix only; Windows doesn't support fine-grained mode bits)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(badgePath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o640 {
			t.Errorf("badge.html permissions: got %o, want 640", info.Mode().Perm())
		}
	}
}

func TestRenderNoFiles(t *testing.T) {
	cmd := exec.Command(binaryPath, "render", "--skip-bundle", testdataPath("test-components.js"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit, got success")
	}
	if !strings.Contains(stderr.String(), "no HTML files") {
		t.Errorf("expected 'no HTML files' in stderr, got: %s", stderr.String())
	}
}

func TestRenderNoSource(t *testing.T) {
	cmd := exec.Command(binaryPath, "render", "some-file.html")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit, got success")
	}
	if !strings.Contains(stderr.String(), "--dir or --skip-bundle is required") {
		t.Errorf("expected source required message in stderr, got: %s", stderr.String())
	}
}
