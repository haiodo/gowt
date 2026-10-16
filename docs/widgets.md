# Widgets

Example: [`examples/widgets`](../examples/widgets) (three tabs: input, lists, ranges).

Every widget is made by its parent: `parent.Label(text, opts...)`, `parent.Button(text, onClick, opts...)`. The parent can be a `Window`, `Panel`, `Group`, `Split` or `CoolBar`, or the page `Panel` returned by `Tabs.Tab` or `Pages.Page`. Constructors return a small wrapper with plain Go methods: no `Get` prefix, `int` for sizes, `time.Time` for dates.

## Options

Options go last in the call. Some set a style that must be known at creation, the rest adjust the widget afterwards. A style option on a widget it does not apply to is silently ignored.

| Option | Meaning |
|---|---|
| `Check()`, `Radio()` | button kind; `Check` also makes Table/Tree rows and ToolBar items checkable |
| `Multiline()`, `Password()`, `ReadOnly()` | Text (and `ReadOnly` for Combo) |
| `Border()`, `Flat()`, `Wrap()` | frame; flat tool bar; Label and StyledText text wrapped into lines, tool bar items wrapped into rows |
| `Vertical()` | Scale, Slider, Progress, Sash, Split orientation |
| `MultiSelect()` | List, Table, Tree |
| `Disabled()`, `Tooltip(s)`, `Background(RGB)` | any widget |
| `NoBorder()` | drops the frame of List, Table, Tree |
| `CellSelect()` | Tree selects only the node text instead of the whole row |
| `Search()` | Text as a search field with a magnifier icon and a clear button |
| `Grow()`, `GrowX()`, `GrowY()`, `Span(n)`, `Size(w, h)`, `Place(h, v)`, `Cell(GridCell)`, `InRow(RowCell)`, `Anchor(FormCell)` | position in the parent's layout, see [windows-layouts.md](windows-layouts.md) |
| `Right()`, `Center()` | Table column alignment |
| `Bottom()` | Tabs and CTabs: tabs below the pages |
| `Closable()` | CTabs tab with a close button |
| `AsTime()`, `AsCalendar()`, `DropDown()` | DateTime variants |
| `Simple()`, `Smooth()`, `Indeterminate()` | Combo list always shown; unsegmented progress; progress without a value |

## Catalog

