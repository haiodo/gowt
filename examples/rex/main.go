// Command rex is a two-window Rex demo: a terminal session window and a settings window.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/swt"
)

var quitting bool

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		app.SetAppearance(gowt.AppearanceDark)
		watchTheme(app)

		var settings *gowt.Window
		var settingsList *drawnList
		shown := false
		rex := rexWindow(app, func() {
			if !shown {
				shown = true
				settings.Show()
			} else {
				settings.SetVisible(true)
				settings.Focus()
			}
			swt.QAFocus(&settingsList.canvas.Unwrap().Control)
		})
		settings, settingsList = settingsWindow(app)
		qa.HandleValue("client-origins", func(qa.Args) (any, error) {
			origin := func(w *gowt.Window) [2]int32 {
				b := w.Unwrap().GetBounds()
				p := w.Unwrap().ToDisplay(0, 0)
				return [2]int32{p.X - b.X, p.Y - b.Y}
			}
			return map[string][2]int32{"rex": origin(rex), "settings": origin(settings)}, nil
		})
		rex.OnClose(func() bool {
			quitting = true
			settings.Close()
			app.Quit()
			return true
		})
		rex.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
