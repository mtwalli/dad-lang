package interp_test

import (
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/interp"
)

const formApp = `
مكون الرئيسية() {
    حالة الاسم = ""
    حالة البريد = ""
    حالة كلمة_السر = ""
    حالة التأكيد = ""
    حالة العمر = 30
    حالة البلد = ""
    حالة موافق = خطأ
    حالة الرسالة = ""
    نموذج(مسافة: 12) {
        حقل(الاسم، تسمية: "الاسم"، مطلوب: صح، أدنى_طول: 2)
        حقل(البريد، تسمية: "البريد"، صيغة: "بريد"، مطلوب: صح)
        حقل(كلمة_السر، صيغة: "كلمة_سر"، أدنى_طول: 8، أقصى_طول: 20)
        حقل(التأكيد، تحقق: { إذا هو == كلمة_السر { عدم } وإلا { "لا تطابق كلمة السر" } })
        حقل(العمر، أدنى: 18، أقصى: 120)
        قائمة_منسدلة(البلد، ["مصر"، "ألمانيا"]، تسمية: "البلد"، مطلوب: صح)
        مربع_اختيار(موافق، نص: "أوافق"، مطلوب: صح)
        زر("سجّل") { الرسالة = "أهلاً {الاسم}" }
        زر("إلغاء"، إرسال: خطأ) { الرسالة = "أُلغي" }
    }
    فقرة(الرسالة)
}
`

