// Command hello is the smallest gowt program: one window, a label and a button.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Hello")
		w.SetLayout(gowt.Row{Vertical: true, Margin: 12, Spacing: 8})
		w.Label("Hello, gowt")
		w.Button("Quit", app.Quit)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
