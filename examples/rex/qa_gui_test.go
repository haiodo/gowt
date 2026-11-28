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

func TestSidebarToolbarSlotWidths(t *testing.T) {
	for _, tc := range []struct {
		width, viewer, slots int
		plus                 bool
		min, gap, right      int
	}{
		{300, 36, 3, true, 200, 16, 284},
		{300, 44, 2, false, 216, 16, 284},
		{280, 36, 3, false, 200, 8, 272},
		{280, 44, 2, false, 216, 16, 264},
		{220, 36, 1, false, 200, 16, 204},
		{220, 44, 1, false, 216, 4, 216},
		{216, 36, 1, false, 200, 16, 200},
		{216, 44, 1, false, 216, 0, 216},
		{200, 36, 1, false, 200, 0, 200},
		{200, 44, 1, false, 216, 0, 216},
	} {
		if got := sidebarMinWidth(tc.viewer); got != tc.min {
			t.Errorf("viewer %d: minimum %d, want %d", tc.viewer, got, tc.min)
		}
		effective := max(tc.width, tc.min)
		if got := sidebarToolbarSlots(effective, tc.viewer); got != tc.slots {
			t.Errorf("width %d viewer %d: slots %d, want %d", tc.width, tc.viewer, got, tc.slots)
		}
		if got := sidebarPlusFits(effective, tc.viewer); got != tc.plus {
			t.Errorf("width %d viewer %d: plus %v, want %v", tc.width, tc.viewer, got, tc.plus)
		}
		if got := sidebarTrailingGap(tc.width, tc.viewer); got != tc.gap {
			t.Errorf("width %d viewer %d: gap %d, want %d", tc.width, tc.viewer, got, tc.gap)
		}
		if right := effective - sidebarTrailingGap(tc.width, tc.viewer); right != tc.right {
			t.Errorf("width %d viewer %d: More right %d, want %d", tc.width, tc.viewer, right, tc.right)
		}
		if right := effective - sidebarTrailingGap(tc.width, tc.viewer); right > effective+2 {
			t.Errorf("width %d viewer %d: More right %d exceeds edge %d", tc.width, tc.viewer, right, effective)
		}
	}
	if got := sidebarToolbarSlots(33, 36); got != 0 {
		t.Errorf("hide animation below the minimum keeps %d slots, want 0", got)
	}
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

	divider := 0
	for _, width := range []int{300, 280, 260, 220, 200} {
		app.State("sidebar-width", "width", width)
		app.Settle()
		app.Shot(fmt.Sprintf("rex-width-%d", width))
		checkToolbar(t, app, width, &divider)
	}

	app.State("sidebar-width", "width", 100)
	app.Settle()
	app.Shot("rex-width-min")
	checkToolbar(t, app, 200, &divider)

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

func checkToolbar(t *testing.T, app *qatest.Client, width int, divider *int) {
	t.Helper()
	edge := app.Find(qa.ID("rex-content")).Node().Bounds[0]
	raw := app.State("sidebar-metrics")
	actualWidth := 0
	if err := json.Unmarshal(raw, &actualWidth); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" {
		if edge != actualWidth {
			t.Errorf("width %d: content starts at %d, sidebar width %d", width, edge, actualWidth)
		}
	} else {
		if *divider == 0 {
			*divider = edge - actualWidth
		}
		if want := actualWidth + *divider; edge != want {
			t.Errorf("width %d: content starts at %d, want sidebar %d plus divider %d", width, edge, actualWidth, *divider)
		}
	}
	want := []string{"toggle-sidebar", "more"}
	if runtime.GOOS == "darwin" {
		viewer := app.Find(qa.ID("toggle-sidebar")).Node().Bounds[2]
		slots := sidebarToolbarSlots(actualWidth, viewer)
		want = []string{"toggle-sidebar"}
		switch slots {
		case 3:
			want = append(want, "toolbar-eye", "toolbar-command")
			if sidebarPlusFits(actualWidth, viewer) {
				want = append(want, "toolbar-plus")
			} else {
				want = append(want, "more")
			}
		case 2:
			want = append(want, "toolbar-eye", "more")
		case 1:
			want = append(want, "more")
		}
	}
	for _, id := range []string{"toggle-sidebar", "toolbar-eye", "toolbar-command", "toolbar-plus", "more"} {
		e := app.Find(qa.ID(id))
		has := e.Exists()
		if runtime.GOOS == "darwin" && has != contains(want, id) {
			t.Errorf("width %d: %s present %v, want %v", width, id, has, contains(want, id))
		}
		if has {
			b := e.Node().Bounds
			if right := b[0] + b[2]; right > edge+2 {
				t.Errorf("width %d: %s right %d, sidebar edge %d", width, id, right, edge)
			}
		}
	}
	if runtime.GOOS != "darwin" {
		return
	}
	viewer := app.Find(qa.ID("toggle-sidebar")).Node().Bounds[2]
	wantGap := sidebarTrailingGap(actualWidth, viewer)
	last := 0
	for _, id := range want {
		if id == "toggle-sidebar" {
			continue
		}
		b := app.Find(qa.ID(id)).Node().Bounds
		last = max(last, b[0]+b[2])
	}
	if last != 0 && edge-last != wantGap {
		t.Errorf("width %d: the toolbar cluster ends %d points from the sidebar edge, want %d", width, edge-last, wantGap)
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
