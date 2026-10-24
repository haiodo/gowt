package qatest_test

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

func events(t *testing.T, app *qatest.Client) []string {
	t.Helper()
	var ev []string
	if err := json.Unmarshal(app.State("events"), &ev); err != nil {
		t.Fatal(err)
	}
	return ev
}

// wantEvents compares the events without the focus ones, which depend on where the focus was before.
func wantEvents(t *testing.T, app *qatest.Client, want ...string) {
	t.Helper()
	var got []string
	for _, e := range events(t, app) {
		if !strings.HasPrefix(e, "focus ") {
			got = append(got, e)
		}
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("events %q, want %q", got, want)
	}
}

func actErr(app *qatest.Client, sel qa.Selector, action string, args any) error {
	_, err := app.Call("act", map[string]any{"selector": sel, "action": action, "args": args})
	return err
}

func wantErr(t *testing.T, err error, part string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), part) {
		t.Errorf("error %v, want one with %q", err, part)
	}
}

func TestActivate(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.Text("b").In(qa.ID("list"))).Activate()
	app.Find(qa.Text("beta")).Activate()
	app.Find(qa.ID("name")).Activate()
	wantEvents(t, app, "activate list 1", "activate table 1", "activate name")
	wantErr(t, actErr(app, qa.Class("Shell"), "activate", nil), "default button")
	wantErr(t, actErr(app, qa.ID("count"), "activate", nil), "does not apply")
}

func TestExpandLazy(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	// A repeated run finds the children of the last one, collapsed.
	if ns := app.Find(qa.Text("disk")).Nodes(); len(ns) > 0 && ns[0].Visible {
		t.Fatal("the lazy children are shown before the expand")
	}
	app.Find(qa.Text("remote")).Expand()
	if n := app.Find(qa.Text("disk")).Node(); !n.Visible {
		t.Errorf("disk after expand: %+v", n)
	}
	if app.Find(qa.Text("loading")).Exists() {
		t.Error("the placeholder is still there")
	}
	app.Find(qa.Text("disk")).Activate()
	app.Find(qa.Text("remote")).Collapse()
	wantEvents(t, app, "expand remote", "activate tree disk", "collapse remote")
	wantErr(t, actErr(app, qa.Text("disk"), "click", nil), "not visible")
	wantErr(t, actErr(app, qa.ID("list"), "expand", nil), "TreeItem")
}

func TestKeys(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.ID("click")).Key("Mod+K")
	app.Find(qa.ID("name")).Key("a")
	app.Find(qa.ID("name")).Key("Enter")
	app.Find(qa.ID("click")).Key("Mod+R")
	app.Key("Mod+K") // to the control the line above focused
	wantEvents(t, app, "key mod+k", "keydown name 97", "keydown name 13", "activate name", "menu run", "key mod+k")
	wantErr(t, actErr(app, qa.ID("click"), "key", map[string]string{"keys": "Hyper+K"}), "modifier")
}

func TestFocus(t *testing.T) {
	guiOnly(t)
	before := frontApp()
	app := qatest.App(t)
	app.Find(qa.ID("click")).Focus()
	app.Find(qa.ID("name")).Focus()
	if n := app.Find(qa.ID("name")).Node(); !n.Focused {
		t.Errorf("name after Focus: %+v", n)
	}
	app.Find(qa.ID("click")).Focus()
	ev := events(t, app)
	if n := len(ev); n < 2 || ev[n-2] != "focus name true" || ev[n-1] != "focus name false" {
		t.Errorf("events %q, want focus name true, then false at the end", ev)
	}

	// Between two windows the old control gets FocusOut before the new one gets FocusIn.
	app.Find(qa.ID("name")).Focus()
	app.Find(qa.ID("popup")).Click()
	app.Find(qa.ID("pname")).Focus()
	app.Find(qa.ID("name")).Focus()
	ev = events(t, app)
	want := []string{"focus name false", "focus pname true", "focus pname false", "focus name true"}
	if n := len(ev); n < 4 || !slices.Equal(ev[n-4:], want) {
		t.Errorf("events %q, want %q at the end", ev, want)
	}
	app.Find(qa.Selector{Class: "Shell", Text: "Popup"}).Close()
	if after := frontApp(); after != before {
		t.Errorf("the frontmost app changed from %q to %q", before, after)
	}
}

