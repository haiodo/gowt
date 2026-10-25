# Windows and layouts

<img src="img/art/windows-layouts.png" alt="Gopher laying out panels on a grid with a ruler" width="220" align="right">

Example: [`examples/layouts`](../examples/layouts).

<br clear="right">

## Windows

```go
w := app.Window("Title")   // hidden until Show; its children stack as in a VBox
w.Add(
	gowt.Label("Name"),
	gowt.Text(gowt.GrowX(), gowt.Hint("Full name")),
	gowt.Button("Close", gowt.OnClick(w.Close)))
w.SetSize(480, 320)        // optional; without it Show packs the window to its content
w.Show()
```

- `app.Window(title, opts...)` creates a top-level window. It is hidden until `Show`. Its options are `Gap`, `Padding` and children.
- The window stacks its children top to bottom like a `VBox`, `Spacing.SM` apart and `Spacing.PageMargin` from the window edge. `Padding(0)` puts the content against the edge.
- Children can go straight into `app.Window(title, children...)`. A child whose callback uses the window goes into `w.Add(...)` after that call, because `w` is not set while `app.Window` runs.
- `Show` sizes the window to its content unless `SetSize` was called. `SetMinSize(w, h)` keeps the user from making it smaller; `app.ScreenSize()` is the area a window can take on the primary monitor.
- `SetTitle`, `Close` (as if the user closed it) and `OnClose(func() bool)` exist; returning false from `OnClose` keeps the window open.
- More than one window: call `app.Window` again. `Run` returns when the last one is closed.

## Containers

A container is created with its layout and its children. Nest containers to build any arrangement; the nesting of the calls is the nesting on screen.

| Container | Lays children out | Per-child options |
|---|---|---|
| `VBox(opts...)` | top to bottom; a `Spacer()` takes the free height | `Grow`, `GrowX`, `GrowY`, `Size`, `Place`, `Cell` |
| `HBox(opts...)` | left to right, one column per shown child; a `Spacer()` takes the free width | the same |
| `Grid(columns, opts...)` | in `columns` columns, left to right, top to bottom | the same, and `Span(n)` |
| `Flow(opts...)` | in a row at natural size, wrapping to the next row; `Vertical()` makes it a wrapping column | `Size` |
| `Fill(opts...)` | in equal parts of the width, or of the height with `Vertical()`; one child fills it | none; `Grow`, `GrowX`, `GrowY` are accepted and change nothing |
| `Form(opts...)` | by edges attached to the container or to siblings | `Anchor(FormCell{...})`, `Size` |
| `Pages(opts...)` | one child at a time at full size; `Show(i)`, `Current()` | the same as `Fill` |

`Group(title, ...)`, a `Tab(title, ...)` page and the content of `Scroll(...)` stack their children like a `VBox`; for another arrangement put one `HBox`, `Grid`, ... inside. `Split` lays its children side by side with sashes, each pane filling its part as `SetWeights` sizes it, so a pane takes no placement options except `Grow`, `GrowX` and `GrowY`, which are accepted there and change nothing; the same holds for a `Tab` page. `Section` stacks them as rows of a card (see [widgets.md](widgets.md)).

Each container takes `Gap(points)` (space between children; rows and columns of a Grid alike) and `Padding(points)` (space from its edge). Gap defaults to `Spacing.SM` everywhere. Padding defaults to `Spacing.PageMargin` for a Window and a Tab page, `Spacing.SM` for a Group and 0 elsewhere, so nested boxes stay flush with each other. `Gap(0)` puts children edge to edge.

```go
gowt.Grid(2, gowt.GrowX(), gowt.Gap(gowt.Spacing.LG),
	gowt.Label("Name"), gowt.Text(gowt.GrowX()),
	gowt.Label("Notes"), gowt.Text(gowt.Multiline(), gowt.Grow(), gowt.Size(0, 80)),
	gowt.Separator(gowt.Span(2)),
	gowt.Spacer(), gowt.Button("Save", gowt.OnClick(onSave)))
```

