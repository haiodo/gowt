package qatest_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

func dndLog(t *testing.T, app *qatest.Client) []string {
	t.Helper()
	var log []string
	if err := json.Unmarshal(app.State("dnd"), &log); err != nil {
		t.Fatal(err)
	}
	return log
}

func wantDnd(t *testing.T, app *qatest.Client, want ...string) {
	t.Helper()
	if got := dndLog(t, app); !slices.Equal(got, want) {
		t.Errorf("drag log %q, want %q", got, want)
	}
}

// The source offers copy and move and so does the target: a plain drag moves (DND_MOVE is 2).
func TestDragTreeToTree(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.Text("apple").In(qa.ID("dnd-src"))).DragDropTo(app.Find(qa.Text("crate").In(qa.ID("dnd-dst"))))
	wantDnd(t, app, "start", "tree over crate", `tree drop "apple" op 2`, "done 2")
}

// The row under the press is the source: it is selected first and its text is the data.
func TestDragSecondRow(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.Text("pear").In(qa.ID("dnd-src"))).DragDropTo(app.Find(qa.Text("basket").In(qa.ID("dnd-dst"))))
	wantDnd(t, app, "start", "tree over basket", `tree drop "pear" op 2`, "done 2")
}

// Over refuses the drop on empty space: no Drop, the source reports no operation.
func TestDragOverRefuses(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.Text("apple").In(qa.ID("dnd-src"))).DragDropToAt(app.Find(qa.ID("dnd-dst")), 100, 60)
	wantDnd(t, app, "start", "tree over nothing", "done 0")
}

// A canvas target with the default operation (copy) takes the drag of a source offering copy and move.
func TestDragTreeToCanvas(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	app.Find(qa.Text("pear").In(qa.ID("dnd-src"))).DragDropToAt(app.Find(qa.ID("gfx")), 100, 80)
	wantDnd(t, app, "start", "canvas over 100,80", `canvas drop "pear" op 1`, "done 1")
}

func TestDragNeedsSourceAndTarget(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	wantErr(t, actErr(app, qa.ID("dnd-dst"), "drag", map[string]any{"to": qa.ID("gfx")}), "no drag source")
	wantErr(t, actErr(app, qa.ID("dnd-src"), "drag", map[string]any{"to": qa.ID("count")}), "no drop target")
	wantErr(t, actErr(app, qa.ID("dnd-src"), "drag", nil), "needs a target")
}
