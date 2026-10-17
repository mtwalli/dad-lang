package interp_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/parser"
)

const tasks = `
جدول مهمة {
  عنوان: نص
  منجزة = خطأ
  أولوية = 1
}

خادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)
خادم دالة بدل(م: مهمة) = م.نسخة(منجزة: ليس م.منجزة).احفظ()
دالة سرية() = 1

مسار جلب "/api/مهام" = مهمة.الكل()
مسار جلب "/api/مهام/{رقم}" = مهمة.بالرقم(رقم)
مسار أرسل "/api/مهام" {
  أرجع مهمة.أنشئ(عنوان: البيانات["عنوان"])
}
`

func newInterp(t *testing.T, src string, store interp.Store) (*interp.Interp, *bytes.Buffer) {
	t.Helper()
	file, diags := parser.ParseFile(src)
	if file == nil {
		t.Fatalf("parse failed: %v", diags)
	}
	var out bytes.Buffer
	in := interp.New(&out)
	in.MaxSteps = 1_000_000
	if store != nil {
		in.Store = store
	}
	if err := in.Run(file); err != nil {
		t.Fatal(err)
	}
	return in, &out
}

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestTables(t *testing.T) {
	got, err := run(t, tasks+`
ثابت أ = مهمة.أنشئ(عنوان: "اشترِ خبزاً")
ثابت ب = مهمة.أنشئ(عنوان: "اكتب التقرير"، أولوية: 2)
اطبع(أ.رقم، ب.رقم، مهمة.عدد())
ب.نسخة(منجزة: صح).احفظ()
اطبع(مهمة.حيث { هو.منجزة }.حول { هو.عنوان })
اطبع(مهمة.بالرقم(1).عنوان، مهمة.بالرقم(9))
أ.احذف()
اطبع(مهمة.الكل())
ثابت جديدة = مهمة(عنوان: "لم تُحفظ")
اطبع(جديدة.رقم، مهمة.عدد())
`)
	if err != nil {
		t.Fatal(err)
	}
	want := `1 2 2
[«اكتب التقرير»]
اشترِ خبزاً عدم
[مهمة(عنوان: «اكتب التقرير»، منجزة: صح، أولوية: 2، رقم: 2)]
0 1
`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestTableErrors(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"جدول س { رقم: صحيح }", "رقم"},
		{"جدول س { اسم }", "نوع"},
	} {
		_, diags := parser.ParseFile(c.src)
		if len(diags) == 0 || !strings.Contains(diags[0].String(), c.want) {
			t.Errorf("%s: got %v, want an error mentioning %q", c.src, diags, c.want)
		}
	}
	_, err := run(t, "جدول س { اسم: نص }\nس(اسم: \"أ\").احذف()")
	if err == nil || !strings.Contains(err.Error(), "لم يُحفظ") {
		t.Errorf("deleting an unsaved record: got %v", err)
	}
}

func TestFileStorePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".dad", "data.json")
	s1, err := interp.OpenFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	newInterp(t, tasks+`مهمة.أنشئ(عنوان: "أ")
مهمة.أنشئ(عنوان: "ب").احذف()
مهمة.أنشئ(عنوان: "ج"، أولوية: 3)`, s1)

	s2, err := interp.OpenFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	_, out := newInterp(t, tasks+`اطبع(مهمة.الكل().حول { هو.رقم }، مهمة.بالرقم(3).أولوية)
اطبع(مهمة.أنشئ(عنوان: "د").رقم)`, s2)
	if got := out.String(); got != "[1، 3] 3\n4\n" {
		t.Errorf("after reopening: got %q", got)
	}
}

func TestServerAPI(t *testing.T) {
	in, _ := newInterp(t, tasks, nil)

	fns := in.ServerFunctions()
	if got := jsonString(fns); got != `[{"name":"أضف","params":[{"name":"عنوان","type":"نص","optional":false}]},{"name":"بدل","params":[{"name":"م","type":"مهمة","optional":false}]}]` {
		t.Errorf("server functions: %s", got)
	}
	res, err := in.CallServer("أضف", map[string]any{"عنوان": "اشترِ خبزاً"})
	if err != nil || jsonString(res) != `{"أولوية":1,"رقم":1,"عنوان":"اشترِ خبزاً","منجزة":false}` {
		t.Fatalf("أضف: %v %v", jsonString(res), err)
	}
	res, err = in.CallServer("بدل", map[string]any{"م": res})
	if err != nil || res.(map[string]any)["منجزة"] != true {
		t.Fatalf("بدل: %v %v", jsonString(res), err)
	}
	for name, args := range map[string]map[string]any{
		"سرية": {},                                // not a server function
		"أضف":  {"عنوان": 5.0},                    // wrong type
		"بدل":  {"م": map[string]any{}, "س": 1.0}, // unknown parameter
	} {
		if _, err := in.CallServer(name, args); err == nil {
			t.Errorf("%s(%v): want an error", name, args)
		}
	}

	r, params := in.MatchRoute("GET", "/api/مهام/1")
	if r == nil || params["رقم"] != "1" {
		t.Fatalf("route match: %v %v", r, params)
	}
	res, err = in.ServeRoute(r, params, nil)
	if err != nil || res.(map[string]any)["عنوان"] != "اشترِ خبزاً" {
		t.Fatalf("GET one: %v %v", jsonString(res), err)
	}
	r, params = in.MatchRoute("POST", "/api/مهام")
	res, err = in.ServeRoute(r, params, map[string]any{"عنوان": "من الويب"})
	if err != nil || res.(map[string]any)["رقم"] != int64(2) {
		t.Fatalf("POST: %v %v", jsonString(res), err)
	}
	if r, _ := in.MatchRoute("DELETE", "/api/مهام"); r != nil {
		t.Error("DELETE should not match")
	}
	tables, err := in.Tables()
	if err != nil || len(tables) != 1 || len(tables[0].Rows) != 2 || strings.Join(tables[0].Fields, ",") != "عنوان,منجزة,أولوية,رقم" {
		t.Errorf("tables: %s %v", jsonString(tables), err)
	}
}