func TestForms(t *testing.T) {
	in, _ := newInterp(t, formApp, nil)
	sess := interp.NewUISession()
	in.Open(sess, "/")
	html := page(t, in, sess)
	for _, want := range []string{
		`<form class="dad-form" novalidate style="gap:12px">`,
		// a label above the field, marked when the field is required
		`<div class="dad-field"><label class="dad-label" for="f1">الاسم<span class="dad-req" aria-hidden="true"> *</span></label><input type="text" value="" data-input="`,
		`" id="f1" data-blur="`,
		`<input type="email" value="" data-input="`,
		// no mistakes shown before the field is left or the form sent
		`<button type="submit" data-click="`,
		`<button type="button" data-click="`,
		`<label class="dad-label" for="f6">البلد<span class="dad-req" aria-hidden="true"> *</span></label><select data-choose="`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("want\n  %s\nin\n%s", want, html)
		}
	}
	if strings.Contains(html, "dad-error") {
		t.Errorf("mistakes shown too early: %s", html)
	}
	// the button does nothing while a field is wrong, and shows every mistake
	click := func(text string) {
		t.Helper()
		if err := in.Event(sess, sess.Version, handlerID(t, html, "data-click", text), nil); err != nil {
			t.Fatal(err)
		}
		html = page(t, in, sess)
	}
	click("سجّل")
	for _, want := range []string{
		`aria-invalid="true" aria-describedby="f1-e"><p class="dad-error" id="f1-e">هذا الحقل مطلوب</p>`,
		`<p class="dad-error" id="f2-e">هذا الحقل مطلوب</p>`,
		`<p class="dad-error" id="f6-e">اختر واحداً</p>`,
		`<p class="dad-error" id="f7-e">حدّد هذا المربع للمتابعة</p>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("after sending: want\n  %s\nin\n%s", want, html)
		}
	}
	if strings.Contains(html, "أهلاً") || strings.Contains(html, `id="f3-e"`) || strings.Contains(html, `id="f5-e"`) {
		t.Errorf("an empty optional field, or a right one, has a mistake; or the button ran: %s", html)
	}
	// the cancel button doesn't check
	click("إلغاء")
	if !strings.Contains(html, "<p>أُلغي</p>") {
		t.Errorf("cancel: %s", html)
	}
	type step struct{ field, value, mistake string }
	input := func(id, value string) {
		t.Helper()
		if err := in.Event(sess, sess.Version, inputOf(t, html, id), value); err != nil {
			t.Fatal(err)
		}
		html = page(t, in, sess)
	}
	for _, s := range []step{
		{"f1", "م", "عدد الأحرف 2 على الأقل"},
		{"f2", "ليس-بريداً", "أدخل بريداً إلكترونياً صحيحاً، مثل name@example.com"},
		{"f3", "قصيرة", "عدد الأحرف 8 على الأقل"},
		{"f3", strings.Repeat("ك", 21), "عدد الأحرف 20 على الأكثر"},
		{"f4", "شيء", "لا تطابق كلمة السر"},
		{"f5", "12", "أقل قيمة مسموحة 18"},
		{"f5", "200", "أكبر قيمة مسموحة 120"},
		{"f5", "كثير", "أدخل رقماً"},
	} {
		input(s.field, s.value)
		if !strings.Contains(html, `<p class="dad-error" id="`+s.field+`-e">`+s.mistake+`</p>`) {
			t.Errorf("%s = %q: want %q in\n%s", s.field, s.value, s.mistake, html)
		}
	}
	// put everything right: the button runs, and the form starts over
	input("f1", "مريم")
	input("f2", "maryam@example.com")
	input("f3", "كلمة_سر_طويلة")
	input("f4", "كلمة_سر_طويلة")
	input("f5", "٣٠")
	if err := in.Event(sess, sess.Version, between(html, `<select data-choose="`, `"`), "1"); err != nil {
		t.Fatal(err)
	}
	html = page(t, in, sess)
	if err := in.Event(sess, sess.Version, between(html, `<input type="checkbox" data-toggle="`, `"`), true); err != nil {
		t.Fatal(err)
	}
	html = page(t, in, sess)
	if strings.Contains(html, "dad-error") {
		t.Fatalf("mistakes left: %s", html)
	}
	click("سجّل")
	if !strings.Contains(html, "<p>أهلاً مريم</p>") {
		t.Errorf("the button didn't run: %s", html)
	}
}

// Leaving a field shows its mistake; the others wait.
func TestFormFieldsLeft(t *testing.T) {
	in, _ := newInterp(t, formApp, nil)
	sess := interp.NewUISession()
	in.Open(sess, "/")
	html := page(t, in, sess)
	blur := between(html, `id="f2" data-blur="`, `"`)
	if blur == "" {
		t.Fatalf("no data-blur: %s", html)
	}
	if err := in.Event(sess, sess.Version, blur, nil); err != nil {
		t.Fatal(err)
	}
	html = page(t, in, sess)
	if !strings.Contains(html, `<p class="dad-error" id="f2-e">هذا الحقل مطلوب</p>`) || strings.Contains(html, `id="f1-e"`) {
		t.Errorf("after leaving the email: %s", html)
	}
	if strings.Contains(html, `id="f2" data-blur`) {
		t.Errorf("a field already left asks again: %s", html)
	}
}

func TestFormErrors(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{`حقل(س، مطلوب: صح)`, "«مطلوب» في «حقل» لحقل داخل «نموذج»"},
		{`زر("أ"، إرسال: صح)`, "«إرسال» في «زر» لزر داخل «نموذج»"},
		{"نموذج {\n        نموذج { }\n    }", "لا يوضع «نموذج» داخل «نموذج» آخر"},
		{"نموذج {\n        حقل(س، تحقق: 3)\n    }", "«تحقق» في «حقل» يأخذ دالة"},
		{"نموذج {\n        حقل(س، تحقق: { 3 })\n    }", "«تحقق» في «حقل» تُرجع نص الخطأ أو عدم"},
	} {
		in, _ := newInterpUnchecked(t, "مكون م() {\n    حالة س = \"\"\n    "+c.body+"\n}")
		sess := interp.NewUISession()
		in.Open(sess, "/")
		if _, err := in.RenderPage(sess); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want %q, got %v", c.body, c.want, err)
		}
	}
}

// inputOf is the input handler of the field with an id.
func inputOf(t *testing.T, html, id string) string {
	t.Helper()
	i := strings.Index(html, `id="`+id+`"`)
	if i < 0 {
		t.Fatalf("no field %s in %s", id, html)
	}
	start := strings.LastIndex(html[:i], "<")
	tag := html[start:]
	tag = tag[:strings.Index(tag, ">")]
	h := between(tag, `data-input="`, `"`)
	if h == "" {
		t.Fatalf("field %s has no input handler: %s", id, tag)
	}
	return h
}
