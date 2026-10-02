# gowt facade (package `gowt`)

`swt` stays the generated, Java-shaped low level (full SWT, for ported code). `gowt` is a
hand-written, thin, idiomatic layer on top; it is the API we document. Prototype: `gowt.go`,
`layout.go`, `cmd/hellogowt`.

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
| Layouts | value types: `Fill{Vertical}`, `Grid{Columns, EqualWidth, Margin, Spacing}`; per-child `Cell(GridCell{...})` | Plain structs with zero-value meaning "none/natural", enum `Align*` instead of int bit masks. Applied with `SetLayout`. Row/Form/Stack come with TSK-02. |
| Naming | Go names: `Checked`, `SetText`, `OnClick`; no `Get` prefix; ints for sizes (not int32); `time.Duration` for time; no `Like`/`As*`/`Overload` | |

## Not wrapped

- Anything past the widgets listed in TSK-02; reach it through `Unwrap()`.
- The SWT event/listener type hierarchy, `Display` as a public type (`App.Unwrap()` exists), `Runnable`, `Internal_*`.
- Resource ownership (fonts, colors, images) is not hidden; TSK-02 decides how Image/GC are exposed.

## Open points

- `Sync` called from the UI thread runs inline (SWT behaviour); documented, not changed.
- Test `gowt_test.go` opens windows; it skips unless `GOWT_GUI_TEST=1` (Linux Xvfb stand).
