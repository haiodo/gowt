// Command hello is the smallest gowt program: one window, a label and a button.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Hello",
			gowt.Label("Hello, gowt"),
			gowt.Button("Quit", app.Quit),
		)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
