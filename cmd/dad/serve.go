package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const serveUsage = "الاستخدام: ضاد انشر <ملف.dad> [--port 8080] [--host 0.0.0.0] [--data مسار.db]"

// cmdServe runs a program as a production web server: «dad serve app.dad».
//
// It is the server of «dad dev» without what is only for development: no
// reloading, no developer tools (they show every table's rows), and a program
// with mistakes doesn't start. It listens on every network interface, takes its
// port from $PORT as hosting platforms set it, logs each request, and stops
// cleanly on Ctrl+C or SIGTERM, closing the database.
func cmdServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	defPort := 8080
	if p, err := strconv.Atoi(os.Getenv("PORT")); err == nil && p > 0 {
		defPort = p
	}
	port := fs.Int("port", defPort, "")
	fs.IntVar(port, "منفذ", defPort, "")
	host := fs.String("host", "0.0.0.0", "")
	data := fs.String("data", "", "")
	fs.StringVar(data, "بيانات", "", "")
	var file string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		file, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, serveUsage)
		return 2
	}
	rest := fs.Args()
	if file == "" && len(rest) == 1 {
		file, rest = rest[0], nil
	}
	if file == "" || len(rest) > 0 {
		fmt.Fprintln(os.Stderr, serveUsage)
		return 2
	}

	// A program with mistakes doesn't start: there is no «last good version» to keep.
	if _, ok := loadProgram(file); !ok {
		return 1
	}
	db := *data
	if db == "" {
		db = dataPath(file)
	}
	store, err := openStoreAt(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() {
		if c, ok := store.(io.Closer); ok {
			c.Close()
		}
	}()
	s := &devServer{path: file, store: store, out: os.Stdout, production: true}
	s.load()
	if s.err != "" {
		return 1 // an error while the program starts; load has shown it
	}

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "تعذر فتح المنفذ %d: %v\n", *port, err)
		return 1
	}
	srv := &http.Server{
		Handler:           logRequests(secure(s), os.Stderr),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		fmt.Fprintln(os.Stderr, "يتوقف الخادم…")
		shut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shut)
		close(done)
	}()

	shown := *host
	if shown == "0.0.0.0" || shown == "::" || shown == "" {
		shown = "localhost"
	}
	fmt.Fprintf(os.Stderr, "ضاد ينشر «%s» على http://%s — البيانات في %s\nللإيقاف: Ctrl+C\n",
		file, net.JoinHostPort(shown, strconv.Itoa(*port)), db)
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	<-done
	return 0
}

// secure adds the headers a public site should send.
func secure(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hd := w.Header()
		hd.Set("X-Content-Type-Options", "nosniff")
		hd.Set("X-Frame-Options", "SAMEORIGIN")
		hd.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.ServeHTTP(w, r)
	})
}

// logRequests writes one line per request: time, method, path, status and duration.
func logRequests(h http.Handler, out io.Writer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		h.ServeHTTP(rec, r)
		fmt.Fprintf(out, "%s %s %s %d %s\n", start.UTC().Format(time.RFC3339), r.Method, r.URL.Path, rec.status,
			time.Since(start).Round(time.Millisecond))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
