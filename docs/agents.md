# gowt in one page

The idioms below are the ones to write. Longer explanations are in [windows-layouts.md](windows-layouts.md), [widgets.md](widgets.md) and [events.md](events.md). `go doc github.com/haiodo/gowt.<Name>` gives the full signature, and each constructor's doc ends with an "Options:" line.

## A window

```go
err := gowt.Run(func(app *gowt.App) {
	var name *gowt.TextView                                  // filled by Ref when the field is created
	w := app.Window("Contacts")                              // hidden until Show; children stack top to bottom
	w.Add(
		gowt.HBox(gowt.GrowX(),
			gowt.Label("Contacts"),
			gowt.Spacer(),                                   // pushes the button to the right
			gowt.Button("Clear", func() { name.SetText("") }),
		),
		gowt.Grid(2, gowt.GrowX(),                           // a form: label | field per row
			gowt.Label("Name"), gowt.Text(gowt.GrowX(), gowt.Ref(&name)),
			gowt.Label("Notes"), gowt.Text(gowt.Multiline(), gowt.Grow(), gowt.Size(0, 80)),
		),
		gowt.HBox(gowt.GrowX(),
			gowt.Spacer(),
			gowt.Button("Cancel", w.Close),
			gowt.Button("Save", func() { save(name.Text()) }),
		),
	)
	name.SetHint("Full name")                                // after the Add that created it
	w.Show()                                                 // packs to the content size unless SetSize was called
})
if err != nil {
	log.Fatal(err)
}
```

- A constructor (`gowt.Label`, `gowt.Button`, `gowt.HBox`, ...) returns a description, a `gowt.Option`. The widget is created when the description is passed to its parent: to a container constructor, to `app.Window(title, ...)` or to `container.Add(...)`. The nesting of the calls is the nesting on screen.
- `gowt.Ref(&x)` stores the created widget in `x` (`var x *gowt.TextView`; the widget types end in `View`, containers are `*gowt.Panel`). Call `x`'s methods after the `Add` or `app.Window` call that created it.
- `gowt.Init(func(x *gowt.ComboView) { x.Select(0) })` runs on the widget as soon as it is created: for a helper function that returns a description and must set up its widget.
- A child whose callback uses `w` goes into `w.Add(...)`, after `w` exists. Children that do not can go straight into `app.Window("Title", children...)`.
- `w.Show()` comes last, once the content is built.

## Containers

| On screen | Constructor |
|---|---|
| a column | `gowt.VBox(children...)`; a Window, Group, Tab page and Scroll content are columns too |
| a row | `gowt.HBox(children...)`, one column per child |
| a form or a table of cells | `gowt.Grid(columns, children...)` |
| a row that wraps | `gowt.Flow(children...)` |
| children sharing the space equally, or one filling it | `gowt.Fill(children...)`, `Fill(gowt.Vertical(), ...)` |
| edges attached to siblings | `gowt.Form(children...)` with `Anchor(gowt.FormCell{...})` |
| one page at a time | `gowt.Pages(children...)`, `pages.Show(i)`, `pages.Current()` |
| tabs | `gowt.Tabs(gowt.Tab("General", children...), gowt.Tab("Advanced", ...))` |
| a titled frame | `gowt.Group("Options", children...)` |
| a scrolled column | `gowt.Scroll(children...)` |
| panes with draggable sashes | `gowt.Split(children...)`, `split.SetWeights(1, 3)` |
| settings rows on a card | `gowt.Section(rows...)` |

`Gap(points)` is the space between children (default `Spacing.SM`), `Padding(points)` the space from the container's edge (default `Spacing.PageMargin` for a Window and a Tab page, `Spacing.SM` for a Group, 0 elsewhere).

## Placement

| On screen | Option |
|---|---|
| a field as wide as the row allows | `GrowX()` |
| an area that takes all free space | `Grow()` |
| free height only (a list in a column) | `GrowY()` |
| one cell across n columns of a Grid | `Span(n)` |
| a width or height hint in points | `Size(w, h)`, 0 keeps the natural one |
| aligned right / centred in its cell | `Place(gowt.AlignEnd, gowt.AlignCenter)` |
| everything at once, also `SpanY` | `Cell(gowt.GridCell{...})` |
| empty space that pushes the next children to the end | `gowt.Spacer()` |
| a horizontal or vertical line | `gowt.Separator()`, `gowt.Separator(gowt.Vertical())` |

Options add up; when two set the same thing the later one wins. They work in a VBox, HBox and Grid; `Size` also works in `Flow` and `Form`, `Anchor(FormCell{...})` only in `Form`. An option the parent cannot use is reported to `App.OnError` with the option, the widget and the layout in the message; the widget keeps its natural size.

## Events

```go
gowt.Button("Delete", onDelete)                    // nil means no callback
var field *gowt.TextView
p.Add(gowt.Text(gowt.Ref(&field)))
field.OnChange(func(s string) { status.SetText(s) })
table.OnSelect(func(i int) { show(table.RowAt(i).Text(0)) })

go func() {                                        // slow work off the UI thread
	r := fetch()
	app.Async(func() { status.SetText(r) })        // back on the UI thread; a no-op after Run
}()
app.After(time.Second, tick)                       // once
stop := app.Every(time.Second, tick)               // repeats until stop()
file.Item("&Save", save, gowt.Shortcut("Cmd+S"))   // Cmd+S binds the key and shows it; Ctrl+S on Windows and Linux

app.OnError(func(err error) { status.SetText(err.Error()) }) // errors in callbacks and misused options; the app keeps running
```

