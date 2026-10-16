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
		var name *gowt.TextView
		var greet *gowt.LabelView
		w := app.Window("Hello gowt",
			gowt.Text(gowt.GrowX(), gowt.Size(200, 0), gowt.Ref(&name)),
			gowt.Label("", gowt.GrowX(), gowt.Ref(&greet)),
			gowt.Button("Greet", gowt.OnClick(func() { greet.SetText("Hello, " + name.Text() + "!") })),
		)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// The tree of values reads like the screen: a header row, a form, a row of buttons at the right.
// Children whose callbacks use the window go in w.Add, after w exists.
func ExampleHBox() {
	_ = gowt.Run(func(app *gowt.App) {
		var name *gowt.TextView
		w := app.Window("Contacts")
		w.Add(
			gowt.HBox(gowt.GrowX(),
				gowt.Label("Contacts"),
				gowt.Spacer(),
				gowt.Button("Add", gowt.OnClick(func() { name.SetText("") })),
			),
			gowt.Grid(2, gowt.GrowX(),
				gowt.Label("Name"),
				gowt.Text(gowt.GrowX(), gowt.Ref(&name)),
			),
			gowt.HBox(gowt.GrowX(),
				gowt.Spacer(),
				gowt.Button("Cancel", gowt.OnClick(w.Close)),
				gowt.Button("Save", gowt.OnClick(func() { fmt.Println("save", name.Text()) })),
			),
		)
		name.SetHint("Full name")
		w.Show()
	})
}

// Add fills a container after its creation; Clear empties it for new content.
func ExamplePanel_Add() {
	_ = gowt.Run(func(app *gowt.App) {
		var list *gowt.Panel
		w := app.Window("Log", gowt.VBox(gowt.Grow(), gowt.Ref(&list)))
		w.Add(gowt.Button("More", gowt.OnClick(func() { list.Add(gowt.Label(time.Now().Format(time.TimeOnly))) })))
		w.Add(gowt.Button("Clear", gowt.OnClick(func() { list.Clear() })))
		w.Show()
	})
}

// Work off the UI thread reports back through Async; widgets are touched only on the UI thread.
func ExampleApp_Async() {
	_ = gowt.Run(func(app *gowt.App) {
		var status *gowt.LabelView
		w := app.Window("Download", gowt.Label("working...", gowt.Ref(&status)))
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

// Every repeats a function on the UI thread until the returned stop is called.
func ExampleApp_Every() {
	_ = gowt.Run(func(app *gowt.App) {
		var clock *gowt.LabelView
		w := app.Window("Clock", gowt.Label(time.Now().Format(time.TimeOnly), gowt.Ref(&clock)))
		stop := app.Every(time.Second, func() { clock.SetText(time.Now().Format(time.TimeOnly)) })
		w.OnClose(func() bool { stop(); return true })
		w.Show()
	})
}

// Pages shows one child at a time; Current tells which.
func ExamplePages() {
	_ = gowt.Run(func(app *gowt.App) {
		var pages *gowt.PagesView
		w := app.Window("Wizard",
			gowt.Pages(gowt.Grow(), gowt.Ref(&pages),
				gowt.Label("Step 1"),
				gowt.Label("Step 2"),
			),
		)
		w.Add(gowt.Button("Next", gowt.OnClick(func() { pages.Show((pages.Current() + 1) % 2) })))
		w.Show()
	})
}

// A menu item with Shortcut runs on Cmd+S (Ctrl+S on Windows and Linux) and shows the key.
func ExampleShortcut() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Editor")
		file := w.MenuBar().Submenu("&File")
		file.Item("&Save", func() { fmt.Println("saved") }, gowt.Shortcut("Cmd+S"))
		w.Show()
	})
}

// Grid places children in columns, a row of the form per line of code; placement options such as
// GrowX tune one child.
func ExampleGrid() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Form",
			gowt.Grid(2, gowt.Grow(),
				gowt.Label("Name"), gowt.Text(gowt.GrowX()),
				gowt.Label("Notes"), gowt.Text(gowt.Multiline(), gowt.Grow(), gowt.Size(0, 80)),
			),
		)
		w.Show()
	})
}