| Widget | Constructor | Notes |
|---|---|---|
| Label | `Label(text)` | `SetText`, `SetImage` |
| Spacer, Separator | `Spacer()`, `Separator()` | an empty cell that takes the free width of a Grid; a horizontal line (`Vertical()` for a vertical one). Both return a `*Label` |
| Switch | `Switch(text, onChange)` | on/off toggle; `Checked`, `SetChecked`, `SetText`, `OnChange(func(bool))`. A check box on every OS (SWT creates no NSSwitch or GtkSwitch) |
| Button | `Button(text, onClick)` | push by default; `Check()`/`Radio()`; `Checked`, `SetChecked`, `SetImage`, `OnClick` |
| Text | `Text()` | `Text`, `SetText`, `Append`, `Lines`, `TrimHead(keep)`, `ScrollToEnd`, `SetHint`, `OnChange(func(string))`, `OnActivate(func())` (Enter) |
| Link | `Link(html, onClick)` | text with `<a href="...">`; `onClick` gets the href |
| Combo | `Combo(items)` | `Index` (-1 if none), `Items`, `Select`, `Text`, `OnSelect(func(int))`, `OnChange(func(string))` |
| List | `List(items)` | `Add`, `Remove`, `Clear`, `Index`, `Selected`, `SetSelection`, `OnSelect`, `OnActivate` (double click) |
| Table | `Table()` | `Column(title, width)`, `Row(cells...)` returns `*TableRow` (`SetText(col, s)`, `SetData`, `Data`, `SetChecked`, `SetImage(col, img)`), `RowAt(i)`, selection as for List, `ShowHeader`, `ShowLines`, `ShowRowLines`, `SetSort(col, dir)`, `CellAt(x, y)`, `OnCellClick`; `Column` also has `SetResizable` |
| Tree | `Tree()` | `Node(text)` on the tree and on a node; callbacks get `*Node` (`Text`, `SetExpanded`, `Children`, `SetData`, `Remove`); `Roots`, `Selected`, `Select(node)`, `Clear`; `OnSelect`, `OnActivate`, `OnExpand`, `OnCollapse`. Columns as in Table: `Column(title, width, opts...)` returns `*TreeColumn` (the first holds the tree), `ShowHeader`, `ShowRowLines`, `SetSort`, `CellAt`, `OnCellClick`; per-column cells `Node.TextAt(col)`, `SetTextAt`, `SetImageAt` |
| Scale, Slider, Spinner | `Scale(min, max, value)` | `Value`, `SetValue`, `OnChange(func(int))` |
| Progress | `Progress(max)` | `SetValue`, `SetMax`, `Value` |
| DateTime | `DateTime()` | `Value`/`SetValue` as `time.Time`, `OnChange` |
| Group | `Group(title)` | titled container |
| Section | `Section()` | grouped "inset" card as in macOS System Settings; `Row()` adds a row `*Panel` (no layout until you `SetLayout`), with a separator line before every row but the first. See [theme-look.md](theme-look.md#finder-and-system-settings-look) |
| Tabs, CTabs | `Tabs()`, `CTabs()` | `Tab(title)` returns the page `*Panel`; `Select`, `Index`, `OnSelect`; CTabs adds `OnClose(func(index int) bool)` (false keeps the tab) |
| Pages | `Pages()` | one page at a time, no tabs: `Page()` returns the page `*Panel` (the first is shown), `Show(i)`, `Current()` |
| Split | `Split()` | children are any widgets, created in order (`split.Panel()`, `split.Table()`, ...); `SetWeights(...)` sets the relative sizes |
| Scroll | `Scroll()` | build into `Content()`, then call `Fit()` |
| ToolBar | `ToolBar()` | `Item(text, onClick)` returns a `*ToolItem`; `Separator()` |
| CoolBar, Sash | `CoolBar()`, `Sash()` | `CoolBar.Add(widget)`; `Sash.OnMove` |
| Canvas | `Canvas(onPaint)` | see [graphics.md](graphics.md); `OnMouseDown`, `OnMouseUp`, `OnMouseMove` (`Mouse`: X, Y, Button, Count) |
| StyledText | `StyledText(Scrollbars())` | `Text`, `SetText`, `Selection`, `Caret`, `Append`, `Lines`, `TrimHead(keep)`, `ScrollToEnd`, `Replace`, `SetStyle(start, length, TextStyle)`, `Styles`, `OnStyle(func(line string) []StyleSpan)`, `OnChange`, `OnSelect`, `OnCaret`. Offsets are runes; each call converts to UTF-16 (O(n)). Example: [`examples/styledtext`](../examples/styledtext) |

## Any control

Every widget (`Window` and containers included) has these methods, which are also the methods of the `Widget` interface: `SetEnabled(v)` / `Enabled()` (children of a disabled container follow), `SetVisible(v)` / `Visible()` (in a Grid or Row layout the hidden control gives up its cell and the parent is laid out again; `SetVisible(true)` clears that, and a placement option applied afterwards resets it), `SetFont(Font{Name, Size, Weight, Bold, Italic})` (empty Name and zero Size keep the current ones; `Weight` is `Light`/`Regular`/`Medium`/`Semibold`/`Bold` or any 100..900 and overrides `Bold`, the system UI font gets a real weight on every OS, a named family gets the nearest face it has; the control owns the font and frees the previous one on each call), `SetForeground(RGB)`, `SetBackground(RGB)`, `SetTooltip(s)`, `Focus()`, `Dispose()` (the parent is laid out again; a second call does nothing), `OnDispose(f)`, and the rx bindings `BindVisible(r)` / `BindEnabled(r)` ([events.md](events.md)). The header font of Table and Tree is not reachable; use `Unwrap()`.

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

`Index()` is the first selected item (-1 for none) for Combo, List and Table; `Select(i)` selects one Combo item. List and Table also have `Selected()` (all selected) and `SetSelection(...)`; Combo has neither. Callbacks receive indexes. Tree callbacks receive `*Node`.

## Known limits

- There is no widget font API yet; use `Unwrap()` ([swt-direct.md](swt-direct.md)).
- Table and Tree on macOS use the older cell-based NSTableView/NSOutlineView; the view-based kind (custom cell views) is not started, see [view-based-table.md](view-based-table.md).
