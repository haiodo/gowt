# gowt facade (package `gowt`)

`swt` stays the generated, Java-shaped low level (full SWT, for ported code). `gowt` is a
hand-written, thin, idiomatic layer on top; it is the API we document. Files: `gowt.go` (App,
Window, Panel, base widgets, options), `layout.go`, `widgets_common.go`, `widgets_list.go`,
`widgets_container.go`, `widgets_range.go`, `widgets_text.go`, `clipboard.go`, `dnd.go`, `graphics.go`, `menu.go`, `dialogs.go`, `theme.go`; the platform look is in `look` and the named icons in `icons` (see "Package layout"). Examples:
`cmd/hellogowt` (minimal), `cmd/gowtdemo` (a tab per widget family).

## Decisions

| Topic | Decision | Why |
|---|---|---|
| Entry | `gowt.Run(func(*App)) error` | Replaces LockOSThread + display + ReadAndDispatch/Sleep + Dispose in every main. |
| UI thread | `init()` in gowt calls `runtime.LockOSThread()` | The main goroutine is on the main thread until init ends; AppKit needs it. Users need no init of their own. |
| Loop end | no shells left, or `App.Quit()` | Matches the usual single-window app; Quit covers tray/background apps. |
| Threads | `App.Async/Sync(func())`, `App.After(d, func())`, `App.Every(d, func()) (stop func())` | Wrap `AsyncExec/SyncExec/TimerExec` with `jrt.NewRunnable` (type nobody outside can name). Async/Sync/Quit are safe from any goroutine; everything else is UI-thread only. |
| Constructors | `gowt.Button(text, onClick, opts ...Option) Option`; `app.Window(title, opts...)` | A constructor returns a description that is created in its parent when passed to it (see "Tree of values" below). No `Parent` argument, no `OverloadN`, no `NewXxxStyle`. Required data is positional. |
| Options | functional options; `Option` is a sealed interface (unexported method): style, common, placement, box (`Gap`, `Padding`), `Ref`/`Init`, `Shortcut` and child descriptions | SWT style bits must be known before creation; one type lets every kind share one variadic, so children and options mix in one call. An option the widget cannot use goes to `OnError` (each constructor passes the set of style options it reads). Option structs rejected: per-widget struct types double the API surface. |
| Events | `OnX(func(...))` methods, several calls add several listeners | Listeners attach after construction, so struct fields would not work with fluent building; a method can also register more than one. Callbacks take plain values (`func()`, `func(text string)`), `OnClose(func() bool)` returns false to veto. The SWT event is reachable through `Unwrap()`. |
| Errors | `Run` returns the `error` of `setup`; errors in event handlers go to `App.OnError` and the loop continues; no per-call errors | SWT reports failures by `panic(*SWTException)` and other Java exceptions (all `error`s). In `setup`, `Run` recovers any panic that is an `error` and not a `runtime.Error`, wraps with `%w`. In the loop, SWT's own per-listener catch (`ExceptionStash`, `RunnableLock`, timers) hands each error to the display's runtime exception and error handlers, which `Run` points at `OnError`; a panic that gets past them (a non-error value) is recovered around `ReadAndDispatch`. Default: log with the stack. Getters/setters do not return errors. |
| Escape hatch | `Unwrap()` on every wrapper returns the `*swt.X` | Facade never needs to cover all of SWT; users drop down per widget. Wrappers hold the swt object in a private field (no embedding), so the Java API does not leak into godoc. |
| Layouts | containers: `VBox`, `HBox`, `Grid(columns)`, `Flow`, `Fill`, `Form`, `Pages`, with `Gap` and `Padding`; per-child placement options `Grow`, `GrowX`, `GrowY`, `Span`, `Size`, `Place` or the whole `Cell(GridCell{...})` | The layout is fixed at creation, so placement is checked against it when the child is created. Gap defaults to `Spacing.SM`, Padding to `Spacing.PageMargin` on a Window and a Tab, `Spacing.SM` on a Group, 0 elsewhere (see windows-layouts.md). |
| Theme | `App.Dark()`, `App.OnThemeChange(func(dark bool))`; `Run` calls `Display.FollowSystemTheme()` | Light/dark follows the system while running: macOS KVO on `NSApp.effectiveAppearance`, Windows `WM_SETTINGCHANGE` + dark title bar (content stays light), Linux portal `color-scheme` over GDBus (no portal: GTK theme unchanged). Plain `swt` users call `FollowSystemTheme()` themselves; the callback fires only when dark flips. |
| Naming | Go names: `Checked`, `SetText`, `OnClick`; no `Get` prefix; ints for sizes (not int32); `time.Duration` for time; no `Like`/`As*`/`Overload`; the one exception is `AsComposite` on containers | |

## Wrapped (TSK-02)

