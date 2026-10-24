//go:build linux

package gtk

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Without GTK the package must load, and the first use after init must panic with the load error.
func TestNoGTK(t *testing.T) {
	if os.Getenv("GOWT_NOGTK_CHILD") == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestNoGTK$")
		cmd.Env = append(os.Environ(), "GOWT_NOGTK_CHILD=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("child: %v\n%s", err, out)
		}
		return
	}
	libNames = []string{"libgowt-no-such-lib.so.0"}
	if !missing() || OSG_malloc(8) != 0 {
		t.Fatal("before MarkInitDone a missing GTK must give zero values")
	}
	MarkInitDone()
	defer func() {
		e, ok := recover().(error)
		if !ok || !strings.Contains(e.Error(), "GTK 3 is not installed") {
			t.Fatalf("panic %v, want the GTK 3 is not installed error", e)
		}
	}()
	OSG_malloc(8)
	t.Fatal("no panic")
}
