// Command dnd drags a label's text onto a drop zone and accepts files dropped from a file manager.
package main

import (
	"log"
	"strings"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Drag and drop")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 10, Spacing: 8})
		fill := gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 280})

		src := w.Label("Drag this text", fill, gowt.Border())
		gowt.DragFrom(src, gowt.Drag{
			Ops:    gowt.OpCopy | gowt.OpMove,
			Text:   func() string { return "dragged from gowt" },
			OnDone: func(op gowt.Op) { src.SetText("done, operation " + opName(op)) },
		})

		zone := w.Label("Drop text or files here", fill, gowt.Border())
		gowt.DropOn(zone, gowt.Drop{
			Text:  func(s string, _ gowt.Op) { zone.SetText("text: " + s) },
			Files: func(p []string, _ gowt.Op) { zone.SetText("files: " + strings.Join(p, ", ")) },
		})
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func opName(op gowt.Op) string {
	switch op {
	case gowt.OpCopy:
		return "copy"
	case gowt.OpMove:
		return "move"
	case gowt.OpLink:
		return "link"
	}
	return "none"
}
