// Command hellogowt is the smallest gowt program: a window with a field, a button and a label.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		name, greet := rx.New(""), rx.New("")
		w := app.Window("Hello gowt",
			gowt.Text(gowt.GrowX(), gowt.Size(200, 0), gowt.Bind(name)),
			gowt.Label("", gowt.GrowX(), gowt.BindText(greet)),
			gowt.Button("Greet", gowt.OnClick(func() { greet.Set("Hello, " + name.Get() + "!") })))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