| Family | API |
|---|---|
| Lists | `Combo`, `List`, `Table` (+`Column`, `TableRow`), `Tree` (+`Node`). Index-based `OnSelect(func(index int))`, `OnActivate`; tree callbacks get `*Node`. |
| Containers | `Group`, `Tabs` (TabFolder) and `CTabs` (CTabFolder, `OnClose` veto) with `Tab(title, ...)` children (`Closable()` on a CTabs tab), `Pages` (a composite with a StackLayout, children are pages: `Show(i)`, `Current()`), `Split` (SashForm), `Scroll` (ScrolledComposite: children in its content, fitted), `Sash` (bare), `ToolBar`+`ToolItem`, `CoolBar` (children are bands), `Section` (children are rows) |
| Ranges | `Scale`, `Slider`, `Spinner`, `Progress`, `DateTime` (`time.Time`), `Link` (`OnClick(func(href string))`) |
| Graphics | `Canvas(onPaint func(*GC))`, `GC` (colors, line width, Line/Rect/Oval/Fill*, Text, TextSize, Image), `Image`, `RGB` |
| Menus | `Window.MenuBar()`, `PopupMenu(widget)`, `Menu.Item/Submenu/Separator/Show/OnShow`, `MenuItem`, `Shortcut(keys)` (accelerator plus the "\t" hint); `App.Tray` (nil without a system tray) |
| Dialogs | `Window.MessageBox(Message)`, `Confirm`, `FileDialog(FileDialog)`, `DirDialog`, `ColorDialog`, `FontDialog`: value-type descriptions in, plain values out |
| Layouts | `Flow` (RowLayout, wrapping; `Vertical()`) + `Size`; `Form` + `Anchor(FormCell{Left, Right, Top, Bottom Edge})`; `Fill` (FillLayout; `Vertical()`) |

## Tree of values (step 5)

A screen is written as one nested expression: `w.Add(gowt.HBox(gowt.GrowX(), gowt.Label("Contacts"), gowt.Spacer(), gowt.Button("Add", add)), ...)`. The choices, each with its reason:

| Choice | Reason |
|---|---|
| A constructor returns `Option`; a child is an `Option` too | Go has one variadic per call, so options and children must share a type to mix in `HBox(GrowX(), Label(...), ...)`. A description is one small struct and a closure over the arguments; it is created in the parent at once and nothing of it is kept (no shadow tree, no reflection). |
| Widget types end in `View` (`LabelView`, `TextView`, ...) | Go has one namespace: `func Label` and `type Label` cannot both exist. Constructors appear on every line of a screen, types only where a widget is kept, so the types give up the plain names. `Panel` and `Window` keep theirs: no constructor has those names. |
| Layouts are containers (`VBox`, `HBox`, `Grid(n)`, `Flow`, `Fill`, `Form`, `Pages`), not `SetLayout` values | `gowt.HBox{}` as a value and `gowt.HBox(...)` as a constructor cannot share the name either. The constructor reads like the screen; the value would need a second constructor (`Panel(HBox{}, ...)`) for every box. `SetLayout` goes: a layout fixed at creation lets placement be checked when each child is created. |
| Window, Group, Tab page and Scroll content are columns (a VBox) | One rule instead of a layout option per frame; another arrangement is one box inside, as in SwiftUI. Cost: one more native composite per frame that needs a row or a grid. |
| `Gap` defaults to `Spacing.SM`, `Padding` to `PageMargin` on a Window and a Tab, `SM` on a Group, 0 elsewhere | The default UI is not glued together; nested boxes stay flush, so padding is not doubled. `Gap(0)` / `Padding(0)` opt out. |
| `HBox` gets one column per shown child at each layout pass | `Add`, `Dispose` and `SetVisible` change the count; a `boxLayout` hook over SWT's GridLayout recounts instead of asking callers to. |
| Events and initial values are options: `OnClick`, `OnChange`, `OnSelect`, `OnActivate`; `Hint`, `Checked`, `Selected`, `Value` (string, int or `time.Time`) | The handler sits next to its widget, and no variable per widget is needed. One name per concept: each event option is generic over a closed set of handler types (`OnChange[F func(string) \| func(bool) \| func(int) \| func(time.Time)]`), so a wrong handler type does not compile and a handler the widget does not report (a `func(int)` on a Text) goes to `OnError`. The methods of the same names stay for handlers added later. |
| rx bindings are options: `Bind(state)` two-way, `BindText`, `BindEnabled`, `BindVisible` one-way; `Watch(w, r, apply)` for the rest | Svelte's `bind:value` and `{expr}` in the markup: shared data is state, widgets follow it. They replace the binding methods instead of adding a second way; the lifetime is the widget's, as before. `Bind` is generic over `string \| bool \| int`, which covers every value the facade's widgets hold. |
| No positional callback on `Button`, `Switch`, `Link`; `Scale`, `Slider`, `Spinner` take `Value` instead of a positional value | One way per event: 39 of 88 `Button` and `Switch` calls in the examples passed `nil`. Positional arguments stay for required content: a label's text, a range's bounds, a canvas's paint function. Menu and tool items keep their callback argument: they are methods, not descriptions. |
| Closing parens sit on the line of the last child | gofmt accepts it; a line holding only `)` per container was most of the line growth of round 1 (106 such lines against 32 in v0.7). |
| `Init(func(x *T))` for imperative setup, `Ref(&x)` for a widget a later callback calls into | `Init` fills a Table or Tree, sets Split weights or adds tool items in place; `Ref` stays for `Pages.Show`, `Canvas.Redraw` and the like. Every hook (Ref, Init, events, initial values, bindings) runs after the widget's children exist, so `Selected` finds the tabs and `Init` the panes. |
| Children of `Tabs`/`CTabs` are `Tab(title, ...)`; children of `Pages`, `Section`, `Split`, `CoolBar` are their pages, rows, panes, bands | No `Page()`, `Row()`, `Content()` or `Add(widget)` methods: the container gives the child its role. |
| `container.Add(children...)` after creation; `Clear()` empties | The imperative path for dynamic content, and the way to create children whose callbacks use the window (`w` is not set while `app.Window` runs). |
| A misused option goes to `App.OnError` with the option and widget names; the widget is created without it | Same path as placement errors since step 2; the app keeps running. Style options a widget does not read are reported too (each constructor passes the set it reads). |

