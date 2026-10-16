# Events and the UI thread

Example: [`examples/events`](../examples/events).

## Callbacks

Events are `OnX` methods that take plain functions: `OnClick(func())`, `OnChange(func(text string))`, `OnSelect(func(index int))`. Calling `OnX` again adds another listener; nothing is replaced. Button, menu and tool item callbacks can also be passed to the constructor (`w.Button("OK", onClick)`); `nil` means none.

`Window.OnClose(func() bool)` can veto closing: return false to keep the window open.

## The UI thread rule

All widgets belong to the UI thread: the goroutine inside `gowt.Run`'s setup function and every callback. `Run` locks it to its OS thread. Touching a widget from another goroutine is a bug. Three methods are safe from any goroutine:

| Call | Effect |
|---|---|
| `app.Async(f)` | queue `f` on the UI thread and return |
| `app.Sync(f)` | run `f` on the UI thread and wait (do not call it while the UI thread waits for you) |
| `app.Quit()` | end the event loop after the current event |

After `Run` has returned the three do nothing, so a goroutine that outlives the window may still call them; `f` does not run.

`app.After(d, f)` runs `f` on the UI thread once after `d`; call it again from `f` for a periodic tick. It is UI-thread only.

```go
go func() {
	result := slowWork()
	app.Async(func() { status.SetText(result) })
}()
```

## Reactive state (package rx)

Package `github.com/haiodo/gowt/rx` keeps widgets in step with state, in the style of Svelte 5 runes. A `State` is a value you set, a `Derived` is computed from other signals, an `Effect` reruns when something it read has changed. Nothing is declared: whatever a `Derived` or an `Effect` reads with `Get` is its dependency.

```go
count := rx.New(0)                                          // State[int], compared with ==
label := rx.Compute(func() string {                         // Derived[string], lazy and memoized
	return fmt.Sprintf("%d clicks", count.Get())
})
w.Label("").BindText(label)                                // the label follows the derived
w.Button("+1", func() { count.Update(func(n int) int { return n + 1 }) })

only := rx.New(false)
w.Button("Only running", nil, gowt.Check()).BindChecked(only) // two-way
query := rx.New("")
w.Text().Bind(query)                                       // two-way; typing keeps the caret
find := w.Button("Find", onFind)
gowt.BindEnabled(find, rx.Compute(func() bool { return query.Get() != "" }))

go func() { for v := range updates { count.Post(v) } }()    // from any goroutine
```

| Call | What it does |
|---|---|
| `rx.New(v)`, `rx.NewFunc(v, eq)` | a `State`; `Set` of an equal value does nothing. `eq` nil: every `Set` notifies (slices, maps) |
| `s.Get()`, `s.Peek()`, `s.Set(v)`, `s.Update(f)` | read and subscribe, read only, write, write `f(current)` |
| `s.Post(v)` | the only call safe from any goroutine; the last value wins, one hop to the UI thread per batch |
| `rx.Compute(fn)`, `rx.ComputeFunc(fn, eq)` | a `Derived`: recomputed when read after a source changed; an equal result stops the change there |
| `rx.Effect(fn)` | runs `fn` now and again after its sources changed; returns `dispose` |
| `rx.OnCleanup(f)` | inside an `Effect` or a `Derived`: `f` runs before the next run and on dispose; inside a `Scope`: on dispose |
| `rx.Scope(fn)` | owns the effects and deriveds created in `fn`; returns `dispose` |
| `rx.Untrack(fn)` | reads in `fn` do not subscribe |
| `rx.Batch(fn)`, `rx.Flush()` | run the pending effects now instead of after the current event handler |
| `gowt.Bind(w, r, apply)` | call `apply` with `r`'s value whenever it changes, for as long as `w` lives |
| `Label.BindText`, `Button.BindChecked`, `Text.Bind`, `gowt.BindVisible`, `gowt.BindEnabled` | the common bindings |

Rules:

- `Set` marks the dependents at once (a `Derived` read right after sees the new value), but effects and bindings run once per batch, after the current event handler returns. Code that needs the widgets updated before it continues calls `rx.Flush()` or wraps the writes in `rx.Batch`; inside an effect both only schedule the flush. Parents run before the effects they created.
- Everything except `Post` is UI-thread only. `Set` from another goroutine panics; build with `-tags rxdebug` to check `Get` as well.
- Values that change together go into one `State` (a whole poll result) and are split with `Derived`s; separate `Post`s may reach the UI thread in different batches.
- A binding is disposed with its widget. An `Effect` or `Derived` made inside an `Effect` or `Scope` is disposed with it; a top-level `Effect` lives until its `dispose` is called. A `Derived` subscribes to its sources only while an effect observes it; read elsewhere, it is recomputed after any `Set`.
- An effect that panics does not stop the others: the flush finishes, then the panic goes to `App.OnError` like one from a callback. A failed `Derived` is recomputed on its next read. Effects that keep invalidating each other for 100 rounds panic with "rx: effects did not settle"; a `Derived` that reads itself and a `Set` inside a `Derived` panic too.
- `BindText` lays the window out again on the next event-loop turn, so a growing label is not clipped. `Label.SetText` still does not.

## Errors

An error or panic in an event handler or in an `Async`, `Sync` or `After` callback does not end the program. It goes to the function set with `App.OnError`, and the event loop goes on; without `OnError` it is logged to stderr with the stack where it was raised. Toolkit errors arrive wrapped (`gowt: Widget is disposed`), so `errors.As(err, &swtErr)` works; a panic with another value arrives as `gowt: panic: <value>`.

```go
app.OnError(func(err error) { status.SetText(err.Error()) })
```

A placement option the parent's layout cannot use (`GrowX` under a `Row`, `InRow` under a `Grid`) is reported the same way when the widget is created, naming the option, the widget and the layout; the widget keeps its natural size.

`gowt.Run` returns an `error` raised while `setup` builds the window: then nothing is on screen to recover to. A runtime error in `setup` (nil dereference, index out of range) keeps panicking. Getters and setters have no per-call errors; `App.LoadImage` and `App.ImageFrom` return one because a bad file is routine.

## Key and mouse events

`Canvas` has `OnMouseDown`, `OnMouseUp` and `OnMouseMove`; `Table` and `Tree` have `OnCellClick` with the row, column, button and click count. There are no key listeners and no mouse listeners on other widgets: add them on the swt object, see [swt-direct.md](swt-direct.md).
