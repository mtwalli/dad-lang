package lexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/token"
)

func kinds(toks []token.Token) []token.Kind {
	out := make([]token.Kind, 0, len(toks))
	for _, t := range toks {
		out = append(out, t.Kind)
	}
	return out
}

func lexOK(t *testing.T, src string) []token.Token {
	t.Helper()
	toks, diags := Tokenize(src)
	for _, d := range diags {
		t.Errorf("unexpected diagnostic: %s", d)
	}
	return toks
}

func expectKinds(t *testing.T, src string, want ...token.Kind) {
	t.Helper()
	got := kinds(lexOK(t, src))
	want = append(want, token.NEWLINE, token.EOF)
	if len(got) != len(want) {
		t.Fatalf("%q:\n got  %v\n want %v", src, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%q: token %d: got %v, want %v\n all: %v", src, i, got[i], want[i], got)
		}
	}
}

func TestExamplesTokenize(t *testing.T) {
	files, err := filepath.Glob("../../examples/*.dad")
	if err != nil || len(files) == 0 {
		t.Fatalf("no examples found: %v", err)
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			toks, diags := Tokenize(string(src))
			for _, d := range diags {
				t.Errorf("%s", d)
			}
			for _, tk := range toks {
				if tk.Kind == token.ILLEGAL {
					t.Errorf("illegal token %s", tk)
				}
			}
		})
	}
}

func TestKeywordsAndBinding(t *testing.T) {
	expectKinds(t, "ثابت اسم = \"سارة\"",
		token.CONST, token.IDENT, token.ASSIGN, token.STRING)
}

func TestDiacriticsAndTatweelIgnored(t *testing.T) {
	toks := lexOK(t, "دالـــة حوِّل")
	if toks[0].Kind != token.FUNC {
		t.Errorf("tatweel: got %v, want دالة", toks[0].Kind)
	}
	if toks[1].Value != "حول" {
		t.Errorf("diacritics: got %q, want حول", toks[1].Value)
	}
}

func TestDecomposedHamzaIsComposed(t *testing.T) {
	// "أرجع" written with alef + combining hamza above (U+0654).
	toks := lexOK(t, "ا\u0654رجع")
	if toks[0].Kind != token.RETURN {
		t.Errorf("got %v, want أرجع", toks[0].Kind)
	}
}

func TestElseVersusAnd(t *testing.T) {
	expectKinds(t, "وإلا و وصف", token.ELSE, token.AND, token.IDENT)
}

func TestNumbers(t *testing.T) {
	cases := []struct {
		src   string
		kind  token.Kind
		value string
	}{
		{"42", token.INT, "42"},
		{"3.14", token.FLOAT, "3.14"},
		{"1_000", token.INT, "1000"},
		{"0", token.INT, "0"},
	}
	for _, c := range cases {
		toks := lexOK(t, c.src)
		if toks[0].Kind != c.kind || toks[0].Value != c.value {
			t.Errorf("%q: got %v %q, want %v %q", c.src, toks[0].Kind, toks[0].Value, c.kind, c.value)
		}
	}
}

func TestRangeIsNotFloat(t *testing.T) {
	expectKinds(t, "1..10", token.INT, token.RANGE, token.INT)
}

// Only 0-9 are allowed. Other digit forms are errors that show the fix.
func TestOnlyLatinDigits(t *testing.T) {
	cases := []struct{ src, fix string }{
		{"\u0664\u0662", "«42»"},                                                  // ٤٢
		{"\u0663\u066B\u0661\u0664", "«3.14»"},                                    // ٣٫١٤
		{"\u0661\u066C\u0660\u0660\u0660\u066C\u0660\u0660\u0660", "«1_000_000»"}, // ١٬٠٠٠٬٠٠٠
		{"3\u066B14", "«3.14»"},                                                   // 3٫14
		{"\u06612\u0663", "«123»"},                                                // ١2٣
		{"\u06F4\u06F2", "«42»"},                                                  // Persian ۴۲
	}
	for _, c := range cases {
		toks, diags := Tokenize(c.src)
		if !HasErrors(diags) || !strings.Contains(diags[0].Msg, c.fix) || !strings.Contains(diags[0].Msg, "0-9") {
			t.Errorf("%q: want an error suggesting %s, got %v", c.src, c.fix, diags)
		}
		// The value is still read, so one mistake doesn't cause a cascade of errors.
		if toks[0].Kind != token.INT && toks[0].Kind != token.FLOAT {
			t.Errorf("%q: got %v, want a number token", c.src, toks[0].Kind)
		}
	}
	// Names can't hide Arabic-Indic digits either.
	if _, diags := Tokenize("س\u0661 = 1"); !HasErrors(diags) {
		t.Error("«س١»: want an error for the digit in the name")
	}
	// Digits inside text are just text.
	lexOK(t, "\"رقم \u0665 و \u06F5\"")
}

