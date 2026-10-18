package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuild(t *testing.T) {
	dir := t.TempDir()

	// A program with an interface becomes a static site.
	site := filepath.Join(dir, "site")
	if code := cmdBuild([]string{"../../examples/06-counter.dad", "--out", site}); code != 0 {
		t.Fatalf("exit code %d", code)
	}
	for _, f := range []string{"index.html", "app.js", "dad-runtime.js", "assets/ui.js", "assets/ui.css", "assets/fonts/NotoKufiArabic-Bold.ttf"} {
		if _, err := os.Stat(filepath.Join(site, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	page, _ := os.ReadFile(filepath.Join(site, "index.html"))
	for _, want := range []string{`href="assets/ui.css"`, `<script src="dad-runtime.js"></script>`, `<script src="app.js"></script>`, `<script src="assets/ui.js"></script>`} {
		if !strings.Contains(string(page), want) {
			t.Errorf("index.html lacks %s", want)
		}
	}

	// Any other program becomes app.js for node.
	src := filepath.Join(dir, "hello.dad")
	os.WriteFile(src, []byte("اطبع(\"مرحبا\"، 1 + 2)\n"), 0o644)
	if code := cmdBuild([]string{src}); code != 0 {
		t.Fatalf("exit code %d", code)
	}
	app := filepath.Join(dir, "dist", "hello", "app.js")
	if _, err := os.Stat(app); err != nil {
		t.Fatal(err)
	}
	if nodePath, err := exec.LookPath("node"); err == nil {
		out, err := exec.Command(nodePath, app).CombinedOutput()
		if err != nil || string(out) != "مرحبا 3\n" {
			t.Errorf("node app.js: %v %q", err, out)
		}
	}

	// A program with mistakes is not built.
	bad := filepath.Join(dir, "bad.dad")
	os.WriteFile(bad, []byte("اطبع(غير_معرف)\n"), 0o644)
	if code := cmdBuild([]string{bad}); code != 1 {
		t.Errorf("a program with errors: exit code %d", code)
	}
}
