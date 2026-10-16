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
		var field *gowt.TextView
		var echo *gowt.LabelView
		w := app.Window("Facade and swt",
			gowt.Text(gowt.GrowX(), gowt.Ref(&field)),
			gowt.Label("", gowt.GrowX(), gowt.Ref(&echo)),
		)

		w.SetMinSize(320, 160)

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
		var b *gowt.ButtonView
		w.Add(gowt.Button("Left aligned", nil, gowt.GrowX(), gowt.Ref(&b)))
		b.Unwrap().SetAlignment(swt.LEFT)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
