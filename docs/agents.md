# gowt in one page

The idioms below are the ones to write. Longer explanations are in [windows-layouts.md](windows-layouts.md), [widgets.md](widgets.md) and [events.md](events.md). `go doc github.com/haiodo/gowt.<Name>` gives the full signature, and each constructor's doc ends with an "Options:" line.

## A window

```go
err := gowt.Run(func(app *gowt.App) {
	name := rx.New("")                                       // state shared by the widgets below
	w := app.Window("Contacts")                              // hidden until Show; children stack top to bottom
	w.Add(
		gowt.HBox(gowt.GrowX(),
			gowt.Label("Contacts"),
			gowt.Spacer(),                                   // pushes the button to the right
			gowt.Button("Clear", gowt.OnClick(func() { name.Set("") }))),
		gowt.Grid(2, gowt.GrowX(),                           // a form: label | field per row
			gowt.Label("Name"), gowt.Text(gowt.GrowX(), gowt.Hint("Full name"), gowt.Bind(name)),
			gowt.Label("Notes"), gowt.Text(gowt.Multiline(), gowt.Grow(), gowt.Size(0, 80))),
		gowt.HBox(gowt.GrowX(),
			gowt.Spacer(),
			gowt.Button("Cancel", gowt.OnClick(w.Close)),
			gowt.Button("Save", gowt.OnClick(func() { save(name.Get()) }),
				gowt.BindEnabled(rx.Compute(func() bool { return name.Get() != "" })))))
	w.Show()                                                 // packs to the content size unless SetSize was called
})
if err != nil {
	log.Fatal(err)
}
```

- A constructor (`gowt.Label`, `gowt.Button`, `gowt.HBox`, ...) returns a description, a `gowt.Option`. The widget is created when the description is passed to its parent: to a container constructor, to `app.Window(title, ...)` or to `container.Add(...)`. The nesting of the calls is the nesting on screen.
- Everything about a widget goes into its option list: placement, style, events (`OnClick`, `OnChange`, `OnSelect`, `OnActivate`), initial state (`Hint`, `Checked`, `Selected`, `Value`) and rx bindings (`Bind`, `BindText`, `BindEnabled`, `BindVisible`).
- Data the screen shares lives in `rx` state (`rx.New`, `rx.Compute`); widgets bind to it, callbacks `Set` it. No variable per widget.
- Rows of a Table or Tree that show rx state are declared with `Rows`, `Nodes` (and `Items` for a List or Combo), see below. Imperative setup (columns, fixed rows, `Split.SetWeights`, tool bar items) goes in `gowt.Init(func(t *gowt.TableView) {...})`, which runs once the widget and its children exist. Widget types end in `View`; containers are `*gowt.Panel`.
- `gowt.Ref(&x)` (`var x *gowt.PagesView`) is for a widget a later callback calls into (`pages.Show`, `canvas.Redraw`) when no state fits. Ref is set by the `Add` or `app.Window` call that creates the widget.
- A child whose callback uses `w` goes into `w.Add(...)`, after `w` exists. Children that do not can go straight into `app.Window("Title", children...)`.
- Closing parens go on the line of the last child; the indentation shows the nesting.
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
| moved right or down inside its cell | `Indent(x, y)` |
| aligned right / centred in its cell | `Place(gowt.AlignEnd, gowt.AlignCenter)` |
| everything at once, also `SpanY` | `Cell(gowt.GridCell{...})` |
| empty space that pushes the next children to the end | `gowt.Spacer()` |
| a horizontal or vertical line | `gowt.Separator()`, `gowt.Separator(gowt.Vertical())` |

Options add up; when two set the same thing the later one wins. They work in a VBox, HBox and Grid; `Size` also works in `Flow` and `Form`, `Anchor(FormCell{...})` only in `Form`. An option the parent cannot use is reported to `App.OnError` with the option, the widget and the layout in the message; the widget keeps its natural size.

## Events and state

