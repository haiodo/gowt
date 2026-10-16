// Command hellogowt is the smallest gowt program: a window with a field, a button and a label.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		var name *gowt.TextView
		var greet *gowt.LabelView
		w := app.Window("Hello gowt",
			gowt.Text(gowt.GrowX(), gowt.Size(200, 0), gowt.Ref(&name)),
			gowt.Label("", gowt.GrowX(), gowt.Ref(&greet)),
			gowt.Button("Greet", func() { greet.SetText("Hello, " + name.Text() + "!") }),
		)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
