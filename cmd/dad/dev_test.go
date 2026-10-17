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
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), ".dad", "app.json")); err != nil {
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
