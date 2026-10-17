# Events and the UI thread

Examples: [`examples/events`](../examples/events), [`examples/rows`](../examples/rows) (rows bound to state).

## Callbacks

Events are options of the constructor, with plain functions as handlers. One name per concept; the handler's parameter is what the widget reports:

```go
gowt.Button("OK", gowt.OnClick(ok))
gowt.Text(gowt.Hint("Search"), gowt.OnChange(func(s string) { query.Set(s) }), gowt.OnActivate(find))
gowt.Switch("Wi-Fi", gowt.Checked(true), gowt.OnChange(func(on bool) { wifi(on) }))
gowt.Scale(0, 100, gowt.Value(30), gowt.OnChange(func(v int) { volume(v) }))
gowt.List(names, gowt.Selected(0), gowt.OnSelect(func(i int) { show(names[i]) }), gowt.OnActivate(open))
gowt.Tree(gowt.OnSelect(func(n *gowt.Node) { show(n.Text()) }))
```

| Option | Widgets |
|---|---|
| `OnClick(func())` | Button; `OnClick(func(href string))` Link |
| `OnChange(f)` | Text, Combo, StyledText `func(string)`; Switch, Check/Radio Button `func(bool)`; Scale, Slider, Spinner `func(int)`; DateTime `func(time.Time)` |
| `OnSelect(f)` | Combo, List, Table, Tabs, CTabs `func(int)`; Tree `func(*Node)` |
| `OnActivate(f)` | Text `func()` (Enter); List, Table `func(int)`; Tree `func(*Node)` |

The initial state goes in options too: `Hint(s)` for Text, `Checked(b)` for Switch and Check/Radio Button, `Selected(i)` for Combo, List, Table, Tabs and CTabs, `Value(v)` for Scale, Slider, Spinner and Progress (int), for Text, Combo and StyledText (string) and for DateTime (`time.Time`).

The `OnX` methods of the same names (`text.OnChange(f)`, `table.OnSelect(f)`) add a handler to a widget that exists already, through `Ref`. Calling `OnX` again adds another listener; nothing is replaced. Menu and tool items take their callback as an argument: `menu.Item("Quit", app.Quit)`; `nil` means none.

`Window.OnClose(func() bool)` can veto closing: return false to keep the window open.

## The UI thread rule

All widgets belong to the UI thread: the goroutine inside `gowt.Run`'s setup function and every callback. `Run` locks it to its OS thread. Touching a widget from another goroutine is a bug. Three methods are safe from any goroutine:

| Call | Effect |
|---|---|
| `app.Async(f)` | queue `f` on the UI thread and return |
| `app.Sync(f)` | run `f` on the UI thread and wait (do not call it while the UI thread waits for you) |
| `app.Quit()` | end the event loop after the current event |

After `Run` has returned the three do nothing, so a goroutine that outlives the window may still call them; `f` does not run.

`app.After(d, f)` runs `f` on the UI thread once after `d`. `stop := app.Every(d, f)` runs `f` every `d` until `stop()`; a panic in `f` goes to `OnError` and the ticks go on. Both are UI-thread only; `stop` is safe from any goroutine.

```go
go func() {
	result := slowWork()
	status.Post(result) // or app.Async(func() { ... })
}()
```

## Reactive state (package rx)

Package `github.com/haiodo/gowt/rx` keeps widgets in step with state, in the style of Svelte 5 runes. A `State` is a value you set, a `Derived` is computed from other signals, an `Effect` reruns when something it read has changed. Nothing is declared: whatever a `Derived` or an `Effect` reads with `Get` is its dependency. Widgets bind to a state through options, so the data a screen shares lives in `rx` and needs no variable per widget.

