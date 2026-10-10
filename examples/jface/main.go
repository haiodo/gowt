// Command jface builds one window from gowt containers and JFace factories: the grid layout and
// the label, text field and button come from package jface, the status line is a gowt widget.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/jface"
	"github.com/haiodo/gowt/swt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("JFace factories")
		// A JFace layout factory configures the composite behind the window; AsComposite hands it over.
		jface.GridLayoutFactorySwtDefaults().NumColumns(2).ApplyTo(w.AsComposite())

		fill := func() *swt.GridData { return jface.GridDataFactoryFillDefaults().Grab(true, false).Create() }
		jface.WidgetFactoryLabel(swt.NONE).Text("Name:").Create(w.AsComposite())
		name := jface.WidgetFactoryText(swt.BORDER).Message("Full name").LayoutData(fill()).Create(w.AsComposite())

		// gowt widgets in the same grid take their GridData from Cell.
		status := w.Label("", gowt.Cell(gowt.GridCell{SpanX: 2, Align: gowt.AlignFill, GrowX: true}))
		jface.WidgetFactoryButton(swt.PUSH).Text("Greet").
			LayoutData(jface.GridDataFactoryCreate(swt.END).Span(2, 1).Create()).
			OnSelect(func(e *swt.SelectionEvent) { status.SetText("Hello, " + name.GetText()) }).
			Create(w.AsComposite())

		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
