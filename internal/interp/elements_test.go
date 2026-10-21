package interp_test

import (
	"io"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/parser"
)

const elementsApp = `
تعداد أولوية {
    عادية
    عاجلة
}

جدول مهمة {
    العنوان: نص
    منجزة = خطأ
    الموعد = ""
}

سمة(اللون: "فيروزي"، الخط: "نسخ"، الزوايا: 14، الوضع: "داكن")

مكون الرئيسية() {
    حالة اللون = "أخضر"
    حالة الأولوية = عاجلة
    حالة الملاحظات = "سطر"
    حالة مفتوحة = خطأ
    حالة اللسان = "الثاني"
    مهمة.أنشئ(العنوان: "خبز"، منجزة: صح)
    عمود(مسافة: 8، حشو: 12، خلفية: "ذهبي"، عرض: 600، صنف: "رئيسي مميز") {
        عنوان("العناصر"، لون: "أساسي"، محاذاة: "وسط")
        فقرة("صغيرة"، حجم: "صغير")
        قائمة_منسدلة(اللون، ["أحمر"، "أخضر"]، تلميح: "اختر لوناً")
        اختيار(الأولوية، [عادية، عاجلة])
        منطقة_نص(الملاحظات، أسطر: 3، تلميح: "ملاحظات…")
        حقل(الملاحظات، صيغة: "تاريخ"، أدنى: "2026-01-01")
        جدول_عرض(مهمة.الكل()، أعمدة: ["العنوان"، "منجزة"]) { م => زر("احذف"، نمط: "خطر"، حجم: "صغير") { م.احذف() } }
        جدول_عرض([]، فارغ: "لا مهام")
        قائمة_عناصر(["أ"، "ب"]، مرقمة: صح)
        ألسنة(اللسان) {
            لسان("الأول") { فقرة("في الأول") }
            لسان("الثاني"، أيقونة: "نجمة") { فقرة("في الثاني") }
        }
        زر("افتح"، أيقونة: "أضف"، نمط: "ثانوي"، ملء: صح) { مفتوحة = صح }
        نافذة(مفتوحة، عنوان: "نافذة") { فقرة("داخل النافذة") }
        تنبيه("تم الحفظ"، نمط: "نجاح")
        أيقونة("بحث"، وصف: "بحث"، حجم: "كبير")
        رابط("المزيد"، "https://example.com"، نمط: "ثانوي"، أيقونة: "رابط")
    }
}
`

