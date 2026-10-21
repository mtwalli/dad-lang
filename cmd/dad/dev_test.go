package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const devSrc = `جدول مهمة {
  عنوان: نص
  منجزة = خطأ
}
خادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)
مسار جلب "/api/مهام" = مهمة.الكل()
مسار جلب "/api/مهام/{رقم}" = مهمة.بالرقم(رقم)
`

func startDev(t *testing.T, src string) (*devServer, *httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "app.dad")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows can't delete the temporary folder while the database is open.
	t.Cleanup(func() {
		if c, ok := store.(io.Closer); ok {
			c.Close()
		}
	})
	s := &devServer{path: path, store: store, out: io.Discard}
	s.load()
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)
	return s, ts, path
}

func do(t *testing.T, method, u, body string) (int, map[string]any, []any) {
	t.Helper()
	req, _ := http.NewRequest(method, u, strings.NewReader(body))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	var obj map[string]any
	var list []any
	if json.Unmarshal(data, &obj) != nil {
		json.Unmarshal(data, &list)
	}
	return res.StatusCode, obj, list
}

func TestDevServer(t *testing.T) {
	s, ts, path := startDev(t, devSrc)
	call := ts.URL + "/_dad/call/" + url.PathEscape("أضف")

	code, obj, _ := do(t, "POST", call, `{"عنوان":"اشترِ خبزاً"}`)
	if code != 200 || obj["result"].(map[string]any)["رقم"] != 1.0 {
		t.Fatalf("call: %d %v", code, obj)
	}
	if code, obj, _ = do(t, "POST", call, `{"عنوان":5}`); code != 400 || obj["خطأ"] == nil {
		t.Errorf("bad argument: %d %v", code, obj)
	}
	if code, _, _ = do(t, "GET", call, ""); code != 405 {
		t.Errorf("GET on a call: %d", code)
	}
	if code, _, list := do(t, "GET", ts.URL+"/api/مهام", ""); code != 200 || len(list) != 1 {
		t.Errorf("list route: %d %v", code, list)
	}
	if code, obj, _ = do(t, "GET", ts.URL+"/api/مهام/1", ""); code != 200 || obj["عنوان"] != "اشترِ خبزاً" {
		t.Errorf("one route: %d %v", code, obj)
	}
	if code, _, _ = do(t, "GET", ts.URL+"/api/مهام/7", ""); code != 404 {
		t.Errorf("missing record: %d", code)
	}
	if code, _, _ = do(t, "GET", ts.URL+"/لا", ""); code != 404 {
		t.Errorf("unknown route: %d", code)
	}
	if code, obj, _ = do(t, "GET", ts.URL+"/_dad/info", ""); code != 200 || len(obj["tables"].([]any)) != 1 || len(obj["functions"].([]any)) != 1 {
		t.Errorf("info: %d %v", code, obj)
	}
	res, err := http.Get(ts.URL + "/")
	if err != nil || !strings.Contains(res.Header.Get("Content-Type"), "text/html") {
		t.Errorf("dev page: %v %v", err, res.Header)
	}

	// The data file is written next to the program.
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), ".dad", "app.db")); err != nil {
		t.Errorf("data file: %v", err)
	}

	// A syntax error keeps the last good program running and is reported.
	os.WriteFile(path, []byte(devSrc+"دالة ("), 0o644)
	s.load()
	if code, obj, _ = do(t, "GET", ts.URL+"/_dad/version", ""); obj["error"] == "" {
		t.Errorf("load error not reported: %v", obj)
	}
	if code, _, _ = do(t, "GET", ts.URL+"/api/مهام", ""); code != 200 {
		t.Errorf("old program should still serve: %d", code)
	}

	// Reloading keeps the data.
	os.WriteFile(path, []byte(devSrc+`مسار جلب "/عدد" = مهمة.عدد()`), 0o644)
	s.load()
	res, _ = http.Get(ts.URL + "/" + url.PathEscape("عدد"))
	b, _ := io.ReadAll(res.Body)
	if strings.TrimSpace(string(b)) != "1" {
		t.Errorf("after reload: %s", b)
	}
}

func TestDevWatchReloads(t *testing.T) {
	s, _, path := startDev(t, devSrc)
	stop := make(chan struct{})
	defer close(stop)
	go s.watch(stop)
	later := time.Now().Add(2 * time.Second)
	os.WriteFile(path, []byte(devSrc+"\n"), 0o644)
	os.Chtimes(path, later, later)
	for i := 0; i < 50; i++ {
		time.Sleep(50 * time.Millisecond)
		s.mu.Lock()
		v := s.version
		s.mu.Unlock()
		if v > 1 {
			return
		}
	}
	t.Error("the file change was not picked up")
}

