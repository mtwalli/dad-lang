//go:build !windows

package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// «dad serve» takes its port from $PORT, keeps the data where --data says,
// and stops cleanly on SIGTERM.
func TestServeStopsOnSIGTERM(t *testing.T) {
	dir := t.TempDir()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	app := filepath.Join(dir, "app.dad")
	os.WriteFile(app, []byte(devSrc), 0o644)
	data := filepath.Join(dir, "data", "app.db")
	os.MkdirAll(filepath.Dir(data), 0o755)
	t.Setenv("PORT", strconv.Itoa(port))
	exit := make(chan int)
	go func() { exit <- cmdServe([]string{app, "--host", "127.0.0.1", "--data", data}) }()
	var ok bool
	for i := 0; i < 100 && !ok; i++ {
		time.Sleep(30 * time.Millisecond)
		if res, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/_dad/health"); err == nil {
			res.Body.Close()
			ok = res.StatusCode == 200
		}
	}
	if !ok {
		t.Fatal("the server did not start")
	}
	if _, err := os.Stat(data); err != nil {
		t.Errorf("the database is not where --data says: %v", err)
	}
	syscall.Kill(os.Getpid(), syscall.SIGTERM)
	select {
	case code := <-exit:
		if code != 0 {
			t.Errorf("exit %d after SIGTERM", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the server did not stop")
	}
}
