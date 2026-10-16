// Command clipboard copies a field's text to the system clipboard and pastes it back.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		field := rx.New("")
		status := rx.New("")
		// The clipboard belongs to the App and is disposed with it. A busy clipboard is a normal
		// error on Windows, so setters return it.
		w := app.Window("Clipboard",
			gowt.Grid(2, gowt.Grow(),
				gowt.Text(gowt.GrowX(), gowt.Size(240, 0), gowt.Bind(field)),
				gowt.Label("", gowt.GrowX(), gowt.Span(2), gowt.BindText(status)),
				gowt.Button("Copy", gowt.OnClick(func() {
					if err := app.Clipboard().SetText(field.Get()); err != nil {
						status.Set("copy failed: " + err.Error())
						return
					}
					status.Set("copied")
				})),
				gowt.Button("Paste", gowt.OnClick(func() {
					if s, ok := app.Clipboard().Text(); ok {
						field.Set(s)
						status.Set("pasted")
					} else {
						status.Set("no text on the clipboard")
					}
				}))))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
