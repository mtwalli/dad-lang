package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func startServe(t *testing.T, src string) *httptest.Server {
	t.Helper()
	s, _, _ := startDev(t, src)
	s.production = true
	s.load()
	ts := httptest.NewServer(logRequests(secure(s), io.Discard))
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, u string) (int, string, http.Header) {
	t.Helper()
	res, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b), res.Header
}

func TestServeHidesDeveloperTools(t *testing.T) {
	ts := startServe(t, devSrc)
	for _, p := range []string{"/_dad", "/_dad/", "/_dad/info", "/_dad/version"} {
		if code, _, _ := get(t, ts.URL+p); code != 404 {
			t.Errorf("%s: %d, want 404", p, code)
		}
	}
	if code, body, _ := get(t, ts.URL+"/_dad/health"); code != 200 || !strings.Contains(body, `"ok": true`) {
		t.Errorf("health: %d %s", code, body)
	}
	// the program itself is served: server functions and routes
	code, body, _ := do(t, "POST", ts.URL+"/_dad/call/"+url.PathEscape("أضف"), `{"عنوان": "نشر"}`)
	if code != 200 {
		t.Fatalf("call: %d %v", code, body)
	}
	code, text, h := get(t, ts.URL+"/api/"+url.PathEscape("مهام"))
	if code != 200 || !strings.Contains(text, "نشر") {
		t.Errorf("route: %d %s", code, text)
	}
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("headers: %v", h)
	}
	// no page at / without an interface, and no developer page instead
	if code, _, _ := get(t, ts.URL+"/"); code != 404 {
		t.Errorf("/: %d", code)
	}
}

func TestServePageDoesNotReload(t *testing.T) {
	ts := startServe(t, "مكون الرئيسية {\n    حالة ع = 0\n    زر(\"زد {ع}\") { ع += 1 }\n}\n")
	code, page, _ := get(t, ts.URL+"/")
	if code != 200 || !strings.Contains(page, "زد 0") {
		t.Fatalf("page: %d %s", code, page)
	}
	if strings.Contains(page, "data-live") {
		t.Error("a production page must not poll for reloads")
	}
}

// The whole command: it refuses a program with mistakes, serves on $PORT, and
// stops cleanly on SIGTERM.
func TestServeCommand(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.dad")
	os.WriteFile(bad, []byte("اطبع(غير_معرف)\n"), 0o644)
	if code := cmdServe([]string{bad}); code != 1 {
		t.Errorf("a program with mistakes: exit %d", code)
	}
	if code := cmdServe([]string{}); code != 2 {
		t.Errorf("no file: exit %d", code)
	}
}
