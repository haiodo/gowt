# Changelog

## v0.7.0 (2026-10-15)

New API
- `look.SetToolbar` has a portable fallback: on Windows and Linux (anywhere the native toolbar is not created) a window laid out by `look.SetSidebar` gets the items as a bar of gowt widgets at the top of the content column: flat back/forward buttons dimmed by `Enabled`, the window title (it follows `Window.SetTitle`), a search field calling `OnSearch` on every change, in the content's colour in light and dark. An app declares its toolbar once; `cmd/finderdemo` no longer builds a second one under `if !native`.

Behaviour change
- Windows, Linux: an app that called `look.SetToolbar` and `look.SetSidebar` and built its own replacement bar now shows two; drop the replacement. `SetToolbar` still returns false there, since no native toolbar exists, and a window without `SetSidebar` still gets nothing.

## v0.6.1 (2026-10-15)

Fixes
- macOS: typing into the toolbar search field (`look.ToolbarItem.OnSearch`) put no text in it. The field used the window's field editor, SWT's `SWTEditorView`, which drops the keys of a field that belongs to no SWT widget; the field now has a plain AppKit field editor. The magnifier button also left the focus where it was, since it focused the field before the toolbar had put it in the window; the field now takes the focus once shown.

## v0.6.0 (2026-10-15)

New API
- Finder and System Settings look (`cmd/finderdemo`): `look.SetSidebar(w, sidebar, content, width)` lays a window out with a sidebar, floating Liquid Glass on macOS 26 with the toolbar aligned past it, plain panels on Windows and Linux; `look.ToolbarItem` gets `Group` (segmented capsule such as back/forward), `Nav` (before the window title) `OnSearch` (native search field, collapsed to a magnifier button until clicked) and `Enabled` (an `rx.Reader[bool]` that dims a push item or a segment); `Section()` on any container adds a grouped "inset" card with `Row()` and separators (macOS `quaternarySystemFillColor` over the window colour, 6% toward the text colour elsewhere; it follows a theme switch). `GOWT_SNAP_LAYERS=1` makes macOS snapshots render the layer tree, `GOWT_SNAP_SCREEN=1` captures the composited window with glass and toolbar.

Fixes
- macOS: the toolbar shows icons only, as Finder does; the labels under the items are gone.
- macOS: the toolbar search field was an empty capsule, drawn by the search cell SWT installs for every NSSearchField; it now has AppKit's own cell with the magnifier and the placeholder.
- macOS: `List` text is centred in rows taller than the text (a source list in `look.SetSidebar`); it sat at the top edge of the selection.
- macOS: while the user dragged a window edge the content could stop following the window until the mouse button went up. A deferred layout (`Label.BindText`, `Control.RequestLayout`) made during the drag held every resize layout of the window, because deferred layouts ran only on the next `ReadAndDispatch`; a window resize now runs them first.

## v0.5.0 (2026-10-15)

New API
- Reactive state, package `rx` (Svelte 5 style): `State` (`New`, `NewFunc`, `Get`, `Peek`, `Set`, `Update`, `Post` from any goroutine), `Derived` (`Compute`, `ComputeFunc`), `Effect`, `OnCleanup`, `Scope`, `Untrack`, `Batch`, `Flush`, `Reader`; `Attach` connects it to an event loop and `Run` calls it. Effects run once per batch after the current event handler; `Post` costs one `AsyncExec` per batch. Bindings that live as long as their widget: `Bind`, `Label.BindText`, `Button.BindChecked` and `Text.Bind` (two-way), `BindVisible`, `BindEnabled`. See docs/events.md.
- `rx` rules: everything except `Post` is UI-thread only; `Set` from another goroutine panics, and `-tags rxdebug` checks `Get` as well. A panicking effect does not stop the others: the flush finishes, then the first panic continues from `Run`. Effects that keep invalidating each other for 100 rounds panic with "rx: effects did not settle".

Performance
- `rx`: `Set` followed by `Flush` with one effect costs ~26 ns and does not allocate (Apple M4 Max). A `Post` from a goroutine reaches its effect in 50-70 us on GTK, the same as `App.Async`; concurrent `Post`s coalesce into one effect run.

