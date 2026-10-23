// Command qa is a greeter written to be tested through the QA driver: see qa_test.go and README.md.
package main

import (
	"context"
	"log"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/rx"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		name := rx.New("")
		greeting := rx.New("")
		w := app.Window("Greeter",
			gowt.Text(gowt.ID("name"), gowt.OnChange(func(s string) { name.Set(s) })),
			gowt.Button("Greet", gowt.ID("greet"), gowt.OnClick(func() { greeting.Set("Hello, " + name.Get()) })),
			gowt.Label("", gowt.ID("greeting"), gowt.BindText(greeting)),
			gowt.Button("Load", gowt.ID("load"), gowt.OnClick(func() {
				app.Job("Load greeting", func(ctx context.Context, p *gowt.JobProgress) error {
					time.Sleep(time.Second) // a slow request
					return nil
				}).Then(func(error) { greeting.Set("Loaded") })
			})))
		w.Show()

		// A no-op without -tags gowtqa. qatest calls "reset" before every test.
		qa.Handle("reset", func(qa.Args) error { greeting.Set(""); return nil })
		qa.HandleValue("greeting", func(qa.Args) (any, error) { return greeting.Get(), nil })
	})
	if err != nil {
		log.Fatal(err)
	}
}