// A modal question over a window.
func ExampleWindow_Confirm() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Editor")
		w.Add(gowt.Button("Quit", gowt.OnClick(func() {
			if w.Confirm("Quit", "Discard unsaved changes?") {
				w.Close()
			}
		})))
		w.Show()
	})
}

// A table that fills the window: columns first, then one row per record; SetData keeps an
// application value with a row.
func ExampleTable() {
	_ = gowt.Run(func(app *gowt.App) {
		var t *gowt.TableView
		w := app.Window("Processes", gowt.Table(gowt.Grow(), gowt.MultiSelect(), gowt.Ref(&t)))
		t.Column("Name", 160)
		t.Column("CPU", 60, gowt.Right())
		t.Row("gowt", "1.5").SetData("pid-1")
		t.Row("docker", "12.0")
		t.OnActivate(func(i int) { fmt.Println("open", t.RowAt(i).Text(0), t.RowAt(i).Data()) })
		w.Show()
	})
}

// The paint callback draws the whole area; call Redraw after the model changes.
func ExampleCanvas() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Canvas",
			gowt.Canvas(func(g *gowt.GC) {
				g.SetFill(gowt.RGB{R: 30, G: 120, B: 220})
				g.FillOval(10, 10, 100, 60)
			}, gowt.Size(200, 100)),
		)
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
		var zone *gowt.LabelView
		w := app.Window("Drop", gowt.Label("drop files here", gowt.Ref(&zone)))
		gowt.DropOn(zone, gowt.Drop{Files: func(paths []string, _ gowt.Op) { zone.SetText(paths[0]) }})
		w.Show()
	})
}

// StyledText offsets are runes; spans style ranges of the text.
func ExampleStyledText() {
	_ = gowt.Run(func(app *gowt.App) {
		var ed *gowt.StyledTextView
		w := app.Window("Editor", gowt.StyledText(gowt.Scrollbars(), gowt.Grow(), gowt.Ref(&ed)))
		ed.SetText("bold word")
		ed.SetStyle(0, 4, gowt.TextStyle{Bold: true})
		w.Show()
	})
}

// A check box and a push button pushed to the right end of their row.
func ExampleButton() {
	_ = gowt.Run(func(app *gowt.App) {
		var remember *gowt.ButtonView
		w := app.Window("Buttons",
			gowt.HBox(gowt.GrowX(),
				gowt.Button("Remember me", gowt.Check(), gowt.Ref(&remember)),
				gowt.Spacer(),
				gowt.Button("Sign in", gowt.OnClick(func() { fmt.Println("remember:", remember.Checked()) })),
			),
		)
		w.Show()
	})
}

// A field that takes the free width, a password field and a search field with a hint.
func ExampleText() {
	_ = gowt.Run(func(app *gowt.App) {
		var user, find *gowt.TextView
		w := app.Window("Fields",
			gowt.Grid(2, gowt.GrowX(),
				gowt.Label("User"), gowt.Text(gowt.GrowX(), gowt.Size(200, 0), gowt.Ref(&user)),
				gowt.Label("Password"), gowt.Text(gowt.Password(), gowt.GrowX()),
				gowt.Text(gowt.Search(), gowt.GrowX(), gowt.Span(2), gowt.Ref(&find)),
			),
		)
		find.SetHint("Search")
		user.OnChange(func(s string) { fmt.Println("user:", s) })
		w.Show()
	})
}

// Nodes nest under the tree and under each other; callbacks get the node.
func ExampleTree() {
	_ = gowt.Run(func(app *gowt.App) {
		var tree *gowt.TreeView
		w := app.Window("Files", gowt.Tree(gowt.Grow(), gowt.Ref(&tree)))
		docs := tree.Node("Documents")
		docs.Node("report.pdf")
		docs.SetExpanded(true)
		tree.OnSelect(func(n *gowt.Node) { fmt.Println(n.Text()) })
		w.Show()
	})
}
