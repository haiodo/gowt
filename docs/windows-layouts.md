# Windows and layouts

Example: [`examples/layouts`](../examples/layouts).

## Windows

```go
w := app.Window("Title")   // hidden until Show
w.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
// ... create children ...
w.SetSize(480, 320)        // optional; without it Show packs the window to its content
w.Show()
```

- `app.Window(title, opts...)` creates a top-level window. It is hidden until `Show`.
- `Show` sizes the window to its content unless `SetSize` was called.
- `SetTitle`, `Close` (as if the user closed it) and `OnClose(func() bool)` exist; returning false from `OnClose` keeps the window open.
- A window is also a container: it has the same child constructors as `Panel` (`w.Button`, `w.Label`, `w.Panel`, ...).
- More than one window: call `app.Window` again. `Run` returns when the last one is closed.

## Panels

`w.Panel()` is a plain container with its own layout. Nest panels to build any arrangement. `Group(title)` is a panel with a titled frame; `Tabs`, `CTabs`, `Split` and `Scroll` are containers too (see [widgets.md](widgets.md)).

## Layouts

A layout is a value you pass to `SetLayout`. The zero value of every field means "none" or "natural size".

| Layout | Fields | Per-child option | Use |
|---|---|---|---|
| `Fill` | `Vertical` | - | children share the space equally in a row or column |
| `Grid` | `Columns` (0 means 1), `EqualWidth`, `Margin`, `Spacing` | `Cell(GridCell{...})` | forms and most dialogs |
| `Row` | `Vertical`, `Wrap`, `Fill`, `Center`, `Margin`, `Spacing` | `InRow(RowCell{Width, Height})` | toolbars, button rows, wrapping flows |
| `Form` | `Margin`, `Spacing` | `Anchor(FormCell{Left, Right, Top, Bottom, Width, Height})` | edges attached to the container or to siblings |
| `Stack` | `Margin` | - | one child visible at a time; choose with `panel.ShowTop(child)` |

### Grid

```go
p.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
p.Label("Name")
p.Text(gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true}))
```

`GridCell` fields: `Align` and `VAlign` (`AlignStart`, `AlignCenter`, `AlignEnd`, `AlignFill`), `GrowX`/`GrowY` (take spare space), `SpanX`/`SpanY`, `Width`/`Height` (size hints in points). A child without `Cell` has its natural size, top left.

`Margin` and `Spacing` are in points and zero means none (SWT itself would default to 5).

### Form

Edges are `Edge` values:

- `gowt.Percent(pct, offset)`: a percentage of the container, plus an offset. `Percent(100, -5)` is 5 from the right or bottom.
- `gowt.Beside(w, gap)`: the opposite side of sibling `w`, `gap` away.
- `gowt.Same(w, offset)`: the same side of sibling `w`.

A side left as the zero `Edge` is free. An edge that refers to a sibling needs that sibling to exist already, so create the sibling first.

### Stack

`Stack` keeps all children at the container's full size. `ShowTop(child)` shows one; it panics on a panel that has no `Stack` layout. There is no getter for the page on top: remember it yourself.

## Gaps

There is no minimum window size in the facade; use `w.Unwrap().SetMinimumSize(w, h)` ([swt-direct.md](swt-direct.md)).
