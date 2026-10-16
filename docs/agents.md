# gowt in one page

The idioms below are the ones to write. Longer explanations are in [windows-layouts.md](windows-layouts.md), [widgets.md](widgets.md) and [events.md](events.md). `go doc github.com/haiodo/gowt.<Name>` gives the full signature, and each constructor's doc ends with an "Options:" line.

## A window

```go
err := gowt.Run(func(app *gowt.App) {
	w := app.Window("Contacts")                                      // hidden until Show
	w.SetLayout(gowt.Grid{Columns: 2, Margin: gowt.Spacing.PageMargin, Spacing: gowt.Spacing.SM})

	w.Label("Name")
	name := w.Text(gowt.GrowX())                                     // field takes the free width
	w.Label("Notes")
	w.Text(gowt.Multiline(), gowt.Grow(), gowt.Size(0, 80))          // fills width and height, 80 points tall at least
	w.Separator(gowt.Span(2))                                        // line across both columns

	row := w.Panel(gowt.GrowX(), gowt.Span(2))                       // a nested container is one more Panel
	row.SetLayout(gowt.Grid{Columns: 3, Spacing: gowt.Spacing.SM})
	row.Spacer()                                                     // pushes the buttons to the right
	row.Button("Cancel", w.Close)
	row.Button("Save", func() { save(name.Text()) })

	w.Show()                                                         // packs to the content size unless SetSize was called
})
if err != nil {
	log.Fatal(err)
}
```

- A parent creates its children: `p.Button(...)`, never `NewButton(p)`. The order of the calls is the order on screen: a Grid fills its rows left to right, top to bottom.
- Call `SetLayout` before adding children. A container without a layout gives its children no size.
- `w.Show()` comes last, once the content is built.

## Placement

| On screen | Option |
|---|---|
| a field as wide as the row allows | `GrowX()` |
| an area that takes all free space | `Grow()` |
| free height only (a list in a column) | `GrowY()` |
| one cell across n columns | `Span(n)` |
| a width or height hint in points | `Size(w, h)`, 0 keeps the natural one |
| aligned right / centred in its cell | `Place(gowt.AlignEnd, gowt.AlignCenter)` |
| everything at once, also `SpanY` | `Cell(gowt.GridCell{...})` |
| empty space that pushes the next cells to the end | `p.Spacer()` |
| a horizontal or vertical line | `p.Separator()`, `p.Separator(gowt.Vertical())` |

Options add up; when two set the same thing the later one wins. They work in a `Grid`; `Size` also works in `Row` and `Form`. `InRow(RowCell{...})` belongs to `Row` and `Anchor(FormCell{...})` to `Form`. An option the parent's layout cannot use is reported to `App.OnError` with the option, the widget and the layout in the message; the widget keeps its natural size.

## Events

```go
w.Button("Delete", onDelete)                       // nil means no callback
field.OnChange(func(s string) { status.SetText(s) })
table.OnSelect(func(i int) { show(table.RowAt(i).Text(0)) })

go func() {                                        // slow work off the UI thread
	r := fetch()
	app.Async(func() { status.SetText(r) })        // back on the UI thread; a no-op after Run
}()
app.After(time.Second, tick)                       // once; call After again inside tick to repeat

app.OnError(func(err error) { status.SetText(err.Error()) }) // errors in callbacks; the app keeps running
```

Widgets belong to the UI thread: touch them only in the setup function and in callbacks. `Async`, `Sync` and `Quit` are the calls safe from any goroutine.

## Changing a widget later

Shared setters are package functions that take any widget, including windows and containers:

```go
gowt.SetEnabled(save, false)       // not save.SetEnabled
gowt.SetVisible(details, false)    // in a Grid or Row the neighbours take the space
gowt.SetFont(title, gowt.Font{Weight: gowt.Semibold})
gowt.SetForeground(warn, gowt.RGB{R: 200, G: 40, B: 40})
```

Widget-specific ones are methods: `label.SetText`, `text.Text`, `button.SetChecked`, `progress.SetValue`. Tool items and menu items have their own `SetEnabled` method.

## Tables, trees, tabs

```go
t := w.Table(gowt.Grow())
t.Column("Name", 200)
t.Column("Size", 80, gowt.Right())
t.Row("report.pdf", "1.2 MB").SetData(file)

tree := w.Tree(gowt.Grow())
docs := tree.Node("Documents")
docs.Node("report.pdf")

tabs := w.Tabs(gowt.Grow())
page := tabs.Tab("General")          // the page is a *Panel: give it a layout and children
page.SetLayout(gowt.Grid{Columns: 2})
```

Rows, nodes, tool items and tab pages are not widgets: they come from their owner. Wrappers are fresh per call, so keep per-row state with `SetData`/`Data`.

## Leaving the facade

`Unwrap()` returns the swt object for what the facade lacks (key listeners, a minimum window size, drawing into an image). Before writing `Unwrap()`, check this page and `go doc`: mouse events on `Canvas`, cell clicks on `Table`/`Tree`, `SetVisible`, `SetEnabled`, `TrimHead` and `look.DragsWindow` exist already.

## Option to widgets

The common options work on every control: `Disabled()`, `Tooltip(s)`, `Background(rgb)`, `Custom(f)`, and the placement options above. Items (`Tab`, `ToolBar.Item`, `Menu.Item`, `Column`) and `app.Window` read only the style options listed for them and ignore the rest. A style option on a widget not listed here is ignored.

| Option | Widgets |
|---|---|
| `Check()` | Button, Table, Tree, ToolBar.Item, Menu.Item |
| `Radio()` | Button, ToolBar.Item, Menu.Item |
| `Multiline()`, `Password()`, `Search()` | Text |
| `ReadOnly()` | Text, Combo, Spinner, StyledText |
| `Border()` | Panel, Label, Text, Canvas, Scroll, CTabs, Spinner, StyledText |
| `NoBorder()`, `MultiSelect()` | List, Table, Tree |
| `CellSelect()` | Tree |
| `Simple()` | Combo |
| `Vertical()` | Scale, Slider, Progress, Split, Sash, Separator |
| `Smooth()`, `Indeterminate()` | Progress |
| `Flat()` | ToolBar |
| `Wrap()` | Label, StyledText, ToolBar |
| `Scrollbars()` | StyledText |
| `Bottom()` | Tabs, CTabs |
| `Closable()` | CTabs.Tab |
| `DropDown()` | DateTime, ToolBar.Item |
| `AsTime()`, `AsCalendar()` | DateTime |
| `Right()`, `Center()` | Table.Column, Tree.Column, Label |
| `Grow()`, `GrowX()`, `GrowY()`, `Span(n)`, `Place(h, v)`, `Cell(...)` | any control in a Grid |
| `Size(w, h)` | any control in a Grid, Row or Form |
| `InRow(...)` / `Anchor(...)` | any control in a Row / a Form |
