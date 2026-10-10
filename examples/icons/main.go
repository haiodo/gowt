// Command icons puts named system icons on buttons and tints them again when the theme changes.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Icons")
		w.SetLayout(gowt.Row{Margin: 10, Spacing: 8})
		names := icons.Names()
		if len(names) > 12 {
			names = names[:12]
		}
		buttons := make([]*gowt.Button, len(names))
		for i, n := range names {
			buttons[i] = w.Button("", nil, gowt.Tooltip(n))
		}
		// Get caches per name, size and colour and the App owns the images: no Dispose here.
		fill := func() {
			for i, n := range names {
				buttons[i].SetImage(icons.Get(app, n, icons.Size(24)))
			}
		}
		fill()
		// An icon is tinted for the theme at Get time, so ask again when the theme changes.
		app.OnThemeChange(func(bool) { fill() })
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
