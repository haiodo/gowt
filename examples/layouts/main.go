// Command layouts shows the five layouts: Grid, Row, Fill, Form and Stack.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

var fillX = gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true})
var fillBoth = gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, VAlign: gowt.AlignFill, GrowX: true, GrowY: true})

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
	p.Text(fillX)
	p.Label("Notes")
	p.Text(gowt.Multiline(), fillBoth)
	p.Button("Save", nil, gowt.Cell(gowt.GridCell{SpanX: 2, Align: gowt.AlignEnd}))
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

// Stack: every child has the full size of the panel, ShowTop picks the visible one.
func stack(p *gowt.Panel) {
	p.SetLayout(gowt.Grid{Columns: 1, Margin: 10, Spacing: 8})
	pages := p.Panel(fillBoth)
	pages.SetLayout(gowt.Stack{})
	first := pages.Panel()
	first.SetLayout(gowt.Fill{})
	first.Label("Page one")
	second := pages.Panel()
	second.SetLayout(gowt.Fill{})
	second.Label("Page two")
	pages.ShowTop(first)
	onSecond := false // Stack has no getter for the page on top
	p.Button("Switch", func() {
		onSecond = !onSecond
		if onSecond {
			pages.ShowTop(second)
		} else {
			pages.ShowTop(first)
		}
	})
}
