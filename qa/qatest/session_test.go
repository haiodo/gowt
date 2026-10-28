package qatest

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// A long TMPDIR must not push the socket path over the unix limit.
func TestShortDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows binds a loopback port when the unix socket fails")
	}
	long := filepath.Join(t.TempDir(), strings.Repeat("d", 120))
	if err := os.MkdirAll(long, 0o700); err != nil {
		t.Skip(err)
	}
	t.Setenv("TMPDIR", long)
	t.Setenv("XDG_RUNTIME_DIR", long)
	d, err := shortDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(d)
	if p := filepath.Join(d, "s"); len(p) > maxSockPath {
		t.Errorf("%q is %d bytes", p, len(p))
	}
}

// A connection the app dropped fails the call with the end of the app's log.
func TestLostConnectionShowsLog(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.log"), []byte("panic: boom\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app, conn := net.Pipe()
	app.Close()
	s := &session{conn: conn, dec: json.NewDecoder(conn), exited: make(chan struct{}), dir: dir}
	close(s.exited)
	_, err := s.call("act", nil, time.Second)
	if err == nil || !strings.Contains(err.Error(), "connection to the app lost") || !strings.Contains(err.Error(), "panic: boom") {
		t.Errorf("err = %v, want the lost connection and the app log", err)
	}
}
