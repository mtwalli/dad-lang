package sqlitestore_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/parser"
	"github.com/mtwalli/dad-lang/internal/sqlitestore"
)

func open(t *testing.T, path string) *sqlitestore.Store {
	t.Helper()
	s, err := sqlitestore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// run runs src against store and returns what it printed.
func run(t *testing.T, store interp.Store, src string) (string, error) {
	t.Helper()
	file, diags := parser.ParseFile(src)
	if file == nil {
		t.Fatalf("parse failed: %v", diags)
	}
	var out bytes.Buffer
	in := interp.New(&out)
	in.Store = store
	in.MaxSteps = 1_000_000
	err := in.Run(file)
	return out.String(), err
}

func mustRun(t *testing.T, store interp.Store, src string) string {
	t.Helper()
	out, err := run(t, store, src)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

const tasks = `
جدول مهمة {
  عنوان: نص
  منجزة = خطأ
  أولوية = 1
  وسوم: [نص] = []
  ملاحظة: نص؟ = عدم
}
`

func TestSameBehaviourAsMemory(t *testing.T) {
	prog := tasks + `
ثابت أ = مهمة.أنشئ(عنوان: "اشترِ خبزاً"، وسوم: ["بيت"])
ثابت ب = مهمة.أنشئ(عنوان: "اكتب التقرير"، أولوية: 2)
مهمة.أنشئ(عنوان: "اتصل بأحمد"، أولوية: 3، ملاحظة: "مساءً")
اطبع(أ.رقم، ب.رقم، مهمة.عدد())
ب.نسخة(منجزة: صح).احفظ()
ثابت حد = 2
اطبع(مهمة.حيث { هو.منجزة }.حول { هو.عنوان })
اطبع(مهمة.حيث { ليس هو.منجزة و هو.أولوية >= حد }.حول { هو.عنوان })
اطبع(مهمة.حيث { م => م.أولوية == 1 أو م.ملاحظة != عدم }.حول { هو.رقم })
اطبع(مهمة.حيث { هو.عنوان.يحتوي("خبز") }.حول { هو.رقم })
اطبع(مهمة.حيث { 3 > هو.أولوية }.حول { هو.رقم })
اطبع(مهمة.بالرقم(1).وسوم، مهمة.بالرقم(9))
أ.احذف()
اطبع(مهمة.الكل().حول { هو.رقم }، مهمة.عدد())
`
	want := mustRun(t, interp.NewMemoryStore(), prog)
	got := mustRun(t, open(t, filepath.Join(t.TempDir(), "d.db")), prog)
	if got != want {
		t.Errorf("SQLite and memory differ\n--- sqlite\n%s--- memory\n%s", got, want)
	}
	if !strings.HasPrefix(want, "1 2 3\n[«اكتب التقرير»]\n[«اتصل بأحمد»]\n[1، 3]\n[1]\n[1، 2]\n[«بيت»] عدم\n[2، 3] 2\n") {
		t.Errorf("unexpected output:\n%s", want)
	}
}

func TestPersistsAcrossRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.db")
	s1 := open(t, path)
	mustRun(t, s1, tasks+`مهمة.أنشئ(عنوان: "أ")
مهمة.أنشئ(عنوان: "ب").احذف()
مهمة.أنشئ(عنوان: "ج"، أولوية: 3، وسوم: ["س"، "ص"])`)
	s1.Close()
	got := mustRun(t, open(t, path), tasks+`اطبع(مهمة.الكل().حول { هو.رقم }، مهمة.بالرقم(3).وسوم)
اطبع(مهمة.أنشئ(عنوان: "د").رقم)`)
	if got != "[1، 3] [«س»، «ص»]\n4\n" {
		t.Errorf("after reopening: %q", got)
	}
}

func TestMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.db")
	v1 := "جدول منتج {\n  اسم: نص\n  سعر: نص\n  قديم = 1\n}\n"
	s := open(t, path)
	mustRun(t, s, v1+`منتج.أنشئ(اسم: "قلم"، سعر: "5")
منتج.أنشئ(اسم: "دفتر"، سعر: "12")`)

	// Adding a field with a default fills old rows; changing سعر to صحيح converts
	// it; removing قديم keeps its data hidden.
	v2 := "جدول منتج {\n  اسم: نص\n  سعر: صحيح\n  متوفر = صح\n}\n"
	got := mustRun(t, s, v2+`اطبع(منتج.الكل().حول { "{هو.اسم} {هو.سعر + 1} {هو.متوفر}" })
اطبع(منتج.حيث { هو.سعر > 10 }.حول { هو.اسم })`)
	if got != "[«قلم 6 صح»، «دفتر 13 صح»]\n[«دفتر»]\n" {
		t.Errorf("after migration: %q", got)
	}
	if len(s.Notes) != 1 || !strings.Contains(s.Notes[0], "قديم") {
		t.Errorf("notes: %v", s.Notes)
	}

	// A new field without a default can't be added to a table with rows.
	_, err := run(t, s, "جدول منتج {\n  اسم: نص\n  سعر: صحيح\n  متوفر = صح\n  وزن: عشري\n}\n")
	if err == nil || !strings.Contains(err.Error(), "قيمة افتراضية") {
		t.Errorf("field without default: %v", err)
	}

	// A type change that doesn't fit every row is refused and changes nothing.
	mustRun(t, s, v2+`منتج.أنشئ(اسم: "ممحاة"، سعر: 3)`)
	_, err = run(t, s, "جدول منتج {\n  اسم: صحيح\n  سعر: صحيح\n  متوفر = صح\n}\n")
	if err == nil || !strings.Contains(err.Error(), "1، 2، 3") {
		t.Errorf("bad conversion: %v", err)
	}
	if got := mustRun(t, s, v2+`اطبع(منتج.عدد()، منتج.بالرقم(3).اسم)`); got != "3 ممحاة\n" {
		t.Errorf("after refused change: %q", got)
	}

	// Bringing a removed field back brings its data back.
	if got := mustRun(t, s, v2[:len(v2)-2]+"  قديم = 1\n}\n"+`اطبع(منتج.بالرقم(1).قديم)`); got != "1\n" {
		t.Errorf("re-added field: %q", got)
	}
}