## v0.4.0 (2026-10-15)

Behaviour change
- macOS: `Table`, `Tree` and `List` are now view-based (NSTableView/NSOutlineView with cell views). Cell-based drawing added ~227 MB of GPU memory while cells updated; view-based uses 74-81 MB in the same workload. `GOWT_VIEW_TABLE=0` restores the cell-based path for this release only.
- macOS: set `GOWT_VIEW_TABLE=0` to go back to cell-based drawing.

New API
- Spacing tokens: `Spacing` (`SpacingScale`: `XS` 4, `SM` 8, `MD` 12, `LG` 16, `XL` 32, `PageMargin` LG on macOS and MD elsewhere), tunable at startup and applied to layouts created afterwards; `Grid`/`Row`/`Form` `.Spaced()` and `Grid`/`Row`/`Form`/`Stack` `.Padded()`; `Golden(total)` splits a size 38:62. Zero `Margin`/`Spacing` still means none.

Performance
- Generated code walks strings in O(n): `String.charAt` now maps to `jrt.CharAt`, and `StringLength`, `Substring` and `IndexFrom` share a one-string UTF-16 cache. Appending 1000 log lines to a `StyledText` in one call took 80 s and now takes 6 ms.

Fixes
- windows/arm64: GDI+ float arguments go in float registers. `Transform`, pens, font sizes, arcs and every `GC.drawImage` that built a transform failed with "No more handles".
- Windows: `Tree.SetRedraw(false)` panicked (`jrt.ListOf` kept an array of children as one element).

## v0.3.0 (2026-10-14)

New API
- Font weights: `Font.Weight` (`Light`, `Regular`, `Medium`, `Semibold`, `Bold`, CSS 100..900) for `SetFont` and `TextStyle.Font`; `SetFont` with an empty `Name` keeps the control's current family (the system UI font for a default control); `NewFontDeviceNameHeightWeight` with an empty name is the system UI font (SF Pro on macOS via `systemFontOfSize:weight:`, Pango weight on Linux, `lfWeight` on Windows). `swt.NewFontDeviceNameHeightWeight` makes the same font for direct use.

Fixes
- Windows: a `Text` with `SEARCH|ICON_SEARCH` or `ICON_CANCEL` no longer panics with "No more handles". The icons came from swt.dll resources that gowt does not have; they are now drawn at run time (light and dark variants).

## v0.2.0 (2026-10-14)

Behaviour change
- `Tree` now selects whole rows (FULL_SELECTION) on every OS, so a click anywhere in a row finds its node. macOS did this already; on Windows and Linux v0.1.0 selected only the text. `CellSelect()` restores the old behaviour.

New API
- Tree columns: `Tree.Column` returning `*TreeColumn` (`SetTitle`, `SetWidth`, `SetResizable`, `Pack`, `OnClick`, `Unwrap`), `Tree.ShowHeader`, `Node.TextAt`, `SetTextAt`, `SetImageAt`.
- Sorting indicator: `SortDir` (`SortNone`, `SortAscending`, `SortDescending`), `Table.SetSort`, `Tree.SetSort`; `Column.SetResizable`.
- Cell hit test: `Table.CellAt`, `Table.OnCellClick` (`TableClick`), `Tree.CellAt`, `Tree.OnCellClick` (`TreeClick`), `Mods`.
- Any control: `SetEnabled`, `Enabled`, `SetVisible`, `Visible`, `SetFont`, `SetForeground`.
- `Switch` widget (`Panel.Switch`): a check box on every OS.
- Logs: `Text.Append`, `Lines`, `TrimHead`, `ScrollToEnd`; `StyledText.TrimHead`, `ScrollToEnd`.
- Options: `NoBorder`, `Search`, `CellSelect`.
- `Table.ShowRowLines`, `Tree.ShowRowLines`: horizontal separators only on macOS.
- `Canvas.OnMouseDown`, `OnMouseUp`, `OnMouseMove` (`Mouse`).
- `look.DragsWindow`: window drag area for custom title bars.

## v0.1.0

First release: the idiomatic `gowt` facade over the translated SWT, `look`, `jface`, `browser` and `webview`.
