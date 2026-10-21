package interp_test

import (
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/interp"
)

const pagesApp = `
جدول مهمة {
    العنوان: نص
}

صفحة "/" = الرئيسية()
صفحة "/مهمة/{رقم}" = تفاصيل(رقم)

صفحة "/عن" {
    حالة ع = 0
    القائمة()
    عنوان("عن البرنامج")
    زر("زد {ع}") { ع += 1 }
}

مكون القائمة() {
    صف {
        رابط("الرئيسية"، "/")
        رابط("عن"، "/عن")
        رابط("الواجهة البرمجية"، "/api/مهام")
        رابط("خارجي"، "https://example.com")
    }
}

مكون الرئيسية() {
    عمود {
        القائمة()
        عنوان("المهام")
        زر("أضف") {
            ثابت م = مهمة.أنشئ(العنوان: "خبز")
            انتقل("/مهمة/{م.رقم}")
        }
        زر("الموقع") { انتقل("https://example.com/") }
    }
}

مكون تفاصيل(رقم: صحيح) {
    ثابت م = مهمة.بالرقم(رقم)
    عمود {
        القائمة()
        عنوان(م؟.العنوان ؟؟ "لا توجد مهمة {رقم}")
    }
}

مسار جلب "/api/مهام" = مهمة.الكل()
`

func page(t *testing.T, in *interp.Interp, sess *interp.UISession) string {
	t.Helper()
	html, err := in.RenderPage(sess)
	if err != nil {
		t.Fatal(err)
	}
	return html
}

func TestPages(t *testing.T) {
	in, _ := newInterp(t, pagesApp, nil)
	if !in.HasPages() || len(in.Pages()) != 3 {
		t.Fatalf("pages: %d", len(in.Pages()))
	}
	sess := interp.NewUISession()
	if !in.Open(sess, "/") {
		t.Fatal("no page at /")
	}
	html := page(t, in, sess)
	if sess.Title != "المهام" {
		t.Errorf("title: %q", sess.Title)
	}
	// links to pages open in place; the current one is marked; others are ordinary links
	for _, want := range []string{
		`<a href="/" data-nav aria-current="page">الرئيسية</a>`,
		`<a href="/عن" data-nav>عن</a>`,
		`<a href="/api/مهام">الواجهة البرمجية</a>`,
		`<a href="https://example.com">خارجي</a>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("want %s in\n%s", want, html)
		}
	}

	// «انتقل» after adding moves to the new task's page, a value from the address
	add := handlerID(t, html, "data-click", "أضف")
	if err := in.Event(sess, sess.Version, add, nil); err != nil {
		t.Fatal(err)
	}
	if !sess.Moved || sess.Path != "/مهمة/1" || sess.NotFound {
		t.Fatalf("after أضف: moved %v, path %q", sess.Moved, sess.Path)
	}
	html = page(t, in, sess)
	if !strings.Contains(html, "<h1>خبز</h1>") || sess.Title != "خبز" || strings.Contains(html, "aria-current") {
		t.Errorf("task page: %s", html)
	}

	// a page block keeps its state while it is shown, and starts again after leaving
	in.Open(sess, "/%D8%B9%D9%86/") // «/عن/», as a browser sends it
	html = page(t, in, sess)
	for i := 0; i < 2; i++ {
		if err := in.Event(sess, sess.Version, handlerID(t, html, "data-click", "زد"), nil); err != nil {
			t.Fatal(err)
		}
		html = page(t, in, sess)
	}
	if !strings.Contains(html, "زد 2") || !strings.Contains(html, `<a href="/عن" data-nav aria-current="page">`) {
		t.Errorf("about page: %s", html)
	}
	in.Open(sess, "/")
	page(t, in, sess)
	in.Open(sess, "/عن")
	if html = page(t, in, sess); !strings.Contains(html, "زد 0") {
		t.Errorf("state after leaving: %s", html)
	}

	// an address with no page
	if in.Open(sess, "/لا/شيء") || !sess.NotFound {
		t.Fatal("found a page at /لا/شيء")
	}
	html = page(t, in, sess)
	if !strings.Contains(html, "الصفحة غير موجودة") || !strings.Contains(html, "«/لا/شيء»") ||
		!strings.Contains(html, `<a href="/" data-nav>`) || sess.Title != "الصفحة غير موجودة" {
		t.Errorf("not found: %s", html)
	}

	// «انتقل» to another site
	in.Open(sess, "/")
	html = page(t, in, sess)
	if err := in.Event(sess, sess.Version, handlerID(t, html, "data-click", "الموقع"), nil); err != nil {
		t.Fatal(err)
	}
	if sess.Away != "https://example.com/" || sess.Moved {
		t.Errorf("away: %q moved %v", sess.Away, sess.Moved)
	}
}

func TestProgramWithoutPagesShowsItsMainComponent(t *testing.T) {
	in, _ := newInterp(t, "مكون عداد() {\n    عنوان(\"عدّاد\")\n}", nil)
	if in.HasPages() || len(in.Pages()) != 1 || in.Pages()[0].Path != "/" {
		t.Fatalf("pages: %v", in.Pages())
	}
	sess := interp.NewUISession()
	in.Open(sess, "/")
	if html := page(t, in, sess); !strings.Contains(html, "<h1>عدّاد</h1>") {
		t.Errorf("html: %s", html)
	}
	if in.Open(sess, "/أخرى") {
		t.Error("a page at /أخرى")
	}
}

func TestNavigateErrors(t *testing.T) {
	in, _ := newInterp(t, "صفحة \"/\" {\n    انتقل(\"/أ\")\n}\nصفحة \"/ب\" {\n    زر(\"أ\") { انتقل(\"أ\") }\n}", nil)
	sess := interp.NewUISession()
	in.Open(sess, "/")
	if _, err := in.RenderPage(sess); err == nil || !strings.Contains(err.Error(), "لا أثناء عرض الصفحة") {
		t.Errorf("render: %v", err)
	}
	in.Open(sess, "/ب")
	html := page(t, in, sess)
	err := in.Event(sess, sess.Version, handlerID(t, html, "data-click", "أ"), nil)
	if err == nil || !strings.Contains(err.Error(), "يبدأ بـ «/»") {
		t.Errorf("event: %v", err)
	}
	if _, err := run(t, "انتقل(\"/أ\")"); err == nil || !strings.Contains(err.Error(), "في صفحة ويب") {
		t.Errorf("outside a page: %v", err)
	}
}