Round 2 (events, values and bindings as options, Init after the children, closing parens on the last child line) took the examples and demos from 1957 non-blank lines to 1732, against 1781 in v0.7, and `Ref` calls from 115 to 25.

Measured on the Linux stand (`BenchmarkTree200`: 50 HBox rows of a Label, a Text and a Button, created and disposed): the tree costs about 3 allocations per widget over the panel methods of v0.7 (196 282 against 195 713 allocations, 4.40 against 4.37 MB per op, time within noise), all in the descriptions; SWT's own work dominates.

## Decisions added in TSK-02

| Topic | Decision | Why |
|---|---|---|
| Widget | `Widget` interface with an unexported `control()` and the common methods (`SetEnabled`, `Enabled`, `SetVisible`, `Visible`, `SetTooltip`, `SetFont`, `SetForeground`, `SetBackground`, `Focus`, `Dispose`, `OnDispose`, `BindVisible`, `BindEnabled`); every wrapper embeds the unexported `base` that implements them | Layouts (`Beside`, `Same`), `PopupMenu`, `DropOn` refer to widgets without `*swt.Control` leaking. Methods on the embedded `base` show in `go doc` of each widget type, where free functions were not found. |
| Base style | Constructors pass `swt.NONE` plus the option bits, never a default style | `Widget.checkBits` keeps the FIRST set bit of a group in its own order (PUSH before CHECK, HORIZONTAL before VERTICAL, LEFT before RIGHT, DROP_DOWN before SIMPLE, SINGLE before MULTI), so an OR-ed default overrides the option. SWT adds the default itself when no bit is set. The prototype's `Button(Check())` was affected. |
| Items | Items are not widgets: `table.Row(cells...)`, `table.Column(title, width)`, `tree.Node(text)`, `node.Node(text)`, `toolbar.Item(text, onClick)`, `menu.Item(text, onClick)`; a `Tab(title, ...)` description creates the item and its page and gives the page `*Panel` to `Ref` | The item/page pairs always go together; the user wants the page. Item options only contribute style bits (`Check()`, `Radio()`, `DropDown()`, `Closable()`, `Right()`); any other option goes to `OnError`. |
| Re-wrapping | `Row`, `Node`, `Tree.OnX` hand out fresh wrappers per call; use `SetData/Data` for per-item state | Wrappers are one pointer; caching them would need a map keyed by swt item. |
| Popup menu | package function `PopupMenu(w Widget)`, not a method | Go has no extension methods and every wrapper would repeat it. |
| Selection | `Index() int` (first selected, -1 none) / `Selected() []int` / `SetSelection(...int)` for Combo/List/Table; ints, not int32 | |
| Image ownership | The caller owns an `Image`: `Dispose()` (idempotent) when done. Widgets (`SetImage`) neither copy nor dispose, so keep it alive while shown. `App` also disposes every image still alive when `Run` returns. `LoadImage`/`ImageFrom` return `error` (the one place a failure is routine), via the same recover as `Run` | SWT images are native resources that must be disposed explicitly. |
| Colors, fonts | Value types `RGB`, `Font{Name, Size, Weight, Bold, Italic}`; `GC.SetColor/SetFill` take `RGB` and create the SWT color internally (`swt.NewColorRedGreenBlue`, not a disposable resource). Widget fonts: the `SetFont` method | Avoids the Color/Font dispose protocol in the common path. |
| GC | `*GC` exists only inside the paint callback (SWT creates and disposes it per event); `Canvas` is DOUBLE_BUFFERED by default | Nothing to dispose, nothing to leak. Drawing into an `Image` offscreen is not wrapped (`Unwrap` + `swt.NewGCDrawable`). |
| Placement | `Grow`, `GrowX`, `GrowY`, `Span`, `Size`, `Place` (and the whole-struct `Cell`, `Anchor`) add up in one `placement` per widget; `finish` turns it into the `GridData`, `RowData` or `FormData` the parent's layout reads. A mismatch goes to `OnError` with the option, widget and layout names and sets no layout data | SWT layouts cast the layout data and fail with a `ClassCastException` on the next layout pass, far from the line that chose the option. |
| Form | `Form` edges are `Edge` values built by `Percent(pct, offset)`, `Beside(&w, gap)`, `Same(&w, offset)`, the zero `Edge` is unattached. `Beside` and `Same` take the address of a `Ref` variable and read it when the child is placed | Mirrors `Grid`/`GridCell`; no `FormAttachment` nil/zero confusion; a sibling in the same tree is not created when the edge is written. |
| Dialogs | Methods on `Window` (the parent); settings as one struct (`Message`, `FileDialog`), results as values (`Answer`, `[]string` or nil on cancel, `(value, ok)`) | A dialog is a one-shot call; builder objects add nothing. |

