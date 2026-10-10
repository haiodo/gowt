// Command swtdirect drops from the facade to package swt for what the facade does not wrap:
// Unwrap gives the swt object of a widget, AsComposite the parent for a raw swt widget.
package main

import (
	"fmt"
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/swt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Facade and swt")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 10, Spacing: 8})
		fillX := gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true})

		// The facade has no minimum window size: the shell has.
		w.Unwrap().SetMinimumSize(320, 160)

		field := w.Text(fillX)
		echo := w.Label("", fillX)

		// A key listener on the swt Text; the facade only has OnChange and OnActivate.
		field.Unwrap().AddKeyListener(swt.KeyListenerKeyPressedAdapter(func(e *swt.KeyEvent) {
			echo.SetText(fmt.Sprintf("key code %d, character %q", e.KeyCode, rune(e.Character)))
		}))

		// A separator line is a Label with the SEPARATOR style in SWT; the facade has no such widget.
		sep := swt.NewLabel(w.AsComposite(), swt.SEPARATOR|swt.HORIZONTAL)
		gd := swt.NewGridData()
		gd.HorizontalAlignment, gd.GrabExcessHorizontalSpace = swt.FILL, true
		sep.SetLayoutData(gd)

		// Left-aligned button text: swt.Button has SetAlignment.
		b := w.Button("Left aligned", nil, fillX)
		b.Unwrap().SetAlignment(swt.LEFT)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
