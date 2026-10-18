package interp_test

import (
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/parser"
)

// handlerID finds the id of the handler on the element whose HTML matches re.
func handlerID(t *testing.T, html, attr, near string) string {
	t.Helper()
	re := regexp.MustCompile(`<[^>]*` + attr + `="(h\d+)"[^>]*>` + regexp.QuoteMeta(near))
	if m := re.FindStringSubmatch(html); m != nil {
		return m[1]
	}
	re = regexp.MustCompile(`<[^>]*` + attr + `="(h\d+)"`)
	if near == "" {
		if m := re.FindStringSubmatch(html); m != nil {
			return m[1]
		}
	}
	t.Fatalf("no %s near %q in\n%s", attr, near, html)
	return ""
}

func TestTodoApp(t *testing.T) {
	src, err := os.ReadFile("../../examples/05-todo.dad")
	if err != nil {
		t.Fatal(err)
	}
	in, _ := newInterp(t, string(src), nil)
	root := in.RootComponent()
	if root == nil || root.Name != "قائمة_المهام" {
		t.Fatalf("root: %v", root)
	}
	sess := interp.NewUISession()
	html, err := in.Render(sess, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(html)
	if !strings.Contains(html, "مهامي (0 متبقية)") {
		t.Errorf("first render: %s", html)
	}
	// type a task and press أضف
	field := handlerID(t, html, "data-input", "")
	if err := in.Event(sess, sess.Version, field, "اشترِ خبزاً"); err != nil {
		t.Fatal(err)
	}
	html, _ = in.Render(sess, root)
	add := handlerID(t, html, "data-click", "أضف")
	if err := in.Event(sess, sess.Version, add, nil); err != nil {
		t.Fatal(err)
	}
	html, _ = in.Render(sess, root)
	t.Log(html)
	if !strings.Contains(html, "مهامي (1 متبقية)") || !strings.Contains(html, "<p>اشترِ خبزاً</p>") || !strings.Contains(html, `value=""`) {
		t.Errorf("after adding: %s", html)
	}
	// tick it
	box := handlerID(t, html, "data-toggle", "")
	if err := in.Event(sess, sess.Version, box, true); err != nil {
		t.Fatal(err)
	}
	html, _ = in.Render(sess, root)
	if !strings.Contains(html, "مهامي (0 متبقية)") || !strings.Contains(html, "checked") {
		t.Errorf("after ticking: %s", html)
	}
	// an event from an old render is refused
	if err := in.Event(sess, sess.Version-1, box, false); err != interp.ErrStale {
		t.Errorf("stale event: %v", err)
	}
}

func TestComponentsKeepTheirOwnState(t *testing.T) {
	src, err := os.ReadFile("../../examples/06-counter.dad")
	if err != nil {
		t.Fatal(err)
	}
	in, _ := newInterp(t, string(src), nil)
	root := in.RootComponent()
	if root.Name != "الرئيسية" {
		t.Fatalf("root: %s", root.Name)
	}
	sess := interp.NewUISession()
	html, err := in.Render(sess, root)
	if err != nil {
		t.Fatal(err)
	}
	click := func(label string, nth int) {
		t.Helper()
		re := regexp.MustCompile(`data-click="(h\d+)">` + regexp.QuoteMeta(label) + `<`)
		m := re.FindAllStringSubmatch(html, -1)
		if len(m) <= nth {
			t.Fatalf("no button %q #%d in %s", label, nth, html)
		}
		if err := in.Event(sess, sess.Version, m[nth][1], nil); err != nil {
			t.Fatal(err)
		}
		html, _ = in.Render(sess, root)
	}
	click("زد", 0)
	click("زد", 0)
	click("أنقص", 1)
	if !strings.Contains(html, "التفاح: 2") || !strings.Contains(html, "البرتقال: 4") {
		t.Errorf("counters: %s", html)
	}
	if !strings.Contains(html, `<button type="button" data-click`) || strings.Count(html, "disabled") != 0 {
		t.Errorf("no minus button should be disabled: %s", html)
	}
	// a number state gets a number field, and typing keeps it a number
	re := regexp.MustCompile(`type="number" value="1" data-input="(h\d+)"`)
	m := re.FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("number field: %s", html)
	}
	if err := in.Event(sess, sess.Version, m[1], "3"); err != nil {
		t.Fatal(err)
	}
	html, _ = in.Render(sess, root)
	if !strings.Contains(html, "checked") {
		t.Errorf("3 > 1 should tick the box: %s", html)
	}
}

func TestInterfaceErrors(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"زر(\"أ\")", "عنصر واجهة"},
		{"مكون م { فقرة(\"أ\") }\nم()", "ضاد تطوير"},
		{"حالة ع = 0", "داخل مكون"},
	} {
		if _, err := run(t, c.src); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", c.src, err, c.want)
		}
	}
	for _, c := range []struct{ src, want string }{
		{"مكون م { زر(\"أ\"، لون: \"أحمر\") }", "لا يأخذ «لون»"},
		{"مكون م { مربع_اختيار(\"نعم\") }", "قيمة منطقية"},
		{"مكون م { حالة س = 1\nحالة س = 2 }", "معرّف مسبقاً"},
		{"مكون م { رابط(\"أ\") }", "النص والعنوان"},
	} {
		in := uncheckedInterp(t, c.src) // these mistakes are the point of the test
		_, err := in.Render(interp.NewUISession(), in.RootComponent())
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", c.src, err, c.want)
		}
	}
}

func TestRenderEscapesText(t *testing.T) {
	in, _ := newInterp(t, `مكون م { فقرة("<b>\"&") }`, nil)
	html, err := in.Render(interp.NewUISession(), in.RootComponent())
	if err != nil || !strings.Contains(html, "<p>&lt;b&gt;&#34;&amp;</p>") {
		t.Errorf("%s %v", html, err)
	}
}

// uncheckedInterp runs a program without the checker, for tests of run-time errors.
func uncheckedInterp(t *testing.T, src string) *interp.Interp {
	t.Helper()
	file, diags := parser.ParseFile(src)
	if file == nil {
		t.Fatalf("parse failed: %v", diags)
	}
	in := interp.New(io.Discard)
	if err := in.Run(file); err != nil {
		t.Fatal(err)
	}
	return in
}
