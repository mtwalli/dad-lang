// Package editors holds editor integrations. This test keeps their keyword lists in step
// with the compiler, so a new keyword can't be forgotten in the editor support.
package editors

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

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
