package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
	"github.com/haiodo/gowt/swt"
)

// TestMain builds the app with the QA server (-tags gowtqa); opening windows needs GOWT_GUI_TEST=1.
func TestMain(m *testing.M) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		os.Exit(m.Run())
	}
	if bin := os.Getenv("GOWT_QA_APP"); bin != "" {
		qatest.Main(m, bin)
	}
	dir, err := os.MkdirTemp("", "qarex")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	bin := filepath.Join(dir, "app")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-tags", "gowtqa", "-o", bin, ".").CombinedOutput(); err != nil {
		println(string(out))
		os.Exit(1)
	}
	qatest.Main(m, bin)
}

// TestSidebarStates checks the staged toolbar overflow, the hidden state and the narrow right pane.
func TestSidebarStates(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	app := qatest.App(t)
	app.Settle()
	expanded := app.Shot("rex-expanded")
	if !app.Find(qa.Text("~Show Activity")).Exists() {
		t.Fatalf("the expanded toolbar has no Activity item")
	}

	for _, c := range []struct {
		width int
		want  []string
	}{
		{300, []string{"toggle-sidebar", "toolbar-eye", "toolbar-command", "toolbar-plus"}},
		{280, []string{"toggle-sidebar", "toolbar-eye", "toolbar-command", "more"}},
		{260, []string{"toggle-sidebar", "toolbar-eye", "more"}},
		{220, []string{"toggle-sidebar", "more"}},
		{200, []string{"toggle-sidebar", "more"}},
	} {
		app.State("sidebar-width", "width", c.width)
		app.Settle()
		app.Shot(fmt.Sprintf("rex-width-%d", c.width))
		checkToolbar(t, app, c.width, c.want)
	}

	app.State("sidebar-width", "width", 100)
	app.Settle()
	app.Shot("rex-width-min")
	checkToolbar(t, app, 200, []string{"toggle-sidebar", "more"})

	app.State("sidebar-width", "width", 260)
	app.Settle()
	app.State("split", "left", 80, "right", 20)
	app.Settle()
	app.Shot("rex-right-narrow")
	checkRightPane(t, app)
	app.State("split", "left", 50, "right", 50)
	app.Settle()
	app.Shot("rex-right-normal")
	checkRightPane(t, app)

	app.State("toggle-sidebar")
	time.Sleep(40 * time.Millisecond)
	mid := app.Shot("rex-mid")
	checkMoreInside(t, app)
	time.Sleep(300 * time.Millisecond)
	app.Settle()
	collapsed := app.Shot("rex-collapsed")
	if app.Find(qa.Text("~Show Activity")).Exists() {
		t.Errorf("the collapsed toolbar still has the wide items")
	}
	for _, id := range []string{"more", "toolbar-eye", "toolbar-command", "toolbar-plus"} {
		if app.Find(qa.ID(id)).Exists() {
			t.Errorf("the hidden sidebar keeps %s", id)
		}
	}
	if !app.Find(qa.ID("toggle-sidebar")).Exists() {
		t.Errorf("the hidden sidebar has no toggle")
	}
	t.Logf("shots: %s, %s, %s", expanded, mid, collapsed)
}

func checkToolbar(t *testing.T, app *qatest.Client, width int, want []string) {
	t.Helper()
	edge := app.Find(qa.ID("rex-content")).Node().Bounds[0]
	if edge != width {
		t.Errorf("width %d: content starts at %d", width, edge)
	}
	for _, id := range []string{"toolbar-eye", "toolbar-command", "toolbar-plus", "more"} {
		has := app.Find(qa.ID(id)).Exists()
		wanted := contains(want, id)
		if has != wanted {
			t.Errorf("width %d: %s present %v, want %v", width, id, has, wanted)
		}
	}
	last := 0
	for _, id := range want {
		b := app.Find(qa.ID(id)).Node().Bounds
		right := b[0] + b[2]
		if right > edge+2 {
			t.Errorf("width %d: %s right %d, sidebar edge %d", width, id, right, edge)
		}
		if id != "toggle-sidebar" {
			last = max(last, right)
		}
	}
	if last != 0 && (edge-last < 0 || edge-last > 8) {
		t.Errorf("width %d: the toolbar cluster ends %d points from the sidebar edge, want 0..8", width, edge-last)
	}
}

func checkMoreInside(t *testing.T, app *qatest.Client) {
	t.Helper()
	edge := app.Find(qa.ID("rex-content")).Node().Bounds[0]
	if e := app.Find(qa.ID("more")); e.Exists() {
		b := e.Node().Bounds
		if right := b[0] + b[2]; right > edge+2 {
			t.Errorf("mid-animation: More right %d, sidebar edge %d", right, edge)
		}
	}
}