## jface integration (note only, nothing implemented)

User-level page: [jface.md](jface.md).

jface factories take the parent as `swt.CompositeLike` and return the swt widget
(`ButtonFactory.Create(parent) *swt.Button`). From gowt they plug in through `Unwrap()`:
`jface.ButtonFactoryNewButton(swt.PUSH).Text("x").Create(panel.Unwrap())`. The result is a plain swt
widget; gowt has no constructor that wraps an existing `*swt.X` (it would let wrappers be built
around factory output, deliberately not exported yet). jface has no viewers in this repo, so there is nothing to plug
`Table`/`Tree` into.

## Modern look, macOS 26 (TSK-11, 12, 17)

| API | Behavior |
|---|---|
| (none) | Liquid Glass chrome is on by itself: the Go linker records `sdk 26.2` in `LC_BUILD_VERSION` of every binary. |
| `look.Classic()` | Before `Run`: sets `UIDesignRequiresCompatibility` via NSUserDefaults and the main bundle info dictionary. No Info.plist exists for a plain binary, so whether AppKit reads it there is checked only by eye. Reliable alternative: `go build -ldflags=-macsdk=15.0`. |
| `look.SetBackdrop(w, b)` | `Translucent` is an NSVisualEffectView, `Glass` an NSGlassEffectView (26+, else Translucent), the window one sits in the window frame view below the content view, a panel one in the panel parent right below the panel (frame kept in step by a resize listener); SWT puts every new child at the bottom of its parent, so a child backdrop would end up above the widgets. The window turns non-opaque. SWT composites fill their background, so the window and panels also get an alpha-0 background plus `SetBackgroundMode(INHERIT_FORCE)`; |
| `look.SetFullSizeContent(w, on)` | Full-size content view mask plus transparent title bar; the window layout gets the title bar height as top inset (Window.SetTopInset). |
| `look.SetGlass(c, on)` (c is any container: `Panel`, `Window`, `Group`, `Split`, `CoolBar`) | NSGlassEffectView behind a panel (corner radius 12). |
| `look.GlassButton()` | `gowt.Option`: `bezelStyle = .glass` (16), only when NSGlassEffectView exists. |
| `look.SetToolbar(w, items...) bool` | NSToolbar (`toolbarStyle = .unified`) set on the NSWindow, not in the SWT view tree. Items come from a run-time ObjC delegate class (`GowtToolbarDelegate`, target and action `gowtClick:`); item identifiers `gowt.item.<index>`, spaces are `NSToolbarFlexibleSpaceItem`. Other OS: returns false; with `look.SetSidebar` the items become a gowt `Panel` laid out by the sidebar layout above the content (flat `ToolBar` items with SVG icons, a `Label` for the title fed by `Window.SetTitle` through the shell data key `gowt.look.title`, a `Search` `Text`), otherwise nothing. |
| `look.SetToolbar` items: `Group`, `Nav`, `OnSearch` | `Group` is `NSToolbarItemGroup groupWithItemIdentifier:images:selectionMode:labels:target:action:` (momentary, expanded; text segments if a symbol is missing), the pressed segment read from `selectedIndex`. `Nav` sets `navigational`. `OnSearch` is an NSSearchToolbarItem whose search field carries the item identifier and sends `gowtSearch:`. |
| `look.SetSidebar(w, sidebar, content, width)` | A Go `swt.Layout` on the shell: glass frame inset 7 pt (macOS 26), sidebar composite and content below `ContentTopInset`. The glass is an InstallBackdrop sibling moved by the layout (`SetBackdropFrame`), corner radius 18, not through `SetGlass`, whose listeners would pin it to the sidebar frame. Lists, tables and trees in the sidebar get `NSTableViewStyleSourceList`; the SWT table class gets a `setBackgroundColor:` that stores clear for source-list instances (SWT puts its opaque default back whenever the background is refreshed, since an alpha-0 parent background does not count) and a `tableView:rowViewForRow:` that returns never-emphasized row views (`GowtSidebarRowView`) for them, so the selection is a grey pill; `List.vtDrawCell` draws the selected text in `controlAccentColor`. After the window opens it is laid out again and its view tree marked for display once. The toolbar gets a first item `gowt.sidebar`: navigational, unbordered, an empty `GowtToolbarSpacer` view with a width constraint fitted on every window layout and on its own `layout` so that it ends 4 pt past the sidebar edge (AppKit places navigational items and the title right after the traffic lights). An `NSTrackingSeparatorToolbarItem` on a hidden split view was tried first: AppKit kept the navigational items and the title before the separator, so they ran over the sidebar edge. |
| `gowt.Section()` | Composite background with `INHERIT_DEFAULT`, corners painted in the parent colour (radius 12), colour taken again on `SWT.Settings`. macOS: `quaternarySystemFillColor` composited over `windowBackgroundColor` in the app's appearance (`cocoa.GroupedFillRGB`). A layer-only card view behind the section was tried first: SWT composites without a background of their own fill the window colour in `drawRect:` and hid it. Windows/Linux: 6% from `COLOR_WIDGET_BACKGROUND` to `COLOR_WIDGET_FOREGROUND`. |
| `look.SetGlassGroup(c, spacing)` | An NSGlassEffectContainerView subclass (`GowtGlassGroup`, `hitTest:` returns nil) fills `c`'s view; glass views installed for its children go into the container's content view, and the container is kept just below the lowest glass panel (SWT inserts new children at the bottom). |

