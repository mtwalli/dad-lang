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
	if got != string(want) {
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
		{"ثابت س = ١\nس = ٢", "لأنه ثابت"},
		{"اطبع(غير_موجود)", "«غير_موجود» غير معرّف"},
		{"دالة ف(س: صحيح) = س\nف(\"نص\")", "توقعت قيمة من النوع «صحيح» لكن وجدت «نص»"},
		{"دالة ف(س: صحيح، ص: صحيح) = س\nف(١)", "ينقص المعامل «ص»"},
		{"ارفع مشكلة(\"حدث خطأ\")", "مشكلة لم تُعالَج: حدث خطأ"},
		{"تعداد ل { أ، ب }\nاطبع(طابق ب { أ => ١ })", "لا توجد حالة في «طابق»"},
		{"متغير ن = ٠\nطالما صح { ن += ١ }", "حلقة لا تنتهي"},
		{"دالة ف(): نص { اطبع(١) }\nف()", "هل نسيت «أرجع»"},
		{"اطبع([١، ٢][٥])", "خارج الحدود"},
		{"اطبع(١ / ٠)", "قسمة على صفر"},
		{"ثابت س: نص = عدم", "لا يقبل «عدم»"},
		{"دالة ف(ن: صحيح): صحيح = ف(ن + ١)\nف(١)", "عمق الاستدعاءات"},
		{"ثابت ق = [١]\nق.س = ١", "لا تتغير"},
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
	got, err := run(t, "ثابت ن = حاول { ١ / ٠ } التقط خ { اطبع(خ.رسالة)؛ -١ }\nاطبع(ن)")
	if err != nil {
		t.Fatal(err)
	}
	if got != "قسمة على صفر\n-١\n" {
		t.Errorf("got %q", got)
	}
}