```go
gowt.Button("Delete", gowt.OnClick(onDelete))
gowt.Text(gowt.OnChange(func(s string) { query.Set(s) }), gowt.OnActivate(search))
gowt.List(names, gowt.OnSelect(func(i int) { show(names[i]) }), gowt.OnActivate(open))
gowt.Tree(gowt.OnSelect(func(n *gowt.Node) { show(n.Text()) }))
gowt.Switch("Wi-Fi", gowt.Checked(true), gowt.OnChange(func(on bool) { wifi(on) }))
gowt.Scale(0, 100, gowt.Value(30), gowt.OnChange(func(v int) { volume(v) }))

on := rx.New(true)                                 // Svelte-style state
gowt.Switch("Enable", gowt.Bind(on))               // two-way: a click sets on, on.Set(false) unchecks
gowt.Button("Remember", gowt.Check(), gowt.BindEnabled(on))
gowt.Label("", gowt.BindText(rx.Compute(func() string { return fmt.Sprint("on: ", on.Get()) })))

go func() {                                        // slow work off the UI thread
	r := fetch()
	status.Post(r)                                 // rx state from any goroutine; or app.Async(func() { ... })
}()
app.After(time.Second, tick)                       // once
stop := app.Every(time.Second, tick)               // repeats until stop()
file.Item("&Save", save, gowt.Shortcut("Cmd+S"))   // Cmd+S binds the key and shows it; Ctrl+S on Windows and Linux
w.OnKey("Cmd+[", back)                             // a key for the whole window, without a menu item; text fields keep their editing keys

app.OnError(func(err error) { status.Set(err.Error()) }) // errors in callbacks and misused options; the app keeps running
```

One event name per concept: the handler's parameter is what the widget reports (`OnChange(func(string))` for a Text, `OnChange(func(int))` for a Scale). `Bind` takes `*rx.State[string]` (Text, Combo text), `[bool]` (Switch, Check button) or `[int]` (Scale, Slider, Spinner, Combo or List index). A binding lives as long as its widget. `gowt.Watch(w, reader, apply)` binds anything else. The methods of the same names (`text.OnChange(f)`, `table.OnSelect(f)`) add handlers after creation; menu and tool items take their callback as an argument.

Widgets belong to the UI thread: touch them only in the setup function and in callbacks. `Async`, `Sync`, `Quit` and `State.Post` are the calls safe from any goroutine.

## Changing a widget later

Every widget, windows and containers included, has the same common methods (they are also the methods of the `Widget` interface):

```go
save.SetEnabled(false)
details.SetVisible(false)          // in a VBox, HBox, Grid or Flow the neighbours take the space
title.SetFont(gowt.Font{Weight: gowt.Semibold})
warn.SetForeground(gowt.RGB{R: 200, G: 40, B: 40})
row.SetBackground(gowt.Adaptive{Light: gowt.RGB{R: 250, G: 250, B: 250}, Dark: gowt.RGB{R: 40, G: 40, B: 44}})
name.SetTooltip("Full name")
name.Focus()
row.Dispose()                      // removes it; the parent is laid out again
row.OnDispose(stop)                // runs when row is disposed or its window closes
```

`Enabled()` and `Visible()` read the state back. A colour is an `RGB` or an `Adaptive{Light, Dark}`, which the widget switches with the system theme by itself; a background on a container also shows behind its labels and boxes. A `Font` with `Scale` and no `Size` is sized from the system font (`Font{Scale: 1.2, Weight: gowt.Semibold}` for a heading), and widgets with equal fonts share one native font. Widget-specific setters are methods too: `label.SetText`, `text.Text`, `button.SetChecked`, `progress.SetValue`. Tool items and menu items have their own `SetEnabled`.

A container has `Add(children...)` (create more children at its end, also after Show), `Relayout()` (lay the children out again after a text got longer) and `Clear()` (dispose every child, for content that is rebuilt). `table.Batch(f)` and `tree.Batch(f)` repaint once after many row changes in `f`.

## The window

`w.SetSize(800, 600)` fixes the start size in points, `w.SetMinSize(400, 300)` the smallest one the user can drag to, and `app.ScreenSize()` gives the area a window can take on the primary monitor.

## Tables, trees, tool bars

```go
var t *gowt.TableView                              // OnSelect reads the rows later
w.Add(
	gowt.ToolBar(gowt.Init(func(tb *gowt.ToolBarView) { tb.Item("Open", open) })),
	gowt.Table(gowt.Grow(), gowt.Ref(&t), gowt.OnSelect(func(i int) { show(t.RowAt(i).TextAt(0)) }),
		gowt.Init(func(t *gowt.TableView) {
			t.Column("Name", 200)
			t.Column("Size", 80, gowt.Right())
			t.Row("report.pdf", "1.2 MB").SetData(file)
		})),
	gowt.Tree(gowt.Grow(), gowt.Init(func(tree *gowt.TreeView) {
		docs := tree.Node("Documents")
		docs.Node("report.pdf")
	})))
```

