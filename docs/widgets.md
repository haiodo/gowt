# Widgets

Example: [`examples/widgets`](../examples/widgets) (three tabs: input, lists, ranges).

Every widget has a constructor function that returns a description (`gowt.Option`): `gowt.Label(text, opts...)`, `gowt.Button(text, opts...)`. A description becomes a widget when it is passed to its parent: as a child of a container constructor (`gowt.VBox`, `gowt.Group`, `gowt.Split`, ...), to `app.Window(title, children...)` or to `Add(children...)` of a live container. Events, initial values and rx bindings are options too (see below), and data shared between widgets lives in `rx` state ([events.md](events.md)). For imperative work on a widget (filling a Table, `Pages.Show`, `Canvas.Redraw`) declare `var x *gowt.LabelView` and pass `gowt.Ref(&x)`, or run the code in `gowt.Init(func(x *gowt.LabelView) {...})`; both run once the widget and its children exist, `Ref` sets `x` for the code after the `Add` or `app.Window` call. The widget types carry a `View` suffix (`*gowt.LabelView`, `*gowt.TableView`, ...) and have plain Go methods: no `Get` prefix, `int` for sizes, `time.Time` for dates.

## Options

Options go last in the call. Some set a style that must be known at creation, the rest adjust the widget afterwards. A style option on a widget it does not apply to is reported to `App.OnError`; the "Options:" line of `go doc github.com/haiodo/gowt.Label` (and of every other constructor) lists what a widget takes.

| Option | Meaning |
|---|---|
| `Check()`, `Radio()` | button kind; `Check` also makes Table/Tree rows and ToolBar items checkable |
| `Multiline()`, `Password()`, `ReadOnly()` | Text (and `ReadOnly` for Combo) |
| `Border()`, `Flat()`, `Wrap()` | frame; flat tool bar; Label and StyledText text wrapped into lines, tool bar items wrapped into rows |
| `Vertical()` | Scale, Slider, Progress, Sash, Split orientation |
| `MultiSelect()` | List, Table, Tree |
| `Disabled()`, `Tooltip(s)`, `Background(RGB)` | any widget |
| `Gap(n)`, `Padding(n)` | containers: space between children and from the edge, see [windows-layouts.md](windows-layouts.md) |
| `Ref(&x)`, `Init(f)` | store the created widget in `x`; run `f` on it once it and its children exist |
| `OnClick(f)` | Button `func()`, Link `func(href string)` |
| `OnChange(f)` | Text, Combo, StyledText `func(string)`; Switch, Check/Radio Button `func(bool)`; Scale, Slider, Spinner `func(int)`; DateTime `func(time.Time)` |
| `OnSelect(f)` | Combo, List, Table, Tabs, CTabs `func(int)`; Tree `func(*Node)` |
| `OnActivate(f)` | Text `func()` (Enter); List, Table `func(int)`; Tree `func(*Node)` |
| `Hint(s)`, `Checked(b)`, `Selected(i)` | initial state: Text placeholder; Switch and Check/Radio Button; Combo, List, Table, Tabs, CTabs |
| `Value(v)` | initial value: string for Text, Combo, StyledText; int for Scale, Slider, Spinner, Progress; `time.Time` for DateTime |
| `Bind(state)` | two-way: `*rx.State[string]` Text, Combo; `[bool]` Switch, Check Button; `[int]` Scale, Slider, Spinner, Combo, List index (Progress one way) |
| `BindText(r)`, `BindEnabled(r)`, `BindVisible(r)` | Label, Button, Link text; enabled and visible state of any widget, see [events.md](events.md) |
| `NoBorder()` | drops the frame of List, Table, Tree |
| `CellSelect()` | Tree selects only the node text instead of the whole row |
| `Search()` | Text as a search field with a magnifier icon and a clear button |
| `Grow()`, `GrowX()`, `GrowY()`, `Span(n)`, `Size(w, h)`, `Place(h, v)`, `Cell(GridCell)`, `Anchor(FormCell)` | position in the parent's layout, see [windows-layouts.md](windows-layouts.md) |
| `Right()`, `Center()` | Table column alignment |
| `Bottom()` | Tabs and CTabs: tabs below the pages |
| `Closable()` | CTabs tab with a close button |
| `AsTime()`, `AsCalendar()`, `DropDown()` | DateTime variants |
| `Simple()`, `Smooth()`, `Indeterminate()` | Combo list always shown; unsegmented progress; progress without a value |

