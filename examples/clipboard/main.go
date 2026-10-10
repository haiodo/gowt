// Command clipboard copies a field's text to the system clipboard and pastes it back.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Clipboard")
		w.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
		field := w.Text(gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 240}))
		status := w.Label("", gowt.Cell(gowt.GridCell{SpanX: 2, Align: gowt.AlignFill, GrowX: true}))

		// The clipboard belongs to the App and is disposed with it. A busy clipboard is a normal
		// error on Windows, so setters return it.
		w.Button("Copy", func() {
			if err := app.Clipboard().SetText(field.Text()); err != nil {
				status.SetText("copy failed: " + err.Error())
				return
			}
			status.SetText("copied")
		})
		w.Button("Paste", func() {
			if s, ok := app.Clipboard().Text(); ok {
				field.SetText(s)
				status.SetText("pasted")
			} else {
				status.SetText("no text on the clipboard")
			}
		})
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
