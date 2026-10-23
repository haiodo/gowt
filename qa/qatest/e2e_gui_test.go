package qatest_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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

func state(t *testing.T, app *qatest.Client) (m struct {
	Off   int
	Name  string
	Agree bool
	Pick  int
}) {
	t.Helper()
	if err := json.Unmarshal(app.State("form"), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestFindAndClick(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	if n := app.Find(qa.ID("count")).Node(); n.Class != "Label" || n.Text != "clicks: 0" {
		t.Fatalf("count: %+v", n)
	}
	if app.Find(qa.ID("no-such-id")).Exists() {
		t.Error("a missing id exists")
	}
	app.Find(qa.ID("click")).Click()
	if got := string(app.State("get")); got != "1" {
		t.Fatalf("one click ran the handler %s times", got)
	}
	if n := app.Find(qa.ID("count")).Node(); n.Text != "clicks: 1" {
		t.Errorf("label after click: %q", n.Text)
	}
	if n := app.Find(qa.Text("Click")).Node(); n.ID != "click" {
		t.Errorf("by text: %+v", n)
	}
	if n := app.Find(qa.Text("~gre")).Node(); n.ID != "agree" {
		t.Errorf("by substring: %+v", n)
	}
}

func TestInputs(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.ID("name")).Type("Ada")
	app.Find(qa.ID("agree")).Check(true)
	app.Find(qa.ID("list")).Select(2)
	if m := state(t, app); m.Name != "Ada" || !m.Agree || m.Pick != 2 {
		t.Errorf("form: %+v", m)
	}
	if n := app.Find(qa.ID("name")).Node(); n.Value != "Ada" {
		t.Errorf("name node: %+v", n)
	}
}

func TestRefusedAndStrict(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	_, err := app.Call("act", map[string]any{"selector": qa.ID("off"), "action": "click"})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Errorf("click on a disabled button: %v", err)
	}
	if m := state(t, app); m.Off != 0 {
		t.Errorf("the disabled handler ran %d times", m.Off)
	}
	_, err = app.Call("act", map[string]any{"selector": qa.Text("Twin"), "action": "click"})
	if err == nil || !strings.Contains(err.Error(), "2 controls match") {
		t.Errorf("click on two matches: %v", err)
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

func TestJobSettles(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	start := time.Now()
	app.Find(qa.ID("load")).Click()
	// No sleep: the click settled, so the 1.5 s job and its Then are done.
	if n := app.Find(qa.ID("report")).Node(); n.Text != "report ready" {
		t.Fatalf("report label right after the click: %q", n.Text)
	}
	if d := time.Since(start); d < 1400*time.Millisecond {
		t.Errorf("click returned after %v, before the job ended", d)
	}
	jobs := app.Jobs()
	if len(jobs) != 1 || jobs[0].Name != "poll" || !jobs[0].Background {
		t.Errorf("jobs after settle: %+v, want only the background poll", jobs)
	}
}

func TestSettleTimeoutNamesJob(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	_, err := app.Call("act", map[string]any{"selector": qa.ID("load"), "action": "click", "timeout_ms": 300})
	if err == nil || !strings.Contains(err.Error(), `waiting: job "Download report"`) {
		t.Errorf("settle timeout: %v", err)
	}
	app.Settle() // the job outlives this test otherwise
}
