package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/module"
	"github.com/mtwalli/dad-lang/internal/sqlitestore"
	"github.com/mtwalli/dad-lang/internal/web"
)

//go:embed devpage.html
var devPage []byte

// dataPath is where a program's tables are kept by default: .dad/<name>.db next to it.
func dataPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return filepath.Join(filepath.Dir(path), ".dad", base+".db")
}

// openStore opens the database for a program, or keeps data in memory for stdin.
// Data from the JSON file earlier versions wrote (.dad/<name>.json) is moved
// into a new database.
func openStore(path string) (interp.Store, error) {
	if path == "-" {
		return interp.NewMemoryStore(), nil
	}
	return openStoreAt(dataPath(path))
}

// openStoreAt opens the database in a file of its own.
func openStoreAt(db string) (interp.Store, error) {
	_, statErr := os.Stat(db)
	st, err := sqlitestore.Open(db)
	if err != nil {
		return nil, err
	}
	old := strings.TrimSuffix(db, ".db") + ".json"
	if _, err := os.Stat(old); err == nil && os.IsNotExist(statErr) {
		if err := st.ImportJSON(old); err != nil {
			return nil, err
		}
	}
	return st, nil
}

// storeNotes prints what the store has to say about migrations, once.
func storeNotes(store interp.Store) {
	st, ok := store.(*sqlitestore.Store)
	if !ok {
		return
	}
	for _, n := range st.Notes {
		fmt.Fprintln(os.Stderr, "ملاحظة: "+n)
	}
	st.Notes = nil
	st.FinishImport()
}

// stylePath is a program's own stylesheet: app.css next to app.dad. Its classes
// are given to elements with «صنف».
func stylePath(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ".css"
}

// devServer runs one Ḍād program as a web server. For «dad dev» it reloads the
// program when a file changes and serves the developer tools; for «dad serve»
// (production) it runs the program it started with and serves only the program.
type devServer struct {
	path       string
	store      interp.Store
	out        io.Writer
	production bool

	mu       sync.Mutex
	in       *interp.Interp
	sessions map[string]*interp.UISession // open pages, by id
	order    []string                     // session ids, oldest first
	lastUsed map[string]time.Time         // when each page last sent an event
	err      string                       // the last load error, in Arabic
	version  int
	mtimes   map[string]time.Time // every file of the program, as last loaded
}

