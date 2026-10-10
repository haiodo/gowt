package gowt_test

import (
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
)

// A window with a field, a button and a label; Run returns when the window is closed.
func ExampleRun() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Hello gowt")
		w.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
		name := w.Text(gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 200}))
		greet := w.Label("", gowt.Cell(gowt.GridCell{SpanX: 2, Align: gowt.AlignFill, GrowX: true}))
		w.Button("Greet", func() { greet.SetText("Hello, " + name.Text() + "!") })
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Work off the UI thread reports back through Async; widgets are touched only on the UI thread.
func ExampleApp_Async() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Download")
		status := w.Label("working...")
		w.Show()
		go func() {
			time.Sleep(time.Second) // stands for slow work
			app.Async(func() { status.SetText("done") })
		}()
	})
}

// After runs a function on the UI thread once the delay has passed.
func ExampleApp_After() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Timer")
		w.Show()
		app.After(5*time.Second, app.Quit)
	})
}

// Grid places children in columns; GridCell tunes one child.
func ExampleGrid() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Form")
		w.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 6})
		w.Label("Name")
		w.Text(gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true}))
		w.Label("Notes")
		w.Text(gowt.Multiline(), gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, GrowY: true, Height: 80}))
		w.Show()
	})
}

// A modal question over a window.
func ExampleWindow_Confirm() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Editor")
		w.Button("Quit", func() {
			if w.Confirm("Quit", "Discard unsaved changes?") {
				w.Close()
			}
		})
		w.Show()
	})
}

// Table rows hold one text per column; SetData keeps an application value with a row.
func ExampleTable() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Files")
		t := w.Table()
		t.Column("Name", 200)
		t.Column("Size", 80, gowt.Right())
		t.Row("a.txt", "12").SetData("id-1")
		t.OnSelect(func(i int) { fmt.Println(t.RowAt(i).Data()) })
		w.Show()
	})
}

// The paint callback draws the whole area; call Redraw after the model changes.
func ExamplePanel_Canvas() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Canvas")
		w.Canvas(func(g *gowt.GC) {
			g.SetFill(gowt.RGB{R: 30, G: 120, B: 220})
			g.FillOval(10, 10, 100, 60)
		}, gowt.Cell(gowt.GridCell{Width: 200, Height: 100}))
		w.Show()
	})
}

// The clipboard belongs to the App; a setter fails with an error when the system clipboard is busy.
func ExampleApp_Clipboard() {
	_ = gowt.Run(func(app *gowt.App) {
		if err := app.Clipboard().SetText("hello"); err != nil {
			log.Print(err)
			return
		}
		text, ok := app.Clipboard().Text()
		fmt.Println(text, ok)
	})
}

// Any widget can be a drag source and a drop target; callbacks run on the UI thread.
func ExampleDropOn() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Drop")
		zone := w.Label("drop files here")
		gowt.DropOn(zone, gowt.Drop{Files: func(paths []string, _ gowt.Op) { zone.SetText(paths[0]) }})
		w.Show()
	})
}

// StyledText offsets are runes; spans style ranges of the text.
func ExampleStyledText() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Editor")
		ed := w.StyledText(gowt.Scrollbars())
		ed.SetText("bold word")
		ed.SetStyle(0, 4, gowt.TextStyle{Bold: true})
		w.Show()
	})
}
