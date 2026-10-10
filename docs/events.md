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

`app.After(d, f)` runs `f` on the UI thread once after `d`; call it again from `f` for a periodic tick. It is UI-thread only.

```go
go func() {
	result := slowWork()
	app.Async(func() { status.SetText(result) })
}()
```

## Errors

`gowt.Run` returns toolkit failures as an `error`. Programmer errors (nil dereference, index out of range) keep panicking. Getters and setters have no per-call errors; `App.LoadImage` and `App.ImageFrom` return one because a bad file is routine.

## Key and mouse events

The facade has no key or mouse listeners. Add them on the swt object, see [swt-direct.md](swt-direct.md).