`Button`, `Switch`, `Link` and `Canvas` take their main callback as an argument. Other events are methods: `Ref` the widget and call them after the `Add`, or call them in `Init`.

Widgets belong to the UI thread: touch them only in the setup function and in callbacks. `Async`, `Sync` and `Quit` are the calls safe from any goroutine.

## Changing a widget later

Every widget, windows and containers included, has the same common methods (they are also the methods of the `Widget` interface):

```go
save.SetEnabled(false)
details.SetVisible(false)          // in a VBox, HBox, Grid or Flow the neighbours take the space
title.SetFont(gowt.Font{Weight: gowt.Semibold})
warn.SetForeground(gowt.RGB{R: 200, G: 40, B: 40})
row.SetBackground(gowt.RGB{R: 250, G: 250, B: 250})
name.SetTooltip("Full name")
name.Focus()
row.Dispose()                      // removes it; the parent is laid out again
row.OnDispose(stop)                // runs when row is disposed or its window closes
save.BindEnabled(canSave)          // rx: follows an rx.Reader[bool]; BindVisible likewise
```

`Enabled()` and `Visible()` read the state back. Widget-specific setters are methods too: `label.SetText`, `text.Text`, `button.SetChecked`, `progress.SetValue`. Tool items and menu items have their own `SetEnabled`.

A container has `Add(children...)` (create more children at its end, also after Show), `Relayout()` (lay the children out again after a text got longer) and `Clear()` (dispose every child, for content that is rebuilt). `table.Batch(f)` and `tree.Batch(f)` repaint once after many row changes in `f`.

## The window

`w.SetSize(800, 600)` fixes the start size in points, `w.SetMinSize(400, 300)` the smallest one the user can drag to, and `app.ScreenSize()` gives the area a window can take on the primary monitor.

## Tables, trees, tool bars

```go
var t *gowt.TableView
var tree *gowt.TreeView
var tb *gowt.ToolBarView
w.Add(
	gowt.ToolBar(gowt.Ref(&tb)),
	gowt.Table(gowt.Grow(), gowt.Ref(&t)),
	gowt.Tree(gowt.Grow(), gowt.Ref(&tree)),
)
tb.Item("Open", open)
t.Column("Name", 200)
t.Column("Size", 80, gowt.Right())
t.Row("report.pdf", "1.2 MB").SetData(file)
docs := tree.Node("Documents")
docs.Node("report.pdf")
```

Rows, nodes, columns, tool items and menu items are not widgets: they come from their owner's methods. Wrappers are fresh per call, so keep per-row state with `SetData`/`Data`.

## Leaving the facade

`Unwrap()` returns the swt object for what the facade lacks (key listeners outside menus, drawing into an image). Before writing `Unwrap()`, check this page and `go doc`: mouse events on `Canvas`, cell clicks on `Table`/`Tree`, the common methods above, `Relayout`, `Batch`, `SetMinSize`, `Shortcut`, `TrimHead` and `look.DragsWindow` exist already.

## Option to widgets

The common options work on every widget: `Disabled()`, `Tooltip(s)`, `Background(rgb)`, `Custom(f)`, `Ref(&x)`, `Init(f)`, and the placement options above. `Gap`, `Padding` and children work on containers. Items (`Tab`, `ToolBar.Item`, `Menu.Item`, `Column`) read only the style options listed for them. Any other option is reported to `App.OnError` and the widget is created without it.

| Option | Widgets |
|---|---|
| `Check()` | Button, Table, Tree, ToolBar.Item, Menu.Item |
| `Shortcut(keys)` | Menu.Item |
| `Radio()` | Button, ToolBar.Item, Menu.Item |
| `Multiline()`, `Password()`, `Search()` | Text |
| `ReadOnly()` | Text, Combo, Spinner, StyledText |
| `Border()` | VBox, HBox, Grid, Flow, Fill, Form, Pages, Label, Text, Canvas, Scroll, CTabs, Spinner, StyledText |
| `NoBorder()`, `MultiSelect()` | List, Table, Tree |
| `CellSelect()` | Tree |
| `Simple()` | Combo |
| `Vertical()` | Scale, Slider, Progress, Split, Sash, Separator, Flow, Fill |
| `Smooth()`, `Indeterminate()` | Progress |
| `Flat()` | ToolBar |
| `Wrap()` | Label, StyledText, ToolBar |
| `Scrollbars()` | StyledText |
| `Bottom()` | Tabs, CTabs |
| `Closable()` | Tab in a CTabs |
| `DropDown()` | DateTime, ToolBar.Item |
| `AsTime()`, `AsCalendar()` | DateTime |
| `Right()`, `Center()` | Table.Column, Tree.Column, Label |
| `Gap(n)`, `Padding(n)` | Window, VBox, HBox, Grid, Flow, Fill, Form, Group, Tab, Scroll |
| `Grow()`, `GrowX()`, `GrowY()`, `Span(n)`, `Place(h, v)`, `Cell(...)` | any widget in a VBox, HBox or Grid |
| `Size(w, h)` | any widget in a VBox, HBox, Grid, Flow or Form |
| `Anchor(...)` | any widget in a Form |
