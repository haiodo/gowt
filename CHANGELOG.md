# Changelog

## Unreleased

gfx HUD layer
- New: `Compositor.HUD()` returns a second `Scene` in view points that is drawn over the scene, the lifted nodes and the sprites and ignores the camera (edge indicators, rulers, minimap, HUD text). It is drawn into one image of the bounds of its content, again only when it changes or the view size or backing scale does. `Compositor.OnViewport(f)` and `Viewport()` place nodes on resize and scale change; `Compositor.HUDHit` and `view.View.HUDHitTest` hit-test it; `view.Pointer.HUD` is the HUD node under the pointer (and `Pointer.Node` is nil then). New layer group `gfx.GroupHUD` above `GroupOverlay`: every `gfx.Host` must keep a fourth group (cg, cairo and d2d hosts do). Checked headless with a fake host, on cg and cairo hosts, in GUI tests on Linux (Docker, Xvfb), on a second Mac and on the Windows 11 arm64 VM (d2d host and `TestHUDGUI`).

QA environment (Tenniarb)
- New: a QA run has its own clipboard. `gowt.Clipboard` and `swt.Clipboard` read and write `clipboard.json` under `GOWT_QA_DATA` (text, RTF, HTML, URL, PNG image, files, custom types), so the system clipboard keeps what the user copied. `Client.Clipboard()` and `Client.SetClipboard(qa.Clipboard)` read and set it; `App(t)` empties it. The hook is in `swt/qa_clipboard_manual.go` and `tooling/port.sh` (the five `Clipboard` methods of each platform), only in a `-tags gowtqa` build with `GOWT_QA_SOCKET` set. The native copy and paste of a text control still use the system clipboard.
- Fixed: QA `Key("Tab")` and `Key("Shift+Tab")` send the `Traverse` event and do the default traversal unless vetoed; `Esc` and `Enter` send `Traverse` to the listeners too. A `Tree` that takes Tab can be tested.
- Fixed: the QA socket was under `TMPDIR`, whose macOS path is near the unix limit; with a longer one the app failed with `bind: invalid argument`. The socket now lives in a short directory (`$XDG_RUNTIME_DIR`, `/tmp`), and a path that is too long fails with a message naming the limit.
- Tests: `TestVirtualClipboard` (checks `pbpaste` on macOS), `TestKeyTraverse`, `TestShortDir`: Docker Xvfb, the second Mac and the Windows 11 arm64 VM (`TestShortDir` is skipped on Windows, where the app uses a loopback port).

Packaging (Tenniarb stage 9)
- New: `mkapp.sh` takes `BUNDLE_ID`, `VERSION`, `BUILD`, `ICON` (an `.icns` or a PNG, converted with `sips` and `iconutil`, written as `AppIcon.icns` with `CFBundleIconFile`), a PNG for `DOC_ICON` and `DOC_UTI_CONFORMS` (default `public.data`). Tested on macOS by reading the plist back.
- New: `tooling/mksyso` writes one `.syso` with a manifest, an icon (`.ico` or PNG sizes) and a VERSIONINFO resource (`-icon`, `-version`, `-product`, `-company`, `-description`, `-copyright`, `-original`, `-internal`), for amd64 and arm64; `-manifest` is optional. The object is parsed back and a linked program's `.rsrc` is read. `-tags gowt_nomanifest` stops the facade from importing `winmanifest`, for an app with its own manifest. Checked on the Windows 11 arm64 VM: Explorer shows the icon and the version fields read back.
- New: `tooling/assoc -kind icons` writes hicolor mimetype (or app) icons from a PNG; `-kind mime -icon` names the icon in the XML; GNOME Files on Ubuntu 26.04 showed it for a file of the type. docs/packaging.md.

QA windows and popovers (Tenniarb)
- New: `Client.ShotOf(name, sel)`, `ShotTitle(name, title)` and `ShotWindow(name, index)`: a shot of the window that holds a control, has a title, or is at an index of the dump; the `shot` command takes `selector`, `title` or `window`. Before, only window 0 could be shot. On Windows the window order is the stacking order, so use the selector or the title there.
- Fixed: QA `Key("Esc")`, with the focus in the popover or not, now reaches an open `Popover` (its `OnEscape`, then close); `Close` on a `Popover` root closes it. Esc posted to a control did not pass the display filter the popover listens with.
- Changed: on macOS a `Shot` draws a popover from its window as the window server has it (CGWindowListCreateImage, like `GOWT_SNAP_SCREEN`) and places it by its screen rectangle; the view capture of the NSPopover window came out black with a block of colour. The popover shows the right shape and arrow in the shot; its content is blank on the second Mac, not understood yet.
- Fixed: a macOS popover shell is not in `Display.GetShells` while its window holds the placeholder, so the QA dump lost it on later opens; the driver keeps its own list. A popover that is opening (macOS attaches its window up to half a second late when the app is inactive) makes `settle` wait ("a popover opening").
- Tests: `TestShotOfOtherWindow`, `TestPopoverEscape`, `TestPopoverEscapeOnControl`, `TestPopoverCloseRoot` on Xvfb, the Windows VM and the second Mac.

macOS: a nil NSString no longer kills the app
- Fixed: `Window.DirDialog` without a start path crashed on macOS 26 with `NSInvalidArgumentException ... nil string parameter`: SWT's `DirectoryDialog` passes a null path to `NSURL.fileURLWithPath`, which os.c's `@try` used to hide. 28 selectors that throw on a nil NSString (menu and window titles, `setStringValue:`, `appendString:`, `stringByAppendingString:`, `fileURLWithPath:` and others) now go through hand-written wrappers in `internal/cocoa/nonnilstring_manual.go`: a nil text is sent as an empty string, a nil path gives nil.
- `examples/quit` is 360x150 instead of 96 points wide.

