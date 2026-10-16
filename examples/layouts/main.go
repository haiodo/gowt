// Command layouts shows the five layouts: Grid, Row, Fill, Form and Stack.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Layouts")
		w.SetLayout(gowt.Fill{})
		tabs := w.Tabs()
		grid(tabs.Tab("Grid"))
		row(tabs.Tab("Row"))
		form(tabs.Tab("Form"))
		stack(tabs.Tab("Stack"))
		w.SetSize(480, 320)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Grid: labels in column 0, fields stretch in column 1, the notes field takes the spare height.
func grid(p *gowt.Panel) {
	p.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
	p.Label("Name")
	p.Text(gowt.GrowX())
	p.Label("Notes")
	p.Text(gowt.Multiline(), gowt.Grow())
	p.Button("Save", nil, gowt.Span(2), gowt.Place(gowt.AlignEnd, gowt.AlignStart))
}

// Row: children keep their natural size and wrap to the next row when the panel is narrow.
func row(p *gowt.Panel) {
	p.SetLayout(gowt.Row{Wrap: true, Margin: 10, Spacing: 6})
	for _, name := range []string{"One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight"} {
		p.Button(name, nil, gowt.InRow(gowt.RowCell{Width: 90}))
	}
}

// Form: the field sits right of the label and stretches to 5 points from the right edge,
// the button hangs under the field.
func form(p *gowt.Panel) {
	p.SetLayout(gowt.Form{Margin: 10})
	label := p.Label("Address", gowt.Anchor(gowt.FormCell{
		Left: gowt.Percent(0, 0), Top: gowt.Percent(0, 4),
	}))
	field := p.Text(gowt.Anchor(gowt.FormCell{
		Left: gowt.Beside(label, 8), Right: gowt.Percent(100, 0), Top: gowt.Percent(0, 0),
	}))
	p.Button("Go", nil, gowt.Anchor(gowt.FormCell{
		Right: gowt.Same(field, 0), Top: gowt.Beside(field, 8),
	}))
}

// Stack: Pages gives every page the full size of the panel, Show picks the visible one.
func stack(p *gowt.Panel) {
	p.SetLayout(gowt.Grid{Columns: 1, Margin: 10, Spacing: 8})
	pages := p.Pages(gowt.Grow())
	pages.Page().SetLayout(gowt.Fill{}).Label("Page one")
	pages.Page().SetLayout(gowt.Fill{}).Label("Page two")
	p.Button("Switch", func() { pages.Show(1 - pages.Current()) })
}
