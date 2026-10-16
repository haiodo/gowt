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
		var names = icons.Names()
		var buttons = make([]*gowt.ButtonView, len(names))
		var cells = []gowt.Option{gowt.Grow()}
		for i, n := range names {
			cells = append(cells, gowt.Button("", nil, gowt.Tooltip(n), gowt.Ref(&buttons[i])))
		}
		w.Add(gowt.Grid(8, cells...))
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
