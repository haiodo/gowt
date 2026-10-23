package qatest_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

// TestMain builds the fixture with -tags gowtqa, or takes a prebuilt one from GOWT_QA_APP (a machine
// without Go); the window tests need GOWT_GUI_TEST=1 and a display.
func TestMain(m *testing.M) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		os.Exit(m.Run())
	}
	if bin := os.Getenv("GOWT_QA_APP"); bin != "" {
		qatest.Main(m, bin)
	}
	dir, err := os.MkdirTemp("", "qafixture")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	bin := filepath.Join(dir, "app")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-tags", "gowtqa", "-o", bin, "./testdata/app").CombinedOutput(); err != nil {
		println(string(out))
		os.Exit(1)
	}
	qatest.Main(m, bin)
}

func guiOnly(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
}

func TestStateAndSettle(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	if got := string(app.State("get")); got != "0" {
		t.Fatalf("after reset: %s", got)
	}
	app.State("add", "n", 3)
	app.Settle()
	if got := string(app.State("get")); got != "3" {
		t.Fatalf("after add: %s", got)
	}
}

func TestResetBetweenTests(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	if got := string(app.State("get")); got != "0" {
		t.Fatalf("reset did not run, clicks: %s", got)
	}
}

func TestShot(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Settle()
	b, err := os.ReadFile(app.Shot("fixture"))
	if err != nil || !strings.HasPrefix(string(b), "\x89PNG") {
		t.Fatalf("not a PNG: %v, %d bytes", err, len(b))
	}
}

func TestErrors(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	for _, c := range []struct{ cmd, name, want string }{
		{"state", "nope", `no handler "nope"`},
		{"state", "fail", "asked to fail"},
		{"state", "panic", "asked to panic"},
	} {
		_, err := app.Call(c.cmd, map[string]any{"name": c.name})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s %s: got %v, want %q", c.cmd, c.name, err, c.want)
		}
	}
	if _, err := app.Call("nope", nil); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("unknown command: %v", err)
	}
	// Tree commands may be unsupported or answer for real; either way they answer, and the app lives on.
	for _, cmd := range []string{"dump", "windows"} {
		if _, err := app.Call(cmd, nil); err != nil && !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("%s: %v", cmd, err)
		}
	}
	if _, err := app.Call("find", qa.ID("no-such-id")); err != nil && !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("find: %v", err)
	}
	_, err := app.Call("act", map[string]any{"selector": qa.ID("no-such-id"), "action": "click"})
	if err == nil || strings.Contains(err.Error(), "panic") {
		t.Errorf("act on a missing control: %v", err)
	}
	var v any
	r, err := app.Call("ping", nil)
	if err != nil || json.Unmarshal(r, &v) != nil {
		t.Errorf("ping after errors: %v", err)
	}
}