func TestCommaBetweenDigitsWarns(t *testing.T) {
	_, diags := Tokenize("ثابت ن = 1,5")
	if len(diags) != 1 || !diags[0].Warning || !strings.Contains(diags[0].Msg, "«1.5»") {
		t.Errorf("want a warning suggesting 1.5, got %v", diags)
	}
	if _, diags := Tokenize("[1, 5]"); len(diags) != 0 {
		t.Errorf("a list with a space after the comma must not warn: %v", diags)
	}
}

func TestArabicPunctuation(t *testing.T) {
	expectKinds(t, "ف(أ، ب)؛ م؟.بريد ؟؟ \"\"",
		token.IDENT, token.LPAREN, token.IDENT, token.COMMA, token.IDENT, token.RPAREN, token.SEMI,
		token.IDENT, token.SAFE_DOT, token.IDENT, token.COALESCE, token.STRING)
	expectKinds(t, "نص؟", token.IDENT, token.QUESTION)
}

func TestStrings(t *testing.T) {
	toks := lexOK(t, `"مرحبا {اسم} و {ف("}")}" «صباح {س}»`)
	if toks[0].Kind != token.STRING || toks[0].Value != `مرحبا {اسم} و {ف("}")}` {
		t.Errorf("interpolated string: got %v %q", toks[0].Kind, toks[0].Value)
	}
	if toks[1].Kind != token.STRING || toks[1].Value != "صباح {س}" {
		t.Errorf("guillemet string: got %v %q", toks[1].Kind, toks[1].Value)
	}
	if _, d := Tokenize("\"بلا نهاية\n"); !HasErrors(d) {
		t.Error("expected unterminated string error")
	}
	toks = lexOK(t, "\"\"\"سطر\nسطر\"\"\"")
	if toks[0].Value != "سطر\nسطر" {
		t.Errorf("triple-quoted: got %q", toks[0].Value)
	}
}

func TestBidiControlsRejectedOutsideStrings(t *testing.T) {
	if _, d := Tokenize("ثابت أ\u202E = 1"); !HasErrors(d) {
		t.Error("expected error for RLO in code")
	}
	if _, d := Tokenize("// تعليق \u2066 خفي"); !HasErrors(d) {
		t.Error("expected error for LRI in comment")
	}
	lexOK(t, "\"نص \u200F مع علامة\"") // allowed inside strings
}

func TestNewlinesIgnoredInsideParens(t *testing.T) {
	expectKinds(t, "ف(\n1،\n2\n)",
		token.IDENT, token.LPAREN, token.INT, token.COMMA, token.INT, token.RPAREN)
	// …but not inside braces.
	expectKinds(t, "{\nأ\n}", token.LBRACE, token.NEWLINE, token.IDENT, token.NEWLINE, token.RBRACE)
}

func TestKeywordSuggestion(t *testing.T) {
	toks, diags := Tokenize("اذا")
	if HasErrors(diags) || len(diags) != 1 || !strings.Contains(diags[0].Msg, "إذا") {
		t.Fatalf("want one warning suggesting إذا, got %v", diags)
	}
	if toks[0].Kind != token.IDENT {
		t.Errorf("misspelled keyword should stay an identifier, got %v", toks[0].Kind)
	}
}

func TestPositionsCountRunes(t *testing.T) {
	toks := lexOK(t, "ثابت س = 1\nاطبع(س)")
	last := toks[len(toks)-4] // س inside اطبع(س)
	if last.Value != "س" || last.Pos != (token.Pos{Line: 2, Col: 6}) {
		t.Errorf("got %q at %v, want س at 2:6", last.Value, last.Pos)
	}
}

func TestWindowsLineEndings(t *testing.T) {
	toks, diags := Tokenize("ثابت ن = \"\"\"أ\r\nب\"\"\"\r\nاطبع(ن)\r\n")
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	var str token.Token
	for _, tk := range toks {
		if tk.Kind == token.STRING {
			str = tk
		}
	}
	if str.Value != "أ\nب" {
		t.Errorf("multi-line string = %q, want %q", str.Value, "أ\nب")
	}
	last := toks[len(toks)-2] // the token before EOF
	if last.Pos.Line != 3 {
		t.Errorf("line numbers are off: last token %v on line %d, want 3", last, last.Pos.Line)
	}
}
