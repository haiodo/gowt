package qatest_test

import (
	"image"
	_ "image/png"
	"os"
	"runtime"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

func pngSize(t *testing.T, path string) (int, int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	return c.Width, c.Height
}

// A shot takes the window that holds the element, the window with a title, or the one at an index.
func TestShotOfOtherWindow(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.ID("popup")).Click()
	defer app.Find(qa.Selector{Class: "Shell", Text: "Popup"}).Close()
	w0, h0 := pngSize(t, app.ShotTitle("first", "QA fixture"))
	w1, h1 := pngSize(t, app.ShotOf("of-popup-field", qa.ID("pname")))
	w2, h2 := pngSize(t, app.ShotTitle("by-title", "Popup"))
	idx := -1 // the dump lists the windows in the toolkit's order, which differs by OS
	for _, n := range app.Dump() {
		if n.Class == "Popover" {
			continue
		}
		idx++
		if n.Text == "Popup" {
			break
		}
	}
	w3, h3 := pngSize(t, app.ShotWindow("by-index", idx))
	w4, h4 := pngSize(t, app.ShotOf("of-button", qa.ID("click")))
	if w1 >= w0 || h1 >= h0 {
		t.Errorf("the shot of the second window is %dx%d, the first %dx%d", w1, h1, w0, h0)
	}
	// Windows lists its windows in stacking order, which each capture changes: no index check there.
	if w2 != w1 || h2 != h1 || runtime.GOOS != "windows" && (w3 != w1 || h3 != h1) {
		t.Errorf("title %dx%d, index %dx%d, element %dx%d", w2, h2, w3, h3, w1, h1)
	}
	if w4 != w0 || h4 != h0 {
		t.Errorf("shot of a control of the first window is %dx%d, want %dx%d", w4, h4, w0, h0)
	}
	_, err := app.Call("shot", map[string]any{"title": "Nope"})
	wantErr(t, err, "no window titled")
}

// freshApp starts a new app process: on macOS a popover opened a second time in an inactive process
// never gets its window, so every test that opens one gets its own.
func freshApp(t *testing.T) *qatest.Client {
	t.Helper()
	qatest.RestartPerTest = true
	defer func() { qatest.RestartPerTest = false }()
	return qatest.App(t)
}

func openPopover(t *testing.T, app *qatest.Client) *qatest.Element {
	t.Helper()
	app.Find(qa.ID("pop")).Click()
	pop := app.Find(qa.Class("Popover"))
	if !pop.Exists() {
		t.Fatal("the popover did not open")
	}
	return pop
}

func wantClosed(t *testing.T, app *qatest.Client, pop *qatest.Element, how string) {
	t.Helper()
	if pop.Exists() || !hasEvent(t, app, "pop closed") {
		t.Fatalf("%s did not close the popover: %q", how, events(t, app))
	}
}

// Esc reaches the open popover, whatever has the focus.
func TestPopoverEscape(t *testing.T) {
	guiOnly(t)
	app := freshApp(t)
	pop := openPopover(t, app)
	app.Key("Esc")
	wantClosed(t, app, pop, "app.Key(Esc)")
}

func TestPopoverEscapeOnControl(t *testing.T) {
	guiOnly(t)
	app := freshApp(t)
	pop := openPopover(t, app)
	app.Find(qa.ID("pop-ok")).Key("Esc")
	wantClosed(t, app, pop, "Esc on a control of the popover")
}

// Close on the root of a popover closes it like a click outside.
func TestPopoverCloseRoot(t *testing.T) {
	guiOnly(t)
	app := freshApp(t)
	pop := openPopover(t, app)
	pop.Close()
	wantClosed(t, app, pop, "Close")
}