Every class and selector newer than macOS 13 is looked up at run time; other systems get no-ops.
The toolbar, glass group and background extension (TSK-34) are compiled but not yet checked by eye.

## Not wrapped

- Custom drag and drop transfers, RTF/URL clipboard formats, the X11 selection clipboard, StyledText bullets/indents/undo listeners (`Unwrap()`).
- The SWT event/listener type hierarchy, `Display` as a public type (`App.Unwrap()` exists), `Runnable`, `Internal_*`.
- Cursors, keyboard listeners and mouse listeners other than `Table`/`Tree.OnCellClick` and the `Canvas` ones (`Unwrap()`), `ExpandBar`, `Browser`, `CCombo`, `Shell` kinds other than `Window` (dialog shells, tool tips, `Decorations` images), key bindings outside menu items, tray menus, `Sash` repositioning helpers (use `Split`), sorting the rows (only the header indicator `SetSort`), `ImageData`/palettes/transforms/paths/patterns, `GC` clipping, fonts, alpha, polygons.

## Package layout (applied, TSK-31)

The root keeps what every app needs; the platform look and the icon set live in their own packages.

| Package | Contents | Files |
|---|---|---|
| `gowt` | `Run`, `SetAppName`, `App` (threads, `Tray`, images, `Dark`/`OnThemeChange`), `Window`, `Panel` and every widget, options, layouts, dialogs, menus, `GC`/`Image`/`RGB`/`Font` | `gowt.go clipboard.go dialogs.go dnd.go graphics.go layout.go menu.go theme.go theme_darwin.go theme_other.go widgets_*.go manifest_windows.go` + 2 tests |
| `gowt/look` | window material, macOS 26 glass, Windows corners and dark content | `look.go look_darwin.go look_windows.go look_other.go` |
| `gowt/icons` | named icons (`icons/lucide` stays below it) | the seven `icons*.go` files, names unchanged |

Theme stays in the root: custom painting and `icons` read `Dark`, and `Run` already follows the system theme.

One `look` package, because:
- per-OS `gowt/macos` + `gowt/windows`: `Backdrop` is one value on both systems (Glass is Liquid Glass and Mica), so it would be declared twice, a portable app imports both, and the names hint at build tags although both must compile everywhere;
- `gowt/appearance`: same contents, but on macOS "appearance" means light/dark (`NSAppearance`), which stays in the root;
- `look`: one import, short, already the word in `ClassicLook`.

Go constraints:
- No methods on `gowt.Window` from outside: window settings become functions that take the window, `w.SetBackdrop(b)` -> `look.SetBackdrop(w, b)`. `look` reaches the shell through `w.Unwrap()`; the Windows-only `Window.backdropOn` field becomes `shell.SetDataKeyValue`.
- `SetGlass` was promoted to `Panel`, `Window`, `Group`, `Split`, `CoolBar`, whose `Unwrap()` return different swt types. They share the promoted `AsComposite() *swt.Composite` (declared once on the unexported `panel`), and `look.Container` is the one-method interface over it, so `g.SetGlass(true)` -> `look.SetGlass(g, true)`; jface factories can take the same interface. `panel.clear` moves with it; it uses only public swt calls.
- `Option` fields are unexported, so `look.GlassButton()` cannot build one. Hook 1: `gowt.Custom(f func(*swt.Control)) Option`, run after creation, no style bits. Exporting `style`/`apply` is rejected: every `Option` would carry raw SWT bits in godoc.
- `icons` needs the display (`app.Unwrap()`), the theme (`app.Dark()`) and an `Image` the App disposes when `Run` returns; `Image.i` is unexported. Hook 2: `App.ImageFromProvider(p swt.ImageDataProvider) *Image` (`a.track` of `swt.NewImageDeviceImageDataProvider`); it also turns `svg.NewImageDataProvider` output into a `gowt.Image`. The cache moves from `App.icons` to a `map[*gowt.App]...` in `icons`. Ceiling: entries of a finished `Run` are never freed, fine for one `Run` per process; the way out is an App end-of-run hook.
- No cycle: after the move the root has no `App.icons`/`iconKey`, no `Window.backdropOn`, no `GlassButton`. `look` and `icons` import `gowt`, `swt`, `svg`, `internal/cocoa`, `internal/win32`; `gowt` imports neither.

