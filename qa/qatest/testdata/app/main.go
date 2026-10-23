// Command app is the fixture of qatest's end-to-end test: a button, a label and the QA handlers.
package main

import (
	"fmt"
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/rx"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		clicks := rx.New(0)
		text := rx.Compute(func() string { return fmt.Sprintf("clicks: %d", clicks.Get()) })
		w := app.Window("QA fixture",
			gowt.Label("", gowt.BindText(text)),
			gowt.Button("Click", gowt.OnClick(func() { clicks.Set(clicks.Get() + 1) })))
		w.Show()
		qa.Handle("reset", func(qa.Args) error { clicks.Set(0); return nil })
		qa.Handle("add", func(a qa.Args) error { clicks.Set(clicks.Get() + a.Int("n")); return nil })
		qa.HandleValue("get", func(qa.Args) (any, error) { return clicks.Get(), nil })
		qa.Handle("fail", func(qa.Args) error { return fmt.Errorf("asked to fail") })
		qa.Handle("panic", func(qa.Args) error { panic("asked to panic") })
	})
	if err != nil {
		log.Fatal(err)
	}
}
