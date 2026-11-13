//go:build gowtqa

package main_test

// GUI checks of the chatmock demo. Build the app first:
//
//	go build -tags gowtqa -o .work/ui-mockup/ui-builder/chatmock ./examples/chatmock
//	CHATMOCK_BIN=.work/ui-mockup/ui-builder/chatmock GOWT_GUI_TEST=1 \
//	  GOWT_QA_ARTIFACTS=.work/ui-mockup/ui-builder go test -tags gowtqa ./examples/chatmock

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

func TestMain(m *testing.M) {
	// A fresh app per test: the shot of one test must not carry another test's messages.
	qatest.RestartPerTest = true
	qatest.Main(m, os.Getenv("CHATMOCK_BIN"))
}

func TestChannelSwitch(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.ID("ch-mobile")).Click()
	if n := app.Find(qa.ID("header")).Node(); n.Text != "# mobile" {
		t.Errorf("header after mobile: %q, want # mobile", n.Text)
	}
	app.Find(qa.ID("ch-design")).Click()
	if n := app.Find(qa.ID("header")).Node(); n.Text != "# design" {
		t.Errorf("header after design: %q, want # design", n.Text)
	}
	app.Find(qa.ID("ch-jordan")).Click()
	if n := app.Find(qa.ID("header")).Node(); n.Text != "Jordan Brooks" {
		t.Errorf("header after dm: %q, want Jordan Brooks", n.Text)
	}
	app.Find(qa.ID("ch-flight-path")).Click()
	if n := app.Find(qa.ID("header")).Node(); n.Text != "# flight-path" {
		t.Errorf("header after flight-path: %q, want # flight-path", n.Text)
	}
}

func TestSend(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.ID("composer")).Type("hello from qa")
	app.Find(qa.ID("send")).Click()
	if !app.Find(qa.Text("hello from qa")).Exists() {
		t.Error("the sent message is not in the feed")
	}
	n := len(app.Find(qa.Text("hello from qa")).Nodes())
	if n != 1 {
		t.Errorf("sent message found %d times, want 1", n)
	}
}

// TestBottomZonesVisible pins the layout contract of the acceptance: the composer, the
// Honey status and the sidebar user block are fully inside the window, at the start size
// and at the min size.
func TestBottomZonesVisible(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	check := func() {
		shell := app.Find(qa.Class("Shell")).Node()
		h := shell.Bounds[3]
		for _, sel := range []qa.Selector{qa.ID("composer"), qa.ID("send"), qa.ID("honey"), qa.ID("user")} {
			n := app.Find(sel).Node()
			if bottom := n.Bounds[1] + n.Bounds[3]; bottom > h || n.Bounds[1] < 0 {
				t.Errorf("%s bottom = %d, outside the window height %d", sel.ID, bottom, h)
			}
			if !n.Visible {
				t.Errorf("%s is not visible", sel.ID)
			}
		}
	}
	check()
	// The feed scrolls, the bottom zones do not move.
	app.Find(qa.ID("feed")).ScrollBy(0, 100000)
	check()

	// Geometry of the acceptance: the sidebar is flush with the left, top and bottom
	// edges of the client area; the white card floats inside the content with an inset.
	shell := app.Find(qa.Class("Shell")).Node()
	side := app.Find(qa.ID("sidebar")).Node()
	card := app.Find(qa.ID("card")).Node()
	search := app.Find(qa.ID("search")).Node()
	var inset int
	if err := json.Unmarshal(app.State("inset"), &inset); err != nil {
		t.Fatalf("inset state: %v", err)
	}
	if side.Bounds[0] != 0 {
		t.Errorf("sidebar x = %d, want 0 (flush left)", side.Bounds[0])
	}
	// The material is full height: it runs under the traffic lights to the window top.
	if side.Bounds[1] > 2 {
		t.Errorf("sidebar y = %d, want the top edge", side.Bounds[1])
	}
	if bottom := side.Bounds[1] + side.Bounds[3]; bottom < shell.Bounds[3]-2 {
		t.Errorf("sidebar bottom = %d, want the bottom edge %d", bottom, shell.Bounds[3])
	}
	// The search starts below the title bar strip the traffic lights sit in.
	if search.Bounds[1] < inset-2 {
		t.Errorf("search top = %d, want it below the title bar strip (%d)", search.Bounds[1], inset)
	}
	if card.Bounds[0] < side.Bounds[0]+side.Bounds[2] {
		t.Errorf("card x = %d, want it right of the sidebar", card.Bounds[0])
	}
	if card.Bounds[1] < inset-2 {
		t.Errorf("card top = %d, want it below the title bar strip (%d)", card.Bounds[1], inset)
	}
	if right := card.Bounds[0] + card.Bounds[2]; right > shell.Bounds[2]-2 {
		t.Errorf("card right = %d, want an inset from the window edge %d", right, shell.Bounds[2])
	}
	if bottom := card.Bounds[1] + card.Bounds[3]; bottom > shell.Bounds[3]-2 {
		t.Errorf("card bottom = %d, want an inset from the bottom edge %d", bottom, shell.Bounds[3])
	}
}

// TestSidebarResize pins the row width contract: widening the sidebar with the splitter
// (driven programmatically through the "sidebar-width" handler) must stretch the channel
// rows with it, not leave them at their start width.
func TestSidebarResize(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	bounds := func(id string) [4]int { return app.Find(qa.ID(id)).Node().Bounds }
	dump := func(tag string) {
		for _, id := range []string{"sidebar", "rows-scroll", "rows", "ch-flight-path", "search"} {
			t.Logf("%s %s: %v", tag, id, bounds(id))
		}
	}
	dump("before")
	rowBefore := bounds("ch-flight-path")
	app.State("sidebar-width", "width", 340)
	dump("after")
	rowAfter := bounds("ch-flight-path")
	sideAfter := bounds("sidebar")
	if rowAfter[2] <= rowBefore[2] {
		t.Errorf("row width = %d after widening the sidebar to 340, want it above %d", rowAfter[2], rowBefore[2])
	}
	if rowAfter[2] > sideAfter[2] {
		t.Errorf("row width = %d, wider than the sidebar %d", rowAfter[2], sideAfter[2])
	}
}

// TestNoOptionErrors fails on any option the facade rejected during window construction
// (App.OnError collects them); a misplaced option is a startup bug, not a warning.
func TestNoOptionErrors(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	var errs []string
	if err := json.Unmarshal(app.State("errors"), &errs); err != nil {
		t.Fatalf("errors state: %v", err)
	}
	if len(errs) > 0 {
		t.Errorf("options rejected by the facade: %q", errs)
	}
}

func TestShot(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Shot("final")
}

func guiOnly(t *testing.T) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
}