func TestNewElements(t *testing.T) {
	in, _ := newInterp(t, elementsApp, nil)
	sess := interp.NewUISession()
	in.Open(sess, "/")
	html := page(t, in, sess)
	for _, want := range []string{
		// the theme: its variables first, and the dark mode
		`<div class="dad-app" data-mode="dark"><style>:root{--theme-primary:#0e8f9a;--theme-on-primary:#ffffff;--theme-font:var(--font-naskh);--theme-radius:14px}</style>`,
		// options that change the look
		`<div class="dad-col رئيسي مميز" style="gap:8px;padding:12px;max-width:600px;background:color-mix(in srgb, #c9a04a 16%, var(--paper))">`,
		`<h1 style="color:var(--lapis);text-align:center">العناصر</h1>`,
		`<p class="dad-small">صغيرة</p>`,
		// choices: the state's value is selected; a hint first
		`<option value="" disabled>اختر لوناً</option><option value="0">أحمر</option><option value="1" selected>أخضر</option></select>`,
		`<div class="dad-choices" role="radiogroup"><label class="dad-check"><input type="radio" name="`,
		`value="1" checked data-choose="`, `<span>عاجلة</span>`,
		`<textarea rows="3" placeholder="ملاحظات…" data-input="`, `>سطر</textarea>`,
		`<input type="date" value="سطر" min="2026-01-01"`,
		// a table of records, with a column for the row's block
		`<div class="dad-table"><table><thead><tr><th>العنوان</th><th>منجزة</th><th></th></tr></thead><tbody><tr><td>خبز</td><td>✓</td><td class="dad-actions"><button type="button" class="dad-danger dad-small" data-click="`,
		`<p class="dad-empty">لا مهام</p>`,
		`<ol class="dad-list"><li>أ</li><li>ب</li></ol>`,
		// tabs: the open one from the state's title
		`<button type="button" role="tab" aria-selected="false" data-click="`,
		`<svg class="dad-icon" viewBox="0 0 24 24" aria-hidden="true"><path`, `<span>الثاني</span></button>`,
		`<div class="dad-tabpanel" role="tabpanel"><p>في الثاني</p></div>`,
		`<button type="button" class="dad-secondary dad-fill" data-click="`,
		`<div class="dad-alert dad-alert-success" role="status"><svg class="dad-icon"`,
		`<svg class="dad-icon dad-large" viewBox="0 0 24 24" role="img" aria-label="بحث">`,
		`<a href="https://example.com" class="dad-button dad-secondary"><svg class="dad-icon"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("want\n  %s\nin\n%s", want, html)
		}
	}
	if strings.Contains(html, "داخل النافذة") || strings.Contains(html, "في الأول") {
		t.Errorf("closed dialog or tab shown: %s", html)
	}

	// choose a colour and a priority
	sel := between(html, `<select data-choose="`, `"`)
	if err := in.Event(sess, sess.Version, sel, 0.0); err != nil {
		t.Fatal(err)
	}
	html = page(t, in, sess)
	radio := between(html, `value="0" data-choose="`, `"`)
	if err := in.Event(sess, sess.Version, radio, "0"); err != nil {
		t.Fatal(err)
	}
	html = page(t, in, sess)
	if !strings.Contains(html, `<option value="0" selected>أحمر</option>`) || !strings.Contains(html, `value="0" checked data-choose`) {
		t.Errorf("after choosing: %s", html)
	}
	// open the first tab
	first := handlerID(t, html, "data-click", "الأول")
	in.Event(sess, sess.Version, first, nil)
	if html = page(t, in, sess); !strings.Contains(html, "<p>في الأول</p>") {
		t.Errorf("first tab: %s", html)
	}
	// open the dialog, then close it
	in.Event(sess, sess.Version, between(html, `class="dad-secondary dad-fill" data-click="`, `"`), nil)
	html = page(t, in, sess)
	if !strings.Contains(html, `<div class="dad-dialog" role="dialog" aria-modal="true" tabindex="-1" aria-label="نافذة"><div class="dad-dialog-head"><h2>نافذة</h2><button type="button" class="dad-close" aria-label="إغلاق"`) ||
		!strings.Contains(html, "<p>داخل النافذة</p>") {
		t.Fatalf("dialog: %s", html)
	}
	in.Event(sess, sess.Version, between(html, `aria-label="إغلاق" data-click="`, `"`), nil)
	if html = page(t, in, sess); strings.Contains(html, "dad-dialog") {
		t.Errorf("dialog still open: %s", html)
	}
	// typing in the text area
	in.Event(sess, sess.Version, between(html, `<textarea rows="3" placeholder="ملاحظات…" data-input="`, `"`), "\nسطران")
	if html = page(t, in, sess); !strings.Contains(html, ">\n\nسطران</textarea>") {
		t.Errorf("text area: %s", html)
	}
}

func TestTabsKeepTheirOwnState(t *testing.T) {
	in, _ := newInterp(t, "مكون م() {\n    ألسنة {\n        لسان(\"أ\") { فقرة(\"١\") }\n        لسان(\"ب\") { فقرة(\"٢\") }\n    }\n}", nil)
	sess := interp.NewUISession()
	in.Open(sess, "/")
	html := page(t, in, sess)
	if !strings.Contains(html, "<p>١</p>") {
		t.Fatalf("first tab: %s", html)
	}
	in.Event(sess, sess.Version, handlerID(t, html, "data-click", "ب"), nil)
	if html = page(t, in, sess); !strings.Contains(html, "<p>٢</p>") || !strings.Contains(html, `aria-selected="true" data-click="h2">ب`) {
		t.Errorf("second tab: %s", html)
	}
}

func TestElementErrors(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{`زر("أ"، نمط: "رئيسي")`, "«نمط» في «زر» يكون «أساسي» أو «ثانوي» أو «خفيف» أو «خطر»"},
		{`فقرة("أ"، لون: "سماوي")`, "«لون» في «فقرة» يأخذ اسم لون"},
		{`أيقونة("بحثث")`, "لا توجد أيقونة باسم «بحثث»"},
		{`فقرة("أ"، صنف: "a;b")`, "«صنف» يأخذ أسماء"},
		{`لسان("أ") { }`, "«لسان» يُستخدم داخل «ألسنة»"},
		{"ألسنة {\n        فقرة(\"أ\")\n    }", "داخل «ألسنة» تُكتب الألسنة فقط"},
		{`جدول_عرض([1، 2]، أعمدة: ["الاسم"])`, "لا يوجد حقل باسم «الاسم» في قيمة من النوع «صحيح»"},
		{`قائمة_منسدلة("أ")`, "«قائمة_منسدلة» يأخذ القيمة المختارة وقائمة الخيارات"},
		{`نافذة(1)`, "«نافذة» تأخذ قيمة منطقية"},
		{`سمة(اللون: "أحمر")`, "«سمة» تُكتب في أعلى البرنامج"},
	} {
		in, _ := newInterpUnchecked(t, "مكون م() {\n    "+c.body+"\n}")
		sess := interp.NewUISession()
		in.Open(sess, "/")
		if _, err := in.RenderPage(sess); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want %q, got %v", c.body, c.want, err)
		}
	}
	if _, err := run(t, "سمة(الحجم: 50)"); err == nil || !strings.Contains(err.Error(), "«الحجم» في «سمة» بين 12 و28") {
		t.Errorf("theme size: %v", err)
	}
}

func between(s, a, b string) string {
	i := strings.Index(s, a)
	if i < 0 {
		return ""
	}
	s = s[i+len(a):]
	return s[:strings.Index(s, b)]
}

// newInterpUnchecked runs a program the checker would refuse, to test the errors
// the interpreter gives on its own (for instance with values computed at run time).
func newInterpUnchecked(t *testing.T, src string) (*interp.Interp, error) {
	t.Helper()
	file, diags := parser.ParseFile(src)
	if file == nil {
		t.Fatalf("parse failed: %v", diags)
	}
	in := interp.New(io.Discard)
	return in, in.Run(file)
}