Small cleanups in the same change, no API change:
- `extras.go` goes: `SetAppName` next to `Run`, `Background` next to `Tooltip`, `Window.SetTitle` next to `Window.SetSize` (all `gowt.go`), `List.SetItem` to `widgets_list.go`.
- No-op stubs: five `*_other.go` files today, one per feature. In `look` each OS file defines the whole unexported set: `look_darwin.go` adds 2 no-op lines (corners, dark content), `look_windows.go` 4 (classic, full-size content, glass, glass button), `look_other.go` (`!darwin && !windows`) is all no-ops. The root keeps only the `theme_darwin.go`/`theme_other.go` pair.

| Before (`gowt`) | After |
|---|---|
| `type Backdrop`, `BackdropNone`, `BackdropTranslucent`, `BackdropGlass` | `look.Backdrop`, `look.BackdropNone`, `look.BackdropTranslucent`, `look.BackdropGlass` |
| `Window.SetBackdrop(b)` | `look.SetBackdrop(w *gowt.Window, b look.Backdrop)` |
| `Window.SetFullSizeContent(on)` | `look.SetFullSizeContent(w *gowt.Window, on bool)` |
| `Window.SetRoundedCorners(on)` | `look.SetRoundedCorners(w *gowt.Window, on bool)` |
| `Panel/Window/Group/Split/CoolBar.SetGlass(on)` | `look.SetGlass(c look.Container, on bool)` |
| `ClassicLook()` | `look.Classic()` |
| `GlassButton() Option` | `look.GlassButton() gowt.Option` |
| `App.SetDarkContent(on)` | `look.SetDarkContent(app *gowt.App, on bool)` |
| `App.Icon(name, opts...)` | `icons.Get(app *gowt.App, name string, opts ...icons.Option) *gowt.Image` |
| `type IconOption` | `icons.Option` |
| `IconSize(points)`, `IconColor(c)` | `icons.Size(points)`, `icons.Color(c gowt.RGB)` |
| `IconNames()` | `icons.Names()` |
| (new) | `gowt.Custom(f func(*swt.Control)) Option`, `App.ImageFromProvider(p swt.ImageDataProvider) *Image`, `AsComposite() *swt.Composite` on `Panel`, `Window`, `Group`, `Split`, `CoolBar`, `look.Container` |

Impact:
- Callers: `cmd/backdropdemo`, `cmd/glassdemo`, `cmd/icondemo`. `cmd/minibrowser` and `cmd/webviewdemo` use only `SetAppName`, which stays. `examples/` and `README.md` use none of the moved symbols.
- Tests: `icons_test.go` moves to `icons/` as is. `icons_gui_test.go` needs its own `TestMain`/`guiRun` in `icons/` (about 25 lines), since the root's live in package `gowt` tests.
- `tooling/apidump`: `facadeDirs` += `look`, `icons`, header text updated; the golden is rewritten with `-facade -update` (removals intended, pre-1.0).
- `Makefile` `xcheck`: `./look ./icons` join the three-OS build line, and `look/*.go icons/*.go` the platform-import grep.
- `docs/facade.md`: the file list at the top and the "Modern look" table get the new call shapes.
- Unchanged: every other `gowt` symbol, `swt`, `svg`, `webview`, `browser`, `jface`, and what the moved functions do.

## Clipboard, drag and drop, StyledText (applied)

Refs TSK-2026-10-03-product-24 (clipboard, dnd), TSK-2026-10-03-product-25 (StyledText). Every call maps to existing `swt` symbols; `swt` is unchanged. Examples: `examples/clipboard`, `examples/dnd`, `examples/styledtext`; user pages in `docs/widgets.md` and `docs/events.md`.

### New symbols and files

| File (new) | Symbols | Maps to |
|---|---|---|
| `clipboard.go` | `App.Clipboard() *Clipboard`; `Clipboard.Text() (string, bool)`, `SetText(s) error`, `HTML() (string, bool)`, `SetHTML(html, plain string) error`, `Files() []string`, `SetFiles([]string) error`, `Image() (*Image, bool)`, `SetImage(*Image) error`, `Clear()`, `Unwrap() *swt.Clipboard` | `swt.NewClipboard(display)`, `GetContents(TextTransfer/HTMLTransfer/FileTransfer/ImageTransfer instance)`, `SetContents(data, transfers)`, `ClearContents` |
| `dnd.go` | `DragFrom(w Widget, d Drag) *DragHandle`, `DropOn(w Widget, d Drop) *DropHandle`, `Drag`, `Drop`, `Op` (`OpNone`, `OpCopy`, `OpMove`, `OpLink`), `Unwrap()` on both handles | `swt.NewDragSource/NewDropTarget`, `SetTransfer`, `AddDragListener/AddDropListener`, `DNDDROP_*` |
| `widgets_text.go` | `StyledText(opts ...Option) Option` creating a `*StyledTextView`, its methods below, `TextStyle`, `StyleSpan`, option `Scrollbars()` | `swt.NewStyledText`, `StyleRange`, `SetStyleRange(s)`, listeners |
| `gowt.go` | `App.clip`, `Run` disposes the clipboard before the display; `apps` map (display -> App) so a drop callback can create App-owned images | `Clipboard.Dispose` |

