# Icons

Example: [`examples/icons`](../examples/icons).

`icons.Get(app, name, opts...)` returns a named icon from the system set of the current OS: SF Symbols on macOS, Segoe Fluent Icons on Windows, freedesktop symbolic icons on Linux. A symbol the OS lacks comes from an embedded Lucide set, so the same names work everywhere.

```go
b := w.Button("", nil)
b.SetImage(icons.Get(app, "search", icons.Size(24)))
```

- `icons.Names()` lists the valid names (39 today: add, back, check, close, copy, cut, delete, down, download, edit, error, file, filter, folder, forward, heart, help, home, info, lock, menu, open, paste, pause, play, print, redo, refresh, remove, save, search, settings, star, stop, undo, up, upload, user, warning). `Get` panics on an unknown one.
- `icons.Size(points)`: the side in points, default 16.
- `icons.Color(rgb)`: the tint. Default is light gray in dark mode and near-black in light mode.
- The image is drawn at every zoom.
- The App owns icons from `Get` and caches them per name, size and colour. Do not call `Dispose` on them.
- The tint is chosen at call time. After a theme change call `Get` again and set the image anew:

```go
app.OnThemeChange(func(bool) { button.SetImage(icons.Get(app, "search", icons.Size(24))) })
```

## Known gaps

- Without the Fluent font (CrossOver) Windows uses the Lucide fallback; the Fluent glyph path was not checked on a real Windows 11.
