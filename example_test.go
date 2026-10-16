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
		name := w.Text(gowt.GrowX(), gowt.Size(200, 0))
		greet := w.Label("", gowt.GrowX(), gowt.Span(2))
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

// Grid places children in columns; placement options such as GrowX tune one child.
func ExampleGrid() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Form")
		w.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 6})
		w.Label("Name")
		w.Text(gowt.GrowX())
		w.Label("Notes")
		w.Text(gowt.Multiline(), gowt.Grow(), gowt.Size(0, 80))
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
		}, gowt.Size(200, 100))
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

// A push button, a check box and a button pushed to the right end of its row.
func ExamplePanel_Button() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Buttons")
		w.SetLayout(gowt.Grid{Columns: 3, Margin: 10, Spacing: 8})
		remember := w.Button("Remember me", nil, gowt.Check())
		w.Spacer()
		w.Button("Sign in", func() { fmt.Println("remember:", remember.Checked()) })
		w.Show()
	})
}

// A field that takes the free width, a password field and a search field with a hint.
func ExamplePanel_Text() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Fields")
		w.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
		w.Label("User")
		user := w.Text(gowt.GrowX(), gowt.Size(200, 0))
		w.Label("Password")
		w.Text(gowt.Password(), gowt.GrowX())
		find := w.Text(gowt.Search(), gowt.GrowX(), gowt.Span(2))
		find.SetHint("Search")
		user.OnChange(func(s string) { fmt.Println("user:", s) })
		w.Show()
	})
}

// A table that fills the window: columns first, then one row per record.
func ExamplePanel_Table() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Processes")
		w.SetLayout(gowt.Grid{Columns: 1})
		t := w.Table(gowt.Grow(), gowt.MultiSelect())
		t.Column("Name", 160)
		t.Column("CPU", 60, gowt.Right())
		t.Row("gowt", "1.5")
		t.Row("docker", "12.0")
		t.OnActivate(func(i int) { fmt.Println("open", t.RowAt(i).Text(0)) })
		w.Show()
	})
}

// Nodes nest under the tree and under each other; callbacks get the node.
func ExamplePanel_Tree() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Files")
		w.SetLayout(gowt.Grid{Columns: 1})
		tree := w.Tree(gowt.Grow())
		docs := tree.Node("Documents")
		docs.Node("report.pdf")
		docs.SetExpanded(true)
		tree.OnSelect(func(n *gowt.Node) { fmt.Println(n.Text()) })
		w.Show()
	})
}
