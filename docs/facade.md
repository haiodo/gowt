# gowt facade (package `gowt`)

`swt` stays the generated, Java-shaped low level (full SWT, for ported code). `gowt` is a
hand-written, thin, idiomatic layer on top; it is the API we document. Files: `gowt.go` (App,
Window, Panel, base widgets, options), `layout.go`, `widgets_common.go`, `widgets_list.go`,
`widgets_container.go`, `widgets_range.go`, `graphics.go`, `menu.go`, `dialogs.go`, `theme.go`; the platform look is in `look` and the named icons in `icons` (see "Package layout"). Examples:
`cmd/hellogowt` (minimal), `cmd/gowtdemo` (a tab per widget family).

## Decisions

| Topic | Decision | Why |
|---|---|---|
| Entry | `gowt.Run(func(*App)) error` | Replaces LockOSThread + display + ReadAndDispatch/Sleep + Dispose in every main. |
| UI thread | `init()` in gowt calls `runtime.LockOSThread()` | The main goroutine is on the main thread until init ends; AppKit needs it. Users need no init of their own. |
| Loop end | no shells left, or `App.Quit()` | Matches the usual single-window app; Quit covers tray/background apps. |
| Threads | `App.Async/Sync(func())`, `App.After(d, func())` | Wrap `AsyncExec/SyncExec/TimerExec` with `jrt.NewRunnable` (type nobody outside can name). Async/Sync/Quit are safe from any goroutine; everything else is UI-thread only. |
| Constructors | `parent.Button(text, onClick, opts ...Option)`; `app.Window(title, opts...)` | The parent is the receiver: no `Parent` argument, no `OverloadN`, no `NewXxxStyle`. Required data is positional. |
| Options | functional options, one `Option{style, apply}` type for all widgets | SWT style bits must be known before creation; a single type lets common options (`Cell`, `Tooltip`, `Disabled`) and style options (`Check`, `Multiline`, `Password`) share one variadic. Cost: a style option on the wrong widget is silently ignored by SWT; accepted. Option structs rejected: per-widget struct types double the API surface and cannot express "style or post-setter" uniformly. |
| Events | `OnX(func(...))` methods, several calls add several listeners | Listeners attach after construction, so struct fields would not work with fluent building; a method can also register more than one. Callbacks take plain values (`func()`, `func(text string)`), `OnClose(func() bool)` returns false to veto. The SWT event is reachable through `Unwrap()`. |
| Errors | `Run` returns `error`; no per-call errors | SWT reports failures by `panic(*SWTException)` and other Java exceptions (all `error`s). `Run` recovers any panic that is an `error` and not a `runtime.Error`, wraps with `%w`. Programmer bugs (nil deref etc.) keep panicking. Getters/setters do not return errors. |
| Escape hatch | `Unwrap()` on every wrapper returns the `*swt.X` | Facade never needs to cover all of SWT; users drop down per widget. Wrappers hold the swt object in a private field (no embedding), so the Java API does not leak into godoc. |
| Layouts | value types: `Fill{Vertical}`, `Grid{Columns, EqualWidth, Margin, Spacing}`; per-child `Cell(GridCell{...})` | Plain structs with zero-value meaning "none/natural", enum `Align*` instead of int bit masks. Applied with `SetLayout`. `Row`, `Form`, `Stack` follow the same rule (see below). |
| Theme | `App.Dark()`, `App.OnThemeChange(func(dark bool))`; `Run` calls `Display.FollowSystemTheme()` | Light/dark follows the system while running: macOS KVO on `NSApp.effectiveAppearance`, Windows `WM_SETTINGCHANGE` + dark title bar (content stays light), Linux portal `color-scheme` over GDBus (no portal: GTK theme unchanged). Plain `swt` users call `FollowSystemTheme()` themselves; the callback fires only when dark flips. |
| Naming | Go names: `Checked`, `SetText`, `OnClick`; no `Get` prefix; ints for sizes (not int32); `time.Duration` for time; no `Like`/`As*`/`Overload`; the one exception is `AsComposite` on containers | |

## Wrapped (TSK-02)

| Family | API |
|---|---|
| Lists | `Combo`, `List`, `Table` (+`Column`, `TableRow`), `Tree` (+`Node`). Index-based `OnSelect(func(index int))`, `OnActivate`; tree callbacks get `*Node`. |
| Containers | `Group`, `Tabs` (TabFolder), `CTabs` (CTabFolder, `Closable()`, `OnClose` veto), `Split` (SashForm), `Scroll` (ScrolledComposite: `Content()` then `Fit()`), `Sash` (bare), `ToolBar`+`ToolItem`, `CoolBar` |
| Ranges | `Scale`, `Slider`, `Spinner`, `Progress`, `DateTime` (`time.Time`), `Link` (`OnClick(func(href string))`) |
| Graphics | `Canvas(onPaint func(*GC))`, `GC` (colors, line width, Line/Rect/Oval/Fill*, Text, TextSize, Image), `Image`, `RGB` |
| Menus | `Window.MenuBar()`, `PopupMenu(widget)`, `Menu.Item/Submenu/Separator/Show/OnShow`, `MenuItem`; `App.Tray` (nil without a system tray) |
| Dialogs | `Window.MessageBox(Message)`, `Confirm`, `FileDialog(FileDialog)`, `DirDialog`, `ColorDialog`, `FontDialog`: value-type descriptions in, plain values out |
| Layouts | `Row{Vertical, Wrap, Fill, Center, Margin, Spacing}` + `InRow(RowCell)`; `Form{Margin, Spacing}` + `Anchor(FormCell{Left, Right, Top, Bottom Edge})`; `Stack{Margin}` + `Panel.ShowTop(w)` |