A container's content can change after `Show`: `Add(children...)` creates children at its end and measures only them (the children already there keep their sizes, so adding to a long feed costs the new rows), `Relayout()` lays the children out again (after a change made through `Unwrap()`; the facade setters that change a size, such as `SetText`, lay out by themselves), `Clear()` disposes every child so that the content can be built again, and `child.Dispose()` removes one child and lays the parent out again.

## Placement

A child without placement options has its natural size, top left of its cell. The options add up, and when two set the same thing the later one wins:

| Option | Effect in a VBox, HBox or Grid |
|---|---|
| `GrowX()`, `GrowY()` | fill the cell's width (height) and take the free width (height) |
| `Grow()` | both |
| `Span(n)` | cover n columns of a Grid |
| `Size(w, h)` | width and height hints in points, 0 keeps the natural one; also works in Flow and Form |
| `Place(h, v)` | alignment in the cell: `AlignStart`, `AlignCenter`, `AlignEnd`, `AlignFill` |
| `Indent(x, y)` | move the child x points right and y points down inside its cell; the cell grows by as much |
| `Cell(GridCell{...})` | every field at once, including `SpanY`; options after it change single fields |

`Spacer()` is an empty cell that takes the free space along a VBox or HBox, or the free width of a Grid row, pushing the cells after it to the end. `Separator()` is a line across its cell; `Vertical()` turns it.

A placement option the parent cannot use (`GrowX` in a `Flow`, `Anchor` in a `Grid`, `Size` on a `Split` pane) is reported to `App.OnError` when the widget is created, with the option, the widget and the parent's layout in the message, and the widget keeps its natural size.

### Spacing tokens

`gowt.Spacing` is a `SpacingScale` of points: `XS` 4, `SM` 8, `MD` 12, `LG` 16, `XL` 32, and `PageMargin` (`LG` on macOS, `MD` elsewhere). Multiples of 4 stay whole pixels at 100..200% scaling. `Gap(gowt.Spacing.LG)` and `Padding(gowt.Spacing.XL)` pick other steps.

An application tunes the scale once at startup, e.g. `gowt.Spacing.SM = 6` or `gowt.Spacing = gowt.SpacingScale{...}`. Containers created before the change keep their values.

`gowt.Golden(total)` returns `small, large` (about 38% and 62%, summing to `total`), e.g. for a sidebar width or an initial window size; for `Split.SetWeights` the integer weights 5 and 8 give the same ratio.

### Form

```go
var ok *gowt.ButtonView
gowt.Form(gowt.Grow(),
	gowt.Button("OK", gowt.OnClick(onOK), gowt.Ref(&ok), gowt.Anchor(gowt.FormCell{Right: gowt.Percent(100, 0), Bottom: gowt.Percent(100, 0)})),
	gowt.Button("Cancel", gowt.OnClick(onCancel), gowt.Anchor(gowt.FormCell{Right: gowt.Beside(&ok, 8), Bottom: gowt.Same(&ok, 0)})))
```

Edges are `Edge` values:

- `gowt.Percent(pct, offset)`: a percentage of the container, plus an offset. `Percent(100, -5)` is 5 from the right or bottom.
- `gowt.Beside(&w, gap)`: the opposite side of sibling `w`, `gap` away.
- `gowt.Same(&w, offset)`: the same side of sibling `w`.

`Beside` and `Same` take the address of the variable a `Ref` fills and read it when the child is placed, so the sibling must come earlier in the same Form. A side left as the zero `Edge` is free.

### Pages

```go
var pages *gowt.PagesView
w.Add(
	gowt.Pages(gowt.Grow(), gowt.Ref(&pages),
		gowt.Label("Page one"),          // the first page is shown
		gowt.Label("Page two")),
	gowt.Button("Switch", gowt.OnClick(func() { pages.Show(1 - pages.Current()) })))
```

Every child of `Pages` is a page. `Show(i)` counts pages from 0 in the order they were added and panics on an index out of range; `Current()` is -1 when no page is shown. `pages.Add(...)` appends pages; a page added while none is shown is shown. `page.Dispose()` removes a page: the pages after it move down by one, and when it was shown the page before it is shown (the next one when it was the first, none when it was the last page). A `List` that picks the page is kept in step by the app: `list.Remove(i)`, `page.Dispose()`, then `list.Select(pages.Current())`.
