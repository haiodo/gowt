// Command counter is the 7GUIs counter task: a button counts its clicks.
package main

import (
	"log"
	"strconv"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

func counter() gowt.Option {
	count := rx.New(0)
	value := rx.Compute(func() string { return strconv.Itoa(count.Get()) })
	return gowt.HBox(
		gowt.Text(gowt.Disabled(), gowt.Size(80, 0), gowt.BindText(value)),
		gowt.Button("Count", gowt.OnClick(func() { count.Set(count.Get() + 1) })),
	)
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Counter", counter())
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
