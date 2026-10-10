# Using swt directly

Example: [`examples/swtdirect`](../examples/swtdirect).

Package `github.com/haiodo/gowt/swt` is the full Eclipse SWT API translated to Go. It follows Java naming (`NewShell`, `SetText`, `AddSelectionListener`, `int32` sizes). The facade does not cover all of it and does not need to: you can drop down for a single widget and stay in the facade for the rest.

| From the facade | Get |
|---|---|
| `widget.Unwrap()` | the swt object of that wrapper: `*swt.Button`, `*swt.Text`, `*swt.Table`, ... |
| `window.Unwrap()` | `*swt.Shell` |
| `app.Unwrap()` | `*swt.Display` |
| `window.AsComposite()`, `panel.AsComposite()` (also `Group`, `Split`, `CoolBar`) | the `*swt.Composite` to use as a parent for swt widgets, or for `webview.New`, `browser.NewBrowser` |
| `gowt.Custom(func(*swt.Control))` | an option that runs on the new widget right after creation |

```go
w.Unwrap().SetMinimumSize(320, 160)
field.Unwrap().AddKeyListener(swt.KeyListenerKeyPressedAdapter(func(e *swt.KeyEvent) { ... }))
sep := swt.NewLabel(w.AsComposite(), swt.SEPARATOR|swt.HORIZONTAL)
```

Rules:

- Names in `swt` are the SWT ones. Find them with `go doc github.com/haiodo/gowt/swt Button`. SWT documentation on eclipse.org describes the behaviour; Java overloads are separate Go names that spell out the parameters (for example `NewShellDisplayStyle`).
- An swt widget made by hand is not wrapped. A facade layout still positions it, because it is a child of the same composite; give it `LayoutData` in swt terms (`swt.NewGridData()`).
- `swt` resources that SWT disposes by hand (images, fonts, GC outside a paint event) are yours to `Dispose`.
- The same UI-thread rule applies.
- Other SWT-level packages: `jface` (layout and widget factories), `browser`, `svg`.

`Clipboard`, `DragSource`/`DropTarget` and `StyledText` are in `swt` and wrapped by the facade for the common cases; custom `Transfer` types, RTF/URL formats, the X11 selection clipboard and the rest of `StyledText` (bullets, line indents, undo listeners) need `Unwrap()`. `Accessible` is a stub with no facade wrapper.
