// Command icondemo shows every dictionary icon as a button and rebuilds them when the system
// theme changes. Check that the OS set is used (SF Symbols, Segoe Fluent Icons, freedesktop
// symbolic) and that the Lucide fallback, used for symbols the OS lacks, looks alike.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("gowt icons")
		w.SetLayout(gowt.Grid{Columns: 8, Margin: 10, Spacing: 8})
		var buttons []*gowt.Button
		var names = gowt.IconNames()
		for _, n := range names {
			buttons = append(buttons, w.Button("", nil, gowt.Tooltip(n)))
		}
		var shown []*gowt.Image
		fill := func() {
			for _, img := range shown {
				img.Dispose()
			}
			shown = shown[:0]
			for i, n := range names {
				img := app.Icon(n, gowt.IconSize(24))
				shown = append(shown, img)
				buttons[i].SetImage(img)
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