```go
count := rx.New(0)                                          // State[int], compared with ==
label := rx.Compute(func() string {                         // Derived[string], lazy and memoized
	return fmt.Sprintf("%d clicks", count.Get())
})
onlyRunning := rx.New(false)
query := rx.New("")

w.Add(
	gowt.Label("", gowt.BindText(label)),                       // the text follows the derived
	gowt.Button("+1", gowt.OnClick(func() { count.Update(func(n int) int { return n + 1 }) })),
	gowt.Button("Only running", gowt.Check(), gowt.Bind(onlyRunning)), // two-way
	gowt.Text(gowt.Bind(query)),                                // two-way; typing keeps the caret
	gowt.Button("Find", gowt.OnClick(find),
		gowt.BindEnabled(rx.Compute(func() bool { return query.Get() != "" }))))

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
| `app.Flush()` | `rx.Flush()`, then the layouts the bindings put off (a longer `BindText` text) instead of on the next event-loop turn: for timing one update or measuring a widget right after a change |
| `gowt.Bind(state)` | option, two-way: `*rx.State[string]` Text and Combo text, `[bool]` Switch and Check Button, `[int]` Scale, Slider, Spinner value or Combo and List index (Progress one way) |
| `gowt.BindText(r)` | option: the text of a Label, Button or Link follows an `rx.Reader[string]` (a `State` or a `Derived`) |
| `gowt.BindEnabled(r)`, `gowt.BindVisible(r)` | options on any widget, `rx.Reader[bool]` |
| `gowt.Rows(list, key, fill)`, `gowt.Nodes(list, key, children, fill)`, `gowt.Items(list)` | options: one Table row, Tree node or List/Combo entry per element of an `rx.Reader[[]T]`, matched by key, see below |
| `gowt.Watch(w, r, apply)` | call `apply` with `r`'s value whenever it changes, for as long as `w` lives: the binding for what no option covers |

Rules:

- `Set` marks the dependents at once (a `Derived` read right after sees the new value), but effects and bindings run once per batch, after the current event handler returns. Code that needs the widgets updated before it continues calls `rx.Flush()` or wraps the writes in `rx.Batch`; inside an effect both only schedule the flush. Parents run before the effects they created.
- Everything except `Post` is UI-thread only. `Set` from another goroutine panics; build with `-tags rxdebug` to check `Get` as well.
- Values that change together go into one `State` (a whole poll result) and are split with `Derived`s; separate `Post`s may reach the UI thread in different batches.
- A binding is disposed with its widget. An `Effect` or `Derived` made inside an `Effect` or `Scope` is disposed with it; a top-level `Effect` lives until its `dispose` is called. A `Derived` subscribes to its sources only while an effect observes it; read elsewhere, it is recomputed after any `Set`.
- An effect that panics does not stop the others: the flush finishes, then the panic goes to `App.OnError` like one from a callback. A failed `Derived` is recomputed on its next read. Effects that keep invalidating each other for 100 rounds panic with "rx: effects did not settle"; a `Derived` that reads itself and a `Set` inside a `Derived` panic too.
- `BindText` lays the window out again on the next event-loop turn, so a growing label is not clipped. `Label.SetText` still does not.

### Rows bound to state

`Rows`, `Nodes` and `Items` are Svelte's keyed `{#each list as item (key)}` for a Table, a Tree and a List or Combo. The app keeps the list in one `State` (a new slice per change, `rx.NewFunc(v, nil)`) or computes it in a `Derived`; the binding keeps the widget in step:

```go
type proc struct{ pid int; name, cpu string }       // comparable: compared with == to find changed rows
procs := rx.NewFunc([]proc(nil), nil)
gowt.Table(gowt.Grow(),
	gowt.Init(func(t *gowt.TableView) { t.Column("Name", 200); t.Column("CPU", 60, gowt.Right()) }),
	gowt.Rows(procs, func(p proc) int { return p.pid }, func(r *gowt.TableRow, p proc) {
		r.SetTextAt(0, p.name)
		r.SetTextAt(1, p.cpu)
	}))
go func() { for range time.Tick(time.Second) { procs.Post(poll()) } }()
```

- A row is matched by `key`. `fill` runs for a new key and for a row whose element is no longer `==` to the one it showed; equal rows are not touched. A removed key's row goes, with its selection.
- Rows whose keys keep their order keep their native items, so selection, check boxes and scroll position stay. A key that moved is made again at its new place (`fill` runs) and keeps its selection; on a Tree it keeps the expansion and selection of its whole subtree.
- `Nodes` takes `children func(T) []T` (nil for a flat tree) and `fill(n, item, created)`: it runs after the node's children exist, with `created` true when the node is made, so the initial expansion goes there.
- The binding owns the rows of its widget: rows added or removed by hand are dropped and the list is shown again on the next change. Columns go in `Init`.
- Keys should be unique among siblings; a repeated key gets a row made again on every change. A node moved to another parent is a new node there.

Cost, per change: the common prefix and suffix are compared by key; the middle goes through a key map and the longest increasing subsequence of old positions, O(n log n), and only rows outside it are made again. When more than half the rows of a level would go, the level is cleared and made again, which is what a hand-written refill costs. A change without new, removed or moved keys allocates nothing beyond the widget calls. The binding keeps per row the key, the last element and the native item; the buffers of the diff grow to the largest change seen.

Linux (GTK, arm64, Docker), 1000 rows, two columns, per change after `app.Flush()`:

| Change | `Rows` | clear and refill by hand |
|---|---|---|
| 10 rows changed | 20 µs, 75 allocs | 5.3 ms, 35108 allocs |
| 10 rows inserted | 0.13 ms, 371 allocs | 5.3 ms |
| all keys new | 5.4 ms, 35116 allocs | 5.4 ms |
| reversed | 5.3 ms, 35191 allocs | 5.3 ms |
| Tree, 100 nodes with 10 children each, 10 changed | 36 µs, 75 allocs | 5.9 ms, 47791 allocs |

Setting the texts of the 10 changed rows by hand, knowing which they are, takes 17 µs and 64 allocs.

## Errors

An error or panic in an event handler or in an `Async`, `Sync` or `After` callback does not end the program. It goes to the function set with `App.OnError`, and the event loop goes on; without `OnError` it is logged to stderr with the stack where it was raised. Toolkit errors arrive wrapped (`gowt: Widget is disposed`), so `errors.As(err, &swtErr)` works; a panic with another value arrives as `gowt: panic: <value>`.

```go
app.OnError(func(err error) { status.Set(err.Error()) })
```

A placement option the parent's layout cannot use (`GrowX` under a `Flow`, `Anchor` outside a `Form`) is reported the same way when the widget is created, naming the option, the widget and the layout; the widget keeps its natural size.

`gowt.Run` returns an `error` raised while `setup` builds the window: then nothing is on screen to recover to. A runtime error in `setup` (nil dereference, index out of range) keeps panicking. Getters and setters have no per-call errors; `App.LoadImage` and `App.ImageFrom` return one because a bad file is routine.

## Key and mouse events

`Canvas` has `OnMouseDown`, `OnMouseUp` and `OnMouseMove`; `Table` and `Tree` have `OnCellClick` with the row, column, button and click count. A key combination that runs a command belongs to a menu item when there is one: `menu.Item("Back", back, gowt.Shortcut("Alt+Left"))` binds the key and shows it in the menu. Without a menu item, `w.OnKey(keys, f)` binds it in the whole window, written as for `Shortcut`:

```go
if runtime.GOOS == "darwin" {
	w.OnKey("Cmd+[", back)
	w.OnKey("Cmd+]", forward)
} else {
	w.OnKey("Alt+Left", back)
	w.OnKey("Alt+Right", forward)
}
```

The bound key does not reach the focused widget. A focused text field (`Text`, `StyledText`, `Spinner`, an editable `Combo`) keeps the combinations it types or edits with, and `f` does not run for them there:

- a character, an arrow, `Home`, `End`, `PageUp`, `PageDown`, `Enter` or `Tab`, alone or with `Shift`;
- `Backspace` and `Delete` with any modifiers;
- `Cmd+A`, `C`, `V`, `X`, `Y`, `Z` (Command on macOS, Ctrl elsewhere), with or without `Shift`;
- on macOS the arrows, `Home`, `End`, `PageUp` and `PageDown` with any modifiers, and a character with `Alt` (Option) or `Ctrl`;
- on Windows and Linux those keys with `Ctrl`, and a character with `Ctrl+Alt` (AltGr).

So `Cmd+[`, `Alt+Left` on Windows and Linux, `Esc` and `F1`-`F12` work while the user types in a field, and `Alt+Left` on macOS or a bare letter do not. There are no other key listeners and no mouse listeners on other widgets: add them on the swt object, see [swt-direct.md](swt-direct.md).

`w.OnDispose(f)` runs `f` when the widget goes away: by `Dispose`, with its parent, or when its window closes.
