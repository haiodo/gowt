package gowt

import (
	"runtime"
	"testing"

	"github.com/haiodo/gowt/swt"
)

func TestSetBadge(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		app.SetBadge("7")
		later := app.Window("later")
		defer later.shell.Dispose()
		// The items that show it: the app's on macOS, each window's on Windows; Linux has no TaskBar.
		var shells []*swt.Shell
		switch runtime.GOOS {
		case "darwin":
			shells = []*swt.Shell{nil}
		case "windows":
			shells = []*swt.Shell{w.shell, later.shell}
		}
		overlays := func() []string {
			var got []string
			for _, s := range shells {
				got = append(got, app.display.GetSystemTaskBar().GetItemShell(s).GetOverlayText())
			}
			return got
		}
		for i, got := range overlays() {
			if got != "7" {
				t.Errorf("item %d: overlay %q, want \"7\"", i, got)
			}
		}
		app.SetBadge("")
		for i, got := range overlays() {
			if got != "" {
				t.Errorf("item %d: overlay %q after SetBadge(\"\")", i, got)
			}
		}
	})
}
