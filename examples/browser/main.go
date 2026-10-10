// Command browser uses SWT's Browser widget (package browser) inside a gowt window.
package main

import (
	"fmt"
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/browser"
	"github.com/haiodo/gowt/swt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Browser")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 8, Spacing: 8})
		status := w.Label("", gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true}))

		// Browser takes an swt composite as its parent: AsComposite hands it over.
		b := browser.NewBrowser(w.AsComposite(), swt.NONE)
		gd := swt.NewGridData()
		gd.HorizontalAlignment, gd.VerticalAlignment = swt.FILL, swt.FILL
		gd.GrabExcessHorizontalSpace, gd.GrabExcessVerticalSpace = true, true
		gd.WidthHint, gd.HeightHint = 480, 280
		b.SetLayoutData(gd)

		b.AddLocationListener(browser.LocationListenerChangedAdapter(func(e *browser.LocationEvent) {
			status.SetText(fmt.Sprint("location: ", e.Location))
		}))
		b.SetText(`<h2>Hello from SWT Browser</h2><p>Page text is set with SetText.</p>`)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
