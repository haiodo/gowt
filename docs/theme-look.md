# Theme and platform look

Example: [`examples/theme`](../examples/theme).

## Light and dark

`gowt.Run` makes the app follow the system theme. Two calls expose it:

```go
dark := app.Dark()                       // UI thread
app.OnThemeChange(func(dark bool) { })   // runs after widget colours were refreshed
```

How the theme is read: macOS `NSApp.effectiveAppearance`, Windows the apps theme setting, Linux the `color-scheme` value of xdg-desktop-portal. Where each OS stands:

| OS | What follows the theme |
|---|---|
| macOS | the whole UI |
| Windows | the title bar; content stays light unless you call `look.SetDarkContent(app, true)` before creating windows (scroll bars, tables, trees, buttons; it uses undocumented uxtheme exports and does not switch live) |
| Linux | the theme and accent colour come from the portal; this was not verified against a live portal |

Colours you set yourself (`Background`, `GC` colours) do not change by themselves: redraw them in `OnThemeChange`.

## Package look

`github.com/haiodo/gowt/look` holds options for the platform's own appearance. Every call does nothing on a system without the feature, so it is safe to call everywhere.

| Call | Effect |
|---|---|
| `look.SetBackdrop(w, look.BackdropNone / BackdropTranslucent / BackdropGlass)` | window material. macOS: blurred material or Liquid Glass (26+). Windows 11 22H2 (build 22621) and later: Acrylic (`BackdropTranslucent`) or Mica (`BackdropGlass`); older builds keep the opaque window. Linux ignores it |
| `look.SetGlass(container, true)` | Liquid Glass surface behind a `Panel`, `Window`, `Group`, `Split` or `CoolBar`. macOS 26; older macOS shows a blurred material, other systems nothing |
| `look.GlassButton()` | option for a push button: glass bezel on macOS 26 |
| `look.SetFullSizeContent(w, true)` | content extends under a transparent title bar (macOS); the window layout (Grid, Row, Form) gets the title bar and toolbar height as extra top margin (`Window.SetTopInset`), updated on resize and SetToolbar |
| `look.DragsWindow(c)` | `c` (Panel, Window, Group...) and its labels and plain panels move the window on a left press and zoom or maximize on a double click, like a title bar; buttons, text fields and Canvas keep their clicks. Call it after building `c`. macOS `performWindowDragWithEvent:` and the `AppleActionOnDoubleClick` setting; Windows `WM_NCLBUTTONDOWN` with `HTCAPTION`, double click toggles maximized; Linux `gtk_window_begin_move_drag` (GTK 3), double click toggles maximized. Not run on any OS yet |
| `look.SetRoundedCorners(w, true)` | force rounded or square window corners on Windows 11 |
| `look.SetDarkContent(app, true)` | Windows dark content, see above |
| `look.SetToolbar(w, items...)` | native unified toolbar (NSToolbar) with SF Symbol items; Liquid Glass items on macOS 26. Returns false where it cannot exist (Windows, Linux): nothing is created, build a `gowt.ToolBar` instead |
| `look.SetSidebar(w, sidebar, content, width)` | Finder / System Settings window: sidebar left, content right. macOS 26: floating Liquid Glass sidebar under the traffic lights, toolbar items and title past it. Windows, Linux: two plain panels. See below |
| `look.SetGlassGroup(parent, spacing)` | glass surfaces of the children of `parent` share one NSGlassEffectContainerView and merge when closer than `spacing`; call before their `SetGlass`. macOS 26, else nothing |
| `look.Classic()` | before `Run`: ask macOS 26 for the pre-Liquid-Glass look |

On macOS 26 the system chrome is Liquid Glass without any call, because Go 1.27 records SDK 26.2 in the binary.

`look.Classic()` was not confirmed to work for a binary without an app bundle. The reliable way is building with `-ldflags=-macsdk=15.0`.

## Toolbar

```go
if !look.SetToolbar(w,
	look.ToolbarItem{Title: "Add", Symbol: "plus", OnClick: add},
	look.ToolbarItem{Space: true},
	look.ToolbarItem{Title: "Share", Symbol: "square.and.arrow.up", OnClick: share},
) {
	w.ToolBar().Item("Add", add) // Windows, Linux, older macOS without the call
}
```

Demo: `GOWT_TOOLBAR=1 go run ./cmd/glassdemo`; its glass group uses spacing 4, smaller than the 10 px between the panels, so they stay separate shapes (a spacing above 10 fuses neighbours on purpose). Not checked by eye yet: the toolbar and the glass group were only compiled.

NSBackgroundExtensionView is not wrapped.

## Finder and System Settings look

Demo: `go run ./cmd/finderdemo` (`GOWT_FINDERDEMO_SECS=n` quits after n seconds).

