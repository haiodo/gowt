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
| `Grid` | `Columns` (0 means 1), `EqualWidth`, `Margin`, `Spacing` | `Grow`, `GrowX`, `GrowY`, `Span`, `Size`, `Place`, `Cell(GridCell{...})` | forms and most dialogs |
| `Row` | `Vertical`, `Wrap`, `Fill`, `Center`, `Margin`, `Spacing` | `Size`, `InRow(RowCell{Width, Height})` | toolbars, button rows, wrapping flows |
| `Form` | `Margin`, `Spacing` | `Anchor(FormCell{Left, Right, Top, Bottom, Width, Height})`, `Size` | edges attached to the container or to siblings |
| `Stack` | `Margin` | - | one child visible at a time; choose with `panel.ShowTop(child)` |

### Grid

```go
p.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
p.Label("Name")
p.Text(gowt.GrowX())
p.Label("Notes")
p.Text(gowt.Multiline(), gowt.Grow(), gowt.Size(0, 80))
p.Separator(gowt.Span(2))
p.Spacer()
p.Button("Save", onSave)
```

A child without placement options has its natural size, top left of its cell. The options add up, and when two set the same thing the later one wins:

| Option | Effect in a Grid |
|---|---|
| `GrowX()`, `GrowY()` | fill the cell's width (height) and take the free width (height) of the grid |
| `Grow()` | both |
| `Span(n)` | cover n columns |
| `Size(w, h)` | width and height hints in points, 0 keeps the natural one; also works in Row and Form |
| `Place(h, v)` | alignment in the cell: `AlignStart`, `AlignCenter`, `AlignEnd`, `AlignFill` |
| `Cell(GridCell{...})` | every field at once, including `SpanY`; options after it change single fields |

`p.Spacer()` is an empty cell that takes the free width, pushing the cells after it to the end of the row (`Spacer(gowt.GrowY())` also takes the free height). `p.Separator()` is a line across its cell; `Vertical()` turns it.

A placement option the parent's layout cannot use (`GrowX` under `Row`, `InRow` under `Grid`) is reported to `App.OnError` when the widget is created, with the option, the widget and the layout in the message, and the widget keeps its natural size. Set the layout before adding children: a `SetLayout` that comes later drops and reports the children placed for another layout.

`Margin` and `Spacing` are in points and zero means none (SWT itself would default to 5).

### Spacing tokens

`gowt.Spacing` is a `SpacingScale` of points: `XS` 4, `SM` 8, `MD` 12, `LG` 16, `XL` 32, and `PageMargin` (`LG` on macOS, `MD` elsewhere). Multiples of 4 stay whole pixels at 100..200% scaling. They fit the existing int fields:

```go
w.SetLayout(gowt.Grid{Columns: 1}.Padded())                                  // Margin = Spacing.PageMargin
p.SetLayout(gowt.Grid{Columns: 2, Margin: gowt.Spacing.LG, Spacing: gowt.Spacing.SM})
p.SetLayout(gowt.Row{}.Spaced())                                            // Spacing = Spacing.SM
```

`Spaced()` (Grid, Row, Form) and `Padded()` (Grid, Row, Form, Stack) return a copy with `Spacing = Spacing.SM` or `Margin = Spacing.PageMargin`. A plain zero still means none, so nested layouts stay flush; put `Padded()` on the root layout only.

An application tunes the scale once at startup, e.g. `gowt.Spacing.SM = 6` or `gowt.Spacing = gowt.SpacingScale{...}`. Layouts created before the change keep their values.

`gowt.Golden(total)` returns `small, large` (about 38% and 62%, summing to `total`), e.g. for a sidebar width or an initial window size; for `Split.SetWeights` the integer weights 5 and 8 give the same ratio.

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
