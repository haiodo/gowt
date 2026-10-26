package gowt

import (
	"os"
	"testing"
	"time"
)

// TestQuitPatternWindowIsVisible is examples/quit in short: OnClose, OnQuit, then Show. The window must be
// visible, on a screen and not collapsed.
func TestQuitPatternWindowIsVisible(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var visible, onScreen bool
	var w, h int32
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			win := app.Window("Untitled")
			win.Add(Text(GrowX(), Hint("Type to make the document dirty")), Label("Saved"), Button("Save"))
			win.OnClose(func() bool { return true })
			app.OnQuit(func() bool { return true })
			win.Show()
			app.After(300*time.Millisecond, func() {
				b := win.shell.GetBounds()
				visible, w, h = win.shell.IsVisible(), b.Width, b.Height
				for _, m := range app.display.GetMonitors() {
					onScreen = onScreen || m.GetBounds().IntersectsRect(b)
				}
				app.Quit()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !visible || !onScreen || w < 80 || h < 80 {
		t.Errorf("window after Show: visible=%v on a screen=%v size=%dx%d", visible, onScreen, w, h)
	}
}
