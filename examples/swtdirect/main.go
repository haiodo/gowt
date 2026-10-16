// Command swtdirect drops from the facade to package swt for what the facade does not wrap:
// Unwrap gives the swt object of a widget, AsComposite the parent for a raw swt widget.
package main

import (
	"fmt"
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		echo := rx.New("")
		w := app.Window("Facade and swt",
			// A key listener on the swt Text; the facade only has OnChange and OnActivate.
			gowt.Text(gowt.GrowX(), gowt.Init(func(t *gowt.TextView) {
				t.Unwrap().AddKeyListener(swt.KeyListenerKeyPressedAdapter(func(e *swt.KeyEvent) {
					echo.Set(fmt.Sprintf("key code %d, character %q", e.KeyCode, rune(e.Character)))
				}))
			})),
			gowt.Label("", gowt.GrowX(), gowt.BindText(echo)))

		w.SetMinSize(320, 160)

		// A separator line is a Label with the SEPARATOR style in SWT; the facade has no such widget.
		sep := swt.NewLabel(w.AsComposite(), swt.SEPARATOR|swt.HORIZONTAL)
		gd := swt.NewGridData()
		gd.HorizontalAlignment, gd.GrabExcessHorizontalSpace = swt.FILL, true
		sep.SetLayoutData(gd)

		// Left-aligned button text: swt.Button has SetAlignment.
		w.Add(gowt.Button("Left aligned", gowt.GrowX(),
			gowt.Init(func(b *gowt.ButtonView) { b.Unwrap().SetAlignment(swt.LEFT) })))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
