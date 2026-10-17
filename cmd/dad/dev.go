package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/parser"
)

//go:embed devpage.html
var devPage []byte

// dataPath is where a program's tables are kept: .dad/<name>.json next to it.
func dataPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return filepath.Join(filepath.Dir(path), ".dad", base+".json")
}

// openStore opens the data file for a program, or keeps data in memory for stdin.
func openStore(path string) (interp.Store, error) {
	if path == "-" {
		return interp.NewMemoryStore(), nil
	}
	return interp.OpenFileStore(dataPath(path))
}

// devServer runs one Ḍād program as a web server and reloads it when the file changes.
type devServer struct {
	path  string
	store interp.Store
	out   io.Writer

	mu      sync.Mutex
	in      *interp.Interp
	err     string // the last load error, in Arabic
	version int
	mtime   time.Time
}

// load parses and runs the file. A program that doesn't parse leaves the last good
// one running, so a typo doesn't take the server down.
func (s *devServer) load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fi, err := os.Stat(s.path); err == nil {
		s.mtime = fi.ModTime()
	}
	s.version++
	src, err := os.ReadFile(s.path)
	if err != nil {
		s.err = fmt.Sprintf("تعذرت قراءة الملف «%s»: %v", s.path, err)
		return
	}
	file, diags := parser.ParseFile(string(src))
	var msgs []string
	for _, d := range diags {
		msgs = append(msgs, d.String())
		if !d.Warning {
			fmt.Fprintf(os.Stderr, "%s: %s\n", s.path, d)
		}
	}
	if file == nil {
		s.err = strings.Join(msgs, "\n")
		return
	}
	in := interp.New(s.out)
	in.Store = s.store
	in.MaxSteps = 50_000_000
	s.err = ""
	if err := in.Run(file); err != nil {
		s.err = err.Error()
		fmt.Fprintf(os.Stderr, "%s: %s\n", s.path, err)
	}
	if s.version == 1 && len(in.Notices) > 0 {
		fmt.Fprintf(os.Stderr, "ملاحظة: لم يُشغَّل %s — الواجهات تأتي في المرحلة 2.\n", strings.Join(in.Notices, "، "))
	}
	s.in = in
}

// watch reloads the program whenever its modification time changes.
func (s *devServer) watch(stop <-chan struct{}) {
	t := time.NewTicker(300 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			fi, err := os.Stat(s.path)
			s.mu.Lock()
			changed := err == nil && !fi.ModTime().Equal(s.mtime)
			s.mu.Unlock()
			if changed {
				fmt.Fprintf(os.Stderr, "↻ أُعيد تحميل %s\n", s.path)
				s.load()
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"خطأ": msg})
}

// readBody decodes a JSON request body; an empty body is nil.
func readBody(r *http.Request) (any, error) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, errors.New("جسم الطلب ليس JSON صحيحاً")
	}
	return v, nil
}

func (s *devServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := r.URL.Path

	switch {
	case path == "/_dad/version":
		writeJSON(w, 200, map[string]any{"version": s.version, "error": s.err})
		return
	case path == "/_dad/info":
		s.info(w)
		return
	case strings.HasPrefix(path, "/_dad/call/"):
		s.call(w, r, strings.TrimPrefix(path, "/_dad/call/"))
		return
	case path == "/_dad" || path == "/_dad/":
		s.page(w)
		return
	}

	if s.in == nil {
		writeError(w, 500, s.err)
		return
	}
	route, params := s.in.MatchRoute(r.Method, path)
	if route == nil {
		if path == "/" && r.Method == http.MethodGet {
			s.page(w)
			return
		}
		writeError(w, 404, fmt.Sprintf("لا يوجد مسار «%s %s»", r.Method, path))
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	result, err := s.in.ServeRoute(route, params, body)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if result == nil && r.Method == http.MethodGet {
		writeError(w, 404, "غير موجود")
		return
	}
	writeJSON(w, 200, result)
}

func (s *devServer) page(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(devPage)
}

type routeInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
}

func (s *devServer) info(w http.ResponseWriter) {
	out := map[string]any{
		"file":      filepath.Base(s.path),
		"version":   s.version,
		"error":     s.err,
		"routes":    []routeInfo{},
		"functions": []interp.ServerFunc{},
		"tables":    []interp.TableInfo{},
	}
	if fs, ok := s.store.(*interp.FileStore); ok {
		out["data"] = fs.Path()
	}
	if s.in != nil {
		routes := []routeInfo{}
		for _, r := range s.in.Routes() {
			routes = append(routes, routeInfo{r.Method, r.Path, r.Pos.Line})
		}
		out["routes"] = routes
		if fns := s.in.ServerFunctions(); fns != nil {
			out["functions"] = fns
		}
		tables, err := s.in.Tables()
		if err != nil {
			out["error"] = err.Error()
		} else if tables != nil {
			out["tables"] = tables
		}
	}
	writeJSON(w, 200, out)
}

// call runs a server function: POST /_dad/call/<name> with {"param": value, …}.
func (s *devServer) call(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "استدعاء دوال الخادم يكون بطلب POST")
		return
	}
	if s.in == nil {
		writeError(w, 500, s.err)
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	args, ok := body.(map[string]any)
	if body != nil && !ok {
		writeError(w, 400, "المعاملات تُرسل كائناً، مثل: {\"عنوان\": \"…\"}")
		return
	}
	result, err := s.in.CallServer(name, args)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"result": result})
}

func cmdDev(args []string) int {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	port := fs.Int("port", 8080, "")
	fs.IntVar(port, "منفذ", 8080, "")
	fs.SetOutput(io.Discard)
	// Allow the file before or after the flags.
	var file string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		file, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "الاستخدام: ضاد تطوير <ملف.dad> [--port 8080]")
		return 2
	}
	if file == "" && fs.NArg() == 1 {
		file = fs.Arg(0)
	}
	if file == "" {
		fmt.Fprintln(os.Stderr, "الاستخدام: ضاد تطوير <ملف.dad> [--port 8080]")
		return 2
	}
	if _, err := os.Stat(file); err != nil {
		fmt.Fprintf(os.Stderr, "تعذرت قراءة الملف «%s»: %v\n", file, err)
		return 1
	}
	store, err := openStore(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	s := &devServer{path: file, store: store, out: os.Stdout}
	s.load()
	go s.watch(nil)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "تعذر فتح المنفذ %d: %v\n(جرّب منفذاً آخر: --port 8081)\n", *port, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "ضاد يعمل على http://localhost:%d — البيانات في %s\nيُعاد التحميل عند حفظ الملف. للإيقاف: Ctrl+C\n",
		*port, dataPath(file))
	if err := http.Serve(ln, s); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
