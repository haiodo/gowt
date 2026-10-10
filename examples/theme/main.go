// Command theme follows the system light/dark setting and tries the platform look options of package look.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		// Before the first window: dark scroll bars, tables and buttons on Windows.
		look.SetDarkContent(app, true)

		w := app.Window("Theme and look")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 16, Spacing: 8})
		mode := w.Label("", gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 280}))
		show := func(dark bool) {
			if dark {
				mode.SetText("system theme: dark")
			} else {
				mode.SetText("system theme: light")
			}
		}
		show(app.Dark())
		app.OnThemeChange(show)

		// Each call does nothing where the OS has no such feature.
		look.SetBackdrop(w, look.BackdropGlass)
		look.SetRoundedCorners(w, true)

		panel := w.Group("Glass panel")
		panel.SetLayout(gowt.Grid{Columns: 1, Margin: 12, Spacing: 8})
		look.SetGlass(panel, true)
		panel.Button("Glass button", nil, look.GlassButton())
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
