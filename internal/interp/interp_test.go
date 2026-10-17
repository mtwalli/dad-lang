package interp_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/parser"
)

func run(t *testing.T, src string) (string, error) {
	t.Helper()
	file, diags := parser.ParseFile(src)
	if file == nil {
		t.Fatalf("parse failed: %v", diags)
	}
	var out bytes.Buffer
	in := interp.New(&out)
	in.MaxSteps = 1_000_000
	err := in.Run(file)
	return out.String(), err
}

func golden(t *testing.T, srcPath, wantPath string) {
	t.Helper()
	src, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	got, runErr := run(t, string(src))
	if runErr != nil {
		t.Fatalf("%s: %v", srcPath, runErr)
	}
	// Expected-output files may have Windows line endings after a checkout on Windows.
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Errorf("%s output mismatch\n--- got\n%s\n--- want\n%s", srcPath, got, want)
	}
}

func TestExamples(t *testing.T) {
	outs, _ := filepath.Glob("../../examples/output/*.txt")
	if len(outs) == 0 {
		t.Fatal("no golden outputs found")
	}
	for _, out := range outs {
		name := strings.TrimSuffix(filepath.Base(out), ".txt")
		t.Run(name, func(t *testing.T) {
			golden(t, "../../examples/"+name+".dad", out)
		})
	}
}

func TestFeatures(t *testing.T) {
	golden(t, "testdata/features.dad", "testdata/features.out")
}

func TestErrors(t *testing.T) {
	cases := []struct{ src, want string }{
		{"ثابت س = 1\nس = 2", "لأنه ثابت"},
		{"اطبع(غير_موجود)", "«غير_موجود» غير معرّف"},
		{"دالة ف(س: صحيح) = س\nف(\"نص\")", "توقعت قيمة من النوع «صحيح» لكن وجدت «نص»"},
		{"دالة ف(س: صحيح، ص: صحيح) = س\nف(1)", "ينقص المعامل «ص»"},
		{"ارفع مشكلة(\"حدث خطأ\")", "مشكلة لم تُعالَج: حدث خطأ"},
		{"تعداد ل { أ، ب }\nاطبع(طابق ب { أ => 1 })", "لا توجد حالة في «طابق»"},
		{"متغير ن = 0\nطالما صح { ن += 1 }", "حلقة لا تنتهي"},
		{"دالة ف(): نص { اطبع(1) }\nف()", "هل نسيت «أرجع»"},
		{"اطبع([1، 2][5])", "خارج الحدود"},
		{"اطبع(1 / 0)", "قسمة على صفر"},
		{"ثابت س: نص = عدم", "لا يقبل «عدم»"},
		{"دالة ف(ن: صحيح): صحيح = ف(ن + 1)\nف(1)", "عمق الاستدعاءات"},
		{"ثابت ق = [1]\nق.س = 1", "لا تتغير"},
	}
	for _, c := range cases {
		file, diags := parser.ParseFile(c.src)
		var err error
		if file == nil {
			if len(diags) == 0 {
				t.Fatalf("%q: parse failed without diagnostics", c.src)
			}
			err = errorString(diags[len(diags)-1].Msg)
		} else {
			in := interp.New(&bytes.Buffer{})
			in.MaxSteps = 100_000
			err = in.Run(file)
		}
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: want error containing %q, got %v", c.src, c.want, err)
		}
	}
}

type errorString string

func (e errorString) Error() string { return string(e) }

func TestCaughtProblem(t *testing.T) {
	got, err := run(t, "ثابت ن = حاول { 1 / 0 } التقط خ { اطبع(خ.رسالة)؛ -1 }\nاطبع(ن)")
	if err != nil {
		t.Fatal(err)
	}
	if got != "قسمة على صفر\n-1\n" {
		t.Errorf("got %q", got)
	}
}

// Code uses 0-9 only, but text typed on an Arabic or Persian keyboard still converts.
func TestTextWithArabicDigitsConverts(t *testing.T) {
	got, err := run(t, "اطبع(\"\u0664\u0662\" ك صحيح + 1، \"\u06F5\" ك صحيح، \"\u0663\u066B\u0665\" ك عشري)")
	if err != nil {
		t.Fatal(err)
	}
	if got != "43 5 3.5\n" {
		t.Errorf("got %q", got)
	}
}
