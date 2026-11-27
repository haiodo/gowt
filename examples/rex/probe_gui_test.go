package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

func TestToolbarProbe(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	app := qatest.App(t)
	app.Settle()
	for _, width := range []int{300, 260, 220, 180, 150} {
		app.State("sidebar-width", "width", width)
		app.Settle()
		content := app.Find(qa.ID("rex-content")).Node().Bounds
		t.Logf("width %d content %v", width, content)
		for _, id := range []string{"toggle-sidebar", "toolbar-eye", "toolbar-command", "toolbar-plus", "more"} {
			if e := app.Find(qa.ID(id)); e.Exists() {
				t.Logf("%s %v", id, e.Node().Bounds)
			}
		}
		app.Shot(fmt.Sprintf("probe-%d", width))
	}
}
