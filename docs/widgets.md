# Widgets

Example: [`examples/widgets`](../examples/widgets) (three tabs: input, lists, ranges).

Every widget is made by its parent: `parent.Label(text, opts...)`, `parent.Button(text, onClick, opts...)`. The parent can be a `Window`, `Panel`, `Group`, `Split` or `CoolBar`, or the page `Panel` returned by `Tabs.Tab`. Constructors return a small wrapper with plain Go methods: no `Get` prefix, `int` for sizes, `time.Time` for dates.

## Options

Options go last in the call. Some set a style that must be known at creation, the rest adjust the widget afterwards. A style option on a widget it does not apply to is silently ignored.

| Option | Meaning |
|---|---|
| `Check()`, `Radio()` | button kind; `Check` also makes Table/Tree rows and ToolBar items checkable |
| `Multiline()`, `Password()`, `ReadOnly()` | Text (and `ReadOnly` for Combo) |
| `Border()`, `Flat()`, `Wrap()` | frame; flat tool bar; tool bar that wraps |
| `Vertical()` | Scale, Slider, Progress, Sash, Split orientation |
| `MultiSelect()` | List, Table, Tree |
| `Disabled()`, `Tooltip(s)`, `Background(RGB)` | any widget |
| `Cell(GridCell)`, `InRow(RowCell)`, `Anchor(FormCell)` | position in the parent's layout, see [windows-layouts.md](windows-layouts.md) |
| `Right()`, `Center()` | Table column alignment |
| `Closable()` | CTabs tab with a close button |
| `AsTime()`, `AsCalendar()`, `DropDown()` | DateTime variants |
| `Simple()`, `Smooth()`, `Indeterminate()` | Combo list always shown; unsegmented progress; progress without a value |

## Catalog

| Widget | Constructor | Notes |
|---|---|---|
| Label | `Label(text)` | `SetText`, `SetImage` |
| Button | `Button(text, onClick)` | push by default; `Check()`/`Radio()`; `Checked`, `SetChecked`, `SetImage`, `OnClick` |
| Text | `Text()` | `Text`, `SetText`, `SetHint`, `OnChange(func(string))`, `OnActivate(func())` (Enter) |
| Link | `Link(html, onClick)` | text with `<a href="...">`; `onClick` gets the href |
| Combo | `Combo(items)` | `Index` (-1 if none), `Items`, `Select`, `Text`, `OnSelect(func(int))`, `OnChange(func(string))` |
| List | `List(items)` | `Add`, `Remove`, `Clear`, `Index`, `Selected`, `SetSelection`, `OnSelect`, `OnActivate` (double click) |
| Table | `Table()` | `Column(title, width)`, `Row(cells...)` returns `*TableRow` (`SetText(col, s)`, `SetData`, `Data`, `SetChecked`, `SetImage(col, img)`), `RowAt(i)`, selection as for List, `ShowHeader`, `ShowLines` |
| Tree | `Tree()` | `Node(text)` on the tree and on a node; callbacks get `*Node` (`Text`, `SetExpanded`, `Children`, `SetData`, `Remove`); `OnSelect`, `OnActivate`, `OnExpand`, `OnCollapse` |
| Scale, Slider, Spinner | `Scale(min, max, value)` | `Value`, `SetValue`, `OnChange(func(int))` |
| Progress | `Progress(max)` | `SetValue`, `SetMax`, `Value` |
| DateTime | `DateTime()` | `Value`/`SetValue` as `time.Time`, `OnChange` |
| Group | `Group(title)` | titled container |
| Tabs, CTabs | `Tabs()`, `CTabs()` | `Tab(title)` returns the page `*Panel`; `Select`, `Index`, `OnSelect`; CTabs adds `OnClose(func(index int) bool)` (false keeps the tab) |
| Split | `Split()` | children are panels in order; `SetWeights(...)` sets the sizes |
| Scroll | `Scroll()` | build into `Content()`, then call `Fit()` |
| ToolBar | `ToolBar()` | `Item(text, onClick)` returns a `*ToolItem`; `Separator()` |
| CoolBar, Sash | `CoolBar()`, `Sash()` | `CoolBar.Add(widget)`; `Sash.OnMove` |
| Canvas | `Canvas(onPaint)` | see [graphics.md](graphics.md) |

Items are not widgets: rows, nodes, tool items and tab pages come from their owner and are used right away.

Wrappers returned by `Row`, `Node` and the `Tree.On*` callbacks are fresh each call. Keep per-item state with `SetData`/`Data`, not in a map keyed by the wrapper.

## Selection

For Combo, List and Table: `Index()` is the first selected item (-1 for none), `Selected()` all selected, `SetSelection(...)` sets them. Callbacks receive indexes. Tree callbacks receive `*Node`.

## Known limits

- There is no widget font API yet; use `Unwrap()` ([swt-direct.md](swt-direct.md)).
- Table and Tree on macOS use the older cell-based NSTableView/NSOutlineView; the view-based kind (custom cell views) is not started, see [view-based-table.md](view-based-table.md).