// frontApp is the name of the frontmost application on macOS, "" elsewhere.
func frontApp() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	out, _ := exec.Command("sh", "-c", "lsappinfo info -only name $(lsappinfo front)").Output()
	return strings.TrimSpace(string(out))
}

func TestScroll(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	inView := func(row string) bool {
		f, r := app.Find(qa.ID("feed")).Node(), app.Find(qa.ID(row)).Node()
		return r.Bounds[1] >= f.Bounds[1] && r.Bounds[1]+r.Bounds[3] <= f.Bounds[1]+f.Bounds[3]
	}
	if inView("row-50") {
		t.Fatal("row-50 is in view before any scroll")
	}
	// OnScroll coalesces scrolls within a frame; a fast local round trip (Windows) would merge the three.
	frame := func() { time.Sleep(50 * time.Millisecond) }
	app.Find(qa.ID("feed")).ScrollTo(qa.ID("row-50"))
	frame()
	if !inView("row-50") {
		t.Error("row-50 is not in view after ScrollTo")
	}
	app.Find(qa.ID("feed")).ScrollBy(0, -100000)
	frame()
	if !inView("row-0") {
		t.Error("row-0 is not in view after ScrollBy to the top")
	}
	app.Find(qa.ID("row-30")).ScrollIntoView()
	frame()
	if !inView("row-30") {
		t.Error("row-30 is not in view after ScrollIntoView")
	}
	ev := events(t, app)
	if len(ev) != 3 || !strings.HasPrefix(ev[0], "scroll ") || ev[1] != "scroll 0" {
		t.Errorf("OnScroll calls: %q", ev)
	}
	wantErr(t, actErr(app, qa.ID("feed"), "scroll", nil), `"to"`)
}

func TestClose(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	popup := app.Find(qa.Selector{Class: "Shell", Text: "Popup"})
	app.Find(qa.ID("popup")).Click()
	app.State("veto", "on", true)
	popup.Close()
	if !popup.Exists() {
		t.Fatal("the window closed although OnClose vetoed")
	}
	app.State("veto", "on", false)
	popup.Close()
	if popup.Exists() {
		t.Error("the window is still open")
	}
	wantEvents(t, app, "close popup", "close popup")
	wantErr(t, actErr(app, qa.ID("click"), "close", nil), "window")
}

func TestDialogs(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Answer(qa.Dialog{Kind: "confirm", Reply: true})
	app.Find(qa.ID("delete")).Click()
	app.Answer(qa.Dialog{Kind: "confirm", Cancel: true})
	app.Find(qa.ID("delete")).Click()
	app.Answer(qa.Dialog{Kind: "file", Reply: []string{"/a.txt", "/b.txt"}})
	app.Find(qa.ID("open")).Click()
	wantEvents(t, app, "deleted", "opened /a.txt", "opened /b.txt")
	ds := app.Dialogs()
	if len(ds) != 3 || ds[0].Kind != "confirm" || ds[0].Title != "Delete" || ds[0].Message != "Delete the file?" ||
		!ds[1].Cancel || !ds[2].Multi || len(ds[2].Filters) != 1 || ds[2].Filters[0] != "Text|*.txt" {
		t.Errorf("dialogs: %+v", ds)
	}

	// No scripted answer: the action fails and the app got Cancel instead of hanging.
	start := time.Now()
	wantErr(t, actErr(app, qa.ID("delete"), "click", nil), "no scripted answer")
	if time.Since(start) > 5*time.Second {
		t.Error("the unanswered dialog held the action")
	}
	wantEvents(t, app, "deleted", "opened /a.txt", "opened /b.txt")
	if ds := app.Dialogs(); len(ds) != 4 || !ds[3].Unanswered {
		t.Errorf("dialogs after the unanswered one: %+v", ds)
	}

	// A reply of the wrong shape fails the action too.
	app.Answer(qa.Dialog{Kind: "confirm", Reply: "yes"})
	wantErr(t, actErr(app, qa.ID("delete"), "click", nil), "got the reply")
	if got := app.Dialogs(); len(got) != 5 {
		t.Errorf("dialogs: %+v", got)
	}
	app.Settle()
}

func TestEventually(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	n := 0
	app.Eventually(2*time.Second, func() (any, bool) { n++; return n, n >= 3 })
}
