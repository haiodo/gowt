// Command icondemo shows every dictionary icon as a button and rebuilds them when the system
// theme changes. Check that the OS set is used (SF Symbols, Segoe Fluent Icons, freedesktop
// symbolic) and that the Lucide fallback, used for symbols the OS lacks, looks alike.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("gowt icons")
		w.SetLayout(gowt.Grid{Columns: 8, Margin: 10, Spacing: 8})
		var buttons []*gowt.Button
		var names = icons.Names()
		for _, n := range names {
			buttons = append(buttons, w.Button("", nil, gowt.Tooltip(n)))
		}
		// icons.Get caches per name, size and colour and the App owns the images, so no Dispose here.
		fill := func() {
			for i, n := range names {
				buttons[i].SetImage(icons.Get(app, n, icons.Size(24)))
			}
		}
		fill()
		app.OnThemeChange(func(bool) { fill() })
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
