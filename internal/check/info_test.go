package check_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/mtwalli/dad-lang/internal/check"
)

const program = `نوع نقطة(س: عشري، ص: عشري) {
    دالة بعد() = جذر(س * س + ص * ص)
}

دالة مربع(عدد: صحيح): صحيح = عدد * عدد

ثابت ن = نقطة(س: 3، ص: 4)
ثابت م = ن.نسخة(س: 1)
اطبع(ن.بعد()، ن.س، مربع(عدد: 2)، مربع(3))
لكل ع من 1 إلى 3 {
    اطبع(مربع(ع))
}
`

// pos finds the n-th occurrence (from 0) of a word: its line and column, from 1.
func pos(t *testing.T, src, word string, n int) (int, int) {
	t.Helper()
	for i, line := range strings.Split(src, "\n") {
		runes := []rune(line)
		w := []rune(word)
		for col := 0; col+len(w) <= len(runes); col++ {
			boundary := (col == 0 || !unicode.IsLetter(runes[col-1])) &&
				(col+len(w) == len(runes) || !unicode.IsLetter(runes[col+len(w)]))
			if string(runes[col:col+len(w)]) == word && boundary {
				if n == 0 {
					return i + 1, col + 1
				}
				n--
			}
		}
	}
	t.Fatalf("%q #%d not found", word, n)
	return 0, 0
}

func TestDefinitionsAndUses(t *testing.T) {
	info, diags := check.Analyze(program)
	if info == nil || len(diags) > 0 {
		t.Fatalf("%v", diags)
	}
	at := func(word string, n int) *check.Def {
		t.Helper()
		l, c := pos(t, program, word, n)
		d, ok := info.DefAt(l, c)
		if !ok {
			t.Fatalf("no definition at %q #%d", word, n)
		}
		return d
	}

	// a function: every call is a use, and the declaration says what it is
	sq := at("مربع", 2)
	if sq.Kind != "function" || sq.Detail != "دالة مربع(عدد: صحيح): صحيح" {
		t.Errorf("مربع: %+v", sq)
	}
	if uses := info.Uses(sq); len(uses) != 4 {
		t.Errorf("مربع is used %d times: %v", len(uses), uses)
	}
	// a field: the declaration, «ن.س», «س: 3» in the constructor, «نسخة(س: …)» and the method's «س»
	field := at("س", 5) // ن.س
	if field.Kind != "field" || field.Parent == nil || field.Parent.Name != "نقطة" {
		t.Errorf("س: %+v", field)
	}
	if uses := info.Uses(field); len(uses) != 6 {
		t.Errorf("the field س appears %d times: %v", len(uses), uses)
	}
	// a parameter, used by name at a call site
	param := at("عدد", 3) // مربع(عدد: 2)
	if param.Kind != "parameter" || len(info.Uses(param)) != 4 {
		t.Errorf("عدد: %+v uses %v", param, info.Uses(param))
	}
	// a constant's type comes from its value
	if n := at("ن", 1); n.Type.String() != "نقطة" || n.Kind != "constant" {
		t.Errorf("ن: %+v", n)
	}
	// a method, from a call
	if m := at("بعد", 1); m.Kind != "method" || m.Parent.Name != "نقطة" {
		t.Errorf("بعد: %+v", m)
	}
	// the loop variable
	if e := at("ع", 1); e.Kind != "variable" || e.Type.String() != "صحيح" {
		t.Errorf("ع: %+v", e)
	}
	// the type, from its use in the constructor
	if typ := at("نقطة", 1); typ.Kind != "type" || len(typ.Children) != 3 {
		t.Errorf("نقطة: %+v", typ)
	}
}

func TestMembersAndVisibleNames(t *testing.T) {
	info, _ := check.Analyze(program)
	l, c := pos(t, program, "اطبع(مربع(ع))", 0)
	var names []string
	for _, d := range info.Visible(l, c) {
		names = append(names, d.Name)
	}
	joined := " " + strings.Join(names, " ") + " "
	for _, want := range []string{"ن", "م", "مربع", "نقطة", "ع"} {
		if !strings.Contains(joined, " "+want+" ") {
			t.Errorf("%q is not visible: %v", want, names)
		}
	}
	tp := info.TypeOf([]string{"ن"}, l, c)
	var members []string
	for _, m := range info.Members(tp) {
		members = append(members, m.Name)
	}
	if strings.Join(members, " ") != "بعد س ص نسخة" {
		t.Errorf("members of نقطة: %v", members)
	}
	if ms := info.Members(info.TypeOf([]string{"ن", "س"}, l, c)); len(ms) == 0 || ms[0].Name == "" {
		t.Errorf("members of عشري: %v", ms)
	}
}