func TestDevServesTheInterface(t *testing.T) {
	src := devSrc + `
مكون الرئيسية {
    حالة جديدة = ""
    عمود {
        عنوان("مهامي ({مهمة.عدد()})")
        حقل(جديدة)
        زر("أضف") {
            أضف(جديدة)
            جديدة = ""
        }
    }
}
`
	_, ts, _ := startDev(t, src)
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(res.Body)
	html := string(page)
	for _, want := range []string{`dir="rtl"`, `<h1>مهامي (0)</h1>`, "/_dad/assets/ui.js", `data-live="1"`} {
		if !strings.Contains(html, want) {
			t.Fatalf("page lacks %q:\n%s", want, html)
		}
	}
	session := between(html, `data-session="`, `"`)
	field := between(html, `data-input="`, `"`)
	button := between(html, `data-click="`, `"`)

	event := func(v int, id string, value any) map[string]any {
		body, _ := json.Marshal(map[string]any{"s": session, "v": v, "id": id, "value": value})
		_, obj, _ := do(t, "POST", ts.URL+"/_dad/ui/event", string(body))
		return obj
	}
	r := event(1, field, "اشترِ خبزاً")
	if r["version"] != 2.0 {
		t.Fatalf("typing: %v", r)
	}
	r = event(2, button, nil)
	if !strings.Contains(r["html"].(string), "مهامي (1)") || !strings.Contains(r["html"].(string), `value=""`) {
		t.Errorf("after adding: %v", r)
	}
	// an old page gets the current render back, and nothing else happens
	r = event(1, button, nil)
	if !strings.Contains(r["html"].(string), "مهامي (1)") {
		t.Errorf("stale event: %v", r)
	}
	// an unknown page is told to reload
	session = "nope"
	if r = event(1, button, nil); r["reload"] != true {
		t.Errorf("unknown session: %v", r)
	}
	// the assets are served
	for _, a := range []string{"ui.js", "ui.css", "fonts/NotoKufiArabic-Bold.ttf"} {
		if res, err := http.Get(ts.URL + "/_dad/assets/" + a); err != nil || res.StatusCode != 200 {
			t.Errorf("%s: %v %v", a, err, res.StatusCode)
		}
	}
	// the developer page moved to /_dad
	if res, _ := http.Get(ts.URL + "/_dad"); res.StatusCode != 200 {
		t.Errorf("dev page: %d", res.StatusCode)
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

// A change to an imported file reloads the program too.
func TestDevWatchesImports(t *testing.T) {
	s, ts, path := startDev(t, "استورد التحية من \"./نصوص\"\nمسار جلب \"/\" = التحية\n")
	lib := filepath.Join(filepath.Dir(path), "نصوص.dad")
	os.WriteFile(lib, []byte("ثابت التحية = \"أهلاً\"\n"), 0o644)
	s.load()
	if s.err != "" {
		t.Fatal(s.err)
	}
	stop := make(chan struct{})
	defer close(stop)
	go s.watch(stop)
	later := time.Now().Add(2 * time.Second)
	os.WriteFile(lib, []byte("ثابت التحية = \"مرحباً\"\n"), 0o644)
	os.Chtimes(lib, later, later)
	for i := 0; i < 50; i++ {
		time.Sleep(50 * time.Millisecond)
		res, err := http.Get(ts.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if strings.Contains(string(b), "مرحباً") {
			return
		}
	}
	t.Error("the change to the imported file was not picked up")
}

const pagesSrc = `جدول مهمة {
    العنوان: نص
}

صفحة "/" = الرئيسية()
صفحة "/مهمة/{رقم}" = تفاصيل(رقم)

مكون الرئيسية() {
    عمود {
        عنوان("المهام")
        رابط("عن"، "/مهمة/1")
        زر("أضف") {
            ثابت م = مهمة.أنشئ(العنوان: "خبز")
            انتقل("/مهمة/{م.رقم}")
        }
    }
}

مكون تفاصيل(رقم: صحيح) {
    عنوان(مهمة.بالرقم(رقم)؟.العنوان ؟؟ "لا توجد مهمة {رقم}")
}

مسار جلب "/api/مهام" = مهمة.الكل()
`

func getPage(t *testing.T, u string, html bool) (int, string) {
	t.Helper()
	req, _ := http.NewRequest("GET", u, nil)
	if html {
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

func TestDevServesPages(t *testing.T) {
	s, ts, _ := startDev(t, pagesSrc)
	code, html := getPage(t, ts.URL+"/", true)
	if code != 200 || !strings.Contains(html, "<title>المهام</title>") || !strings.Contains(html, `<a href="/مهمة/1" data-nav>`) {
		t.Fatalf("/: %d\n%s", code, html)
	}
	session := between(html, `data-session="`, `"`)
	button := between(html, `data-click="`, `"`)

	// a page with a value in its address, asked for directly
	code, html = getPage(t, ts.URL+"/"+url.PathEscape("مهمة")+"/7", true)
	if code != 200 || !strings.Contains(html, "<h1>لا توجد مهمة 7</h1>") {
		t.Fatalf("/مهمة/7: %d\n%s", code, html)
	}
	// an address with no page: a page that says so, for a browser; a plain 404 otherwise
	code, html = getPage(t, ts.URL+"/لا-شيء", true)
	if code != 404 || !strings.Contains(html, "الصفحة غير موجودة") || !strings.Contains(html, `data-session="`) {
		t.Errorf("not found: %d\n%s", code, html)
	}
	sessions := len(s.sessions)
	if code, _ = getPage(t, ts.URL+"/favicon.ico", false); code != 404 || len(s.sessions) != sessions {
		t.Errorf("favicon: %d, sessions %d → %d", code, sessions, len(s.sessions))
	}
	// routes still answer with JSON
	if code, _, list := do(t, "GET", ts.URL+"/api/مهام", ""); code != 200 || list == nil {
		t.Errorf("route: %d", code)
	}

	event := func(ev map[string]any) map[string]any {
		ev["s"] = session
		body, _ := json.Marshal(ev)
		_, obj, _ := do(t, "POST", ts.URL+"/_dad/ui/event", string(body))
		return obj
	}
	// «انتقل» after a click: the reply says where the page is now
	r := event(map[string]any{"v": 1, "id": button})
	if r["path"] != "/مهمة/1" || r["title"] != "خبز" || !strings.Contains(r["html"].(string), "<h1>خبز</h1>") {
		t.Fatalf("after أضف: %v", r)
	}
	// a link or the back button opens a page in the same session
	r = event(map[string]any{"v": 2, "nav": "/"})
	if r["path"] != "/" || r["title"] != "المهام" || !strings.Contains(r["html"].(string), "<h1>المهام</h1>") {
		t.Errorf("nav to /: %v", r)
	}
	r = event(map[string]any{"v": 3, "nav": "/لا-شيء"})
	if r["path"] != "/لا-شيء" || r["title"] != "الصفحة غير موجودة" {
		t.Errorf("nav to nowhere: %v", r)
	}

	// the developer page lists the pages
	_, info, _ := do(t, "GET", ts.URL+"/_dad/info", "")
	if pages := info["pages"].([]any); len(pages) != 2 || pages[1].(map[string]any)["path"] != "/مهمة/{رقم}" {
		t.Errorf("info pages: %v", info["pages"])
	}
}

// «dad serve» serves pages the same way.
func TestServeServesPages(t *testing.T) {
	ts := startServe(t, pagesSrc)
	if code, html := getPage(t, ts.URL+"/مهمة/2", true); code != 200 || !strings.Contains(html, "لا توجد مهمة 2") || strings.Contains(html, "data-live") {
		t.Errorf("page: %d\n%s", code, html)
	}
	if code, html := getPage(t, ts.URL+"/لا-شيء", true); code != 404 || !strings.Contains(html, "الصفحة غير موجودة") {
		t.Errorf("not found: %d\n%s", code, html)
	}
}

// A program's own stylesheet is served after the built-in one, and changing it
// reloads the page in «dad dev».
func TestDevServesTheProgramsStylesheet(t *testing.T) {
	s, ts, path := startDev(t, "مكون الرئيسية() {\n    فقرة(\"أ\"، صنف: \"مميز\")\n}\n")
	if _, html := getPage(t, ts.URL+"/", true); strings.Contains(html, "/_dad/style.css") {
		t.Errorf("a stylesheet without a file:\n%s", html)
	}
	css := filepath.Join(filepath.Dir(path), "app.css")
	os.WriteFile(css, []byte(".مميز { color: teal; }\n"), 0o644)
	s.load()
	_, html := getPage(t, ts.URL+"/", true)
	if !strings.Contains(html, `<link rel="stylesheet" href="/_dad/style.css?v=`) || !strings.Contains(html, `<p class="مميز">أ</p>`) {
		t.Fatalf("page:\n%s", html)
	}
	res, err := http.Get(ts.URL + "/_dad/style.css")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	if res.Header.Get("Content-Type") != "text/css; charset=utf-8" || string(body) != ".مميز { color: teal; }\n" {
		t.Errorf("style.css: %s %q", res.Header.Get("Content-Type"), body)
	}
	if _, ok := s.mtimes[css]; !ok {
		t.Error("the stylesheet is not watched")
	}
}
