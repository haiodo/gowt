// Command layouts shows the layouts: VBox and HBox, Grid, Flow, Form and Pages.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Layouts",
			gowt.Tabs(gowt.Grow(),
				gowt.Tab("VBox / HBox", boxes()),
				gowt.Tab("Grid", grid()),
				gowt.Tab("Flow", flow()),
				gowt.Tab("Form", form()),
				gowt.Tab("Pages", pages())))
		w.SetSize(480, 320)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// VBox: children stack top to bottom; an HBox lines its children up, a Spacer pushes the buttons to the end.
func boxes() gowt.Option {
	return gowt.VBox(gowt.GrowX(),
		gowt.Label("A VBox stacks its children"),
		gowt.Button("One"),
		gowt.Button("Two"),
		gowt.HBox(gowt.GrowX(),
			gowt.Label("An HBox lines them up"),
			gowt.Spacer(),
			gowt.Button("Cancel"),
			gowt.Button("OK")))
}

// Grid: labels in column 0, fields stretch in column 1, the notes field takes the spare height.
func grid() gowt.Option {
	return gowt.Grid(2, gowt.Grow(),
		gowt.Label("Name"), gowt.Text(gowt.GrowX()),
		gowt.Label("Notes"), gowt.Text(gowt.Multiline(), gowt.Grow()),
		gowt.Button("Save", gowt.Span(2), gowt.Place(gowt.AlignEnd, gowt.AlignStart)))
}

// Flow: children keep their natural size and wrap to the next row when the panel is narrow.
func flow() gowt.Option {
	opts := []gowt.Option{gowt.GrowX()}
	for _, name := range []string{"One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight"} {
		opts = append(opts, gowt.Button(name, gowt.Size(90, 0)))
	}
	return gowt.Flow(opts...)
}

// Form: the field sits right of the label and stretches to 5 points from the right edge,
// the button hangs under the field.
func form() gowt.Option {
	var label *gowt.LabelView
	var field *gowt.TextView
	return gowt.Form(gowt.Grow(),
		gowt.Label("Address", gowt.Ref(&label), gowt.Anchor(gowt.FormCell{
			Left: gowt.Percent(0, 0), Top: gowt.Percent(0, 4),
		})),
		gowt.Text(gowt.Ref(&field), gowt.Anchor(gowt.FormCell{
			Left: gowt.Beside(&label, 8), Right: gowt.Percent(100, 0), Top: gowt.Percent(0, 0),
		})),
		gowt.Button("Go", gowt.Anchor(gowt.FormCell{
			Right: gowt.Same(&field, 0), Top: gowt.Beside(&field, 8),
		})))
}

// Pages: every child is a page of the full size, Show picks the visible one.
func pages() gowt.Option {
	var pages *gowt.PagesView
	return gowt.VBox(gowt.Grow(),
		gowt.Pages(gowt.Grow(), gowt.Ref(&pages),
			gowt.Label("Page one"),
			gowt.Label("Page two")),
		gowt.Button("Switch", gowt.OnClick(func() { pages.Show(1 - pages.Current()) })))
}
