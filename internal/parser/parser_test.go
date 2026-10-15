package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/parser"
)

func TestExamplesParse(t *testing.T) {
	files, _ := filepath.Glob("../../examples/*.dad")
	if len(files) == 0 {
		t.Fatal("no examples")
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		file, diags := parser.ParseFile(string(src))
		if file == nil || lexer.HasErrors(diags) {
			t.Errorf("%s: %v", f, diags)
		}
	}
}

func TestSyntaxErrors(t *testing.T) {
	cases := []struct{ src, want string }{
		{"ثابت = ٥", "توقعت اسماً"},
		{"دالة ف(س صحيح) = س", "توقعت «:»"},
		{"اطبع(١", "لم يُغلق"},
		{"إذا صح { ١ } وإلا", "توقعت «{»"},
		{"ثابت س = ١ ٢", "توقعت نهاية السطر"},
		{"طابق ١ { ١ ٢ }", "توقعت «=>»"},
		{"\"قيمة {١ +}\"", "داخل النص"},
	}
	for _, c := range cases {
		file, diags := parser.ParseFile(c.src)
		if file != nil {
			t.Errorf("%q: expected a syntax error", c.src)
			continue
		}
		msg := diags[len(diags)-1].Msg
		if !strings.Contains(msg, c.want) {
			t.Errorf("%q: want %q in %q", c.src, c.want, msg)
		}
	}
}

func TestTrailingLambdaNotInConditions(t *testing.T) {
	src := "ثابت س = ١\nإذا س { اطبع(س) }\nلكل ع في [١] { اطبع(ع) }"
	if file, diags := parser.ParseFile(src); file == nil {
		t.Fatal(diags)
	}
}
