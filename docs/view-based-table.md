# View-based Table and Tree on macOS: assessment

Status: not started. Decision: the change does not fit the generator's mechanisms (see "Why not now"), and it cannot be verified
without GUI runs. Task: TSK-2026-10-03-product-13.

## Where the code depends on NSCell

All of it is translated 1:1 from SWT's `Table.java` / `Tree.java` / `TableItem.java` / `TreeItem.java` / `TableColumn.java` / `TreeColumn.java`
(about 9800 lines in `swt/widgets_{table,tree,tablecolumn,treecolumn,tableitem,treeitem}_darwin.go`).

| Area | Sites | What it does |
|---|---|---|
| Cell classes | `widgets_display_darwin.go` ~1983 (`SWTImageTextCell`), ~2166 (`SWTTableHeaderCell`), `internal/cocoa` `SWTImageTextCell`, `SWTTableHeaderCell` | Runtime-registered NSTextFieldCell / NSTableHeaderCell subclasses with methods added by `OSClass_addMethod` |
| Data cells | `Table.dataCell`/`buttonCell`, `Tree.dataCell`/`buttonCell` (create ~441-461, 503; release ~579, 1720); `NSTableColumn.SetDataCell` | One shared cell per table, one NSButtonCell for the check column |
| Custom drawing | `Table.drawInteriorWithFrame_inView_` (table:804-1036), `Tree` equivalent (~936) | EraseItem / MeasureItem / PaintItem events, background and foreground colors, image + text layout, selection highlight, via a GC on the cell's view |
| Per-cell state | `willDisplayCell` (table:2870, tree:1894): font, color, alignment, attributed string, image set on the shared cell right before drawing | Becomes per-view configuration in `viewForTableColumn` |
| Hit testing / tracking | `hitTestForEvent_` (table:1469), `shouldTrackCell` (table:2825, tree:1883), `PreparedCellAtColumn` + `ImageRectForBounds` (table:1621, tree:1746) | Check box clicks and image hit areas |
| Sizing | `expansionFrameWithFrame_inView_` (table:1108, tree:1282), `titleRectForBounds_` (table:3028), `CellSize` (table:1204, tree:1343), `TableItem`/`TreeItem` `getBounds`/`getImageBounds`/`getTextBounds` (`FrameOfCellAtColumn`, `dataCell.CellSize`) | Bounds API and tooltips |
| Data source | `objectValueForTableColumn`, `setObjectValue` (check state, editing) | Returns the item's string or the check state |
| Header | `HeaderCell`, `DrawSortIndicatorWithFrame`, `SortIndicatorRectForBounds` in `TableColumn`/`TreeColumn` | Header cells can stay cell-based: NSTableHeaderView is not part of the view-based switch |

`FrameOfCellAtColumn` / `FrameOfOutlineCellAtRow` keep working on view-based tables, so TableEditor / TreeEditor placement is not affected.

## Approach

1. `NSTableView.viewForTableColumn:row:` / `outlineView:viewForTableColumn:item:` returns an `SWTTableCellView` (NSTableCellView
   subclass: image view, text field, optional NSButton for the check column), reused via `makeViewWithIdentifier:`.
2. `drawRect:` of that view takes over what `drawInteriorWithFrame` does: same SWT event flow (Erase, Measure, Paint) on a GC over the view.
   The row/column are taken from the view (`rowForView:`, `columnForView:`) instead of the `SWT_ROW`/`SWT_COLUMN` ivars.
3. `willDisplayCell` fields move into view configuration; the check box becomes an NSButton with a target/action that sets the item's state
   and sends SWT.Selection with SWT.CHECK.
4. Item bounds: keep `FrameOfCellAtColumn`, replace the `dataCell.CellSize` measurements by view `fittingSize` or the same text/image measurement.
5. Remove `dataCell`, `buttonCell`, `hitTestForEvent_`, `shouldTrackCell`, `expansionFrame...`, `titleRectForBounds_`.

## Why not now

- The translated bodies are generated. In the generator the only per-method override is `Manual.MANUAL_METHODS`: skip a Java method and hand-write it in
  `swt/*_manual_darwin.go` (used above for `TaskItem.updateImage`). Doing that for the ~25 cell-bound methods above duplicates thousands of
  translated lines by hand, and they keep drifting from upstream on every `make gen`.
- The Java source of `Table.java` / `Tree.java` is upstream (`SWT_REPO`) and not patched by `port.sh`; there is no source-patch stage. A patch stage
  (apply `*.patch` to the Java files before parsing) is the one generator-level mechanism that would carry this change; it does not exist yet.
- Verification is GUI-only: `make test-swt` has about 270 Table/Tree lines in `tests/expected.txt`, and ControlExample snapshots cover painting. Neither can be
  run by this task, so a rewrite would be unverified.

## Steps (estimate 4-6 working days plus GUI iterations)

1. Add a source patch stage to `tooling/port.sh` (and `Manual` for added native selectors); decide patch vs `_manual` per method. 0.5-1 day.
2. `SWTTableCellView` / `SWTOutlineCellView` classes in `widgets_display_darwin.go` + `internal/cocoa` bindings (NSTableCellView, NSButton target/action). 1 day.
3. Port the drawing methods (drawInteriorWithFrame x2) to `drawRect:`; port willDisplayCell. 1.5 days.
4. Check boxes, hit testing, bounds, tooltips, editing. 1 day.
5. Run the SWT suite and ControlExample snapshots, fix diffs; update `tests/expected.txt` and snapshots only for real differences. 1-2 days.

## Risks

- Pixel differences in every Table/Tree snapshot (row insets, view-based tables add cell padding and default row heights); the expected files will change.
- SWT's `MeasureItem` / `EraseItem` / `PaintItem` clip and offset assumptions are built on cell frames; a view's coordinate origin differs by the cell inset.
- Performance with large virtual tables (SWT.VIRTUAL): view reuse changes when `SetData` events fire.
- Selection highlight and focus ring colors are drawn by the cell today; the row view (`NSTableRowView`) draws them in the view-based mode, which can double-paint.
- Behavior drift from upstream SWT, which stays cell-based: every future `make gen` re-touches the overridden methods.
