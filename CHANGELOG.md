# Changelog

## Unreleased (v0.2.0)

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
