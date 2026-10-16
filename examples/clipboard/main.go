// Command clipboard copies a field's text to the system clipboard and pastes it back.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		var field *gowt.TextView
		var status *gowt.LabelView
		// The clipboard belongs to the App and is disposed with it. A busy clipboard is a normal
		// error on Windows, so setters return it.
		w := app.Window("Clipboard",
			gowt.Grid(2, gowt.Grow(),
				gowt.Text(gowt.GrowX(), gowt.Size(240, 0), gowt.Ref(&field)),
				gowt.Label("", gowt.GrowX(), gowt.Span(2), gowt.Ref(&status)),
				gowt.Button("Copy", func() {
					if err := app.Clipboard().SetText(field.Text()); err != nil {
						status.SetText("copy failed: " + err.Error())
						return
					}
					status.SetText("copied")
				}),
				gowt.Button("Paste", func() {
					if s, ok := app.Clipboard().Text(); ok {
						field.SetText(s)
						status.SetText("pasted")
					} else {
						status.SetText("no text on the clipboard")
					}
				}),
			),
		)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
