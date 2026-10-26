package qatest_test

import (
	"image"
	_ "image/png"
	"os"
	"strings"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

func hasEvent(t *testing.T, app *qatest.Client, prefix string) bool {
	t.Helper()
	for _, e := range events(t, app) {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

// Clicks and drags reach the router of the view, a press focuses it, and Type and Key keep the focus
// of the inline editor, which commits when it loses it.
func TestGfxPointerAndEditor(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	v := app.Find(qa.ID("gfx"))

	v.ClickAt(50, 60)
	wantEvents(t, app, "gfx kind=0 button=1 count=1 at 50,60 drag=false node=true mods=false",
		"gfx kind=2 button=1 count=0 at 50,60 drag=true node=true mods=false")
	if !v.Node().Focused {
		t.Error("a press did not focus the view")
	}

	app.State("reset")
	v.DragTo(10, 10, 90, 50)
	ev := events(t, app)
	if len(ev) != 6 || !strings.Contains(ev[1], "kind=1 button=1") || !strings.Contains(ev[1], "drag=true") ||
		!strings.Contains(ev[5], "kind=2") || !strings.Contains(ev[5], "at 90,50") {
		t.Errorf("drag events %q", ev)
	}

	app.State("reset")
	v.Pointer(qa.PointerEvent{Kind: "down", X: 5, Y: 5, Button: 3, Mods: "Shift"})
	v.Pointer(qa.PointerEvent{Kind: "move", X: 6, Y: 6})
	v.Pointer(qa.PointerEvent{Kind: "up", X: 6, Y: 6, Button: 3})
	wantEvents(t, app, "gfx kind=0 button=3 count=1 at 5,5 drag=false node=true mods=true",
		"gfx kind=1 button=3 count=0 at 6,6 drag=true node=true mods=false",
		"gfx kind=2 button=3 count=0 at 6,6 drag=true node=true mods=false")

	app.State("reset")
	v.DoubleClickAt(150, 150)
	edit := app.Find(qa.ID("gfx-edit"))
	if !edit.Node().Focused {
		t.Fatal("the inline editor has no focus")
	}
	edit.Type("renamed")
	if hasEvent(t, app, "commit") || hasEvent(t, app, "cancel") {
		t.Fatalf("Type ended the edit: %q", events(t, app))
	}
	if got := edit.Node().Value; got != "renamed" {
		t.Fatalf("the field holds %q", got)
	}
	app.Key("Enter")
	if !hasEvent(t, app, "commit renamed") || edit.Exists() {
		t.Errorf("Enter did not commit: %q", events(t, app))
	}

	v.DoubleClickAt(150, 150)
	edit.Type("x")
	edit.Key("Esc")
	if !hasEvent(t, app, "cancel edit") || edit.Exists() {
		t.Errorf("Esc did not cancel: %q", events(t, app))
	}

	wantErr(t, actErr(app, qa.ID("gfx"), "pointer", map[string]string{"kind": "tap"}), "kind")
	wantErr(t, actErr(app, qa.ID("gfx"), "pointer", map[string]any{"kind": "down", "button": 7}), "button")
}

// A control without a router gets the listener events.
func TestPointerPlainControl(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	c := app.Find(qa.ID("plain"))
	c.ClickAt(10, 20)
	c.RightClickAt(11, 21)
	c.DoubleClickAt(12, 22)
	c.Pointer(qa.PointerEvent{Kind: "move", X: 3, Y: 4})
	wantEvents(t, app,
		"plain down 10,20 b1 c1 shift=false", "plain up 10,20 b1",
		"plain down 11,21 b3 c1 shift=false", "plain up 11,21 b3",
		"plain down 12,22 b1 c1 shift=false", "plain up 12,22 b1", "plain down 12,22 b1 c2 shift=false", "plain up 12,22 b1",
		"plain move 3,4")
	if !c.Node().Focused {
		t.Error("a press did not focus the canvas")
	}
}

// A context menu is in the dump; running an item shows the menu first, as a right click does.
func TestPopupMenu(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	menu := app.Find(qa.Class("Menu").In(qa.ID("menuhost")))
	if n := len(menu.Node().Children); n != 3 {
		t.Fatalf("the menu has %d items, want 3", n)
	}
	app.Find(qa.Text("Rename").In(qa.ID("menuhost"))).Click()
	app.Find(qa.Text("Wrap").In(qa.ID("menuhost"))).Click()
	app.Find(qa.Text("Deep").In(qa.ID("menuhost"))).Click()
	wantEvents(t, app, "menu show", "menu rename", "menu show", "menu wrap", "menu show", "menu deep")
	if !*app.Find(qa.Text("Wrap").In(qa.ID("menuhost"))).Node().Checked {
		t.Error("the check item did not toggle")
	}
	wantErr(t, actErr(app, qa.Text("More").In(qa.ID("menuhost")), "click", nil), "submenu")
}

// An open popover is in the dump, its items run, and Shot includes it.
func TestPopover(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	before := app.Shot("popover-closed")
	if app.Find(qa.Class("Popover")).Exists() {
		t.Fatal("a popover before it was shown")
	}
	app.Find(qa.ID("pop")).Click()
	pop := app.Find(qa.Class("Popover"))
	if !pop.Node().Visible || !pop.Exists() {
		t.Fatalf("popover node %+v", pop.Node())
	}
	app.Find(qa.ID("pop-ok")).Click()
	wantEvents(t, app, "pop ok")

	after := app.Shot("popover-open")
	b0, _ := os.ReadFile(before)
	b1, _ := os.ReadFile(after)
	if string(b0) == string(b1) {
		t.Error("the shot with the popover open equals the shot without it")
	}
	c0, _, err0 := image.DecodeConfig(strings.NewReader(string(b0)))
	c1, _, err1 := image.DecodeConfig(strings.NewReader(string(b1)))
	if err0 != nil || err1 != nil || c1.Width < c0.Width || c1.Height < c0.Height {
		t.Errorf("the shot with the popover is %dx%d, the window alone %dx%d (%v, %v)", c1.Width, c1.Height, c0.Width, c0.Height, err0, err1)
	}
	pop.Close()
	if pop.Exists() || !hasEvent(t, app, "pop closed") {
		t.Errorf("the popover did not close: %q", events(t, app))
	}
}

// Items an OnShow handler adds to an empty context menu are reachable after a right click, on a gfx
// view and on a tree, and by Find alone.
func TestPopupMenuFilledInOnShow(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.ID("gfx")).RightClickAt(20, 20)
	if !app.Find(qa.Text("Duplicate")).Exists() {
		t.Fatal("Duplicate is not in the dump after the right click")
	}
	app.Find(qa.Text("Duplicate")).Click()
	if !hasEvent(t, app, "duplicate") {
		t.Errorf("events %q", events(t, app))
	}
	app.Find(qa.ID("lazy")).RightClickAt(5, 5)
	app.Find(qa.Text("Refresh")).Click()
	if !hasEvent(t, app, "refresh") {
		t.Errorf("events %q", events(t, app))
	}
}
