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

		// Adaptive colours switch with the system theme by themselves; the labels in the card show its background.
		card := gowt.Adaptive{Light: gowt.RGB{R: 232, G: 240, B: 254}, Dark: gowt.RGB{R: 36, G: 46, B: 66}}
		accent := gowt.Adaptive{Light: gowt.RGB{R: 29, G: 99, B: 237}, Dark: gowt.RGB{R: 120, G: 165, B: 255}}

		var mode *gowt.LabelView
		var panel *gowt.GroupView
		w := app.Window("Theme and look",
			gowt.Label("", gowt.GrowX(), gowt.Size(280, 0), gowt.Ref(&mode)),
			gowt.VBox(gowt.GrowX(), gowt.Padding(gowt.Spacing.MD), gowt.Background(card),
				gowt.Label("Adaptive colours", gowt.Typeface(gowt.Font{Scale: 1.3, Weight: gowt.Semibold}), gowt.Foreground(accent)),
				gowt.Label("Font sizes are relative to the system font", gowt.Typeface(gowt.Font{Scale: 0.9}))),
			gowt.Group("Glass panel",
				gowt.Ref(&panel),
				gowt.Button("Glass button", look.GlassButton())))
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