## Catalog

| Widget | Constructor | Notes |
|---|---|---|
| Label | `Label(text)` | `SetText`, `SetImage` |
| Spacer, Separator | `Spacer()`, `Separator()` | an empty cell that takes the free space of a VBox, HBox or Grid row; a horizontal line (`Vertical()` for a vertical one). Both create a `*LabelView` |
| Switch | `Switch(text, opts...)` | on/off toggle; options `Checked`, `OnChange`, `Bind`; methods `Checked`, `SetChecked`, `SetText`, `OnChange(func(bool))`. A check box on every OS (SWT creates no NSSwitch or GtkSwitch) |
| Button | `Button(text, opts...)` | push by default; `Check()`/`Radio()`; `OnClick(f)`; `Checked`, `SetChecked`, `SetImage`, `OnClick` |
| Text | `Text()` | `Text`, `SetText`, `Append`, `Lines`, `TrimHead(keep)`, `ScrollToEnd`, `SetHint` (or the option `Hint`), `OnChange(func(string))`, `OnActivate(func())` (Enter) |
| Link | `Link(html, opts...)` | text with `<a href="...">`; `OnClick(func(href string))` gets the href |
| Combo | `Combo(items)` | `Index` (-1 if none), `Items`, `Select`, `Text`, `OnSelect(func(int))`, `OnChange(func(string))` |
| List | `List(items)` | `Add`, `Remove`, `Clear`, `Index`, `Selected`, `Select(i...)`, `OnSelect`, `OnActivate` (double click) |
| Table | `Table()` | `Column(title, width)`, `Row(cells...)` returns `*TableRow` (`TextAt(col)`, `SetTextAt(col, s)`, `SetImageAt(col, img)`, `SetData`, `Data`, `SetChecked`), `RowAt(i)`, selection as for List, `ShowHeader`, `ShowLines`, `ShowRowLines`, `SetSort(col, dir)`, `CellAt(x, y)`, `OnCellClick`; `Column` also has `SetResizable` |
| Tree | `Tree()` | `Node(text)` on the tree and on a node; callbacks get `*Node` (`Text`, `SetExpanded`, `Children`, `SetData`, `Remove`); `Roots`, `Selected`, `Select(node)`, `Clear`; `OnSelect`, `OnActivate`, `OnExpand`, `OnCollapse`. Columns as in Table: `Column(title, width, opts...)` returns the same `*Column` type (the first holds the tree), `ShowHeader`, `ShowRowLines`, `SetSort`, `CellAt`, `OnCellClick`; per-column cells `Node.TextAt(col)`, `SetTextAt`, `SetImageAt` |
| Scale, Slider, Spinner | `Scale(min, max, opts...)` | start value with the option `Value(n)` (default `min`); `Value`, `SetValue`, `OnChange(func(int))` |
| Progress | `Progress(max, opts...)` | option `Value(n)`; `SetValue`, `SetMax`, `Value` |
| DateTime | `DateTime()` | `Value`/`SetValue` as `time.Time`, `OnChange` |
| Group | `Group(title, children...)` | titled container, children stacked as in a VBox |
| Section | `Section(rows...)` | grouped "inset" card as in macOS System Settings; every child is a row (usually an `HBox` or `Grid`), with a separator line before every row but the first; `SectionView.Add` appends rows. See [theme-look.md](theme-look.md#finder-and-system-settings-look) |
| Tabs, CTabs | `Tabs(tabs...)`, `CTabs(tabs...)` | children are `Tab(title, children...)` pages (`Ref` gives the page `*Panel`; `TabsView.Add` appends; `Dispose` on the page removes the tab, and a closed CTabs tab disposes its page); `Select`, `Index`, `OnSelect`; CTabs adds `OnClose(func(index int) bool)` (false keeps the tab) |
| Pages | `Pages(pages...)` | one page at a time, no tabs: every child is a page (the first is shown), `Add` appends, `Show(i)`, `Current()`; `Dispose` on a page removes it and shows a neighbour |
| Split | `Split(panes...)` | children are the panes, any widgets in order; each pane fills its part, `SetWeights(...)` sets the relative sizes. A pane takes no placement options: `Grow`, `GrowX`, `GrowY` are accepted and change nothing, `Size`, `Span`, `Place` and the rest go to `App.OnError` |
| Scroll | `Scroll(children...)` | children stack in the scrolling area as in a VBox and fit; `ScrollView.Add` adds more and fits again |
| ToolBar | `ToolBar()` | `Item(text, onClick)` returns a `*ToolItem` (call it after the `Add` that creates the tool bar); `Separator()` |
| CoolBar, Sash | `CoolBar(bands...)`, `Sash()` | each child of the cool bar becomes a band; `Sash.OnMove` |
| Canvas | `Canvas(onPaint)` | see [graphics.md](graphics.md); `OnMouseDown`, `OnMouseUp`, `OnMouseMove` (`Mouse`: X, Y, Button, Count) |
| StyledText | `StyledText(Scrollbars())` | `Text`, `SetText`, `Selection`, `Caret`, `Append`, `Lines`, `TrimHead(keep)`, `ScrollToEnd`, `Replace`, `SetStyle(start, length, TextStyle)`, `Styles`, `OnStyle(func(line string) []StyleSpan)`, `OnChange`, `OnSelect`, `OnCaret`. Offsets are runes; each call converts to UTF-16 (O(n)). Example: [`examples/styledtext`](../examples/styledtext) |

A List row has text only (`swt.List` has no item images). A row with an icon is a Table without columns, header and grid lines; in a `look.SetSidebar` sidebar it gets the List look ([theme-look.md](theme-look.md)), on macOS the selected row's text and icon in the accent colour. `icons.Get` tints the icon for the theme at call time, so set the icons again in `App.OnThemeChange`. See `examples/sidebartable`:

```go
var files *gowt.TableView
w.Add(gowt.Table(gowt.NoBorder(), gowt.Ref(&files), gowt.Init(func(t *gowt.TableView) {
	t.ShowHeader(false)
	t.ShowLines(false)
	t.Row("a.cpuprofile").SetImageAt(0, icons.Get(app, "file"))
})))
app.OnThemeChange(func(bool) { files.RowAt(0).SetImageAt(0, icons.Get(app, "file")) })
```

All images of one Table or Tree, in every row and column, get the size of the first image it was given; this is SWT's model on every platform (macOS `Table.imageBounds`, Windows one image list per table, GTK one fixed cell size). On macOS a larger or smaller image is scaled into that box; on Linux `GetImageBounds` reports it (a 28x20 badge in column 1 after a 16x16 icon in column 0 gets 16-point-wide bounds). Images of different widths, such as count badges next to icons, therefore go onto transparent canvases of one size.

`Column.SetWidth` changes that column only: SWT turns AppKit's column autoresizing off, and the sidebar look restores the widths AppKit changes when it switches the style. On macOS a Table or Tree in a `look.SetSidebar` sidebar has the source-list style, which pads both row ends (31 points on macOS 26); its client area (`Unwrap().GetClientArea()`) leaves the padding out, so columns that add up to the client width fit the visible width.

## Any control

Every widget (`Window` and containers included) has these methods, which are also the methods of the `Widget` interface: `SetEnabled(v)` / `Enabled()` (children of a disabled container follow), `SetVisible(v)` / `Visible()` (in a Grid, VBox or HBox layout the hidden control gives up its cell and the parent is laid out again; `SetVisible(true)` clears that, and a placement option applied afterwards resets it), `SetFont(Font{Name, Size, Scale, Weight, Bold, Italic})` (empty Name and zero Size keep the current ones; with Size 0, `Scale` is a multiple of the system font size, `app.SystemFont()`; `Weight` is `Light`/`Regular`/`Medium`/`Semibold`/`Bold` or any 100..900 and overrides `Bold`, the system UI font gets a real weight on every OS, a named family gets the nearest face it has; controls given equal fonts share one native font, freed after its last user), `SetForeground(paint)`, `SetBackground(paint)` (an `RGB`, or an `Adaptive{Light, Dark}` that follows the system theme; on a container the labels and boxes inside show the background too; nil restores the default), `SetTooltip(s)`, `Focus()`, `Dispose()` (the parent is laid out again; a second call does nothing), `OnDispose(f)`, and the options `BindVisible(r)` / `BindEnabled(r)` that follow an rx value ([events.md](events.md)). The header font of Table and Tree is not reachable; use `Unwrap()`.

`Table.Batch(f)` and `Tree.Batch(f)` suspend drawing while `f` runs, so many row changes repaint once.

## Cell clicks

`Table.OnCellClick` and `Tree.OnCellClick` run on every mouse press with the row (or node), the column index from the item's per-column bounds, `Button` (SWT numbering: 1 left, 2 middle, 3 right), `Count` (2 on a double click) and the modifier keys `Ctrl`, `Shift`, `Alt`, `Cmd` (a macOS Ctrl-click is Button 1 with Ctrl), so a handler ignores what it does not need. The cell comes from a hit test, not from the selection, which may not have moved yet. `CellAt(x, y)` does the same for any point, e.g. a context menu; the expander arrow maps to its cell, but a point right of the last column has `Col` -1 (empty space is never a cell). `Tree` selects whole rows so that a click in any column finds its node.

## Logs in Text and StyledText

`Append(s)` adds at the end without `SetText`; `Lines()` counts lines; `TrimHead(keep)` deletes the oldest lines (`TrimHead(0)` clears; it works on a read-only view) (Text rewrites its content, so trim when `Lines()` exceeds a cap by a margin; StyledText deletes just the head); `ScrollToEnd()` follows the tail.

## Clipboard and drag and drop

`app.Clipboard()` returns the system clipboard (created on first use, disposed when `Run` returns): `Text`/`HTML`/`Files`/`Image` return `ok` or nil, `SetText`/`SetHTML`/`SetFiles`/`SetImage` return an `error` because a busy clipboard is routine on Windows. Images are owned by the App. UI thread only. Example: [`examples/clipboard`](../examples/clipboard).

`DragFrom(widget, Drag{Text: ..., Files: ..., Image: ...})` and `DropOn(widget, Drop{Text: ..., Files: ..., Image: ..., Over: ...})` make any widget a drag source or drop target; a control has at most one of each. Images dropped on a target are owned by the App. Custom transfers go through `Unwrap()` on the returned handle. Example: [`examples/dnd`](../examples/dnd). Not verified with a real mouse on any OS.

Items are not widgets: rows, nodes, tool items and tab pages come from their owner and are used right away.

Wrappers returned by `Row`, `Node` and the `Tree.On*` callbacks are fresh each call. Keep per-item state with `SetData`/`Data`, not in a map keyed by the wrapper.

## Selection

`Index()` is the first selected item (-1 for none) for Combo, List and Table; `Select(i)` selects one Combo item. List and Table also have `Selected()` (all selected), and their `Select(i...)` takes several indexes (none clears the selection). The option `Selected(i)` sets the initial selection of Combo, List, Table, Tabs and CTabs. Callbacks receive indexes. Tree callbacks receive `*Node`.

## Known limits

- There is no widget font API yet; use `Unwrap()` ([swt-direct.md](swt-direct.md)).
- Table and Tree on macOS use the older cell-based NSTableView/NSOutlineView; the view-based kind (custom cell views) is not started, see [view-based-table.md](view-based-table.md).