// load parses and runs the file. A program that doesn't parse leaves the last good
// one running, so a typo doesn't take the server down.
func (s *devServer) load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.version++
	prog, diags := check.Load(s.path, readFile)
	s.mtimes = map[string]time.Time{}
	for _, p := range append(prog.Paths(), stylePath(s.path)) {
		if fi, err := os.Stat(p); err == nil {
			s.mtimes[p] = fi.ModTime()
		}
	}
	for _, d := range diags {
		if !d.Warning {
			fmt.Fprintln(os.Stderr, d)
		}
	}
	if module.HasErrors(diags) {
		// The last program that worked keeps running until this one is fixed.
		var errs []string
		for _, d := range diags {
			if !d.Warning {
				errs = append(errs, s.relative(d))
			}
		}
		s.err = strings.Join(errs, "\n")
		return
	}
	in := interp.New(s.out)
	in.Store = s.store
	in.MaxSteps = 50_000_000 // for each request: a loop that never ends doesn't hang the server
	s.err = ""
	if err := in.RunProgram(prog); err != nil {
		s.err = err.Error()
		fmt.Fprintf(os.Stderr, "%s: %s\n", errorFile(s.path, err), err)
	}
	storeNotes(s.store)
	s.sessions = map[string]*interp.UISession{} // pages reload with the new program
	s.order = nil
	s.lastUsed = map[string]time.Time{}
	if s.version == 1 && len(in.Notices) > 0 {
		fmt.Fprintf(os.Stderr, "ملاحظة: لم يُشغَّل %s.\n", strings.Join(in.Notices, "، "))
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
			s.mu.Lock()
			changed := ""
			for p, mt := range s.mtimes {
				if fi, err := os.Stat(p); err == nil && !fi.ModTime().Equal(mt) {
					changed = p
				}
			}
			s.mu.Unlock()
			if changed != "" {
				fmt.Fprintf(os.Stderr, "↻ أُعيد تحميل %s\n", changed)
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

	if path == "/_dad/style.css" {
		s.style(w)
		return
	}
	if s.production {
		switch {
		case path == "/_dad/health":
			writeJSON(w, 200, map[string]any{"ok": s.in != nil})
			return
		case strings.HasPrefix(path, "/_dad/call/"):
			s.call(w, r, strings.TrimPrefix(path, "/_dad/call/"))
			return
		case strings.HasPrefix(path, web.AssetsPath):
			web.Handler().ServeHTTP(w, r)
			return
		case path == "/_dad/ui/event":
			s.uiEvent(w, r)
			return
		case path == "/_dad" || strings.HasPrefix(path, "/_dad/"):
			// the developer tools show every table's rows: never in production
			writeError(w, 404, "غير موجود")
			return
		}
	}
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
	case strings.HasPrefix(path, web.AssetsPath):
		web.Handler().ServeHTTP(w, r)
		return
	case path == "/_dad/ui/event":
		s.uiEvent(w, r)
		return
	}

	if s.in == nil {
		writeError(w, 500, s.err)
		return
	}
	route, params := s.in.MatchRoute(r.Method, path)
	if route == nil {
		if r.Method == http.MethodGet && len(s.in.Pages()) > 0 {
			// A page, or, for a browser asking for a page, the one that says there's none.
			if p, _ := s.in.MatchPage(path); p != nil || strings.Contains(r.Header.Get("Accept"), "text/html") {
				s.appPage(w, path)
				return
			}
		}
		if path == "/" && r.Method == http.MethodGet && !s.production {
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

// appPage shows the program's page at an address as a live page.
func (s *devServer) appPage(w http.ResponseWriter, path string) {
	id := newSessionID()
	sess := interp.NewUISession()
	found := s.in.Open(sess, path)
	body, err := s.in.RenderPage(sess)
	msg := ""
	if err != nil {
		msg = err.Error()
	} else {
		s.addSession(id, sess)
	}
	live := s.version
	if s.production {
		live = 0 // nothing reloads in production
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if !found {
		w.WriteHeader(http.StatusNotFound)
	}
	var styles []string
	if fi, err := os.Stat(stylePath(s.path)); err == nil {
		styles = []string{"/_dad/style.css?v=" + strconv.FormatInt(fi.ModTime().UnixNano(), 36)}
	}
	io.WriteString(w, web.Page(web.PageOptions{
		Title: s.title(sess), Body: body, Session: id, Version: sess.Version, Error: msg, Live: live, Styles: styles,
	}))
}

// style serves the program's own stylesheet.
func (s *devServer) style(w http.ResponseWriter) {
	css, err := os.ReadFile(stylePath(s.path))
	if err != nil {
		writeError(w, 404, "لا يوجد ملف تنسيق للبرنامج")
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(css)
}

// title is what the browser tab shows: the page's first heading, or the program's name.
func (s *devServer) title(sess *interp.UISession) string {
	if sess.Title != "" {
		return sess.Title
	}
	return strings.TrimSuffix(filepath.Base(s.path), filepath.Ext(s.path))
}

// Open pages keep their state on the server. «dad dev» keeps the last 200;
// «dad serve» keeps up to 10000, and forgets a page left alone for a day.
const (
	maxSessions     = 200
	maxProdSessions = 10000
	sessionIdle     = 24 * time.Hour
)

func (s *devServer) addSession(id string, sess *interp.UISession) {
	limit := maxSessions
	if s.production {
		limit = maxProdSessions
		now := time.Now()
		kept := s.order[:0]
		for _, old := range s.order {
			if now.Sub(s.lastUsed[old]) > sessionIdle {
				delete(s.sessions, old)
				delete(s.lastUsed, old)
			} else {
				kept = append(kept, old)
			}
		}
		s.order = kept
	}
	s.sessions[id] = sess
	s.lastUsed[id] = time.Now()
	s.order = append(s.order, id)
	if len(s.order) > limit {
		delete(s.sessions, s.order[0])
		delete(s.lastUsed, s.order[0])
		s.order = s.order[1:]
	}
}

func newSessionID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// uiEvent runs a click or typing from a page, or opens another page («nav»: a
// link to a page, or the back button), and sends back the new render.
func (s *devServer) uiEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "الأحداث تُرسل بطلب POST")
		return
	}
	var ev struct {
		Session string `json:"s"`
		Version int    `json:"v"`
		ID      string `json:"id"`
		Value   any    `json:"value"`
		Nav     string `json:"nav"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&ev); err != nil {
		writeError(w, 400, "حدث غير مفهوم")
		return
	}
	sess := s.sessions[ev.Session]
	if sess == nil || s.in == nil || len(s.in.Pages()) == 0 {
		writeJSON(w, 200, map[string]any{"reload": true}) // the program was reloaded
		return
	}
	s.lastUsed[ev.Session] = time.Now()
	writeJSON(w, 200, uiReply(s.in, sess, ev.Version, ev.ID, ev.Value, ev.Nav, s.title))
}

// uiReply runs an event or opens a page, and describes the page to show next:
// its HTML and render version, its title, and the address to show when it moved.
func uiReply(in *interp.Interp, sess *interp.UISession, version int, id string, value any, nav string, title func(*interp.UISession) string) map[string]any {
	reply := map[string]any{}
	if nav != "" {
		in.Open(sess, nav)
		reply["path"] = sess.Path
	} else {
		if err := in.Event(sess, version, id, value); err != nil && !errors.Is(err, interp.ErrStale) {
			reply["error"] = err.Error()
		}
		if sess.Away != "" {
			return map[string]any{"location": sess.Away}
		}
		if sess.Moved {
			reply["path"] = sess.Path
		}
	}
	html, err := in.RenderPage(sess)
	if err != nil {
		reply["error"] = err.Error()
	} else {
		reply["html"], reply["version"] = html, sess.Version
	}
	reply["title"] = title(sess)
	return reply
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
		"pages":     []routeInfo{},
		"functions": []interp.ServerFunc{},
		"tables":    []interp.TableInfo{},
	}
	if fs, ok := s.store.(interface{ Path() string }); ok {
		out["data"] = fs.Path()
	}
	if s.in != nil {
		routes := []routeInfo{}
		for _, r := range s.in.Routes() {
			routes = append(routes, routeInfo{r.Method, r.Path, r.Pos.Line})
		}
		out["routes"] = routes
		pages := []routeInfo{}
		for _, p := range s.in.Pages() {
			pages = append(pages, routeInfo{"GET", p.Path, p.Pos.Line})
		}
		out["pages"] = pages
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
	fmt.Fprintf(os.Stderr, "ضاد يعمل على http://localhost:%d — البيانات في %s\nأدوات المطوّر: http://localhost:%d/_dad\nيُعاد التحميل عند حفظ الملف. للإيقاف: Ctrl+C\n",
		*port, dataPath(file), *port)
	if err := http.Serve(ln, s); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// relative shows a problem the way the page shows it: the file name only for
// imported files.
func (s *devServer) relative(d module.Diag) string {
	if filepath.Clean(d.Path) == filepath.Clean(s.path) {
		return d.Diagnostic.String()
	}
	return filepath.Base(d.Path) + ": " + d.Diagnostic.String()
}