## Decisions added in TSK-02

| Topic | Decision | Why |
|---|---|---|
| Widget | `Widget` interface with an unexported `control()`; every wrapper implements it | Layouts (`Beside`, `Same`), `ShowTop`, `CoolBar.Add`, `PopupMenu` refer to siblings without `*swt.Control` leaking. |
| Base style | Constructors pass `swt.NONE` plus the option bits, never a default style | `Widget.checkBits` keeps the FIRST set bit of a group in its own order (PUSH before CHECK, HORIZONTAL before VERTICAL, LEFT before RIGHT, DROP_DOWN before SIMPLE, SINGLE before MULTI), so an OR-ed default overrides the option. SWT adds the default itself when no bit is set. The prototype's `Button(Check())` was affected. |
| Items | Items are not widgets: `table.Row(cells...)`, `table.Column(title, width)`, `tree.Node(text)`, `node.Node(text)`, `toolbar.Item(text, onClick)`, `menu.Item(text, onClick)`; tab folders return the page `*Panel` from `Tab(title)` | The item/page pairs always go together; the user wants the page. Item options only contribute style bits (`Check()`, `Radio()`, `DropDown()`, `Closable()`, `Right()`). |
| Re-wrapping | `Row`, `Node`, `Tree.OnX` hand out fresh wrappers per call; use `SetData/Data` for per-item state | Wrappers are one pointer; caching them would need a map keyed by swt item. |
| Popup menu | package function `PopupMenu(w Widget)`, not a method | Go has no extension methods and every wrapper would repeat it. |
| Selection | `Index() int` (first selected, -1 none) / `Selected() []int` / `SetSelection(...int)` for Combo/List/Table; ints, not int32 | |
| Image ownership | The caller owns an `Image`: `Dispose()` (idempotent) when done. Widgets (`SetImage`) neither copy nor dispose, so keep it alive while shown. `App` also disposes every image still alive when `Run` returns. `LoadImage`/`ImageFrom` return `error` (the one place a failure is routine), via the same recover as `Run` | SWT images are native resources that must be disposed explicitly. |
| Colors, fonts | Value types `RGB`, `Font{Name, Size, Bold, Italic}`; `GC.SetColor/SetFill` take `RGB` and create the SWT color internally (`swt.NewColorRedGreenBlue`, not a disposable resource). There is no widget-font API yet | Avoids the Color/Font dispose protocol in the common path. |
| GC | `*GC` exists only inside the paint callback (SWT creates and disposes it per event); `Canvas` is DOUBLE_BUFFERED by default | Nothing to dispose, nothing to leak. Drawing into an `Image` offscreen is not wrapped (`Unwrap` + `swt.NewGCDrawable`). |
| Row/Form/Stack | Zero value means none / natural size; `Form` edges are `Edge` values built by `Percent(pct, offset)`, `Beside(w, gap)`, `Same(w, offset)`, the zero `Edge` is unattached. `Stack` keeps the `StackLayout` in the `Panel`, so `ShowTop` panics on a panel without a Stack layout | Mirrors `Grid`/`GridCell`; no `FormAttachment` nil/zero confusion. |
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
| `look.SetFullSizeContent(w, on)` | Full-size content view mask plus transparent title bar. |
| `look.SetGlass(c, on)` (c is any container: `Panel`, `Window`, `Group`, `Split`, `CoolBar`) | NSGlassEffectView behind a panel (corner radius 12). |
| `look.GlassButton()` | `gowt.Option`: `bezelStyle = .glass` (16), only when NSGlassEffectView exists. |

Every class and selector newer than macOS 13 is looked up at run time; other systems get no-ops.
Not wrapped: NSGlassEffectContainerView (glass views of different panels would have to be
descendants of one container, which SWT's view tree does not allow), NSBackgroundExtensionView
(it must own the content view) and `toolbarStyle = .unified` (SWT's ToolBar is a plain view, not an NSToolbar).

## Not wrapped

- The SWT event/listener type hierarchy, `Display` as a public type (`App.Unwrap()` exists), `Runnable`, `Internal_*`.
- Widget fonts/colors/backgrounds, cursors, drag and drop, keyboard and mouse listeners (`Unwrap()`), `ExpandBar`, `Browser`, `StyledText`, `CCombo`, `Shell` kinds other than `Window` (dialog shells, tool tips, `Decorations` images), accelerators on menu items, tray menus, `Sash` repositioning helpers (use `Split`), multi-column `Tree`, table sorting helpers, `ImageData`/palettes/transforms/paths/patterns, `GC` clipping, fonts, alpha, polygons.

## Package layout (applied, TSK-31)

The root keeps what every app needs; the platform look and the icon set live in their own packages.

| Package | Contents | Files |
|---|---|---|
| `gowt` | `Run`, `SetAppName`, `App` (threads, `Tray`, images, `Dark`/`OnThemeChange`), `Window`, `Panel` and every widget, options, layouts, dialogs, menus, `GC`/`Image`/`RGB`/`Font` | `gowt.go dialogs.go graphics.go layout.go menu.go theme.go theme_darwin.go theme_other.go widgets_*.go manifest_windows.go` + 2 tests |
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

## Open points

- `Sync` called from the UI thread runs inline (SWT behaviour); documented, not changed.
- Test `gowt_test.go` opens windows; it skips unless `GOWT_GUI_TEST=1` (Linux Xvfb stand).