```go
sidebar := w.Panel()
sidebar.SetLayout(gowt.Grid{Columns: 1}) // no margin: the rows inset their selection
nav := sidebar.List(pages, gowt.NoBorder(), fill)
content := w.Panel()
content.SetLayout(gowt.Grid{Columns: 1, Margin: 20, Spacing: 8})
s := content.Section(gowt.Cell(gowt.GridCell{GrowX: true, Align: gowt.AlignFill}))
s.Row().SetLayout(gowt.Grid{Columns: 2, Margin: 6}).Label("Appearance")
s.Row().SetLayout(gowt.Grid{Columns: 2, Margin: 6}).Label("Liquid Glass")
look.SetSidebar(w, sidebar, content, 220)
look.SetToolbar(w,
	look.ToolbarItem{Title: "Back/Forward", Nav: true, Group: []look.ToolbarItem{
		{Title: "Back", Symbol: "chevron.backward", OnClick: back},
		{Title: "Forward", Symbol: "chevron.forward", OnClick: forward},
	}},
	look.ToolbarItem{Space: true},
	look.ToolbarItem{Title: "Search", OnSearch: func(text string) {}},
)
w.SetTitle("Appearance") // the page title next to back/forward
```

- `look.SetSidebar(w, sidebar, content, width)` replaces the window layout; the window holds just these two. It turns on full-size content. On macOS 26 the sidebar is Liquid Glass with corner radius 18, 7 points from the window edges (both measured on a Finder screenshot), running under the traffic lights; the sidebar children and the content start below the title bar and toolbar. Lists, tables and trees inside the sidebar when it is called get the source-list style with a transparent background; a List or Table selects the Finder way, a grey pill with the text in the accent colour (a Tree keeps the filled selection). AppKit puts navigational items and the window title right after the traffic lights, so an empty navigational item of fitted width goes first and back/forward and the title start past the sidebar edge. Rows of a source list keep a fixed height (custom row size style, at least the one AppKit chose for the sidebar), which SWT reads as the item height; List text is centred in its row. Older macOS: a flush blurred material. Windows and Linux: two plain panels side by side.
- `ToolbarItem.Group` is one capsule of segments (back/forward), `Nav` puts the item before the window title, `OnSearch` makes a native search field (NSSearchToolbarItem) that reports the text after a typing pause, on Return and on clearing. On macOS 15 and newer it starts as a magnifier button that turns into the field on a click and back when editing ends with the field empty (Esc, the clear button, a click elsewhere), as in Finder. The field gets AppKit's own `NSSearchFieldCell`: SWT sets the cell class of every NSSearchField to its own cell, and with that cell the toolbar field showed an empty capsule. The toolbar shows icons only, no labels under the items. `Enabled` (an `rx.Reader[bool]`, nil for always enabled) dims a push item or a group segment while it is false; the toolbar delegate answers `validateToolbarItem:` from it, so AppKit's autovalidation does not enable it again. The page title is the window title.
- `Section()` on any container is a rounded card with rows; `Row()` adds a row panel and a separator line before it. The card is the section's SWT background, inherited by its rows and labels, with the corners painted in the parent's colour: on macOS `quaternarySystemFillColor` over the window colour, flattened to an opaque colour (SWT fills the background of every label and panel on the card, and translucent fills would add up); on Windows and Linux 6% from the window colour toward the text colour. It is taken again when the system theme changes. Labels, check boxes and panels in a row take the card colour; a `Text` keeps its own white field, so show a read-only value with a `Label`. A heading is a `Label` above the section (`gowt.SetFont(l, gowt.Font{Weight: gowt.Semibold})`).

Footprint on macOS: start the demo, then `footprint -p finderdemo` (the `IOAccelerator` and `IOSurface` lines are GPU memory) or `vmmap --summary $(pgrep finderdemo)`.

Checked with `GOWT_SNAP_SCREEN=1` snapshots of dockerdesk on macOS 26, light appearance: the floating rounded sidebar, the inset selection pill, icon-only toolbar, collapsed and expanded search (`GOWT_TOOLBAR_SEARCH_OPEN=1` starts it expanded), dimmed back/forward segments. Not checked: dark appearance, the shadow, the wallpaper. The glass blends what the window draws behind it, and the window is opaque, so the desktop does not show through as in Finder; a single-window capture could not show it either. `GOWT_SNAP_LAYERS=1` makes `GOWT_SNAP` render the layer tree; glass and backdrops never show there.

Do not give the window itself a background (`Unwrap().SetBackgroundWithColor`) when it has a sidebar: the sidebar panel then paints that colour as an opaque rectangle over the glass, square corners included. Colour the content panel instead. `GOWT_SIDEBAR_DEBUG=1` prints, once the window is open, the SWT background state and the AppKit view chain of every list, table and tree in the sidebar.

In the sidebar every control present when `SetSidebar` runs gets its own alpha-0 background: SWT fills a control without one in the window colour, and an alpha-0 background of a parent does not count. Controls added to the sidebar later paint that colour. The selection pill is drawn by the row view (AppKit's own selection is off): `tertiarySystemFillColor`, black or white at 4.7%, 8 points in from the row sides, as in Finder. Give the sidebar no margin around its list to get Finder's 8-point pill inset; the list's scroll view gets no automatic content inset.

## Known gaps

- On Windows the material shows only where the client area is black, so `SetBackdrop` turns the shell background black. Text drawn in pure black becomes transparent: use a near-black foreground. Widgets with their own opaque background are unaffected.
- Mica, Acrylic and rounded corners were never checked on a real Windows 11 (CrossOver has no DWM).
- Windows per-monitor DPI changes while running (`WM_DPICHANGED`) are not handled.
- The macOS toolbar has no menus or toggles, and its delegate lives as long as the window.