func checkRightPane(t *testing.T, app *qatest.Client) {
	t.Helper()
	pane := app.Find(qa.ID("shell-pane")).Node().Bounds
	right := pane[0] + pane[2]
	last := 0
	for _, id := range []string{"shell-split", "open-settings-right", "shell-close"} {
		e := app.Find(qa.ID(id))
		if !e.Exists() {
			t.Errorf("right pane: %s is missing", id)
			continue
		}
		n := e.Node()
		b := n.Bounds
		if !n.Visible || b[2] <= 0 || b[3] <= 0 || b[0] < pane[0] || b[0]+b[2] > right {
			t.Errorf("right pane %v: %s at %v leaves the pane", pane, id, b)
		}
		if b[0] < last {
			t.Errorf("right pane: %s at %v crosses the previous button", id, b)
		}
		last = b[0] + b[2]
	}
	if gap := right - last; gap < 0 || gap > 14 {
		t.Errorf("right pane %v: the last button ends %d points from the edge", pane, gap)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

type sashState struct {
	X     int `json:"x"`
	Inset int `json:"inset"`
}

func sendSash(t *testing.T, app *qatest.Client, event int32, x int) sashState {
	t.Helper()
	raw := app.State("sidebar-sash", "event", event, "x", x)
	var state sashState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

// TestSidebarDragTitle delivers the same MouseDown, Selection and MouseUp events as a sidebar sash.
// While visible, the title stays at the card edge and therefore follows every drag step in that frame;
// releasing the button changes neither its inset nor its screen position.
func TestSidebarDragTitle(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	app := qatest.App(t)
	app.Settle()
	down := sendSash(t, app, swt.MouseDown, 0)
	app.Shot("drag-before")
	tabX := func() int { return app.Find(qa.ID("codex-tab")).Node().Bounds[0] }
	previousCard := tabX()
	previousTitle := previousCard + 6 + down.Inset
	last := down
	for _, step := range []struct{ delta, move int }{{-20, -20}, {-40, -20}} {
		delta := step.delta
		last = sendSash(t, app, swt.Selection, down.X+delta)
		card := tabX()
		title := card + 6 + last.Inset
		t.Logf("drag %d: sash %d, card %d, inset %d, title %d", delta, last.X, card, last.Inset, title)
		if card-previousCard != step.move {
			t.Errorf("drag %d: card moved by %d, want %d", delta, card-previousCard, step.move)
		}
		if last.Inset != 0 {
			t.Errorf("drag %d: title inset %d, want 0 while the sidebar is visible", delta, last.Inset)
		}
		if title != card+6 {
			t.Errorf("drag %d: title %d, card edge title %d", delta, title, card+6)
		}
		app.Shot(fmt.Sprintf("drag-%d", -delta))
		previousCard, previousTitle = card, title
	}
	up := sendSash(t, app, swt.MouseUp, down.X-40)
	upTitle := tabX() + 6 + up.Inset
	app.Shot("drag-after")
	if up.Inset != last.Inset || upTitle != previousTitle {
		t.Errorf("mouse up jumped the title: before inset %d title %d, after inset %d title %d", last.Inset, previousTitle, up.Inset, upTitle)
	}
}

// cardShown reports whether the control with id is on screen.
func cardShown(app *qatest.Client, id string) bool {
	for _, n := range app.Find(qa.ID(id)).Nodes() {
		if n.Visible {
			return true
		}
	}
	return false
}

// Up and Down walk the settings categories: the group, space and caption rows are skipped, the page
// and the selection follow, and the ends do not wrap. The list takes the focus when the window opens,
// so the arrows work without a click first.
func TestSettingsUpDown(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	app := qatest.App(t)
	app.Settle()
	app.Find(qa.ID("open-settings-sidebar")).Click()
	app.Settle()
	list := app.Find(qa.ID("settings-list"))
	if !list.Node().Focused {
		t.Fatalf("the settings list has no focus after the window opened")
	}
	for _, c := range []struct{ keys, want string }{
		{"Down", "Remote Hosts"},
		{"Down", "Interface"},
		{"Up", "Remote Hosts"},
		{"Up", "General"},
	} {
		app.Key(c.keys)
		app.Settle()
		if !app.Find(qa.Text(c.want)).Exists() {
			t.Errorf("%s: no %q page title", c.keys, c.want)
		}
		// The General page is the only one with the Client Info card: the page itself must turn.
		if shown := cardShown(app, "client-info-card"); shown != (c.want == "General") {
			t.Errorf("%s: Client Info card shown %v, want the %q page", c.keys, shown, c.want)
		}
	}
	for range 2 {
		app.Key("Up")
		app.Settle()
	}
	app.Key("Up")
	app.Settle()
	if !app.Find(qa.Text("Server")).Exists() || cardShown(app, "client-info-card") {
		t.Error("Up above the first category wrapped away from Server")
	}
	for range 12 {
		app.Key("Down")
		app.Settle()
	}
	app.Key("Down")
	app.Settle()
	if !app.Find(qa.Text("Developer")).Exists() {
		t.Error("Down past the last category wrapped away from Developer")
	}
	if cardShown(app, "client-info-card") {
		t.Error("Down past the last category left the General page up")
	}
}
