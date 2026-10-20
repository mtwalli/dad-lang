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