func TestImportsJSONData(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "app.json")
	fs, err := interp.OpenFileStore(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	mustRun(t, fs, "جدول مهمة {\n  عنوان: نص\n  منجزة = خطأ\n}\n"+`مهمة.أنشئ(عنوان: "أ")
مهمة.أنشئ(عنوان: "ب").احذف()
مهمة.أنشئ(عنوان: "ج"، منجزة: صح)`)

	s := open(t, filepath.Join(dir, "app.db"))
	if err := s.ImportJSON(jsonPath); err != nil {
		t.Fatal(err)
	}
	got := mustRun(t, s, tasks+`اطبع(مهمة.الكل().حول { "{هو.رقم} {هو.عنوان} {هو.منجزة} {هو.أولوية}" })`)
	if got != "[«1 أ خطأ 1»، «3 ج صح 1»]\n" {
		t.Errorf("imported: %q", got)
	}
	if waiting := s.FinishImport(); len(waiting) != 0 {
		t.Errorf("waiting: %v", waiting)
	}
	if _, err := os.Stat(jsonPath + ".imported"); err != nil {
		t.Errorf("old file not renamed: %v", err)
	}
}

func TestWhereSQL(t *testing.T) {
	s := open(t, filepath.Join(t.TempDir(), "d.db"))
	mustRun(t, s, tasks+`مهمة.أنشئ(عنوان: "أ"، أولوية: 2)
مهمة.أنشئ(عنوان: "ب"، منجزة: صح)`)
	for _, c := range []struct {
		cond *interp.Cond
		n    int
	}{
		{&interp.Cond{Op: "true", Field: "منجزة"}, 1},
		{&interp.Cond{Op: "not", L: &interp.Cond{Op: "true", Field: "منجزة"}}, 1},
		{&interp.Cond{Op: "eq", Field: "ملاحظة", Value: nil}, 2},
		{&interp.Cond{Op: "ne", Field: "عنوان", Value: "أ"}, 1},
		{&interp.Cond{Op: "or", L: &interp.Cond{Op: "ge", Field: "أولوية", Value: int64(2)}, R: &interp.Cond{Op: "eq", Field: "منجزة", Value: true}}, 2},
	} {
		rows, err := s.Where("مهمة", c.cond)
		if err != nil || len(rows) != c.n {
			t.Errorf("%s: %d rows, %v", c.cond, len(rows), err)
		}
	}
}
