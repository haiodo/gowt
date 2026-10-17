package gowt_test

import (
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

// A window with a field, a button and a label; Run returns when the window is closed.
func ExampleRun() {
	err := gowt.Run(func(app *gowt.App) {
		name, greet := rx.New(""), rx.New("")
		w := app.Window("Hello gowt",
			gowt.Text(gowt.GrowX(), gowt.Size(200, 0), gowt.Bind(name)),
			gowt.Label("", gowt.GrowX(), gowt.BindText(greet)),
			gowt.Button("Greet", gowt.OnClick(func() { greet.Set("Hello, " + name.Get() + "!") })))
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
		name := rx.New("")
		w := app.Window("Contacts")
		w.Add(
			gowt.HBox(gowt.GrowX(),
				gowt.Label("Contacts"),
				gowt.Spacer(),
				gowt.Button("Add", gowt.OnClick(func() { name.Set("") }))),
			gowt.Grid(2, gowt.GrowX(),
				gowt.Label("Name"),
				gowt.Text(gowt.GrowX(), gowt.Hint("Full name"), gowt.Bind(name))),
			gowt.HBox(gowt.GrowX(),
				gowt.Spacer(),
				gowt.Button("Cancel", gowt.OnClick(w.Close)),
				gowt.Button("Save", gowt.BindEnabled(rx.Compute(func() bool { return name.Get() != "" })),
					gowt.OnClick(func() { fmt.Println("save", name.Peek()) }))))
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
		status := rx.New("working...")
		w := app.Window("Download", gowt.Label("", gowt.BindText(status)))
		w.Show()
		go func() {
			time.Sleep(time.Second) // stands for slow work
			app.Async(func() { status.Set("done") })
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
		clock := rx.New(time.Now().Format(time.TimeOnly))
		w := app.Window("Clock", gowt.Label("", gowt.BindText(clock)))
		stop := app.Every(time.Second, func() { clock.Set(time.Now().Format(time.TimeOnly)) })
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
				gowt.Label("Step 2")))
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
				gowt.Label("Notes"), gowt.Text(gowt.Multiline(), gowt.Grow(), gowt.Size(0, 80))))
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
		var t *gowt.TableView // OnActivate reads the rows
		w := app.Window("Processes", gowt.Table(gowt.Grow(), gowt.MultiSelect(), gowt.Ref(&t),
			gowt.OnActivate(func(i int) { fmt.Println("open", t.RowAt(i).TextAt(0), t.RowAt(i).Data()) }),
			gowt.Init(func(t *gowt.TableView) { // fills the table once it exists
				t.Column("Name", 160)
				t.Column("CPU", 60, gowt.Right())
				t.Row("gowt", "1.5").SetData("pid-1")
				t.Row("docker", "12.0")
			})))
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
			}, gowt.Size(200, 100)))
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
		w := app.Window("Editor", gowt.StyledText(gowt.Scrollbars(), gowt.Grow(), gowt.Value("bold word"),
			gowt.Init(func(ed *gowt.StyledTextView) { ed.SetStyle(0, 4, gowt.TextStyle{Bold: true}) })))
		w.Show()
	})
}

// A check box and a push button pushed to the right end of their row.
func ExampleButton() {
	_ = gowt.Run(func(app *gowt.App) {
		remember := rx.New(true)
		w := app.Window("Buttons",
			gowt.HBox(gowt.GrowX(),
				gowt.Button("Remember me", gowt.Check(), gowt.Bind(remember)),
				gowt.Spacer(),
				gowt.Button("Sign in", gowt.OnClick(func() { fmt.Println("remember:", remember.Get()) }))))
		w.Show()
	})
}

// A field that takes the free width, a password field and a search field with a hint.
func ExampleText() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Fields",
			gowt.Grid(2, gowt.GrowX(),
				gowt.Label("User"), gowt.Text(gowt.GrowX(), gowt.Size(200, 0), gowt.OnChange(func(s string) { fmt.Println("user:", s) })),
				gowt.Label("Password"), gowt.Text(gowt.Password(), gowt.GrowX()),
				gowt.Text(gowt.Search(), gowt.GrowX(), gowt.Span(2), gowt.Hint("Search"))))
		w.Show()
	})
}

// Nodes nest under the tree and under each other; callbacks get the node.
func ExampleTree() {
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Files", gowt.Tree(gowt.Grow(),
			gowt.OnSelect(func(n *gowt.Node) { fmt.Println(n.Text()) }),
			gowt.Init(func(t *gowt.TreeView) {
				docs := t.Node("Documents")
				docs.Node("report.pdf")
				docs.SetExpanded(true)
			})))
		w.Show()
	})
}

// Colours for both themes and fonts relative to the system size: the card and its labels switch with
// the system theme, and the heading grows with the system font. Equal fonts share one native font.
func ExampleAdaptive() {
	var (
		surface = gowt.Adaptive{Light: gowt.RGB{R: 255, G: 255, B: 255}, Dark: gowt.RGB{R: 30, G: 31, B: 34}}
		muted   = gowt.Adaptive{Light: gowt.RGB{R: 100, G: 106, B: 118}, Dark: gowt.RGB{R: 150, G: 155, B: 164}}
		title   = gowt.Typeface(gowt.Font{Scale: 1.15, Weight: gowt.Semibold})
		small   = gowt.Typeface(gowt.Font{Scale: 0.85})
	)
	_ = gowt.Run(func(app *gowt.App) {
		w := app.Window("Status",
			gowt.VBox(gowt.Background(surface), gowt.Padding(gowt.Spacing.MD), // labels inside show surface too
				gowt.Label("Containers", title),
				gowt.Label("3 running", small, gowt.Foreground(muted))))
		w.Show()
	})
}
