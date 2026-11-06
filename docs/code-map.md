# gowt feature-to-code map

Where each feature lives. The map answers "which files do I read and change for feature X";
it does not repeat API signatures (`go doc`, [tooling/apidump](../tooling/apidump)) or the
prose owned by [agents.md](agents.md) (facade idioms), [internals.md](internals.md)
(repository layout, translator flow, gates) and [platforms.md](platforms.md) (platform status).

Semantic rows are curated by hand. File cells name the stable base or hand-written files of a
family (`_manual*.go`, `_darwin/_linux/_windows` bases); they do not list generated method
parts. A `*` in a cell marks a glob over one such family. Where a feature has no platform,
test or example split of its own, the cell says `-`.

Sections that own Go packages of their own name them in a `Packages:` line
(Layouts, Widgets, Dialogs and menus, Clipboard and drag and drop and Printing
own none - their code lives in `gowt`, `swt` and `jface`, listed by the owning
sections). `make map-check` (`tooling/mapcheck`) verifies the paths and links
of this map and that the curated package lists cover every package of the
repository (the union of `go list` over darwin, windows and linux). The tool
only checks the map; rows and package lists stay hand-curated.

## Facade

Package `gowt`, the hand-written layer documented in [facade.md](facade.md).
Primary files are in the repository root.

