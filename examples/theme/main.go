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

		var mode *gowt.LabelView
		var panel *gowt.GroupView
		w := app.Window("Theme and look",
			gowt.Label("", gowt.GrowX(), gowt.Size(280, 0), gowt.Ref(&mode)),
			gowt.Group("Glass panel",
				gowt.Ref(&panel),
				gowt.Button("Glass button", nil, look.GlassButton()),
			),
		)
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

		look.SetGlass(panel, true)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