QA context menus filled in OnShow
- Fixed: items that an `OnShow` handler adds to an empty `PopupMenu` (on a gfx canvas, a tree) were not reachable in QA. A dump or `Find` now sends Show to an empty context menu once per action, a QA right click (`RightClickAt`, `Pointer` button 3) sends it to the menu of the control, and `Click` on an item still runs `OnShow` first. No native menu opens. Test `TestPopupMenuFilledInOnShow` on Xvfb, the Windows 11 arm64 VM and the second Mac.
- Fixed: the macOS QA dump panicked on an open `Popover` (the shell's client area is gone once its view is in the NSPopover); `TestPopover` now passes there. `TestFocus` and `TestKeys` fail on that Mac on main too.

QA driver for editors (Tenniarb)
- New: QA mouse input. `Element.ClickAt`, `DoubleClickAt`, `RightClickAt`, `DragTo` and `Pointer(qa.PointerEvent)` (action `pointer`) deliver a press, moves and a release at a point. A gfx view gets them through its input router like the native events (`OnPointer`, `View.Drag`, inline editors); any other control gets `MouseDown`, `MouseMove`, `MouseUp` and `MouseDoubleClick` listener events. A press over a control placed on the view is refused. See docs/qa.md.
- Fixed: QA `Type`, `Key` and `Activate` on a control that has the focus asked for the focus again, which on macOS ends the edit of a text field, so an inline `view.Edit` field committed before the text arrived. They now leave the focus alone when the control has it; `Focus` too. The macOS cause is a hypothesis (the field editor, not the field, is the first responder) and was not reproduced; the Linux and Windows runs pass with and without the change.
- Fixed: a QA pointer press focuses the view without activating the window (the view's own `SetFocus` needs an active window, which a QA run on macOS never has). `docs/qa.md` has the pattern: press, then `app.Key`.
- New: the QA dump lists the context menu of a control (`gowt.PopupMenu`) as a `Menu` node with its `MenuItem`s, and an open `Popover` as a root with class `Popover`. `Click` on a context menu item runs `OnShow` first, then the item. `Close` on a popover closes it. `Shot` draws the open popovers of the window into its image, which grows when one reaches past the window; on Windows it then does not bring the window to the front, which would close the popover. The shot's window index and the `windows` command skip popover shells (a popover created at start was window 0 on Windows).
- The `qa/qatest` suite ran on the Windows 11 arm64 VM and under Xvfb (Docker stand): `TestGfxPointerAndEditor`, `TestPointerPlainControl`, `TestPopupMenu`, `TestPopover`; macOS not run.

QA shot overlay on macOS
- Fixed: a QA `Shot` and `GOWT_SNAP` of a gfx view on macOS drew the tiles twice where `cacheDisplayInRect` already had the layers (Darwin 25.5): translucent pixels such as shadows came out darker (alpha 1/3 gave 0.56). The canvas rectangle is now captured with the tile layers hidden, the tiles are drawn over it from the compositor's images, and the result replaces the rectangle, whichever way the capture treats layers. `TestGfxShotWaitsForFrame` checks a translucent square for the exact colour; the macOS run is still to do.

gfx screen-space shadows
- New: `Shadow.Screen`: `DX`, `DY` and `Blur` are screen points, applied at rasterization and independent of the node and camera scale (CoreGraphics `setShadow` parity); no rotation. The device pixels per point come from `gfx.ScreenScaler.SetScreenScale`, which the `Compositor` calls with the display's dpr before each tile; other targets use 1. Works with `FillShadowPath`, `StrokeShadowPath`, `TextShadow` and `FillShadowRRect` (which skips the sprite) on cg, cairo and d2d. The recorded stream gains one byte per shadow, so pictures recorded by an older build are not readable by this one (they are not stored anywhere).

Drag and drop in a Tree (TSK-2026-10-24-gfx-tenniarb-12 D5)
- New: QA `Element.DragDropTo(target)` and `DragDropToAt(target, x, y)` (action `drag`) run a drag between two controls or rows of the app without an OS drag session: `DragDetect` on the source, `DragStart`, `DragEnter`, `DragOver`, `DragLeave`, `DropAccept`, `DragSetData`, `Drop`, `DragEnd`, with the types and operations the platform's `DropTarget` code picks and the data passed through both `Transfer`s (`swt.QADrag`). `DragFrom` `OnStart`/`OnDone` and `DropOn` `Over` and the drop callbacks run for a Tree, a Table and a canvas. A QA run before had no way to start a drag. See docs/qa.md. Tests `TestDrag*` in `qa/qatest` pass on Linux (Xvfb), the Windows 11 arm64 VM and a second Mac (macOS 27).
- Fixed: `DropOn` refused a plain drag (no key held) whenever the target did not allow move: the default operation became "none" for the default `Ops` (copy), so a canvas or a tree with `Ops` unset never got a drop. The target now asks for DROP_DEFAULT and picks move, else copy, else link. Failed on Linux with a real pointer before (`TestDndRealMouse`) and with `DragTo`.
- Fixed: Windows, `DoDragDrop` reported a wrong or empty effect to the drag source about every second drag (`DragFinished` got `OpNone` or copy and move together after a move): the effect array lived on the goroutine stack, and a callback during the OLE loop moved the stack. The call now keeps its pointer arguments on the heap (`syscallEscaping`, j2go `WinNativeEmitter`). `TestDndRealMouse` failed 5 of 12 runs before and passed 10 of 10 after. The other natives that can call back into Go with a pointer argument are covered under "Windows natives that call back".
- New: `TestDndRealMouse` drags Tree to Tree and Tree to a canvas with the OS pointer (xdotool, SendInput, CGEventPost). Linux and Windows pass; macOS not run: the second Mac has no Accessibility permission for `CGEventPost`, the test skips there.

Print and PDF (TSK-2026-10-24-gfx-tenniarb-14 D7)
- New: `Window.PrintDialog(PrintOptions)` returns a `PrintJob`; `PrintToPDF(path, opts)` is a job with no dialog; `PrintJob.Print(pages, draw)` calls `draw(*PrintPage)` per page of the dialog's range. A `PrintPage` has the paper, the printable area and the area inside `PrintOptions.Margin` in points, `GC()` (an `*swt.GC` in points) and `Native()`. QA: the dialog kind `print` takes the path of a PDF to write.
- New: package `gfx/printing`: `Scene`, `Picture` and `Page` draw gfx on a print page, fitted and centred in the area. macOS draws with CoreGraphics and Linux with cairo straight into the print context (vector, text); Windows renders at the printer's resolution in bands and sends bitmaps.
- New: PDF export. `gfx.PDFRenderer`, `Document`, `Page`, `PDFOptions` (page size, margin, scale, background), `WriteSubtreePDF`, `WritePicturePDF`, `offscreen.PDF` and `offscreen.NewPDF` (several pages). Backends: CoreGraphics PDF context on macOS, cairo PDF surface on Linux, and on Windows the new pure Go writer `gfx/pdf` (embeds the glyphs used from the font file; no shaping; shadows are stacked translucent strokes), chosen over the Microsoft Print to PDF printer because that one is an optional feature and prints Direct2D as bitmaps.
- Tests: `internal/pdfcheck` reads a PDF back (pages, sizes, fonts, images) for `gfx/pdf`, `gfx/cg`, `gfx/cairo`, `gfx/d2d`; on macOS the PDFs of CoreGraphics and of `gfx/pdf` are rendered by CoreGraphics and compared with the bitmap renderer. Print tests (`GOWT_GUI_TEST=1`, a PDF file as the target) ran on Linux in the Docker stand, on the Windows 11 arm64 VM and on the second Mac. A Mac with no printer installed cannot start a job: `Print` returns an error. docs/printing.md, `examples/print`.

Open documents
- New: `App.OnOpenFiles(func(paths []string))` runs on the UI thread with the files to open: Finder (double click, Open With, `open -a App.app file`, at launch and while running) on macOS, command-line arguments that name existing files on every OS, and, with the opt-in `gowt.SingleInstance(id) (bool, error)`, the files of a second launch forwarded to the running app (it also raises its window). Paths are absolute and de-duplicated per call; `file:` URLs become paths; files that arrive before the first handler are kept for it. `swt.OpenDocument` still fires on macOS. Linux had no path at all since the D-Bus code left with the LGPL classes: the new `internal/dbus` is a session bus client written from the D-Bus specification serving `org.freedesktop.Application` (`Open`, `Activate`), so `Exec=app %U` and `DBusActivatable=true` work; Windows uses a Unix socket in the temp folder. A missing session bus gives an error and the program runs without forwarding. `examples/openfiles`.
- New: file association. `tooling/darwin/mkapp.sh` takes `DOC_EXTS`, `DOC_NAME`, `DOC_ICON`, `DOC_UTI` and writes `CFBundleDocumentTypes` with the icon; new `tooling/assoc` writes a Windows `.reg`, a Linux `.desktop`, a D-Bus service file and a shared-mime-info XML. docs/packaging.md, docs/events.md. Tests: `go test . ./internal/dbus ./tooling/assoc` (the two-process tests run on Linux with a session bus and on Windows; `gdbus call Open` against a running example on Linux in Docker; `open -a` with a file at launch and while running on a second Mac, macOS 27). Not run: a double click on a registered type, Open With from a real file manager, `DBusActivatable` start of a stopped program, the Windows `.reg` import.

Windows natives that call back
- Fixed: Windows, the same stale-stack defect as `DoDragDrop` in every other generated native that pumps messages or calls back into Go while it holds a pointer argument: `GetMessage`, `PeekMessage` (a send from another thread is dispatched inside the call; the message came back all zero), `TrackPopupMenu`, `MessageBox`, `ChooseColor`, `ChooseFont`, `PrintDlg`, `DocumentProperties`, `DragDetect`, `ShellExecuteEx`, `EnumDisplayMonitors`, `EnumFontFamilies`, `EnumEnhMetaFile`, `CoCreateInstance`, `CoGetClassObject`, `OleCreate`, `OleCreateFromFile`, `OleCreatePropertyFrame`, `OleGetClipboard`, `SHParseDisplayName` and every COM method call (`VtblCall`). They go through `syscallEscaping`, which keeps the pointer arguments on the heap (j2go `WinNativeEmitter` ESCAPING). Tests in `internal/win32/modal_escape_windows_test.go` fail without it on arm64 and amd64 (emulated) and pass with it. The 29 `SendMessage` overloads that take a pointer (text buffer, `RECT`, `TVITEM`, ...) escape too: a window procedure that fills the buffer after growing the stack wrote to a stale address (`TestSendMessageSurvivesStackGrowth`). The overload with integer arguments does not (an escaping call allocates its argument list). `BenchmarkSendMessageMix` (8 integer messages, one text buffer, one `RECT`): arm64 350 -> 390 ns, amd64 emulated 730 -> 820 ns per mix, 4 allocations. `CallWindowProc` and `DefWindowProc` take integers only, so there is no Go pointer to move.

gfx shadows, ArcTo, SVG (Tenniarb request)
- New: `Recorder.FillShadowPath(p, fill, sh)`, `StrokeShadowPath(p, st, c, sh)` and `TextShadow(s, x, y, font, c, sh)`: a blurred drop shadow under any path fill, any stroke (ellipses, `RoundRectPath4`, boxes with no fill, lines) and under text, on cg, cairo and d2d. New ops `OpFillShadow`, `OpStrokeShadow`, `OpTextShadow`, appended after `OpEndLayer`. `FillShadowRRect` is unchanged.
- New: `Path.ArcTo(x1, y1, x2, y2, r)` with HTML canvas `arcTo` semantics; tests against the canvas results. `Path.Arc` no longer splits an exact quarter turn in two cubics because of rounding.
- New: `gfx.WriteSubtreeSVGWith(..., SVGOptions{Background})` fills a rect behind the content. Text with neither `Name` nor `Family` writes `font-family="Helvetica Neue, Helvetica, Arial, sans-serif"`, which changes the SVG of such text.
- Fixed: cg took the shadow offset as y-down device pixels, so a node under a negative-determinant transform (`Scale(1, -1)`) had a mirrored shadow, and a rotated one an unrotated shadow. The offset is now taken through the CTM, as cairo and d2d always did. Tested on macOS (cg), Linux Docker (cairo) and the Windows 11 arm64 VM (d2d); the shadow tests are new, `TestHashAcrossRuns` in d2d fails the same way before this change.

Viewer gaps (Tenniarb)
- New: `CanvasView.OnResize(func(w, h int))` runs with the client size after layout gave the canvas a new size, and once for the first size known (also when the canvas has one already, then on the next turn of the event loop). `view.View.OnResize` is the same for a gfx view, so fit-to-window no longer waits for `OnFrameReport`. Checked under Xvfb, on the Windows 11 arm64 VM and on the Ubuntu 26.04 Wayland VM (`TestCanvasOnResize`, `TestGfxResize`); not run on macOS yet.
- Changed: a QA `Shot` and a `GOWT_SNAP` capture ask every gfx view in the window for a frame and wait (3 s at most) until one came after the request and no tile is queued, drawn or waiting to be painted, so a shot taken right after a camera change shows it. A view that animates all the time is captured after the 3 s. Before, the shot could hold the frame before the change. `TestGfxShotWaitsForFrame` fails without the wait (Xvfb) and passes on the Windows VM; macOS: the view capture (`cacheDisplayInRect`) does not draw CALayer contents, so the tiles of a gfx view were missing from a QA `Shot` and `GOWT_SNAP`; `cg.Host.Render` now draws the layers' own images over the capture (`TestHostRender`, headless; the shot tests on a Mac still to be run).
- New: QA actions `scrollevent` and `pinch` (`Element.ScrollEvent(qa.ScrollEvent)`, `Element.Pinch(qa.PinchEvent)`) hand scroll deltas with phase and momentum, and pinch magnification, to a gfx view through the router its native events use; `OnScroll`, `OnMagnify` and the built-in pan and zoom run, the camera has moved when the call returns, no OS event is sent. See docs/qa.md.
- Changed: in the QA dump a `MenuItem` has the title in `Text` without the mnemonic `&`, and an accelerator that swt keeps in the text (`Run\tCtrl+R`) in `Value`, so `qa.Text("Run")` finds the item. New test `TestMenuBar` clicks a menu item through the dump (Linux: passes). The macOS dump read the menu from the shell's menu bar object and the code shows no reason for it to be empty; the report that it lists no `MenuItem` is not reproduced and the macOS run is still to do.

App shell (Tenniarb)
- Fixed: `Display.GetSystemMenu()` threw `NSInvalidArgumentException` "characterAtIndex: Range or index out of bounds" on macOS: `Menu.createItem` reads `keyEquivalent.characterAtIndex(0)` and the key equivalent of About and Preferences is empty. SWT's os.c answers 0 when AppKit throws; Go cannot catch it, so `NSString.characterAtIndex` is hand-written with a range check (`internal/cocoa/nsexception_manual.go`, skipped in the translator's `Manual.java`); `gen-check` is green.
- New: `App.OnAbout(f) bool` and `App.OnPreferences(f) bool` hook the About and Preferences items of the macOS application menu. On Windows and Linux they return false and the app puts its own items in the Help and Edit menus (pattern in docs/dialogs-menus.md). Test `TestAppMenuHooks`.
- Changed: `App.After` returns `cancel func()`. Source compatible for callers that ignore the result; the facade API golden changed (`After` gained a result), and code that passes `app.After` as a `func(time.Duration, func())` value no longer compiles. A cancelled short timer leaves `Pending` at once. Test `TestAfterCancel`.
- New: QA `Type` and `Key` work on a Spinner. `Type("42")` sets the focus and the value per character (clamped, decimals up to the spinner's digits, read-only fails); `Key("Up")`, Down, PageUp and PageDown step the value; Enter sends DefaultSelection. Test `TestQATreeSpinner` (`GOWT_GUI_TEST=1`) passes on Linux (Docker, Xvfb), the Windows 11 arm64 VM and the second Mac.

Toolbar QA (Tenniarb)
- Fixed: `look.SetSidebar` after `look.SetToolbar` crashed the QA dump (SIGSEGV) on macOS: the sidebar rebuilds the NSToolbar and the search handle kept the freed `NSSearchField`. The handles now look up the current toolbar on every call, so the order of the `look` calls does not matter; `TestToolbarQA` runs both orders.
- Fixed: after `look.FocusSearch` on macOS the QA `Focused` stayed false. `FocusSearch` now lays the toolbar out and makes the field the first responder at once, without activating the app.
- New: the toolbar items and group segments are `ToolItem` nodes in the QA dump (`ToolbarItem.ID`, `Title` as `Text`) and `Click` runs their `OnClick`, on macOS (NSToolbar) and in the bar of Windows and Linux. An icon-only `ToolItem` shows its tooltip as `Text`.

Toolbar search (Tenniarb)
- New: `look.FocusSearch(w) bool` focuses the toolbar search field made by `look.SetToolbar` (`ToolbarItem.OnSearch`): the NSSearchToolbarItem on macOS (a collapsed one expands), the search `Text` of the bar on Windows and Linux. `ToolbarItem.ID` names it. On Windows and Linux Enter in the field now runs `OnSearch` with the text.
- New: the toolbar search field is in the QA dump (a `Text` with the item ID, placeholder as `Text`, content as `Value`) and takes `Type`, `Key("Enter")`, `Key("Esc")` and `Focus` on macOS too. Test `TestToolbarSearch` (`GOWT_GUI_TEST=1`) on Linux Docker, the Windows VM and the second Mac.

Quit hook
- Fixed: macOS Quit after a Save question crashed with NSRangeException; Dock/logout quit now runs dispose handlers. The delegate's `applicationShouldTerminate:` no longer disposes inside the event loop (an allowed quit ends `Run`), `applicationWillTerminate:` disposes on the terminate: path, and windows AppKit closes after an approved quit skip their `OnClose`. `TestQuitExitsCleanly` (`GOWT_GUI_TEST=1`, macOS).
- New: `App.OnQuit(func() bool)`; false cancels the quit. It runs on Cmd+Q, the app menu and the Dock (macOS), logout and shutdown (macOS, Windows) and the close of the last window, after every open window's `Window.OnClose` agreed (asked in creation order, the first refusal stops). `app.Unwrap().Close()` takes the same path for an own Quit item; `app.Quit()` still ends without asking. A `MessageBox` from a handler works during the system's quit request. Linux has no session-end path (GTK 3 without `GtkApplication`). Several `Window.OnClose` calls on one window now stop at the first false. Tests `TestQuit*` (`GOWT_GUI_TEST=1`) run on Linux under Xvfb and on the Windows VM; macOS not run. `examples/quit`, docs/events.md.

Emoji in text (TSK-2026-10-24-gfx-tenniarb-11 D3, TSK-2026-10-03-product-36)
- Fixed: editing text with surrogate pairs (emoji, ZWJ sequences, flags) no longer panics in `StyledText`. The cause was in `jrt` (UTF-16 indices over UTF-8 strings, no lone surrogates) and was fixed earlier by the WTF-8 work and the `toCharArray`/`charAt`/`new String(char[])` generation; the defect note in the docs was stale. New tests pin it: `swt/custom_defaultcontent_test.go` (no window, fails on the commits before that work) and `swt/custom_styledtext_surrogates_gui_test.go` (`GOWT_GUI_TEST=1`: typing, paste, cut, delete, caret moves, selection, wrap, `StyleRange` inside a pair, `Text`, `Combo`, `Label`). Run on Linux only; macOS and Windows not run.

Controls over the gfx view
- New: `view.View.Overlay(opt, rect, font)` places a gowt control over the view at a scene rectangle; it follows pan, zoom and a backing-scale change, its font follows the zoom, `Remove` disposes it. `view.View.Edit(EditOptions)` is an inline `Text` on it: Esc cancels, Enter (Cmd or Ctrl+Enter when `Multiline`) commits, a press on the view or the focus leaving commits. `CanvasView.AddChild` and `SetChildBounds` are the facade calls under it.
- Fixed: wheel, pinch and smart zoom over a control that lies over the view went to the view (macOS, Windows, GTK), so a `Text` or a scrolling control over the canvas could not be scrolled. They go to the control now. Checked under Xvfb; not run on macOS or Windows yet.
- `examples/gfxeditor`: double-click a topic to edit its title in place.

Popover
- New: `Window.Popover(opts...)` returns a `Popover`, a floating panel with ordinary widgets: `ShowAt(anchor, edge)`, `ShowAtPoint(anchor, x, y, edge)`, `Close`, `Shown`, `OnClose`, `OnEscape`, `SetSize`, `Add` and `Dispose`. It closes by a click outside, Esc and loss of the application focus, and keeps itself on the screen. macOS uses `NSPopover` (not run on a Mac yet); Windows and Linux a borderless window, rounded on Windows 11. The Goto-item popover is in `examples/dialogs`. See docs/dialogs-menus.md.

gfx/pen absolute children
- Fixed: `gfx/pen` reads `layoutPosition: "absolute"`. Such a child sits at its x and y in the parent, takes no gap slot and does not grow a hug parent; before, it was laid out in the flow.

CTabs look
- Changed: the selected `CTabs` tab has no accent line any more: it is the content surface with round top corners (radius 8 on Windows, 6 on GNOME, square on macOS) and covers the strip hairline; unselected neighbours have a thin divider on Windows and macOS. The focus ring shows only after Tab, an arrow key or Space and hides after a mouse press, so not for the focus a window starts with.
- Changed: `CTabs` draws a modern document tab bar on every OS: a flat strip with a hairline in the system separator colour, the selected tab in the folder's background (so it joins the content) with a 2 px accent line, secondary text on the other tabs with a hover fill, the Type.Body font, the tab height from Spacing.ControlHeight, a round close button shown on the selected and the hovered tab, and an accent ring on the selected tab while the folder has the focus. Set the folder's `Background` to the editor's colour so that the tab joins it. A `CTabs` now selects its first tab when created; SWT did so only when the folder got the focus.
- Changed: j2go has `dispatch.properties`: a class listed there dispatches through `impl` and exports its protected constructors, so Go code can subclass it. `swt.CTabFolderRenderer` is the first: embed it, `NewCTabFolderRenderer(folder)`, `SetImpl_(x)`, define `Draw_`, `ComputeSize_`, `ComputeTrim_`, `Dispose_`, then `CTabFolder.SetRenderer(x)`.

Spacing
- Changed: a `look.SetSidebar` sidebar gets an 8 point side margin on Windows and Linux, as the macOS glass has. On Windows its Tables, Trees and Lists take the window background, rows are `Spacing.SidebarRow` high and the pill spans the list with the icon 10 and the text 10 after it; on Linux the sidebar tree view is transparent and cells have 10 points of side room.
- Changed: Windows Tables are `Spacing.RowHeight` high after `SetDensity`; the base `RowHeight` is 28, not 32 (32 on Windows and Linux at native density).
- Changed: on Windows `Destructive()` is a standard button; only `Primary()` together with it gets the critical fill.

API gaps found by examples/apps (TSK-13)
- New: `Primary()` and `Destructive()` for a push `Button`. Primary is the window's default button (Enter) with the system's prominent look; Destructive marks a deleting action. macOS: the accent default button and `hasDestructiveAction` (not run on a Mac yet); GTK: `suggested-action` and `destructive-action`; Windows: the WinUI accent and critical fills with hover and pressed states, painted by gowt.
- New: `GC.Size()`, the drawable size at paint time; a chart can follow its cell and font. `CanvasView.Size()` was already right and gives the same numbers.
- New: `LineNumbers()` and `StyledTextView.SetLineNumbers` for a gutter of line numbers made of SWT bullets: it scrolls with the text, keeps the line and paragraph spacing, follows the font and the theme.
- New: `Fit()` on a Table or Tree column keeps it as wide as header and widest cell after row changes (also with `Rows` and `Nodes`); `FitColumns()` does it once. Linux measures the texts itself, because SWT's `Pack` gave the header width only.
- New: `App.AvatarImage(initials, colour, size)` and `App.CircleImage(img, size)`: round images with a smooth edge, drawn again at every display zoom.
- New: `App.SetAppearance(AppearanceSystem | AppearanceLight | AppearanceDark)` and `App.Appearance()`; `Dark()` and `OnThemeChange` follow it. Live on macOS and Linux; on Windows the title bars switch live and the content only for windows created afterwards, and part of it stays light (docs/cross-platform-look.md). `swt.Display.ForceColorScheme` is the platform call under it.
- New: `Fields(Field(label, control)...)`, a two-column form whose labels are trailing on macOS and leading on Windows and GNOME; `FieldTop()` for tall controls.
- Decision: no badge or chip widget. A pill stays a small `Canvas` (docs/widgets.md "Badges and chips").
- examples/apps use the new calls instead of their workarounds; `kit.Appearance` reads `GOWT_APPEARANCE=dark|light`.

Windows look (Windows 11)
- The window background is the Fluent colour (F3F3F3, 202020 in the dark theme) instead of the system F0F0F0; labels, groups, tab pages and the content of `Scroll` take it. Inputs and tables keep their own white. `look.SetDarkContent` stays opt-in; nothing turns it on by itself.
- With `App.SetDensity`, push `Button`, drop-down `Combo`, `Spinner` and `DateTime` are `Spacing.ControlHeight` high (32 at `DensityNative`), `List` and `Tree` rows `Spacing.RowHeight`. Without `SetDensity` the native heights stay. A `Spinner` line is centred in its taller edit.
- `Progress` is the WinUI bar (hairline track, 3 px rounded accent indicator, red and yellow for error and paused), with its own animation for `Indeterminate()`. `Scale` is the WinUI slider (4 px track, accent up to a round thumb with a dot that grows on hover); the native trackbar still handles mouse, keyboard and accessibility.
- `List` and `Tree` select with the rounded pill of `Table` (accent when focused, grey otherwise), with a hover fill. `Text` with `Search()` has the Fluent frame and a centred line.
- `Section` cards are the WinUI card fill (white at 70% over the window colour) instead of a darkened grey.
Docs images
- `GOWT_SNAP` captures the whole window with its frame on every OS (Windows: the DWM frame bounds copied from the screen; Linux: the CSD title bar with the shadow margin trimmed), and `GOWT_SNAP_SIZE=<w>x<h>` exports it at exactly 2 pixels per point. The catalog opens every sample at one outer size from a table, so the three images of a sample have one size; `make shot-sizes` checks it. `tooling/docs-shots-vm.sh` sets the VM display scale for the run and restores it (`tooling/vm-scale.sh`).

Linux look (GTK)
- A default stylesheet is installed by `Run` at GTK application priority (below any app CSS): flat table and tree headers without column dividers, roomier rows (`-GtkTreeView-vertical-separator: 6`), and `Group` as a card with its title above it instead of in the frame line. Theme named colours only, so Adwaita, Yaru, light and dark follow.
- `look.SetSidebar` marks the sidebar `gowt-sidebar`: a selected row in a List, Table or Tree there is a neutral rounded fill with the normal text colour instead of the accent bar; other lists keep the accent. The rule outranks the portal accent rule. Row insets are not possible with GTK 3 CSS.
- New: `App.SetLinkColor(*Adaptive)` sets the link colour of every `Link` (light and dark, existing and later ones, follows a theme switch); nil restores the system colour. Works wherever SWT `Link.SetLinkForeground` does.
- `Type.Mono` and the system monospace font now come from GSettings in process (no `gsettings` binary needed); "Monospace" without the schema.
- `List`, `Table`, `Tree` and `Scroll` show overlay scroll bars on overflow only, as `Text` with `Multiline` does.
- Fix: a `Text` without a frame was white in a dark window when the portal switched the theme after the display existed: SWT baked the light colour read at creation into the entry CSS. The entry now uses `@theme_base_color`.
- Fix: `GOWT_SNAP` at scale 2 captured only the top-left quarter; the window is read at width and height times the scale factor.
- Known: `List`, `Tree` and `Table` still get their minimum width as natural size, so a window packed around them shows "al..." (cause: the cell renderers end text with an ellipsis, which makes GtkTreeView report the ellipsis width as the column minimum); a `Button` with `Check()` or `Radio()` can lose its last letter (measured and allocated size agree, cause not found).

gfx: retained 2D canvas, still experimental (docs/gfx.md has a quick start)
- New: `gfx`, a scene of nodes drawn from cached tiles for editors and mockups, with a macOS (CoreGraphics), Linux (cairo, Pango) and Windows (Direct2D) backend, `gfx/view` as the gowt widget, `gfx/offscreen` for PNG without a window, `gfx/layout` for auto-layout and `gfx/pen` for `.pen` designs.
- Changed: `gfx.NewScene()` takes no arguments. The grid is sparse and has no world; the old `NewScene(world, cell)` allocated 1.4 GB for a world of 2e6 units and walked every cell of a huge query. Migration: `gfx.NewScene(gfx.R(...), 256)` becomes `gfx.NewScene()`.
- Changed: `gfx.NewNode(0)` makes an unnamed node: it counts in `Scene.Len` and is not found by `Scene.Node`. Before, every node with id 0 shared one slot.
- Changed: `Scene.Lift` and `Scene.Land` are private. They bypassed the compositor's sprites and made nodes vanish; use `Compositor.Lift` and `Compositor.Land`, or `View.Drag`.
- Changed: `layout.Size{Kind: layout.Fixed, V: v}` is `layout.Fixed(v)`, `layout.Size{Kind: layout.Fill}` is `layout.Fill`, `layout.Hug` is a value; the kinds are `layout.KindHug`, `KindFixed`, `KindFill`. `Node.Pad` is `layout.Insets` (`layout.All(8)`, `layout.XY(x, y)`); a `[4]float64` still assigns.
- Changed: `layout.Apply` places a scene node relative to its nearest ancestor that has a scene node, so a nested layout over a flat scene lands right; a nil `target` reads `Tag` as the `*gfx.Node`.
- Changed: text bounds come from the linked backend whatever the order of recording and of creating a renderer. Before, a picture recorded before `cg.NewRenderer` (or `view.New`) kept estimated bounds, so the text was cut at the edge of a tile or of a PNG.
- Changed: `RenderSubtree` and `WriteSubtreeSVG` take a nil root for the whole scene; errors say why nothing was rendered and what the size limit is.
- New: `offscreen.PNG(scene, node, scale, bg)` and `offscreen.Default()`.
- New: `View.Drag(p, nodes...)` lifts, moves, drops (clamped into the view when released outside) and lands nodes, and puts them back on Escape; the demo's 40 lines of drag code.
- New: `gfx.RGB`, `RGBA`, `Color.WithAlpha`, `White`, `Black`; `RectPath`, `RoundRectPath`, `EllipsePath`, `PolygonPath`; `Recorder.FillRect`, `FillRRect`, `FillEllipse`, `StrokeRect`, `StrokeRRect`, `StrokeEllipse`; `gfx.LazyMeasurer`.
- New: `gfx.Font` has `Family`, `Weight` (CSS 100..900, 0 is 400), `Italic` and `Spacing` next to `Name` and `Size`. Without a `Name` the backend picks the face of `Family` closest in weight and slant among installed and registered fonts (CoreText, Pango/fontconfig, DirectWrite); an unknown family is the default sans at that weight. `Name` stays an exact face name and wins when set. `Spacing` is letter spacing in points, added after every character, the last one included, in drawing and in `MeasureText` (`EstimateText` too) alike. `Font` has more fields now: a positional literal `gfx.Font{"Helvetica", 12}` no longer compiles, write `gfx.Font{Name: "Helvetica", Size: 12}`.
- New: `gfx.RoundRectPath4`, `Path.RoundRect4`, `Recorder.FillRRect4`, `StrokeRRect4`: a radius per corner (top left, top right, bottom right, bottom left). `gfx.ParseSVGPath` reads SVG path data into a `*gfx.Path`.
- Fix: a `gfx.Font` family that is not installed falls back to the sans face (Helvetica Neue with its weight and slant on CoreText, a `family,sans-serif` list for fontconfig, Segoe UI on DirectWrite), never a serif one; each backend's test measures it against the sans face and not the serif one.
- Fix: the SVG writer takes `font-family` from `Font.Family` when `Name` is empty and writes `font-weight`, `font-style` and `letter-spacing`. New: `Tx.ClipRRect4`.
- Changed: `pen.ParseSVGPath` moved to `gfx.ParseSVGPath`. `pen.Options.FontName` is gone: `gfx/pen` passes the family, weight and italic of a text node to the renderer, which resolves them; a family the renderer lacks falls back to its default sans there. `pen` reads `fontStyle` and `letterSpacing`.
- Docs: every exported symbol of `gfx` has a doc comment, each package has Example functions, and `examples/gfxcanvas`, `examples/gfxpng` and `examples/gfxeditor` show a window, a PNG and an editor core.

## v0.12.2 (2026-10-25)

Fixes
- Windows: closing a window that used `look.SetGlass` no longer panics with "comparing uncomparable type fluent.listener": the fluent listeners are pointers, so `Display.RemoveFilter` can compare them.
- Windows: a glass surface is one continuous shape. A push button inside it (`Borderless()`, `Circle()`) and a `Label` paint the slice of the surface under them instead of the parent colour, and the surface is no longer overpainted with the parent colour between its children, so the composer capsule keeps its right end and the send button sits inside it.

Type and density tokens (opt-in)
- New: `gowt.Type` (`Caption`, `Body`, `BodyStrong`, `Subtitle`, `Title`, `LargeTitle`, `Mono`), `Font.Px` (logical pixels, float; replaces `Size` and `Scale` when set), `Spacing.RowHeight`, `ControlHeight`, `SidebarRow`, `SidebarWidth`, `InspectorWidth`, `gowt.Scaled`, and `App.SetDensity` with `DensityNative`, `DensitySame` and `DensityFactor(f)`. See docs/cross-platform-look.md.
- Default unchanged in this release: without `SetDensity` the `Spacing` values, the system font and the `look` sizes are what v0.12.1 had (`Type` and `Font.Px` exist but nothing uses them until the app does). Native density becomes the default in a later minor release, which will change sizes on Windows and Linux; it is announced here first.
- With `SetDensity`, `look.SetSidebar` and `SetInspector` minimum widths and the toolbar padding, search width and title follow the density.

Docs and tests
- Docs: Linux targets a Wayland or X11 display; on a Wayland session GTK uses native Wayland and `GDK_BACKEND=x11` gives Xwayland. The README and platforms.md list `expected_windows_vm.txt` (0 fail, 6 flaky) and `expected_linux_desktop.txt` (Ubuntu 26.04 GNOME, Wayland); the stale "3741 passed, 64 failed" line is gone. `make docs-counts` checks both rows.
- Tests: the root GUI tests close the window after the queued `Async` work has run (and `Pending` is zero), not after a fixed 10 ms; on macOS `TestSetTextLaysOut` found the loop not yet idle. The focus action test moved to `TestQATreeFocus` (`-tags gowtqa`), which `make check-gui` now runs as a second command.

## v0.12.1 (2026-10-25)

Behaviour change
- Windows `Multiline()` has no vertical scroll bar: an edit control cannot hide it when there is nothing to scroll (a one-line `AutoGrow` composer showed a dead bar). `Scrollbars()` is accepted on `Text` and brings the always-visible bar back; the mouse wheel scrolls without it. GTK: the bar follows overflow only (policy automatic, overlay scrolling), as the macOS overlay scroller does.
- Windows 11: `Text` gets the WinUI 3 text box frame, with or without `Border()`: 4 px radius, hairline, stronger bottom edge, 2 px accent line under it on focus, text inset 10/6 px, a single line centred in a 32 point minimum height. `NoBorder()` is now accepted on `Text` and keeps the plain edit. A multi-line `Text` gets `Inset(0, 5)` unless the app sets one.
- Windows: `Border()` on a container, list, table, tree or `StyledText` is a flat 1 px hairline instead of the 3D `WS_EX_CLIENTEDGE` edge.
- Windows: `Borderless()` is the WinUI "Subtle" button (no frame, hover and pressed fill), `Circle()` makes a push button round, `look.GlassButton()` the standard WinUI button fill and hairline, `look.GlassBorder()` adds the hairline to a subtle or round button. Hover, pressed, focus and disabled states are drawn by gowt; the colours are the WinUI 3 light and dark values with the system accent (`HKCU\Software\Microsoft\Windows\DWM\AccentColor`); dark follows `look.SetDarkContent`.
- Windows and GTK: `look.SetGlass`, `SetGlassRadius`, `SetGlassBorder`, `SetGlassFocusRing` give a rounded input surface (radius in points, `look.Capsule` too) with a hairline and an accent ring on focus; the `Text` fields inside lose their own frame. Windows: call after the children exist. GTK: GTK CSS with Adwaita named colours.
- Windows: no dotted focus rectangle after a mouse click on a Link, Button, check box, radio button, Table or Tree: `App.Window` sets UISF_HIDEFOCUS on the window, a mouse press sets it again, Tab, an arrow key or Space clears it, so a focus visual shows for keyboard focus only. A SysLink draws its focused item whatever the UI state says, so a click takes the focus off the item.
- Windows `Table`: one selection pill for the whole row (WinUI 3: accent tint when the table has the focus, grey otherwise, hover tint; no seams between columns) for the whole row (no seams between columns), no dotted focus rectangle after a mouse click (it returns on the first arrow key), images in right and centre aligned columns follow the alignment.

Checked on Windows 11 arm64 (build 26100) at 100% and 150%, light theme, with screenshots; not checked: dark content, keyboard focus visuals of Table and Button, pressed state.

Emoji: GDI font linking already gives monochrome Segoe UI Emoji in Label, Text, StyledText, Button, Table and GC.DrawText there, with a private font as well; colour needs DirectWrite and is not done.

## v0.12.0 (2026-10-24)

Behaviour change
- Linux: the swt level no longer raises `OpenDocument`/`OpenUrl` from D-Bus or `QueryEndSession`/`EndSession`; the facade never exposed them. The two LGPL-2.1 upstream classes behind them (GDBus, SessionManagerDBus) are replaced by no-op stubs, so the repository is EPL-2.0 only. The "SWT SessionManagerDBus: Failed to connect" lines on stderr are gone.
- Linux: the GTK libraries load on the first call into them, not when the program starts. Without GTK installed a program no longer dies with a raw panic and exit 2 before `main`: importing `gowt` (a `go test` of a package that only imports it) works, `gowt.Run` returns the "GTK 3 is not installed" error, and a plain `swt` program gets the same error as a panic from `NewDisplay`.
- Browser on every OS: `evaluate` called inside `LocationListener.changing` or inside a `BrowserFunction` is queued and returns null, as SWT's Edge does. Before, the result differed per OS: macOS raised an error inside a `BrowserFunction`, and Windows never answered.
- Windows `webview`: `OnNavigationFinished` (and so Browser's `LocationListener.changed` and `ProgressListener.completed`) runs after the WebView2 NavigationCompleted handler returns, as SWT's Edge posts it with `asyncExec`. A nested `evaluate` in those listeners now gets its result.
- `App.Async`, `Sync`, `After`, `Every` and the facade relayout skip their callback once `Run` is ending; a callback queued during Display release ran on disposed widgets and panicked.

Fixes
- The README hello checks the error of `gowt.Run`; without a display the program exited 0 and printed nothing.
- `examples/catalog` passed `Border()` to `Table` and `Tree`, which have a border already and reject the option (two logged stack traces on Windows). `examples/graphics` stops its timer when the canvas is disposed, instead of logging "Widget is disposed" on exit.
- Windows `webview`: `window.gowt.call` showed the native "This page says" prompt instead of reaching the handler. WebView2 raises `ScriptDialogOpening` only when `AreDefaultScriptDialogsEnabled` is false, and nothing set it. The engine now turns the default dialogs off, so the event fires for every dialog: the call prompt goes to the handler, and the page's own `alert`, `confirm`, `prompt` and `beforeunload` get a gowt message box or input box.
- Windows `webview`: `window.gowt.call` was answered with an empty string on a page loaded with `SetText`: WebView2 reports `about:blank` as the source but a `data:` URL in the dialog event. A subframe still gets the empty answer.
- Windows Browser: `Changing` reports `about:blank` for `SetText` content, a load replaced by a newer one in the same turn still reaches the location policy, and `getText` returns the page without its doctype, as Edge does.
- macOS `webview`: the page's `alert`, `confirm` and `prompt` show as `NSAlert` sheets, and every completion handler is answered exactly once.
- Windows: `TextView.Text` and `OnChange` give `\n` line breaks, not the control's `\r\n`.
- Windows GDI+: a font with an unknown face name falls back to the generic sans serif family instead of failing with "No more handles".
- Windows: `COM.PathToPIDL` is implemented through `SHParseDisplayName` (FileTransfer and HTMLTransfer panicked with "unsupported native").

Licences
- Every j2go-generated file carries the licence header of its Java source; `make licence-check` (in `make check` and CI) fails on a file without one and on any LGPL text. The repository is EPL-2.0 only.

Docs and tooling
- Package documentation for pkg.go.dev (`gowt`, `swt`, `browser`, `jface`, `icons`); install notes: musl/Alpine unsupported, `librsvg2-common` for check marks, WebKitGTK packages verified, the truth about `winmanifest`.
- CONTRIBUTING with DCO, `cmd/fieldcheck` (a one-file report from a live machine), issue templates.
- `make check-gui` and a Linux CI gate for the facade, QA and webview GUI tests; `make docs-counts` checks the test counts in the docs.
- Test results from real machines: `tests/expected_windows_vm.txt` (Windows 11 arm64: 0 fail, 6 flaky) and `tests/expected_linux_desktop.txt` (Ubuntu 26.04 GNOME on Wayland); `tooling/swtdiff.py` compares run logs with an expected file. `examples/qa` runs against a prebuilt app with `GOWT_QA_APP`.

## v0.11.0 (2026-10-24)

New API
- QA driver actions (qa-driver TSK-08): `activate` (DefaultSelection of a double click or Enter on a List, Table or Tree row, a single-line Text, a push Button, a window's default button), `expand` and `collapse` (a Tree row: the Expand or Collapse event runs first, so an app that builds the children lazily does it, then the state changes; a lazy node needs a placeholder child), `focus` (FocusIn and `gowt.OnFocus`; on macOS the control becomes the first responder of its window while the app stays inactive, so QA runs do not take the focus from the user), `close` (Shell.Close: `OnClose` can veto), `scroll` (a ScrollView to a control, by `dx`, `dy`, or a control inside it into view: `ScrollView.OnScroll` and the feed's row clipping run) and `key` (`Mod+K`, `Enter`, `Shift+Tab`; `Mod` is Cmd on macOS and Ctrl elsewhere). `key` runs the menu item with that accelerator, as macOS key equivalents do before the control sees the key; otherwise it sends KeyDown to the display filters (`Window.OnKey`) and the control's listeners (a listener may veto with `Doit`), then KeyUp, and Enter that nobody vetoed also sends DefaultSelection. Without a selector `key` goes to the focus control. `qatest`: `Element.Activate`, `Expand`, `Collapse`, `Focus`, `Close`, `Key`, `ScrollTo`, `ScrollBy`, `ScrollIntoView`, `SelectText` and `Client.Key`.
- QA driver dialogs (qa-driver TSK-10): in a `-tags gowtqa` build with the QA socket, `MessageBox`, `Confirm`, `FileDialog`, `DirDialog`, `ColorDialog` and `FontDialog` do not open a native dialog (it blocks the UI thread in a modal loop). They take the answer a test queued with `app.Answer(qa.Dialog{Kind: "confirm", Reply: true})` and are recorded for `app.Dialogs()` (title, message, buttons, folder, filters). A dialog with no queued answer gets Cancel and fails the action that opened it, instead of hanging. The default build links none of it.
- `qatest`: `Client.Eventually(timeout, cond)` polls for what `Settle` cannot see; the env names `GOWT_QA_ARTIFACTS` and `GOWT_QA_KEEP` are the constants `qa.EnvArtifacts` and `qa.EnvKeep`.
- `docs/qa.md`: the guide to the QA driver: quick start, selectors and the dump, actions, dialogs, waiting and jobs, state and isolation, artifacts, limits.

## v0.10.0 (2026-10-24)

New API
- `App.Job(name, fn, opts...) *Job`, `Job.Then`, `Job.Cancel`, `Job.Wait`, `JobProgress` (`Total`, `Step`, `Text`), `BackgroundJob()`, `App.Track(name) (done func())`, `App.Jobs() []JobInfo` and `App.OnJobs(f)` (qa-driver TSK-13, 14, 15): work the app is busy with, modelled on Eclipse Jobs. `Job` runs `fn(ctx, p)` in a goroutine (ctx ends with `Cancel` and with `Run`; a panic goes to `OnError` and is the job's error); `Then` runs on the UI thread, also when added after the end; `Wait` panics on the UI thread. `Track` counts work gowt does not start until `done()`. `Jobs` and `OnJobs` (coalesced, one call per event-loop turn) feed a status spinner. `Progress` and `Background` were taken by the progress bar and the colour option, hence the names `JobProgress` and `BackgroundJob`.
- `App.Pending` now also counts running jobs and tracks that are not `BackgroundJob`, and `After` timers under one second; a longer `After` and `Every` do not count. The QA driver's `settle` waits for all of it, so a `qatest` click that starts a job returns after the job and its `Then`; the timeout error names what is pending (`waiting: job "Download report" 4.2s, ...`). New `jobs` command and `qatest` `app.Jobs()`.
- `gowt.ID(id)`: names a widget, a Panel or a Window for the QA driver, which picks controls by it. It stores the id under the data key `gowt.id` in every build and changes nothing else. In an `Each`, pass the id from the row data.
- Package `gowt/qa` and `gowt/qa/qatest`: a QA driver. A build with `-tags gowtqa` and `GOWT_QA_SOCKET` serves find, click, type, check, select, dump, settle, shot and app state handlers (`qa.Handle`) on a socket; `qatest` drives it from `go test` without linking the toolkit. Without the tag the app has no server. Example in `examples/qa`; see docs/internals.md. `App.Pending` and `rx.Pending` tell what waits for the next event-loop turn.
- `look.SetGlassFocusRing(c, on)` (client platform-go item 17): while `c` or any control inside it has keyboard focus, the stroke on the glass of a `SetGlass` panel becomes a ring in `NSColor.keyboardFocusIndicatorColor` (the accent colour), 2 points wide, following the corner radius; on blur the `SetGlassBorder` stroke (or nothing) is back. Focus is read from a display filter on `FocusIn` and `FocusOut`, so controls added later count; resizes and `AutoGrow` keep the ring in step, and the colour is resolved again on `SWT.Settings` (accent colour and light/dark switch). Opt-in: macOS 26 Messages draws no ring on its composer, the ring is what a text field has. AppKit's own ring was not used: it is drawn only for the first responder, from its `drawFocusRingMask`, and a layer-backed view cannot draw it outside its layer, while the first responder here is an `NSTextView` inside a scroll view, smaller than the capsule. The ring sits inside the glass (clipped to its corners) and appears without the system's fade. No-op before macOS 26 and on other systems.
- `gowt.OnFocus(func(focused bool))`: an option for any widget, called with true on `FocusIn` and false on `FocusOut` (SWT events, all three systems). A container reports only its own focus, not its children's.
- `StyledTextView.SetParagraphSpacing(line, before)` (client platform-go item 18): `before` points of space above a line, like `NSParagraphStyle.paragraphSpacingBefore`, so the blocks of a feed message get a gap of about 6 pt instead of one solid column of lines. It is `swt.StyledText.SetLineVerticalIndent`, which the client used through `Unwrap`; the facade only names it. A `SetLineBackground` color stops at the gap (SWT paints the widget background there), so a code block keeps a clean edge above its first line and no band shows between a block and the paragraph before it. The gap is part of its line: it follows the line when text is inserted above it, `SetText` drops it, and SWT ignores it while `OnStyle` is set. Checked in the Linux stand (line heights, `GetLineIndex`, hit-testing in the gap, selection, line backgrounds); not run on macOS or Windows.

## v0.9.8 (2026-10-23)

New API
- `look.SetGlassBorder(c, on)` and the option `look.GlassBorder()` (client platform-go item 16): a hairline around a glass panel or a glass button, one device pixel (0.5 pt on Retina) in `NSColor.separatorColor`, following the corner radius, so a glass capsule or circle shows on a white content panel. It sits on the layer of the glass, under the controls. The colour is resolved under the view's appearance and set again when the system switches light and dark; resizes and `AutoGrow` keep it in step. `GlassBorder()` works next to `GlassButton()` and `gowt.Circle()` in either order, `SetGlassBorder` may come before `SetGlass`. No-op before macOS 26 and on other systems.

## v0.9.7 (2026-10-23)

Fixes
- macOS 26.5: a `look.SetGlass` capsule showed a light rectangle with square corners behind it on a white content panel (client platform-go item 15). The glass of a `SetGlass` panel is now clipped to its corner radius (`masksToBounds`), as the glass circle button already was.

## v0.9.6 (2026-10-22)

New API
- `gowt.Borderless()` and `gowt.Circle()` for a push `Button`, and `gowt.AccessibleLabel(s)` for any widget (fusion-desktop gowt-issues 13). An image-only button was a grey rounded rectangle on every platform. macOS: `Borderless` sets `isBordered = false` (AppKit dims the image while pressed); `Circle` sets the circular bezel and, on macOS 26, `borderShape = .circle`; `AccessibleLabel` sets `accessibilityLabel`, which VoiceOver reads for a button with no text. Linux: the GTK style classes `flat` and `circular` (hover and press feedback are the theme's) and the ATK name. Windows: the frame stays and `Accessible` is a stub, so the three do nothing. They go to `App.OnError` on a Check, Radio or Toggle button. Give a round button a square `Size`. Apple's design wins where the request differed: a circle with glass is `Circle()` plus `look.GlassButton()` in either order, not a separate `look.GlassCircleButton()`, since AppKit and SwiftUI keep bezel style and border shape apart; a borderless button has no hover chrome, as `NSButton` without a bezel has none.
- `look.SetGlassRadius(c, r)` and the radius `look.Capsule` (fusion-desktop gowt-issues 10): the corner radius of the glass `look.SetGlass` puts behind a panel, 12 by default. `Capsule` is half the shorter side and follows every resize. Other systems have no glass: no-op. A field that grows over several lines wants a fixed radius of half its one-line height, as the iPad composer has; `Capsule` would turn a tall field into an oval.
- `AutoGrow(minHeight, maxHeight)` and `TextView.SetAutoGrow` (gowt-issues 12): a multi-line `Text` as high as its text, wrapped lines counted, between the two limits (0 is no maximum); beyond the maximum it scrolls. Text shorter than the minimum is centred vertically, so a one-line field looks like the single line of a native field. The field takes the width its layout leaves over instead of the width of its longest line, so a long text does not push it out of the window (macOS; on Linux set the text after `Show`).
- `Inset(x, y)` and `TextView.SetInset` (gowt-issues 12): space between the text and the edges of a multi-line `Text`, the hint included. `x` is for both sides, `y` above and below: AppKit's `textContainerInset` has no per-side value, so the facade keeps one pair on every platform. macOS: `textContainerInset`, Linux: the margins of the `GtkTextView`, Windows: `EM_SETRECT` (not run).
- `TextView.ContentHeight(width)` (gowt-issues 12): the height the text needs at a width, visual lines counted, with the vertical inset and one line at least; the number for `SetHeight` of a field that follows its text without `AutoGrow`.
- `SetHeight(h)` on every widget in a VBox, HBox, Grid, Flow or Form: changes the height `Size` gave at creation, and lays the parents out on the next event-loop turn or at `App.Flush`.
- `Deselect()` on `List`, `Table` and `Tree`, and `Tree.Select(nil)` (njpt gowt-issues 20): clear the selection without `Unwrap`. They call no `OnSelect`, as SWT's `deselectAll` sends no Selection. `Select()` with no index already cleared a List or Table; a Tree had no call.
- `MaxHeight(h)`, `SetMaxHeight(h)` and `SetSize(w, h)` on every widget in a VBox, HBox, Grid, Flow or Form (njpt gowt-issues 21). A widget with `MaxHeight` is as high as its content, up to `h` points with the border, and scrolls beyond it; a Tree or Table follows its rows, so the sidebar Tree of njpt grows with an expanded node and stops at the cap. In a VBox the limit also leaves the other rows their room: with a window lower than a Table's `Size(0, 120)` and the Tree's natural height together, the Table keeps 120 and the Tree takes the rest and scrolls. `SetHeight` and `SetSize` replace the limit. A relative limit is not an option; the VBox rule covers the sidebar case.

Behaviour change
- macOS: the selected row of a Table in a `look.SetSidebar` sidebar tints the icon of the first column only when it is a template image (`NSImage isTemplate`), as AppKit does. An app image is no longer tinted because its pixels are one grey with alpha: a grey "#" glyph and a black-and-white photo with transparent padding keep their pixels (fusion-desktop item 9).
- macOS: `icons.Get` without `icons.Color` returns a template image, so the sidebar tint of the toolkit's own icons stays. A template image is drawn in the label colour by buttons, tool items and menu items (white on a highlight, dimmed when disabled), so such an icon follows the appearance of the view and not the theme of the call. `icons.Get` with `icons.Color`, `icons.FromSVG`, `App.ImageFromProvider`, `App.ImageFrom` and `App.LoadImage` give plain images that keep their colours everywhere.
- macOS: a multi-line `Text` in a `look.SetGlass` panel draws no background of its own, neither the `NSTextView` nor its `NSScrollView` (gowt-issues 11); the glass shows through, as in the iPad composer. A single-line `Text` keeps its native field.
- `Size(w, h)` or `SetHeight(h)` on a widget that also has `Grow` or `GrowY` in a grid sets its minimum height to `h`. The grid used to shrink such a cell to nothing when the window was too low (a Table with `Size(0, 120)` above a tall Tree vanished); it now keeps `h`.
- A Tree or Table asks for a layout on the next event-loop turn after `Node`, `Row`, `Remove`, `Clear`, `SetExpanded` and a user's expand or collapse, because its natural height follows its rows. A Tree in a VBox row that takes its natural height now grows with an expanded node without `Relayout`. `Panel.Relayout` also applies the `MaxHeight` of the widgets inside.
- macOS: a Tree in a `look.SetSidebar` sidebar selects with the Finder pill and accent text, like a Table (njpt look item; one sidebar has one selection look). The outline view's row view comes from `outlineView:rowViewForItem:`, which the source-list hook did not cover, and the Tree cell draws the selected text and template icon in the accent colour. `TestSidebarTreePill` checks the row view class.

Performance
- A chat feed in a `Scroll` (fusion-desktop gowt-issues 14). `Add` of a container (`Panel`, `Window`, `Group`, `Split`, and `ScrollView.Add` with its fit) measures only the new children: the children already there keep the sizes their layouts cached. It flushed every cache and measured the whole content, so 10 rows added to 300 cost a full layout; on the second Mac 120 ms became 42 ms, in the Linux stand 132 ms became 13 ms (`BenchmarkFeedAdd10`, rows of an avatar `Canvas`, a header and a `StyledText`). The result equals a full layout (`TestFeedAddSameAsFullLayout`). `Dispose`, `SetVisible`, `Relayout`, `Clear` and `Add` also drop the sizes cached for the container in its ancestors, so a later partial layout measures it again. A change made through `Unwrap()` (or a `StyledText`/`Text` `SetText`, which keeps the size) is not picked up by a later `Add` any more: `Relayout` the container, or `Fit` the `ScrollView`, as before. The facade setters (`SetText`, `SetImage`, `SetFont`, `SetHeight`, `AutoGrow`) are picked up: their relayout is deferred to the next turn, and an `Add` in the same turn measures the changed row again.
- macOS: a `Scroll` hides its content rows more than a quarter of the view away from the visible part natively (`NSView.hidden`, not `SetVisible`: SWT state and layout stay as they were; the row holding the first responder of the window, key or not, and every row while VoiceOver runs (also when it is turned on later) stay shown; `Focus`, Tab, Shift+Tab and `Print` show a hidden row first, so they reach it as before). On every scroll frame AppKit updated the tracking areas of every view in the content and SWT answered each from Go, so a frame grew with the rows outside the window: with 300 rows 8.4 ms became 4.8 ms, with 60 rows 4.7 ms became 3.5 ms (`BenchmarkFeedScrollFrame`, 40 points a frame and a 1 ms timer). A frame updates only the rows that enter or leave the band.
- `StyledText` with `Wrap()` no longer places its caret while it has no width (`tooling/port.sh` rewrites `StyledText.reset`): it wrapped the text at 1 pixel, a glyph per line, to do that, and `handleResize` places the caret again once the width is known. A feed row with a `StyledText` is created in 1.3 ms instead of 2.6 ms on the second Mac, 0.7 ms instead of 1.7 ms in the Linux stand (`BenchmarkFeedRowCreate`).

Fixes
- macOS 26: `look.GlassButton()` showed a plain button (fusion-desktop gowt-issues 13). SWT's `Button.setBounds` and `setFont` put the push bezel back on every layout, and AppKit draws the glass bezel only for a view class without `drawRect:` and a plain `NSButtonCell`, which SWT's `SWTButton` and `SWTButtonCell` are not. `tooling/port.sh` limits both methods to the two push bezels, and a glass button moves to a new `SWTGlassButton` class (same events, no `drawRect:`, so no `Paint` event) with a plain cell that gets the title, image, font, target, action and default-button state. The press animation and the focus ring are AppKit's. On macOS 26.5 the glass of a circle button also drew a square, darker than the background, over its whole 40x40 frame (macOS 27 does not); a glass circle button is now clipped to its circle through its layer (`cornerRadius`, `masksToBounds`), kept in step on resize.
- macOS 26.5: a `look.SetGlass` panel was a rectangle with a drop shadow, on any background and with any radius, `look.Capsule` included. The `NSGlassEffectView` was a sibling below the panel; AppKit's SwiftUI host fills its whole frame there. The glass is now the lowest child of the panel itself (the content view of a `Group`), it passes clicks through, and it stays below controls added later. A sidebar, an inspector and a glass window keep their glass as before.
- Linux: a Tree that was collapsed and expanded again kept the collapsed height as its preferred height (GtkTreeView reports it from validated rows), so a layout did not make room for the rows. `Tree.computeSize` takes the larger of that height and the count of visible rows (`tooling/port.sh` rewrite, `visibleRowCount` in `swt/widgets_tree_manual_linux.go`).

## v0.9.5 (2026-10-21)

New API
- `ScrollView.OnScroll(func(x, y int))` reports the new origin after any scroll (fusion-desktop gowt-issues 7): the mouse wheel, a trackpad gesture and its momentum, the scroll bars, `SetOrigin`, and a resize that moves the content. It listens to the move of the content, where every scroll of SWT's `ScrolledComposite` ends; the scroll bars' `Selection` misses `SetOrigin` and the resize. Scrolls faster than a frame are coalesced to at most one call per 1/60 s, the last one with the origin where the scroll stopped, and a scroll back to the reported origin calls nothing. `ScrollView.Origin()` and `SetOrigin(x, y)` read and set the origin without `Unwrap`.
- `look.SetInspector(w, content, inspector, width)`, a right-hand inspector pane, alone or next to a `look.SetSidebar` sidebar (sidebar | content | inspector), with `SetInspectorVisible`, `InspectorVisible`, `SetInspectorWidth`, `InspectorWidth` and `OnInspectorChange` as for the sidebar (requested by fusion-desktop). On macOS 26 it is edge-to-edge Liquid Glass alongside the content, flush with the window edges and running under the toolbar, which is how AppKit draws an `NSSplitViewController` inspector (WWDC25 session 310); older macOS shows a flush material, Windows and Linux a plain panel with a hairline divider. The user drags its left edge; the drag stops at 160 points and leaves the content 200, and never hides the inspector. The window minimum counts it. `cmd/finderdemo` has an Info inspector with a `sidebar.right` toolbar toggle and View > Hide Inspector (Ctrl+Cmd+I on macOS).
- `look.SetToolbar` on Windows and Linux draws `sidebar.right`, and gives its bar to a `SetInspector` window without a sidebar too.
- `App.SetBadge(text)` shows a count on the application icon, `""` removes it (fusion-desktop gowt-issues 8). macOS: the badge of the app's Dock tile. Windows: the overlay of the taskbar button of every window, also of windows created later with `App.Window`. Linux: the `com.canonical.Unity.LauncherEntry` count that Ubuntu Dock, dash-to-dock and KDE Plasma draw, whole numbers only, for the desktop file named after the app name (`SetAppName`, the executable name by default). SWT's own `TaskBar` stays as in SWT for Java: on macOS `GetItemShell(shell)` returns nil and only `GetItemShell(nil)`, the application item, exists; on Linux `Display.GetSystemTaskBar` returns nil.

Behaviour change
- macOS: `look.SetGlass` clears the whole content of the panel, as `look.SetSidebar` does: labels, links, check boxes, radio buttons, `StyledText` and the containers inside it, `Group`, `Scroll` and `Split` among them, show the glass through, and lists, tables and trees get the source-list style. It cleared only the panel itself, so a `Scroll` inside showed a white rectangle over the glass.
- macOS: controls created inside a `look.SetGlass` panel or a `look.SetSidebar` sidebar after the call are cleared too, on the next event-loop turn; they used to paint the window colour. A control with a background of its own, set by the app, keeps it in both, also when it existed before the call.
- macOS: a `look.SetSidebar` or `look.SetInspector` window with no List, Table or Tree and no `gowt.Focused` opens with no focused control, the window its own first responder, so no popup or button shows a focus ring until Tab or a click; SWT focused the first control, also on each later activation. Windows and Linux keep SWT's choice.
- macOS and Linux: a Table or Tree cell text wider than its column ends with an ellipsis, as in Finder, instead of being cut at the column edge in the middle of a letter (fusion-desktop gowt-issues 5). SWT clips on both; Windows already truncates with an ellipsis. A Table cell with a bold or italic font follows (macOS); a Linux owner-drawn cell still clips.

Fixes
- `Hint` (and `SetHint`) on a `Text(Multiline())` shows the placeholder on every platform (fusion-desktop gowt-issues 6); SWT sets the message of a single-line field only, and no platform's multi-line editor has a public placeholder. gowt draws it over the empty field, where the first typed character would go, in the field's font (it follows `SetFont`). macOS: `placeholderTextColor` at the text container origin plus the line fragment padding, wrapped and aligned as the text view, hidden while there is marked (IME) text too; no private API. Windows: `COLOR_GRAYTEXT`, as a cue banner, at the edit's formatting rectangle. Linux: the theme's `placeholder_text_color`, or the 50% grey of a GtkEntry placeholder, drawn after the GtkTextView paints. The hint hides once there is text, comes back when the field is empty again, and is repainted after a light/dark switch.
- macOS: in a window that is its own first responder (a `look.SetSidebar` or `look.SetInspector` window without a list opens so), Tab did nothing until the window was activated again; it now focuses the first control and Shift+Tab the last.
- macOS: the `look.SetInspector` glass gets a separator line down its leading edge for the full window height; the content's edge effect under the toolbar ended there in a hard edge. The glass is clipped to its frame, so the grey gradient it cast into the toolbar band of the content is gone.
- macOS: the selected row of a Table in a `look.SetSidebar` sidebar filled any icon with the accent colour since v0.9.4, so a colour image (an avatar) turned into an accent-coloured shape and an opaque one into a solid square. Only the glyph of the first column, the row symbol before the text, is tinted now: a template image, or one grey on a transparent background, as the `icons.Get` and `icons.FromSVG` images in the theme colours are. Images in later columns, a number badge or a "#" glyph, keep their colours. An icon coloured with `icons.Color` keeps its colour.
- macOS: a `StyledText` with a transparent (alpha-0) background painted its lines opaque black when nothing above it had a visible background, since SWT's GC fill ignores the alpha of a colour. `Composite.DrawBackground` now fills nothing for an alpha-0 GC background, so a `StyledText` on glass is transparent.
- macOS: a Table or Tree in a `look.SetSidebar` sidebar keeps the column widths the app set (fusion-desktop gowt-issues 4). When the columns filled the table exactly, switching to the source-list style made AppKit fit them again (`_sizeTableColumnsToFitWidth:`, whatever the column autoresizing style) and take the style's 31 points of padding off the first column with the autoresizing mask: a 44-point badge column became 13 points wide, and the badge showed half. The widths are now restored after the switch.
- macOS: the client area of a Table or Tree in a `look.SetSidebar` sidebar leaves out that padding, measured as 31 points in a macOS 26 window (10 before the first column, 6 inside it, 15 after the last). Columns sized to `GetClientArea` ran that far past the visible width, and the last column was cut.
- macOS: a check box or radio button in a `look.SetSidebar` sidebar (and now an inspector) was a black bar, its box and text hidden. The pane gives it an alpha-0 background, and SWT's `Button` drew its own bezel in the background colour at full opacity whenever a background was set; a transparent background now leaves the bezel to AppKit (`tooling/port.sh` rewrites `drawBezelWithFrame_inView_`). The look GUI test `TestPanesNoBlackFill` captures a window with every kind of control on both panes and fails on a black fill.
- macOS: a `Section` on a transparent parent (a sidebar, inspector or `SetGlass` panel) painted its corners black: they were filled through a GC in the parent colour, and GC fills ignore alpha. A section's view is now clipped to its rounded rectangle through its layer (continuous corner curve), so the corners show whatever is behind the card, glass included, and nothing paints them.
- macOS: a `Text` (and an editable `Combo`, a `Spinner`) on a sidebar, inspector or `SetGlass` panel had no visible field: the panel's alpha-0 background was forced on every control (`INHERIT_FORCE`). The panels now pass it on with `INHERIT_DEFAULT`, which reaches labels, check boxes, scales and panels but leaves fields their own background.

Docs
- docs/widgets.md: all images of a Table or Tree get the size of the first one, on every platform (SWT's model); images of different widths go onto canvases of one size.

## v0.9.4 (2026-10-20)

New API
- The sidebar of a `look.SetSidebar` window can be hidden and resized: `look.SetSidebarVisible`, `SidebarVisible`, `SetSidebarWidth`, `SidebarWidth` and `OnSidebarChange` (njpt issue 19). The user drags the edge between sidebar and content (a sash with the resize cursor); the drag stops at 120 points, or at the room its `Sidebar` toolbar items need where that is more, and leaves the content at least 200 points. A drag never hides the sidebar; `SetSidebarVisible` does, from a toggle button or a menu. A sidebar narrower than that minimum, from `SetSidebar` too, is laid out at the minimum once it is known, and the window gets a minimum size of the sidebar plus 200 points of content (the content alone while hidden), never below a minimum the app gave `Window.SetMinSize`. Hidden, the content takes the whole width, and the `Sidebar` toolbar items move to the start of the toolbar: on macOS after the traffic lights, followed by the navigational items and the title, as in Finder; on Windows and Linux to the start of the bar, while the row in the sidebar hides with it. `OnSidebarChange` reports changes from the calls and once at the end of a drag.

Behaviour change
- Windows and Linux: a `look.SetSidebar` window leaves a 4-pixel gap between sidebar and content, where the sash lies.

Fixes
- `look.SetToolbar` on Windows and Linux draws 24 more SF Symbols as icons, among them `terminal`, `shippingbox`, `folder`, `sidebar.left` (for the button that toggles the sidebar), `gearshape`, `magnifyingglass` and `trash`; it knew only 4 and showed the item title as text for any other. The supported set is listed in docs/theme-look.md.
- macOS: after a light/dark switch the content of a `look.SetSidebar` window kept the old window colour, so its labels could become invisible; every shell's view tree is now redrawn after the switch.
- macOS: the selected row of a Table in a `look.SetSidebar` sidebar draws its text and icon in the accent colour, as a List row does; it drew plain text. The icon is filled with the colour, so a multi-colour icon shows as a silhouette there.
- `examples/sidebartable` and the snippet in docs/widgets.md set the `icons.Get` icons again in `App.OnThemeChange`; they stayed dark on a dark sidebar.

## v0.9.3 (2026-10-19)

New API
- `gowt.Focused()`, an option that gives a widget the keyboard focus when its window is first shown, in place of the first control SWT picks; on macOS with Full Keyboard Access on that was often a button, drawn with a focus ring. The last `Focused` created in a window wins; a `Focused` widget added to a window on screen takes the focus on the next event-loop turn, or when its window is next activated. `Focused` on a Window goes to `App.OnError`.
- `gowt.SecondaryText`, a `Paint` with the system colour of secondary text that follows the system theme like an `Adaptive`: macOS `secondaryLabelColor` (flattened over the window background, as SWT text colours are opaque), Windows `COLOR_GRAYTEXT`, GTK the insensitive label colour. SWT's `COLOR_WIDGET_DISABLED_FOREGROUND` is not used on macOS, where it is a selection background colour.
- `look.ToolbarItem.Sidebar` puts a toolbar item over the sidebar of a `look.SetSidebar` window. On macOS it sits in the title bar level with the traffic lights, right-aligned about 25 points inside the sidebar edge, and back/forward and the title start about 27 points past the edge. On Windows and Linux the item goes into a row at the top of the sidebar, at its right end. Without `SetSidebar` it is an ordinary item.

Behaviour change
- A window laid out by `look.SetSidebar` without a `Focused` widget opens with the focus on its first visible List, Table or Tree, in the content first, then in the sidebar, as Finder does, instead of the first button.
- Disposing the shown page of `Pages` shows the page before it, or the next one when it was the first; it used to leave the `Pages` empty with `Current()` -1. The pages after a disposed one move down by one, as before.
- Disposing a `Tab` page removes its tab from `Tabs` and `CTabs`; it used to leave an empty tab. A `CTabs` tab closed by its button now disposes its page and the bindings in it; the page used to stay hidden in the folder.

Fixes
- `StyledText.SetText` took time quadratic in the number of lines: 10000 lines 0.37 s, 30000 lines 3.5 s. Setting the text resets the renderer's cache through a sorted set of every line index, and the runtime's sorted set searched the whole list and sorted it again on each add. It now inserts by binary search; 30000 lines take about 7 ms on Linux.
- `LabelView.SetText` with a longer text after `Show` kept the label at the size of its first text, so the new one was cut; a `Wrap()` label with `GrowX()` kept the height of its first text. SWT leaves the layout to the app, and the facade did it only for `BindText`. The setters that can change a widget's size now lay its parents out again by themselves: `SetText` of a Label, Button, Switch, Link and tool item, `SetImage` of a Label, Button and tool item, `SetFont`, `GroupView.SetTitle` and `ComboView.SetItems`. The layout runs on the next event-loop turn (or at `App.Flush`), one pass per window over the parent chains of all the widgets changed in that turn; a setter that keeps the widget's size measures it twice and lays nothing out. A Wrap label is measured at its current width, so a text that needs more lines makes it taller. `Relayout` is now needed only after changes made through `Unwrap`.
- `BindText` lays out the parent chain of its widget, only when the size changed, instead of the whole window after every new text.

## v0.9.2 (2026-10-18)

Fixes
- Quitting from the macOS app menu (Cmd+Q) made `Run` return "gowt: Device is disposed": the Quit item disposes the display inside the event loop, and `Run` then asked the disposed display for its shells and disposed the clipboard on it. `Run` now ends without an error when the display is disposed under it, as on Windows logoff (WM_ENDSESSION) or after `Display.Close` in a handler.
- macOS: a tab added to a shown `Tabs` (`TabsView.Add`) kept a strip slot as wide as an empty title, so its title was cut to an overflow arrow and `Window.Relayout` did not help. SWT keeps every NSTabViewItem label empty and sizes the tab itself; the tab view measures a tab again only when its label changes, so `TabItem.SetText` and `SetImage` now change the label and back. `CTabs` and Linux were not affected.
- macOS: the child of a titled `Group` was laid out up to two title heights taller than the box, so its bottom was cut off. The NSBox places its content view while the box is still 0x0 and corrects it only before the first display, after the Group had laid its children out; the Group now lays out again when its content view changes size.
- macOS: a `Table` or `Tree` with a header and rows added before its first layout (in `Init`) showed its first row under the header: scrolled by one row, or by part of a row when the rows almost fit the view. Showing the header tiled the scroll view, which raised the clip view's top inset by the header height but kept its scroll position, so the view started one header height below its top. `setHeaderVisible` now keeps the offset from the top, and a view the app scrolled in `Init` keeps its position.
- macOS: `Table.Row` inside `Batch` was about ten times slower than `Tree.Node`: SWT told NSTableView about each new row (`noteNumberOfRowsChanged` and `tile`). A row appended while redraw is off now only marks the count stale, as Tree does; the view catches up when redraw comes back and before selecting, scrolling or hit-testing rows.
- `Tree.OnExpand` and `OnCollapse` found the node by walking every created node; the event item now maps to its node directly.
- Linux: `Run` could hang for up to 25 s before showing a window when xdg-desktop-portal had to be started and failed to start: the colour scheme and accent were read from the portal synchronously at startup. They are now read asynchronously and applied when the portal answers; on a dark desktop the first frame can be light for a few milliseconds before the switch.
- macOS: after an animated expand (a click on the arrow) the rows below the new children showed only their arrows, no text, until something redrew them. A cell view drew its text at the row's new frame while the animation still held the view at its old place, so the text fell outside the view; cell views now draw relative to their own cell frame. The cell-based mode (`GOWT_VIEW_TABLE=0`) and Linux were not affected.
- Linux: a tab added to a shown `Tabs` and selected showed an empty page until the window was resized; so did a tab other than the first selected before `Show`. `TabFolder` sizes the selected page to the notebook page's allocation, which GTK had not made yet (1x1), and nothing sized the page again. The page now follows its notebook page whenever GTK allocates it.
- Linux: every `Tree` and `Table` column printed `GLib-GObject-CRITICAL ... signal 'dpi-changed' is invalid for instance ... of type 'GtkTreeViewColumn'`. SWT connects that signal on every widget handle, and a column's handle is a GtkTreeViewColumn, which had no such signal.

Behaviour change
- `Grow`, `GrowX` and `GrowY` on a child of `Split`, `Fill`, `Pages`, `Tabs` or `CTabs` (a `Tab` page) are accepted and change nothing, since those give every child its whole part; they used to go to `App.OnError` ("Grow on Group needs a VBox, HBox or Grid parent, not SashFormLayout"). `Size`, `Span`, `Place`, `Indent`, `Cell` and `Anchor` there are still errors, and the message names `Split`, `Tabs` and `CTabs` instead of the SWT layout class.

## v0.9.1 (2026-10-18)

Fixes
- Dialogs and SWT-made texts showed message keys instead of words: the Yes and No buttons of a question read "SWT_Yes" and "SWT_No", likewise OK, Cancel, Abort, Retry, Ignore, the CTabFolder chevron tooltip and accessibility relation names. The English SWT messages are now built in.

## v0.9.0 (2026-10-18)

Tables, trees and lists that follow rx state by key, a `Boundary` that shows an error in place of the part of the screen that failed, colours that follow the system theme, fonts relative to the system size and shared between widgets, one ownership rule for images, and the names the API review found spelled two ways.

```go
// v0.8
w.SetForeground(gowt.RGB{R: 28, G: 32, B: 40}) // and again in OnThemeChange with the dark colour
size := int(float64(win.Unwrap().GetFont().GetFontData()[0].GetHeight())*1.15 + 0.5)
title.SetFont(gowt.Font{Size: size, Weight: gowt.Semibold})

// now
ink := gowt.Adaptive{Light: gowt.RGB{R: 28, G: 32, B: 40}, Dark: gowt.RGB{R: 231, G: 233, B: 237}}
gowt.Label("Containers", gowt.Foreground(ink), gowt.Typeface(gowt.Font{Scale: 1.15, Weight: gowt.Semibold}))
```

New API
- `Paint`, a sealed interface implemented by `RGB` and `Adaptive{Light, Dark RGB}`. `Foreground(paint)` (new option), `Background(paint)`, `SetForeground(paint)` and `SetBackground(paint)` take it; a widget given an `Adaptive` switches to the other colour when the system theme changes, before the `OnThemeChange` callbacks run. nil restores the default colour.
- A background on a container (Window, VBox and the other boxes, Group, Pages, Section, Scroll, Split) shows behind the labels and boxes inside it (SWT `INHERIT_DEFAULT`); text fields, lists, tables and buttons keep the native one.
- `Font.Scale`: with `Size` 0 the size is the system font size times `Scale`, for widgets (`SetFont`, the new option `Typeface(font)`), `GC.SetFont` and `TextStyle.Font`. `App.SystemFont()` returns the family and size of the system UI font.
- `icons.FromSVG(app, data, size, rgb)`: an image from an SVG document with `currentColor` drawn in `rgb`, cached per document, size and colour like `icons.Get`.
- `TableRow.TextAt`, `SetTextAt`, `SetImageAt`, as on `Node`.
- Rows bound to rx state, Svelte's keyed `{#each}`: `Rows(list, key, fill)` on a Table, `Nodes(list, key, children, fill)` on a Tree and `Items(list)` on a List or Combo keep one row, node or entry per element of an `rx.Reader[[]T]`. Rows are matched by key; `fill` runs only for new rows and rows whose element is no longer `==`, the others keep their native item, selection and expansion, and a moved key keeps its selection (and on a Tree the expansion of its subtree). A change of 10 rows in 1000 takes 20 µs and 75 allocations on Linux, against 5.3 ms and 35108 for clearing and refilling the table by hand; a full replace or a reverse costs what the refill costs. `examples/rows` shows a table and a tree that change every second.
- `Boundary(children...)` with `Failed(func(err error, reset func()) Option)`, Svelte's `<svelte:boundary>`: when its children fail to build (a panic, a misused option, a child of a widget that holds none) or a binding or effect inside panics after a state change (`Bind`, `BindText`, `Watch`, `Rows` `fill`, ...), they are disposed with their effects and the fallback is shown in their place; `reset` makes them again. Event handlers are not caught. Without `Failed` the fallback is the message and a Retry button. Example: `examples/boundary`.
- `rx.Catch(fn, handle)`: a `Scope` whose effects hand their panics in a `Flush` to `handle`, the nearest Catch first.

Breaking change
- `List.SetSelection(i...)` and `Table.SetSelection(i...)` are `Select(i...)`, as on Combo, Tabs and Tree.
- `TableRow.Text(col)`, `SetText(col, s)` and `SetImage(col, img)` are `TextAt`, `SetTextAt` and `SetImageAt`.
- `TreeColumn` is gone: `Tree.Column` returns the `*Column` that `Table.Column` returns, and `Tree.SetSort` takes it. `Column.Unwrap()` returns the `*swt.Item`; its `Impl()` is the `*swt.TableColumn` or `*swt.TreeColumn`.
- `Background(c RGB)` is `Background(c Paint)`, and the `Widget` methods `SetForeground` and `SetBackground` take `Paint`: calls with an `RGB` compile as before, code that stores these methods as `func(RGB)` values does not.
- `webview.New`, `webview.NewOn` and `NewWindow.NewOn` take a `webview.Container` (any gowt container or an `*swt.Composite`): `webview.New(host.Unwrap(), opts)` is `webview.New(host, opts)`.
- `svg.NewImageDataProvider(svg, width, height, color)` takes `int` sizes instead of `int32`.

Behaviour change
- Images: the App owns every `Image`, whichever call made it (`LoadImage` and `ImageFrom` said the caller owned theirs), and disposes the ones still alive when `Run` returns. `Dispose` frees one earlier and removes it from the App; disposed images used to stay listed until the end of `Run`. `icons.Get` makes an image again after its cached one was disposed.
- Fonts: controls, canvases and styled texts given equal fonts share one native font per App, released after the last of them is disposed or given another font; each `SetFont` used to make a native font per control. `TextStyle.Font` with an empty `Name` or a zero `Size` now takes them from the widget's font, as `SetFont` does.
- Theme switch: the colours of `Adaptive` paints are Go values (SWT colours hold no native handle on macOS, Windows or Linux), so a switch makes no native colours; images from `icons.Get` and `icons.FromSVG` are made once per colour.
- An `rx.Effect` made in `Init` inside a `Boundary` is disposed with the boundary's children; elsewhere it lives until its `dispose`, as before. Every error a boundary catches also goes to `App.OnError`.

Fixes
- docs: `Column.SetWidth` and `GC.SetLineWidth` said pixels; widths are in points, like every facade size. theme-look.md told apps to recolour widgets in `OnThemeChange` while the facade had no way to set a background after creation.
- docs: `Switch` and a `Check` button are the same check box; the doc now says when each reads right. jface and browser are listed under "SWT port" in docs/index.md and README.md: apps and examples do not use them.

## v0.8.0 (2026-10-17)

A new facade API: a screen is a tree of values, and events, initial values and rx bindings are options next to the widget. Code written for v0.7 does not compile; the breaking changes below say what each old form becomes, and docs/agents.md shows the idioms on one page.

```go
// v0.7
w := app.Window("Contacts")
w.SetLayout(gowt.Grid{Columns: 2, Margin: 12, Spacing: 8})
w.Label("Name")
name := w.Text(gowt.GrowX())
w.Button("Save", func() { save(name.Text()) })

// v0.8
name := rx.New("")
w := app.Window("Contacts")
w.Add(gowt.Grid(2, gowt.GrowX(),
	gowt.Label("Name"), gowt.Text(gowt.GrowX(), gowt.Bind(name)),
	gowt.Button("Save", gowt.OnClick(func() { save(name.Get()) }))))
```

New API
- Tree of values: constructors are package functions that return a description (`Label`, `Button`, `Text`, `Table`, `Tree`, ... and the containers `VBox`, `HBox`, `Grid(n, ...)`, `Flow`, `Fill`, `Form`, `Group`, `Tabs`/`CTabs` with `Tab`, `Pages`, `Scroll`, `Split`, `Section`, `CoolBar`), created in the parent they are passed to: a container constructor, `app.Window(title, children...)` or `Add(children...)` on a live container. `Ref(&x)` stores the created widget, `Init(func(t *gowt.TableView))` sets it up in place once its children exist (fill a Table, set Split weights). `Gap(n)` and `Padding(n)` set a container's spacing.
- Events and initial values as options: `OnClick`, `OnChange`, `OnSelect`, `OnActivate` (one name per concept; the handler type says what the widget reports: `OnChange(func(string))` on a Text, `OnChange(func(int))` on a Scale), `Hint`, `Checked`, `Selected`, `Value` (the text of a Text, the value of a Scale, the date of a DateTime).
- rx bindings as options that live as long as the widget: `Bind(state)` two-way (Text, Combo, Switch, check Button, Scale, Slider, Spinner, List), `BindText(r)`, `BindEnabled(r)`, `BindVisible(r)`; `Watch(w, r, apply)` binds anything else.
- Placement options that do not name the layout and add up to one layout data: `Grow()`, `GrowX()`, `GrowY()` (fill the cell and take the free space), `Span(n)`, `Size(w, h)`, `Place(h, v)`, `Indent(x, y)`; the later option wins on a shared field. `Cell` (`GridCell` gains `IndentX`, `IndentY`) and `Anchor` stay and combine with them. `Spacer()` is an empty cell that takes the free width of a row, `Separator()` a line across its cell (`Vertical()` turns it).
- Common methods on every widget, windows and containers included, also on the `Widget` interface: `SetEnabled`, `Enabled`, `SetVisible`, `Visible`, `SetTooltip`, `SetFont`, `SetForeground`, `SetBackground`, `Focus`, `Dispose` (lays the parent out again), `OnDispose`. Containers have `Add`, `Relayout()` and `Clear()`.
- `Pages(children...)`: one page at a time without tabs, every child is a page; `PagesView.Show(i)`, `Current()`.
- `App.OnError(func(error))` receives errors and panics raised by event handlers, `Async`, `After` and `Every` callbacks, and options a widget cannot use; the event loop continues. Without it they are logged to stderr with their stack.
- `App.Every(d, f) (stop func())` repeats `f` on the UI thread; a panic in `f` goes to `OnError` and the ticks go on.
- Keys: `Shortcut(keys)` on `Menu.Item` binds a combination and shows it (`Shortcut("Cmd+O")` is Command+O on macOS and Ctrl+O elsewhere; `Ctrl`, `Alt`, `Shift`, letters, digits, punctuation, arrows, `F1`-`F12` and other named keys). `Window.OnKey(keys, f)` binds one anywhere in the window with the same syntax; the key does not reach the focused widget, but a focused text field keeps the combinations it types or edits with (the list per OS is in docs/events.md), so `Cmd+[` or `Esc` work while typing and a bare letter does not.
- `GC.SetFont(f)` sets the font of the text drawn after it in a paint callback; `Font{}` goes back to the canvas's font. The canvas makes each font once and releases it when disposed.
- `App.Flush()` runs the pending rx effects and the layouts the bindings put off now instead of on the next event-loop turn.
- `Window.SetMinSize(w, h)`, `App.ScreenSize()`, `Table.Batch(f)` and `Tree.Batch(f)` (drawing suspended while `f` changes many rows).
- docs/agents.md: the canonical idioms on one page and a table from each option to the widgets that read it. Every constructor's doc ends with an "Options:" line. `tooling/api-metrics.sh` counts lines, `Unwrap()` and `swt.` uses per facade program.

Breaking change
- Widgets are created by package functions instead of methods of their parent (`p.Button(...)` is `p.Add(gowt.Button(...))` or a child of a container constructor), and widget types end in `View` (`LabelView`, `ButtonView`, `TextView`, `TableView`, ...; `Panel` and `Window` keep their names). `Option` is a sealed interface: code that wrote `gowt.Option{}` or compared options does not compile.
- Layouts are containers fixed at creation: `SetLayout`, the `Layout` interface, the `Grid{}`, `Row{}`, `Fill{}`, `Form{}` and `Stack{}` values, `Spaced()`, `Padded()`, `Stack`/`ShowTop`, `InRow`/`RowCell` are removed. `Grid{Columns: n}` is `Grid(n, ...)`, `Row{}` is `HBox(...)` (`Flow(...)` when it wraps), `Row{Vertical: true}` is `VBox(...)`, `Stack{}` is `Pages(...)`, `Margin`/`Spacing` are `Padding(n)`/`Gap(n)`, `InRow(RowCell{W, H})` is `Size(w, h)`.
- Default spacing: every container spaces its children by `Spacing.SM` unless `Gap` says otherwise; a Window and a Tab page are padded by `Spacing.PageMargin`, a Group by `Spacing.SM`. A Window, Group, Tab page and Scroll content stack their children as a VBox; put an `HBox` or `Grid` inside for another arrangement.
- `Tabs.Tab(title)` and `CTabs.Tab(title)` are the `Tab(title, children...)` child of `Tabs(...)`/`CTabs(...)`; every child of `Section` is a row (`Section.Row()` is gone); `Scroll.Content()` keeps working but children go to `Scroll(children...)` or `ScrollView.Add`, which fit the content; `CoolBar.Add(widget)` is `CoolBar(children...)`, each child a band.
- `Button`, `Switch` and `Link` lose the positional callback: `Button("Save", save)` is `Button("Save", OnClick(save))`, `Button("Small", nil, Radio())` is `Button("Small", Radio())`. `Scale`, `Slider` and `Spinner` lose the positional value: `Scale(0, 100, 30)` is `Scale(0, 100, Value(30))`.
- Bindings: `LabelView.BindText`, `ButtonView.BindChecked`, `TextView.Bind` and the functions `BindVisible(w, r)`, `BindEnabled(w, r)` are the options `BindText`, `Bind`, `BindVisible`, `BindEnabled`; `gowt.Bind(w, r, apply)` is `gowt.Watch(w, r, apply)`.
- The functions `SetEnabled(w, v)`, `Enabled(w)`, `SetVisible(w, v)`, `Visible(w)`, `SetFont(w, f)` and `SetForeground(w, c)` are the methods `w.SetEnabled(v)` and so on.
- `Beside` and `Same` take the address of the sibling's `Ref` variable: `Beside(&ok, 8)`, read when the child is placed.

Behaviour change
- An error or panic in an event handler no longer ends `Run` and closes the app: it goes to `App.OnError`. `Run` still returns an error raised by `setup`.
- An option a widget cannot use goes to `App.OnError` with the option, the widget and the layout in the message, and the widget is created without it; it used to be ignored, or for a placement option under another layout, to panic with "java.lang.ClassCastException" on the next layout pass. This covers a style option the widget does not read (`Password()` on a Label), `Gap`/`Padding` on a widget that is not a container, children of a widget that holds none, a `Ref` or `Init` of another type, `Shortcut` outside `Menu.Item` and common options on `Table.Column`, `ToolBar.Item` and `Menu.Item`.
- `App.Async`, `App.Sync` and `App.Quit` after `Run` has returned do nothing; they panicked with "Device is disposed".
- `Search()` adds the magnifier icon and the clear button; on macOS and Linux the field had neither.

Fixes
- macOS: `look.SetSidebar` sent a Selection event for each Table, List or Tree in the sidebar that had a selected row (two per widget), so an app had to select its sidebar item after `SetSidebar`. The source-list style is now applied without reporting the reselection.
- `Menu.Item` showed a shortcut hint after "\t" that bound no key; `examples/dialogs` displayed Cmd+O and did nothing on it. It uses `Shortcut` now.
- docs: events.md said the facade had no mouse listeners (`Canvas.OnMouseDown`/`Up`/`Move` and `Table`/`Tree.OnCellClick` exist); facade.md listed widget fonts and colours as not wrapped.

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
