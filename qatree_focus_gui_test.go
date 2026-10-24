//go:build gowtqa

package gowt

import (
	"os"
	"testing"

	"github.com/haiodo/gowt/internal/qatree"
)

// Focus must work in an app that is not active. On macOS only the QA build's QAFocus
// (GOWT_QA_SOCKET read at call time) does; plain SetFocus is refused by a window that is not key.
func TestQATreeFocus(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	keys := 0
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("qa focus")
			w.Add(VBox(Button("go", ID("go")), Button("other", ID("other"))))
			w.OnKey("Cmd+K", func() { keys++ })
			w.Show()
			t.Setenv("GOWT_QA_SOCKET", "unused") // set after Run started, so no QA server listens
			qaAct(t, app, qatree.Selector{ID: "go"}, "focus", "")
			qaAct(t, app, qatree.Selector{}, "key", `{"keys":"Mod+K"}`) // to the focus control
			if keys != 1 {
				t.Errorf("keys=%d, want 1", keys)
			}
			qaAct(t, app, qatree.Selector{Class: "Shell"}, "close", "")
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}