Rows, nodes, columns, tool items and menu items are not widgets: they come from their owner's methods. Wrappers are fresh per call, so keep per-row state with `SetData`/`Data`.

Rows that show data in rx state are declared, as Svelte's `{#each list as item (key)}`: one row per element, matched by key when the state changes. Only new rows and rows whose element is no longer `==` run `fill`; the rest, selection and expansion included, stay as they are. The element type must be comparable: a struct of strings and numbers, or a pointer (compared by identity).

```go
procs := rx.NewFunc([]proc(nil), nil)               // set or Post a new slice per poll
gowt.Table(gowt.Grow(), gowt.Init(func(t *gowt.TableView) { t.Column("Name", 200); t.Column("CPU", 60) }),
	gowt.Rows(procs, func(p proc) int { return p.pid }, func(r *gowt.TableRow, p proc) {
		r.SetTextAt(0, p.name)
		r.SetTextAt(1, p.cpu)
	}))
gowt.Tree(gowt.Grow(), gowt.Nodes(dirs, dir.path, kidsOf, func(n *gowt.Node, d dir, created bool) {
	n.SetText(d.name)
	if created {
		n.SetExpanded(true)                         // initial state only: a node the user collapsed stays collapsed
	}
}))
gowt.List(nil, gowt.Items(names))                  // *rx.State[[]string]; also Combo
```

The binding owns the rows: no `Row`, `Remove` or `Clear` by hand on that widget. `kidsOf(d dir) []dir` returns a node's children (nil for a flat tree); `fill` runs after them, and `created` is false when only the element changed. A key that moves among its siblings gets its row made again (fill runs, selection and expansion are kept). Columns go in `Init`; `Rows` fills after it whatever the order of the options.

## Leaving the facade

`Unwrap()` returns the swt object for what the facade lacks (drawing into an image, key events of one widget). Before writing `Unwrap()`, check this page and `go doc`: mouse events on `Canvas`, cell clicks on `Table`/`Tree`, the common methods above, `Relayout`, `Batch`, `SetMinSize`, `Shortcut`, `Window.OnKey`, `Rows`/`Nodes`/`Items`, `GC.SetFont`, `Font.Scale`, `App.SystemFont`, `Adaptive`, `Indent`, `App.Flush`, `TrimHead` and `look.DragsWindow` exist already. The packages `jface` and `browser` are the SWT port for porting Java code; new screens do not use them.

## Option to widgets

The common options work on every widget: `Disabled()`, `Tooltip(s)`, `Background(paint)`, `Foreground(paint)`, `Typeface(font)`, `Custom(f)`, `Ref(&x)`, `Init(f)`, `BindEnabled(r)`, `BindVisible(r)`, and the placement options above. `Gap`, `Padding` and children work on containers. Items (`Tab`, `ToolBar.Item`, `Menu.Item`, `Column`) read only the style options listed for them. Any other option is reported to `App.OnError` and the widget is created without it.

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
| `OnClick(f)` | Button `func()`, Link `func(href string)` |
| `OnChange(f)` | Text, Combo, StyledText `func(string)`; Switch, Check/Radio Button `func(bool)`; Scale, Slider, Spinner `func(int)`; DateTime `func(time.Time)` |
| `OnSelect(f)` | Combo, List, Table, Tabs, CTabs `func(int)`; Tree `func(*Node)` |
| `OnActivate(f)` | Text `func()`; List, Table `func(int)`; Tree `func(*Node)` |
| `Hint(s)` | Text |
| `Checked(b)` | Switch, Check/Radio Button |
| `Selected(i)` | Combo, List, Table, Tabs, CTabs |
| `Value(n)` | Scale, Slider, Spinner, Progress |
| `Bind(state)` | Text, Combo (`string`); Switch, Check Button (`bool`); Scale, Slider, Spinner, Combo, List (`int`) |
| `BindText(r)` | Label, Button, Link |
| `Rows(list, key, fill)` | Table |
| `Nodes(list, key, children, fill)` | Tree |
| `Items(list)` | List, Combo |
| `Gap(n)`, `Padding(n)` | Window, VBox, HBox, Grid, Flow, Fill, Form, Group, Tab, Scroll |
| `Grow()`, `GrowX()`, `GrowY()`, `Span(n)`, `Place(h, v)`, `Indent(x, y)`, `Cell(...)` | any widget in a VBox, HBox or Grid |
| `Size(w, h)` | any widget in a VBox, HBox, Grid, Flow or Form |
| `Anchor(...)` | any widget in a Form |
