package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The preview is the program's page, with its code inside, its stylesheet and
// the editor's policy; a program with errors or without an interface has none.
func TestPreview(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "lib.dad"), []byte("دالة تحية(اسم: نص): نص = \"أهلاً {اسم}\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "app.css"), []byte(".رأس { color: red }\n"), 0o644)
	app := filepath.Join(dir, "app.dad")
	src := "استورد تحية من \"./lib\"\n\nمكون الرئيسية() {\n    حالة ع = 0\n    عنوان(تحية(\"</script>\"))\n    زر(\"زد\") { ع += 1 }\n}\n"

	p := previewPage(app, src, "vscode-resource:/assets/", "N1", "default-src 'none'")
	if !p.Site || len(p.Diagnostics) != 0 {
		t.Fatalf("preview: %+v", p.Diagnostics)
	}
	for _, want := range []string{
		`<meta http-equiv="Content-Security-Policy" content="default-src &#39;none&#39;">`,
		`<link rel="stylesheet" href="vscode-resource:/assets/ui.css">`,
		"<style>\n.رأس { color: red }",
		`<script nonce="N1" src="vscode-resource:/assets/dad-runtime.js"></script>`,
		`<script nonce="N1">`,
		`<\/script>`,
		`<script nonce="N1" src="vscode-resource:/assets/ui.js"></script>`,
		`data-routing="hash"`,
	} {
		if !strings.Contains(p.HTML, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	if strings.Count(p.HTML, "</script>") != 3 {
		t.Errorf("a script ends early:\n%s", p.HTML)
	}

	bad := previewPage(app, "مكون الرئيسية() {\n    عنوان(س)\n}\n", "", "", "")
	if bad.HTML != "" || len(bad.Diagnostics) != 1 || bad.Diagnostics[0].Line != 2 || !strings.Contains(bad.Diagnostics[0].Message, "«س»") {
		t.Errorf("errors: %+v", bad)
	}
	if plain := previewPage(app, "اطبع(1)\n", "", "", ""); plain.Site || plain.HTML != "" {
		t.Errorf("a program without an interface: %+v", plain)
	}

	assets := filepath.Join(dir, "assets")
	if err := writeAssets(assets); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"ui.js", "ui.css", "dad-runtime.js", "fonts/NotoKufiArabic-Medium.ttf"} {
		if _, err := os.Stat(filepath.Join(assets, filepath.FromSlash(f))); err != nil {
			t.Errorf("assets: %v", err)
		}
	}
}
