package qatest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
