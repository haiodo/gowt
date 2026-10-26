package gowt

import (
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/cocoa"
)

// The folder and file panels used to throw NSInvalidArgumentException when no start path was set
// (examples/dialogs). Each panel is opened with empty paths and stopped from a timer.
func TestDialogsWithEmptyPaths(t *testing.T) {
	quitRun(t, 1, func(app *App, ws []*Window) {
		for name, open := range map[string]func(*Window){
			"DirDialog":       func(w *Window) { w.DirDialog("", "") },
			"FileDialog open": func(w *Window) { w.FileDialog(FileDialog{}) },
			"FileDialog save": func(w *Window) { w.FileDialog(FileDialog{Save: true}) },
		} {
			shown := false
			onUI(app, func() int {
				app.After(700*time.Millisecond, func() {
					shown = true
					cocoa.NSApplicationSharedApplication().StopModal()
				})
				open(ws[0]) // returns once the timer stopped the panel
				return 0
			})
			if !shown {
				t.Errorf("%s: the panel was not open when the timer fired", name)
			}
		}
	})
}
