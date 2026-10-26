package gowt

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestOpenPath(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "a b.txt")
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{abs, abs, true},
		{"file://" + filepath.ToSlash(filepath.Join(filepath.Dir(abs), "a%20b.txt")), abs, runtime.GOOS != "windows"},
		{"file://localhost" + filepath.ToSlash(abs), abs, runtime.GOOS != "windows"},
		{"file://otherhost/x", "", false},
		{"https://example.org/a", "", false},
		{"tenniarb://open?x=1", "", false},
		{"", "", false},
	} {
		got, ok := openPath(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("openPath(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
	if p, ok := openPath("rel/x.txt"); !ok || !filepath.IsAbs(p) {
		t.Errorf("relative path became %q, %v", p, ok)
	}
}

func TestOpenPathWindowsForms(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("drive letters")
	}
	for in, want := range map[string]string{`C:\Users\a\x.txt`: `C:\Users\a\x.txt`, "file:///C:/Users/a/x%20y.txt": `C:\Users\a\x y.txt`} {
		if got, ok := openPath(in); !ok || got != want {
			t.Errorf("openPath(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}

func TestArgFiles(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.tnb")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got := argFiles([]string{"-psn_0_123", "--flag", filepath.Join(dir, "missing"), f, f, "http://x/y"})
	if !reflect.DeepEqual(got, []string{f}) {
		t.Errorf("argFiles = %v, want [%s]", got, f)
	}
}

func TestOpenBoxQueuesBeforeHandler(t *testing.T) {
	var b openBox
	b.deliver([]string{"/a", "/b"})
	b.deliver([]string{"/b", "/c"})
	b.deliver(nil)
	if want := []string{"/a", "/b", "/c"}; !reflect.DeepEqual(b.queue, want) {
		t.Fatalf("queue = %v, want %v", b.queue, want)
	}
	var one, two [][]string
	b.handlers = append(b.handlers, func(p []string) { one = append(one, p) }, func(p []string) { two = append(two, p) })
	b.deliver([]string{"/d"})
	if !reflect.DeepEqual(one, [][]string{{"/d"}}) || !reflect.DeepEqual(two, one) {
		t.Errorf("handlers got %v and %v", one, two)
	}
}

func TestOpenPathsDedupe(t *testing.T) {
	a, _ := filepath.Abs("/x/a")
	b, _ := filepath.Abs("/x/b")
	got := openPaths([]string{"/x/a", "/x/a", "/x/b", "ftp://h/c"})
	if want := []string{a, b}; !reflect.DeepEqual(got, want) {
		t.Errorf("openPaths = %v, want %v", got, want)
	}
}

func TestSingleInstanceBadID(t *testing.T) {
	for _, id := range []string{"", "app", "a..b", "1a.b", "a.b c", "a/b.c"} {
		if first, err := SingleInstance(id); !first || err == nil {
			t.Errorf("SingleInstance(%q) = %v, %v; want true and an error", id, first, err)
		}
	}
}

// TestOpenChild is the second process of TestSingleInstanceForwards.
func TestOpenChild(t *testing.T) {
	id := os.Getenv("GOWT_OPEN_CHILD")
	if id == "" {
		t.Skip("helper process")
	}
	first, err := SingleInstance(id)
	switch {
	case err != nil:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(5)
	case first:
		os.Exit(4)
	}
	os.Exit(3)
}

func runChild(t *testing.T, id string, args ...string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe,append([]string{"-test.run=^TestOpenChild$"}, args...)...)
	cmd.Env = append(os.Environ(), "GOWT_OPEN_CHILD="+id)
	out, err := cmd.CombinedOutput()
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 3 {
		t.Fatalf("second instance: %v, want exit code 3 (forwarded)\n%s", err, out)
	}
}

// With GOWT_GUI_TEST the forwarded files also reach an OnOpenFiles handler on the UI thread.
func TestSingleInstanceForwards(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS: Launch Services keeps one instance")
	}
	id := fmt.Sprintf("org.gowt.Test%d", os.Getpid())
	first, err := SingleInstance(id)
	if err != nil {
		t.Skip("no single instance here: ", err)
	}
	if !first {
		t.Fatal("first call is not the first instance")
	}
	a, b := filepath.Join(t.TempDir(), "a.tnb"), filepath.Join(t.TempDir(), "b.tnb")
	for _, f := range []string{a, b} {
		if err := os.WriteFile(f, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runChild(t, id, a, b, a)
	runChild(t, id)
	want := []string{a, b}
	if os.Getenv("GOWT_GUI_TEST") == "" {
		for i := 0; i < 100; i++ {
			inbox.Lock()
			got, n := openPaths(inbox.backlog), len(inbox.backlog)
			inbox.Unlock()
			if reflect.DeepEqual(got, want) && n >= 2 {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("backlog = %v", inbox.backlog)
	}
	var got []string
	onUI := false
	onMain(func() {
		err := Run(func(app *App) {
			app.OnOpenFiles(func(p []string) {
				got, onUI = p, true
				app.Quit()
			})
			app.Window("open").Show()
			app.After(10*time.Second, app.Quit)
		})
		if err != nil {
			t.Error(err)
		}
	})
	if !onUI || !reflect.DeepEqual(got, want) {
		t.Errorf("handler got %v (called: %v), want %v", got, onUI, want)
	}
}