Nothing goes into `gowt.go`/`widgets_common.go` beyond that line; each topic gets its own file. The golden `tooling/apidump/facade-api.txt` was regenerated; the diff is additions only.

### Clipboard

- `App.Clipboard()` creates the `swt.Clipboard` on first call and caches it in `App`; `Run` disposes it (deferred after `display.Dispose`'s defer is registered, so it runs first). The user never disposes it. A `*Clipboard` is invalid after `Run` returns.
- UI thread only, like every call except Async/Sync/Quit. From another goroutine: `app.Sync(func(){ ... })`.
- Getters return `(value, ok)`; `ok` is false when the clipboard has no such format (`GetContents` returns nil). `Files()` returns nil when empty.
- Setters return `error`: `SetContents` panics with `DNDError(DNDERROR_CANNOT_SET_CLIPBOARD)` on all three OSes when the system clipboard is busy (routine on Windows), so, like `LoadImage`, the facade recovers it with `catch`. This is the one place besides images where a failure is routine.
- `SetHTML(html, plain)` writes both formats in one `SetContents` call (`HTMLTransfer` + `TextTransfer`); setting one format clears the others (SWT behaviour), so combined content must go through one call.
- `Image()` builds an `Image` from the `ImageData` via `a.track`, so the App owns it and disposes it when `Run` returns (`Dispose` earlier is allowed). `SetImage` reads `img.Unwrap().GetImageData()`.
- Not covered: custom `ByteArrayTransfer` types, RTF, URL, several formats beyond HTML+text, async `GetContentsAsync`, the X11 selection clipboard (`clipboards` argument) -> `Unwrap()`.

### Drag and drop

Functions that take the widget, like `PopupMenu(w)` (no extension methods in Go); settings as one value-type struct, like `Message`/`FileDialog`.

```go
type Op int // OpCopy, OpMove, OpLink; zero value = OpCopy
type Drag struct {
	Ops     Op                 // allowed operations, zero = OpCopy
	Text    func() string      // any non-nil getter adds its format; called once per drop (DragSetData)
	Files   func() []string
	Image   func() *Image      // not disposed by the facade
	OnStart func() bool        // false cancels this drag (DragStart.Doit)
	OnDone  func(Op)           // operation performed, OpNone if nobody took it
}
type Drop struct {
	Ops   Op
	Text  func(text string, op Op)
	Files func(paths []string, op Op)
	Image func(img *Image, op Op) // the App owns img (tracked); Dispose may be called earlier
	Over  func(x, y int) bool     // x, y in widget coordinates; false rejects at that point
}
```

- Mapping: `Drag` -> `DragSource` + `SetTransfer` built from the non-nil getters (`TextTransfer`, `FileTransfer`, `ImageTransfer` instances) + a listener whose `DragSetData` sets `event.Data` (field of `TypedEvent`) by `event.DataType`. `Drop` -> `DropTarget` + `SetTransfer` from the non-nil callbacks + a listener: `DragEnter/DragOver` filter `event.CurrentDataType` and the `Over` answer (`event.Detail = DNDDROP_NONE` rejects), `Drop` dispatches by data type. The listener is an unexported struct embedding `swt.DropTargetAdapter`/`DragSourceAdapter`, like `modifier`.
- `DropTargetEvent.X/Y` are display coordinates; the facade converts with `Control.ToControl` before calling `Over`.
- Lifetime: swt `DragSource`/`DropTarget` are widgets registered on the control and disposed with it; the handle only exposes `Unwrap()`. At most one `DropTarget` and one `DragSource` per control: a second call panics with `DNDERROR_CANNOT_INIT_DROP` (an error), returned by `Run`.
- Works on any `Widget` the facade has; whether the native widget accepts drops (Table rows, Tree items, text caret) is the platform's business. Item-level targets (`event.Item`) are not exposed.
- Not covered -> `Unwrap()` and plain `swt`: custom `Transfer` types, RTF/HTML/URL transfers, effects (`DropTargetEffect`, `DragSourceEffect`), drag images (`DragSourceEvent.Image`), `Feedback` flags, `DragOperationChanged`/`DropAccept`, moving data between processes with custom formats.
- Platform caveats (`tests/expected*.txt`): drag was never run with a real mouse by any test, so end-to-end drag between widgets and from other apps is unverified on all three OSes. Cross-process transfer tests are skipped ("no remote clipboard peer: the Swing/RMI process is not ported"), including `FileTransfer`, `HTMLTransfer`, `ImageTransfer`, `RTFTransfer`, `TextTransfer`. `URLTransfer` tests are disabled on macOS, so URLs are left out. `FileTransfer.nativeToJava` is disabled on Windows and macOS in the upstream suite.

### StyledText

`StyledText(opts ...Option) Option`, constructor `swt.NewStyledText(parent, resolve(swt.NONE, opts))`. Existing options apply: `ReadOnly()`, `Border()`, `Wrap()`, `Disabled()`, `Tooltip`, the placement options. New `Scrollbars()` (V_SCROLL|H_SCROLL), since SWT has no default and the base-style rule forbids one.

```go
type TextStyle struct {
	Foreground, Background *RGB // nil = inherit
	Bold, Italic, Underline, Strikeout bool
	Font *Font                  // optional; name/size override, Bold/Italic still apply
}
type StyleSpan struct { Start, Length int; Style TextStyle }
```

| Method | swt call |
|---|---|
| `Text()`, `SetText(s)`, `Append(s)`, `Insert(s)` (at caret), `Replace(start, length int, s string)` | `GetText`, `SetText`, `Append`, `Insert`, `ReplaceTextRange` |
| `Selection() (start, length int)`, `SetSelection(start, length int)`, `SelectedText()`, `Caret() int`, `SetCaret(offset int)`, `Reveal()` | `GetSelectionRange`, `SetSelectionRange`, `GetSelectionText`, `GetCaretOffset`, `SetCaretOffset`, `ShowSelection` |
| `Lines() int`, `LineAt(offset int) int`, `LineStart(line int) int`, `SetTopLine(i int)` | `GetLineCount`, `GetLineAtOffset`, `GetOffsetAtLine`, `SetTopIndex` |
| `SetStyle(start, length int, s TextStyle)`, `Styles() []StyleSpan`, `ClearStyles()` | `SetStyleRange(StyleRange)`, `GetStyleRanges`, `ReplaceStyleRanges(0, GetCharCount, nil)` |
| `SetLineBackground(line, count int, c RGB)` | `SetLineBackground` |
| `SetEditable(bool)`, `SetWrap(bool)`, `SetTabs(n int)` | `SetEditable`, `SetWordWrap`, `SetTabs` |
| `Cut()`, `Copy()`, `Paste()` | same names |
| `OnChange(func(text string))` | `AddModifyListener` (as `Text.OnChange`) |
| `OnSelect(func(start, length int))` | `AddSelectionListener` |
| `OnCaret(func(offset int))` | `AddCaretListener` |
| `OnStyle(func(line string) []StyleSpan)` | `AddLineStyleListener`; spans are relative to the line start, converted to the event's `Styles` |
| `Unwrap() *swt.StyledText` | |

- Offsets and lengths are runes (indexes into `[]rune(Text())`); the facade converts to and from the UTF-16 units SWT uses (`DefaultContent` stores `[]uint16`). Known ceiling: each conversion scans the text, O(n) per call; a line index is the way out for very large documents. `Unwrap()` offsets stay UTF-16. `OnStyle` avoids the scan: its callback gets one line and `Start` counts runes from the line start.
- `TextStyle` -> `swt.StyleRange`: colors via `RGB.color()`; `Bold`/`Italic` go to `StyleRange.FontStyle` (no font object, widget font is reused); `Font` with a name or size creates an `swt.Font` that the `StyledText` wrapper keeps in a cache keyed by the `Font` value and disposes on the widget's dispose event. The user never sees a disposable.
- Not covered -> `Unwrap()`: bullets, line alignment/indent/spacing providers, `GlyphMetrics`, `TextStyle.Rise`/`Data`/borders/underline styles and colors, block and multi-selection, `VerifyListener`/`VerifyKeyListener`/`ExtendedModifyListener` (undo tracking), bidi segments, content replacement (`StyledTextContent`), printing, widget fonts (no widget-font API in the facade yet).
- Platform caveats (`tests/expected*.txt`): style rendering tests (`setStyleRanges_render`, `lineStyleListener_*_render`, `lostStyles`) are skipped on macOS (upstream bugs 553090, 536588), so rendering of styled text is verified only on Linux/Windows; foreground alpha does not exist on Windows; `test_backspaceAndDelete` fails on macOS (`Display.post` does not deliver the key event, test infrastructure, not diagnosed); `caretSizeAndPositionVariableGlyphMetrics` is skipped on Linux; clipboard-carryover test is skipped on macOS.

### Decisions taken

- Drag and drop are functions (`DragFrom`, `DropOn`) with value-type descriptions; `Option`s were rejected because they cannot be added after construction.
- `StyledText` offsets are runes; setters of the clipboard return `error` (a busy clipboard is routine on Windows); images from the clipboard and from a drop are App-owned. Widgets keep no App pointer: the `apps` map is the only link.
- `Scrollbars()` stays in `widgets_text.go`; no X11 selection clipboard.

### Open points

- No test drags with a real mouse; `DragFrom` and `DropOn` have not run end to end on any OS, only compiled and vetted. Needs a stand with synthetic pointer input.
- Editing text that contains surrogate pairs (emoji) panics in `swt` (`DefaultContent.GetLine` -> `jrt.Substring` slice bounds, found on Linux after `ReplaceTextRange`); selection, caret and styles on such text work. A generator-level bug, not fixed here.
- `Styles()` does not report `TextStyle.Font`; `SetStyle` fonts are cached per widget by `Font` value until the widget is disposed.

## Open points

- `Sync` called from the UI thread runs inline (SWT behaviour); documented, not changed.
- Test `gowt_test.go` opens windows; it skips unless `GOWT_GUI_TEST=1` (Linux Xvfb stand).
