# gowt facade (package `gowt`)

`swt` stays the generated, Java-shaped low level (full SWT, for ported code). `gowt` is a
hand-written, thin, idiomatic layer on top; it is the API we document. Files: `gowt.go` (App,
Window, Panel, base widgets, options), `layout.go`, `widgets_common.go`, `widgets_list.go`,
`widgets_container.go`, `widgets_range.go`, `graphics.go`, `menu.go`, `dialogs.go`. Examples:
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
| Naming | Go names: `Checked`, `SetText`, `OnClick`; no `Get` prefix; ints for sizes (not int32); `time.Duration` for time; no `Like`/`As*`/`Overload` | |

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

jface factories take the parent as `swt.CompositeLike` and return the swt widget
(`ButtonFactory.Create(parent) *swt.Button`). From gowt they plug in through `Unwrap()`:
`jface.ButtonFactoryNewButton(swt.PUSH).Text("x").Create(panel.Unwrap())`. The result is a plain swt
widget; gowt has no constructor that wraps an existing `*swt.X` (it would let wrappers be built
around factory output, deliberately not exported yet). jface has no viewers in this repo, so there is nothing to plug
`Table`/`Tree` into.

## Not wrapped

- The SWT event/listener type hierarchy, `Display` as a public type (`App.Unwrap()` exists), `Runnable`, `Internal_*`.
- Widget fonts/colors/backgrounds, cursors, drag and drop, keyboard and mouse listeners (`Unwrap()`), `ExpandBar`, `Browser`, `StyledText`, `CCombo`, `Shell` kinds other than `Window` (dialog shells, tool tips, `Decorations` images), accelerators on menu items, tray menus, `Sash` repositioning helpers (use `Split`), multi-column `Tree`, table sorting helpers, `ImageData`/palettes/transforms/paths/patterns, `GC` clipping, fonts, alpha, polygons.

## Open points

- `Sync` called from the UI thread runs inline (SWT behaviour); documented, not changed.
- Test `gowt_test.go` opens windows; it skips unless `GOWT_GUI_TEST=1` (Linux Xvfb stand).
