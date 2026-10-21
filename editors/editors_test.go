// Package editors holds editor integrations. This test keeps their keyword lists in step
// with the compiler, so a new keyword can't be forgotten in the editor support.
package editors

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/token"
)

func TestVSCodeKnowsEveryKeyword(t *testing.T) {
	js, err := os.ReadFile("vscode/keywords.js")
	if err != nil {
		t.Fatal(err)
	}
	grammarSrc, err := os.ReadFile("vscode/syntaxes/dad.tmLanguage.json")
	if err != nil {
		t.Fatal(err)
	}
	var grammar any
	if err := json.Unmarshal(grammarSrc, &grammar); err != nil {
		t.Fatalf("grammar is not valid JSON: %v", err)
	}
	grammarText := string(grammarSrc)
	// Keywords in keywords.js are quoted object keys: "إذا": [...]
	keyRe := regexp.MustCompile(`"([^"]+)":\s*\[`)
	inJS := map[string]bool{}
	for _, m := range keyRe.FindAllStringSubmatch(string(js), -1) {
		inJS[m[1]] = true
	}
	for kw := range token.Keywords {
		if !inJS[kw] {
			t.Errorf("keyword %q is missing from editors/vscode/keywords.js", kw)
		}
		if !strings.Contains(grammarText, kw) {
			t.Errorf("keyword %q is missing from the VS Code grammar", kw)
		}
	}
}

// Words the compiler knows outside token.Keywords: «صفحة» (a keyword only before an
// address) and the built-in «انتقل» are highlighted and explained everywhere.
func TestEditorsKnowPages(t *testing.T) {
	for _, f := range []string{"vscode/keywords.js", "vscode/syntaxes/dad.tmLanguage.json", "vscode/media/rtl-editor.js",
		"../docs/assets/playground.js", "../tools/pygments-dad/dad_lexer.py"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range []string{"صفحة", "انتقل"} {
			if f == "../tools/pygments-dad/dad_lexer.py" && w == "انتقل" {
				continue // Pygments colours every call alike
			}
			if !strings.Contains(string(b), w) {
				t.Errorf("%q is missing from %s", w, f)
			}
		}
	}
}

// The interface elements are highlighted and explained like the other built-ins.
func TestEditorsKnowEveryElement(t *testing.T) {
	files := map[string]string{}
	for _, f := range []string{"vscode/keywords.js", "vscode/syntaxes/dad.tmLanguage.json", "../docs/assets/playground.js"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		files[f] = string(b)
	}
	for _, name := range interp.Elements {
		for f, text := range files {
			if !strings.Contains(text, name) {
				t.Errorf("element %q is missing from %s", name, f)
			}
		}
	}
}
