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
		{"ثابت = 5", "توقعت اسماً"},
		{"دالة ف(س صحيح) = س", "توقعت «:»"},
		{"اطبع(1", "لم يُغلق"},
		{"إذا صح { 1 } وإلا", "توقعت «{»"},
		{"ثابت س = 1 2", "توقعت نهاية السطر"},
		{"طابق 1 { 1 2 }", "توقعت «=>»"},
		{"\"قيمة {1 +}\"", "داخل النص"},
		{"صفحة \"المهام\" { }", "عنوان الصفحة يبدأ بـ «/»"},
		{"صفحة \"/المهام\"", "توقعت «=» أو «{» بعد عنوان الصفحة"},
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
	src := "ثابت س = 1\nإذا س { اطبع(س) }\nلكل ع في [1] { اطبع(ع) }"
	if file, diags := parser.ParseFile(src); file == nil {
		t.Fatal(diags)
	}
}

func TestAttachedWawAndKafHint(t *testing.T) {
	cases := []struct{ src, hint string }{
		{"إذا أ وب { 1 }", "«و ب»"},
		{"ثابت ن = س كعشري", "«ك عشري»"},
	}
	for _, c := range cases {
		_, diags := parser.ParseFile(c.src)
		if len(diags) == 0 || !strings.Contains(diags[0].Msg, "هل تقصد "+c.hint) {
			t.Errorf("%q: want a hint %s, got %v", c.src, c.hint, diags)
		}
	}
}

func TestTypeTestIsMinNaw(t *testing.T) {
	if _, diags := parser.ParseFile("إذا ش من نوع دائرة { 1 }"); len(diags) != 0 {
		t.Errorf("«من نوع» should parse: %v", diags)
	}
	_, diags := parser.ParseFile("إذا ش هو دائرة { 1 }")
	if len(diags) == 0 || !strings.Contains(diags[0].Msg, "من نوع") {
		t.Errorf("«هو» as a type test should point to «من نوع», got %v", diags)
	}
	// «هو» still means "it" in a lambda, and «من» still starts a range.
	for _, src := range []string{"ثابت ك2 = [1، 2].حول { هو * 2 }", "لكل ع من 1 إلى 3 { اطبع(ع) }", "ثابت م = (من 1 إلى 3)"} {
		if _, diags := parser.ParseFile(src); len(diags) != 0 {
			t.Errorf("%q: %v", src, diags)
		}
	}
}