Packages: `.`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| App entry, event loop, `Run`, timers, jobs | `gowt.go`, `jobs.go`, `on.go`, `bind.go` | `quit_darwin.go`, `quit_other.go` | `gowt_test.go`, `jobs_gui_test.go`, `quit_*_test.go` | `cmd/hellogowt`, `cmd/gowtdemo`; [events.md](events.md) |
| Window, activation, single instance | `gowt.go`, `singleinstance_socket.go`, `openfiles.go` | `singleinstance_{darwin,linux,windows}.go` | `openfiles_test.go`, `singleinstance_socket_test.go` | `cmd/finderdemo`; [windows-layouts.md](windows-layouts.md) |
| Options, placement, `Ref`/`Init` | `gowt.go`, `sizehint.go`, `spacing.go`, `density.go` | `density_sys.go`, `mono_{linux,other}.go`, `style_{linux,other}.go`, `fit_{linux,other}.go` | `sizehint_gui_test.go`, `spacing_test.go`, `density_test.go` | [agents.md](agents.md) "Placement" |
| Reactive bindings, `Each` | `bind.go`, `each.go` | - | `bind_test.go`, `each_test.go`, `each_gui_test.go` | [events.md](events.md) |
| Error surface, `Boundary` | `boundary.go` | - | `boundary_test.go` | [events.md](events.md#boundary-errors-shown-in-place) |
| Canvas, `GC` drawing, images | `graphics.go`, `canvaschild.go` | `textfit_{darwin,linux,windows}.go` | `canvassize_test.go`, `textfit_test.go`, `textfit_gui_test.go` | `cmd/paint`, `examples/gfxcanvas`; [graphics.md](graphics.md) |
| Theme, colours, fonts, appearance | `theme.go`, `linkcolor.go` | `theme_{darwin,other}.go`, `textbg_{linux,other}.go`, `rowlines_{darwin,other}.go` | `theme_test.go`, `appearance_test.go` | [theme-look.md](theme-look.md), [graphics.md](graphics.md) |
| Badges, avatars, buttons kind | `badge.go`, `avatar.go`, `buttonkind.go` | `badge_{linux,other}.go`, `iconbutton_{darwin,linux,windows}.go` | `badge_test.go`, `badge_linux_test.go`, `avatar_test.go`, `buttonkind*_test.go`, `iconbutton*_test.go` | [widgets.md](widgets.md#badges-and-chips) |
| Popover, overlays | `popover.go`, `popover_shell.go` | `popover_{darwin,linux,windows}.go` | `popover_test.go`, `popover_gui_test.go`, `popover_draw_darwin_test.go` | `examples/dialogs` |
| Pills, sections, disclosure, segmented, CTabFolder extras | `section.go`, `disclosure.go`, `segmented.go`, `ctabs_render.go`, `hint.go` | `section_{darwin,other}.go`, `segmented_{darwin,other}.go`, `ctabs_look_{darwin,windows,other}.go`, `hint_{darwin,linux,windows,other}.go` | `section_test.go`, `disclosure_test.go`, `segmented*_test.go`, `ctabs_gui_test.go`, `hint_gui_test.go` | `examples/widgets`, `examples/catalog` |
| Field editor, form fields | `fields.go` | `fieldeditor_{darwin,other}.go`, `fieldstyle_{darwin,linux,windows,other}.go` | `fields_test.go`, `fieldstyle_linux_test.go` | `cmd/form`; [widgets.md](widgets.md) |
| Scroll behaviour, affix | `affix.go` | `scroll_{darwin,other}.go`, `overlay_scroll_windows.go` | `scroll_gui_test.go`, `scroll_*_test.go`, `affix_test.go`, `affix_darwin_test.go` | [windows-layouts.md](windows-layouts.md) |
| Line numbers, styled text helpers | `widgets_text.go` | `textscroll_{linux,other}.go` | `linenumbers_test.go`, `styledtext_bg_gui_test.go` | [widgets.md](widgets.md) |
| App monitor, snapshots (facade side) | `snap.go`, `qa_on.go`, `qa_off.go` | `snap_{darwin,other}.go` | `qatree_*_gui_test.go` | [qa.md](qa.md) |

## Layouts

Facade layout descriptions and the SWT layout classes behind them.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Facade containers and placement engine (VBox, HBox, Grid, Flow, Form, Fill, Pages, Split, Tabs) | `layout.go` | - | `layout_test.go` | `examples/layouts`; [windows-layouts.md](windows-layouts.md) |
| Grid columns sizing, fit | `fitcolumns.go` | - | `fitcolumns_test.go` | [windows-layouts.md](windows-layouts.md) |
| SWT layout classes | `swt/layout_*.go` (shared, no per-platform split) | - | `swt/layout_test.go`, `tests/swttests/junit_test_org_eclipse_swt_layout_*.go` | [swt-direct.md](swt-direct.md) |
| JFace layout factories | `jface/layout_*.go` | one source for every OS | `tests/jfacetests/layout_*test.go` | `examples/jface`, [jface.md](jface.md) |

## Widgets

Facade widget families; the view types end in `View`. SWT widget classes are in
[SWT](#swt) below.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Common widget base, labels, buttons, links | `widgets_common.go` | per-widget `*_{darwin,linux,windows,other}.go` files in root | `example_test.go`, `text_test.go` | `examples/widgets`; [widgets.md](widgets.md) |
| Containers (Group, Scroll, Split, Tabs, CTabs, Boundary shells) | `widgets_container.go` | `section_{darwin,other}.go`, `ctabs_look_*` | `layout_test.go`, `ctabs_gui_test.go` | `examples/widgets` |
| Lists, tables, combos | `widgets_list.go` | per-platform sections of the same file | `fitcolumns_test.go`, `qatree_*_gui_test.go` | `examples/rows`, `examples/sidebartable`, `cmd/tree`; [view-based-table.md](view-based-table.md) |
| Tree | `tree.go` | `tree_edit_{darwin,other}.go` | `tree_test.go`, `tree_edit_gui_test.go`, `tree_edit_gui_darwin_test.go` | `cmd/tree`; [widgets.md](widgets.md) |
| Tool bar | `look/toolbar.go`, `widgets_container.go` (ToolBarView) | `look/toolbar.go` platform sections, `internal/cocoa/*toolbar*` | `look/toolbar_test.go`, `look/toolbar_*_gui_test.go` | `cmd/gowtdemo`; [widgets.md](widgets.md) |
| Text, styled text, search fields, spinners, scales, progress | `widgets_text.go`, `widgets_range.go` | `fieldeditor_*`, `textfit_*`, `scroll_*` | `text_test.go`, `textfit*_test.go`, `sizehint_gui_test.go` | `examples/widgets`; [widgets.md](widgets.md) |
| Canvas and drawing widgets | `graphics.go`, `canvaschild.go` | - | `canvassize_test.go` | `examples/gfxcanvas`, `examples/graphics` |
| ControlExample (widget zoo on `swt`) | - | - | - | `examples/controlexample`, `cmd/controlexample` |

## Dialogs and menus

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Facade dialogs (alert, confirm, prompt, file, folder, color, font, print sheet) | `dialogs.go` | `print_{darwin,linux,windows}.go` | `dialogs_gui_darwin_test.go` (darwin sheets) | `examples/dialogs`; [dialogs-menus.md](dialogs-menus.md) |
| Facade menus, menu bar, shortcuts | `menu.go` | `menu.go` platform sections | `example_test.go` | [dialogs-menus.md](dialogs-menus.md), `examples/dialogs` |
| JFace dialogs (MessageDialog, IDs) | `jface/dialogs_idialogconstants.go`, `jface/dialogs_manual.go` | one source | `tests/jfacetests/*` | [jface.md](jface.md) |

## Clipboard and drag and drop

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Facade clipboard | `clipboard.go` | platform transfer code in `swt/` DnD classes | - | `examples/clipboard` |
| Facade drag and drop | `dnd.go` | `swt/dnd_dragsource_{darwin,linux,windows}.go`, `swt/dnd_droptarget_*.go` | `dnd_gui_test.go`, `dnd_post_*_test.go` | `examples/dnd` |
| SWT DnD framework | `swt/dnd_*.go` | per-platform `swt/dnd_*_{darwin,linux,windows}.go` | `tests/swttests/junit_test_org_eclipse_swt_dnd_*.go` | `examples/dnd` |

## look

Package `look`: window material, sidebar and inspector layouts, glass, window drag.

Packages: `look`, `internal/fluent` (the Windows look helpers).

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Window backdrop, material, corners, dark content | `look/look.go` | `look/look_{darwin,linux,windows,other}.go` | `look/material_*_test.go`, `look/glass*_test.go` | `cmd/glassdemo`, `cmd/backdropdemo`; [cross-platform-look.md](cross-platform-look.md), [theme-look.md](theme-look.md) |
| Sidebar and bottom panel layouts | `look/inspector_flat.go`, `look/toolbar.go` | `look/look_*.go`, `look/windowdrag_*.go` | `look/sidebar*_test.go`, `look/bottompanel*_test.go`, `look/inspector_*_test.go` | `examples/apps`, `examples/sidebartable` |
| Inspector controls, fields, search | `look/inspector_flat.go` | - | `look/inspector_*_gui_darwin_test.go` | `examples/catalog` |
| Window drag by background | `look/windowdrag.go` | `look/windowdrag_{darwin,linux,windows}.go` | `look/toolbar_diag_*_test.go` | [theme-look.md](theme-look.md) |
| Glass button, icon button, caret | `look/look.go`, `look/toolbar.go` | - | `look/glass*_gui*_test.go`, `look/caret_gui_test.go`, `look/iconbutton_gui_darwin_test.go` | `cmd/glassdemo` |
| Windows Fluent look helpers | `internal/fluent/` | windows only | `internal/fluent/glass_windows_test.go`, `internal/fluent/palette_windows_test.go` | - |

## icons

Package `icons`: named system icons plus the embedded Lucide set.

Packages: `icons`, `icons/lucide`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Icon lookup, theming | `icons/icons.go` | `icons/icons_{darwin,linux,windows}.go`, `icons/icons_freedesktop.go`, `icons/icons_template_{darwin,other}.go` | `icons/icons_test.go`, `icons/icons_gui_test.go`, `icons/icons_template_darwin_test.go` | `cmd/icondemo`; [icons.md](icons.md) |
| Lucide embedded set | `icons/lucide/` | - | - | [icons.md](icons.md) |

## svg

Package `svg`: optional SVG rasterizer plugged into `swt.Image`.

Packages: `svg`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| SVG loading and rasterizing for `swt.Image` | `svg/svg.go`, `swt/graphics_svg_manual.go` | - | `svg/svg_test.go`, `svg/example_test.go` | [icons.md](icons.md) |
| Scene-level SVG paths (`gfx`) | `gfx/svg.go`, `gfx/svgpath.go` | - | `gfx/svg_test.go`, `gfx/svgpath_test.go` | `examples/gfxcanvas` |

## rx

Package `rx`: reactive state (Svelte 5 style).

Packages: `rx`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Signals, derived, effect, batching | `rx/rx.go`, `rx/state.go` | - | `rx/rx_test.go`, `rx/bench_test.go`, `rx/graph_test.go` | [events.md](events.md) |
| QA read checks | `rx/reads_on.go`, `rx/reads_off.go` | build tag `rxdebug` / `!rxdebug` | `rx/catch_test.go` | [qa.md](qa.md) |

## SWT

Package `swt`: the translated full SWT API, one package for all platforms.
Files keep the Java class name: `widgets_<class>[_<platform>].go`,
`graphics_<class>[_<platform>].go`, `events_*`, `dnd_*`, `layout_*`, `printing_*`,
`accessibility_*`, `custom_*`, `browser_*`, `internal_*`. Unsuffixed files are shared;
`*_manual*.go` are hand-written.

Packages: `swt`, `swt/swtreflect`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Widget classes (Button, Label, Table, Tree, ToolBar, Shell, Display, ...) | `swt/widgets_<class>_*.go` | the per-OS part of the same family | `tests/swttests/junit_test_org_eclipse_swt_widgets_*.go` | [swt-direct.md](swt-direct.md), `examples/swtdirect` |
| Tree / Table (view-based rewrite, macOS default) | `swt/widgets_tree_*.go`, `swt/widgets_table_*.go` | same families; `swt/widgets_tree_manual_linux.go`, `swt/widgets_table_manual_darwin.go` | `tests/swttests/junit_test_org_eclipse_swt_widgets_{tree,table}*.go` | [view-based-table.md](view-based-table.md) |
| StyledText and custom widgets | `swt/custom_*.go` | per-platform parts of the same families | `tests/swttests/junit_test_org_eclipse_swt_custom_*.go` | `examples/styledtext` |
| Graphics classes (Image, GC, TextLayout, Path, Transform, ...) | `swt/graphics_<class>_*.go` | per-platform parts; `swt/graphics_imagecodec_manual.go` (codecs) | `tests/swttests/junit_test_org_eclipse_swt_graphics_*.go` | [graphics.md](graphics.md) |
| Accessibility classes | `swt/accessibility_*.go` | `swt/accessibility_*_{darwin,windows}.go` delegates | `tests/swttests/junit_test_org_eclipse_swt_accessibility_*.go` | [platforms.md](platforms.md) |
| DnD and layout classes | `swt/dnd_*.go`, `swt/layout_*.go` | per-platform parts | `tests/swttests/junit_test_org_eclipse_swt_{dnd,layout}_*.go` | `examples/dnd`, `examples/layouts` |
| Browser widget (SWT API) | `browser/` package | `browser/webbrowser_manual.go` | `browser/webbrowser_manual_test.go`, `tests/swttests/junit_test_org_eclipse_swt_browser_browser.go` | [webview-browser.md](webview-browser.md) |
| Printing classes | `swt/printing_*.go` | per-platform parts | - | [printing.md](printing.md) |
| Event and listener types | `swt/events_*.go` | - | `tests/swttests/junit_test_org_eclipse_swt_events_*.go` | [events.md](events.md), `swt/doc_manual.go` |

## Native bindings

`internal/cocoa` (AppKit, purego), `internal/win32` (Win32/GDI+/WebView2 COM,
`syscall.SyscallN`), `internal/gtk` (GTK 3/GLib/Pango/cairo, purego, GIR-generated),
`internal/webkit` (WebKitGTK 4.1, GIR-generated). Architecture: [internals.md](internals.md).

Packages: `internal/cocoa`, `internal/win32`, `internal/gtk`, `internal/webkit`, `internal/jrt`, `internal/dbus`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| AppKit bindings | `internal/cocoa/cocoa_<type>.go`, `internal/cocoa/*_manual.go` | darwin only | per-file `*_test.go` | [internals.md](internals.md) |
| Win32 bindings | `internal/win32/win32_<type>.go`, `internal/win32/*_manual.go` | windows only | `internal/win32/*_test.go` | [internals.md](internals.md) |
| GTK 3 bindings | `internal/gtk/gen_funcs.go`, `internal/gtk/gen_consts.go`, `internal/gtk/glue_*.go` | linux only | `internal/gtk/*_test.go` | [internals.md](internals.md) |
| WebKitGTK bindings | `internal/webkit/gen_funcs.go`, `internal/webkit/{load,signals,view,window}.go` | linux only | driven by `webview` GUI tests in the Linux stand | [webview-browser.md](webview-browser.md) |
| Java runtime subset | `internal/jrt/` | all platforms | `internal/jrt/*_test.go` | - |
| D-Bus helpers | `internal/dbus/` | linux | `internal/dbus/*_test.go` | - |

## gfx

Package `gfx` and subpackages: retained-mode scene graph and renderers.

Packages: `gfx`, `gfx/cg`, `gfx/cairo`, `gfx/d2d`, `gfx/internal/cache`, `gfx/internal/fonttest`, `gfx/internal/pdftest`, `gfx/layout`, `gfx/offscreen`, `gfx/pdf`, `gfx/pen`, `gfx/printing`, `gfx/synth`, `gfx/view`, `internal/pdfcheck`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Scene graph, paths, paints, effects (shadow in `gfx/picture.go`) | `gfx/scene.go`, `gfx/path.go`, `gfx/paint.go`, `gfx/comp.go`, `gfx/picture.go`, `gfx/curve.go` | - | `gfx/*_test.go` (`scene`, `path`, `svg`, `curve`, `shadow*`) | [gfx.md](gfx.md), `examples/gfxcanvas`, `examples/gfxeditor` |
| CoreGraphics renderer (macOS) | `gfx/cg/` | darwin (`gfx/cg/bind_darwin.go`) | `gfx/cg/*_darwin_test.go` | [gfx.md](gfx.md) |
| cairo/Pango renderer (Linux) | `gfx/cairo/` | linux (`gfx/cairo/bind_linux.go`) | `gfx/cairo/*_linux_test.go` | [gfx.md](gfx.md) |
| Direct2D renderer (Windows) | `gfx/d2d/` | windows (`gfx/d2d/call_windows_*.go`) | `gfx/d2d/*_windows_test.go` | [gfx.md](gfx.md) |
| Offscreen buffers, HUD | `gfx/offscreen/`, `gfx/hud.go` | per-OS backends in `gfx/offscreen/` | `gfx/offscreen/*_test.go`, `gfx/hud_test.go` | [gfx.md](gfx.md) |
| Layout engine (cassowary-style) | `gfx/layout/` | - | `gfx/layout/*_test.go` | `examples/gfxeditor` |
| PDF backend | `gfx/pdf/`, `gfx/pdf.go` | - | `gfx/pdf_test.go`, `gfx/pdf/*_test.go` | `examples/print` |
| Pen stroke expansion | `gfx/pen/` | - | `gfx/pen/*_test.go` | [gfx.md](gfx.md) |
| Synthetic rendering test corpus | `gfx/synth/` | - | `gfx/synth/*_test.go` | - |
| Interactive view, overlays, autoscroll | `gfx/view/` | per-OS `gfx/view/view_{darwin,linux,windows}.go` | `gfx/view/*_test.go` | `gfx/view/demo` |
| Widgets built on gfx (avatar, badge) | `avatar.go`, `badge.go` (root) | - | `avatar_test.go`, `badge_test.go` | [widgets.md](widgets.md) |

## Printing

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Facade print dialog and job | `print.go` | `print_{darwin,linux,windows}.go` | `print_gui_test.go` | `examples/print`; [printing.md](printing.md) |
| SWT printing classes | `swt/printing_*.go` | per-platform parts | - | [printing.md](printing.md) |
| gfx print rendering | `gfx/printing/`, `gfx/pdf.go` | `gfx/printing/printing_{darwin,linux,windows}.go` | `gfx/printing/printing_gui_test.go` | `examples/print` |

## webview

Hand-written system web view over the native engines ([webview-browser.md](webview-browser.md)).

Packages: `webview`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| WebView facade API, navigation, scripts | `webview/webview.go`, `webview/optional.go` | `webview/webview_{darwin,linux,windows}.go` | `webview/webview_gui_test.go` | `cmd/webviewdemo`, `cmd/minibrowser` |
| WKWebView bridge (macOS) | `webview/webview_darwin.go` | darwin | `webview/export_darwin_test.go` | [webview-browser.md](webview-browser.md) |
| WebView2 bridge (Windows) | `webview/wv2_*.go` | windows | `webview/webview_windows_test.go`, `webview/wv2_util_test.go`, `webview/export_windows_test.go` | [webview-browser.md](webview-browser.md) |
| WebKitGTK bridge (Linux) | `webview/webview_linux.go`, `webview/cookies_linux.go` | linux | `make test-webview-linux` (GUI tests in the Linux stand) | [webview-browser.md](webview-browser.md) |

## browser

SWT `Browser`, translated; the engine bridge is `browser/webbrowser_manual.go`.

Packages: `browser`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Browser widget and listeners | `browser/browser_browser.go`, `browser/browser_*listener*.go` | per-engine bridge in `browser/webbrowser_manual.go` | `browser/webbrowser_manual_test.go`, `tests/swttests/junit_test_org_eclipse_swt_browser_browser.go` | [webview-browser.md](webview-browser.md), `examples/browser` |

## jface

Translated slice of JFace: layout factories, widget factories, dialogs.
One source for every OS; see [jface.md](jface.md).

Packages: `jface`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Layout factories | `jface/layout_*.go` | one source | `tests/jfacetests/layout_*test.go` | `examples/jface` |
| Widget factories | `jface/widgets_*factory.go` | one source | `tests/jfacetests/widgets_*test.go` | `examples/jface`, `cmd/jfacedemo` |
| Dialogs and constants | `jface/dialogs_*.go` | one source | `tests/jfacetests/*` | [jface.md](jface.md) |
| Viewers support | `jface/viewers_*.go` | one source | - | [jface.md](jface.md) |

## QA

The QA driver: server in the app, tree walker, view overlay, test client.

Packages: `qa`, `qa/qatest`, `internal/qatree`, `internal/qaview`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| QA server, handlers, dialog | `qa/qa.go`, `qa/server.go`, `qa/reg.go`, `qa/dialog.go`, `qa/off.go` | `qa/qatest/` client | `qa/` via `make qa-check`, `qatree_*_gui_test.go` | [qa.md](qa.md), `examples/qa` |
| Widget tree dump and actions | `internal/qatree/` | `internal/qatree/act_{darwin,other}.go`, `internal/qatree/act_drag.go` | `internal/qatree/qatree_test.go`, `internal/qatree/types_test.go` | [qa.md](qa.md) |
| On-screen QA overlay | `internal/qaview/qaview.go` | - | `look/focus_qa_gui_test.go`, `look/toolbar_qa_gui_test.go` | [qa.md](qa.md) |

## Tests

Regression gates and translated suites; flow in [internals.md](internals.md).

Packages: `tests/swttests`, `tests/jfacetests`, `internal/shot`, `internal/snapcmp`, `internal/junit`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| SWT translated suites | `tests/swttests/` | - | `make test-swt` against `tests/expected*.txt` | [internals.md](internals.md), `tests/RESULTS.md` |
| JFace translated suites | `tests/jfacetests/` | - | `make test-jface` against `tests/expected_jface*.txt` | [internals.md](internals.md) |
| ControlExample snapshots | `tests/snapshots*/` | per-OS snapshot dirs | `make snap-check` | [internals.md](internals.md) |
| GUI tests (facade, look, gfx) | `*_gui_test.go` in root, `look/`, `gfx/` | per-OS variants such as `look/glass_gui_darwin_test.go` | `make check-gui` (GOWT_GUI_TEST=1) | [qa.md](qa.md) |
| Window capture, PNG compare | `internal/shot/`, `internal/snapcmp/` | per-OS capture | their `*_test.go` | - |

## Generators

Everything that writes committed code; rules and flow in [internals.md](internals.md)
and `tooling/j2go/README.md`.

Packages: `tooling/apidump`, `tooling/assoc`, `tooling/darwin`, `tooling/demovideo`, `tooling/doccounts`, `tooling/fieldreport`, `tooling/girgen`, `tooling/mapcheck`, `tooling/mksyso`, `tooling/testexec`, `tooling/webkitgen`, `winmanifest`.

| Feature | Primary | Platform | Tests | Example / docs |
|---|---|---|---|---|
| Java-to-Go translator (SWT, JFace, tests) | `tooling/j2go/` | runs per platform via `tooling/port.sh` | `tooling/j2go/` tests, `make gen-check` | [internals.md](internals.md), `tooling/j2go/README.md` |
| GIR generator (GTK) | `tooling/girgen/` | linux GIR files in the Linux stand | `make gtk-gen`, `make gen-check` | [internals.md](internals.md) |
| GIR generator (WebKitGTK) | `tooling/webkitgen/` | linux | `make webkit-gen`, `make gen-check` | [internals.md](internals.md) |
| API snapshot gates | `tooling/apidump/` | per-OS exports | `make api-check` | [internals.md](internals.md) |
| Windows manifest objects | `tooling/mksyso/`, `winmanifest/` | windows | `make winmanifest` | [internals.md](internals.md) |
| Port/test helpers | `tooling/port.sh`, `tooling/testexec/`, `tooling/swtdiff.py`, `tooling/swtvmexpected.py` | - | `make test-swt`, `make test-swt-update` | [internals.md](internals.md) |
| Doc test-count gate | `tooling/doccounts/` | - | `make docs-counts` | [platforms.md](platforms.md) |
| Code map gate | `tooling/mapcheck/` | - | `go test ./tooling/mapcheck/`, `make map-check` | this file |

## Commands and example apps

The demo and tool programs in `cmd/` and the sample apps in `examples/`. The
apps are documented one by one in [apps.md](apps.md); `cmd/fieldcheck` pairs
with [field-check.md](field-check.md). Feature demos are linked from the
sections above.

Packages: `cmd/backdropdemo`, `cmd/controlexample`, `cmd/fieldcheck`, `cmd/finderdemo`, `cmd/form`, `cmd/glassdemo`, `cmd/gowtdemo`, `cmd/hello`, `cmd/hellogowt`, `cmd/icondemo`, `cmd/images`, `cmd/jfacedemo`, `cmd/minibrowser`, `cmd/paint`, `cmd/snapcheck`, `cmd/stack`, `cmd/swttest`, `cmd/themecheck`, `cmd/tree`, `cmd/webviewdemo`, `cmd/widgets2`, `cmd/winprobe`, `examples/apps/chat`, `examples/apps/dashboard`, `examples/apps/editor`, `examples/apps/files`, `examples/apps/internal/kit`, `examples/apps/mail`, `examples/apps/monitor`, `examples/apps/monitor/sampler`, `examples/apps/settings`, `examples/apps/tracker`, `examples/boundary`, `examples/browser`, `examples/catalog`, `examples/clipboard`, `examples/controlexample`, `examples/controlexample/res`, `examples/dialogs`, `examples/dnd`, `examples/events`, `examples/gfxcanvas`, `examples/gfxeditor`, `examples/gfxpng`, `examples/graphics`, `examples/hello`, `examples/icons`, `examples/jface`, `examples/layouts`, `examples/materialpanel`, `examples/openfiles`, `examples/print`, `examples/qa`, `examples/quit`, `examples/rows`, `examples/sidebartable`, `examples/styledtext`, `examples/swtdirect`, `examples/theme`, `examples/webview`, `examples/widgets`.
