# Theme and platform look

Example: [`examples/theme`](../examples/theme).

## Light and dark

`gowt.Run` makes the app follow the system theme. Two calls expose it:

```go
dark := app.Dark()                       // UI thread
app.OnThemeChange(func(dark bool) { })   // runs after widget colours were refreshed
```

How the theme is read: macOS `NSApp.effectiveAppearance`, Windows the apps theme setting, Linux the `color-scheme` value of xdg-desktop-portal. Where each OS stands:

| OS | What follows the theme |
|---|---|
| macOS | the whole UI |
| Windows | the title bar; content stays light unless you call `look.SetDarkContent(app, true)` before creating windows (scroll bars, tables, trees, buttons; it uses undocumented uxtheme exports and does not switch live) |
| Linux | the theme and accent colour come from the portal; this was not verified against a live portal |

Colours you set yourself (`Background`, `GC` colours) do not change by themselves: redraw them in `OnThemeChange`.

## Package look

`github.com/haiodo/gowt/look` holds options for the platform's own appearance. Every call does nothing on a system without the feature, so it is safe to call everywhere.

| Call | Effect |
|---|---|
| `look.SetBackdrop(w, look.BackdropNone / BackdropTranslucent / BackdropGlass)` | window material. macOS: blurred material or Liquid Glass (26+). Windows 11 22H2 (build 22621) and later: Acrylic (`BackdropTranslucent`) or Mica (`BackdropGlass`); older builds keep the opaque window. Linux ignores it |
| `look.SetGlass(container, true)` | Liquid Glass surface behind a `Panel`, `Window`, `Group`, `Split` or `CoolBar`. macOS 26; older macOS shows a blurred material, other systems nothing |
| `look.GlassButton()` | option for a push button: glass bezel on macOS 26 |
| `look.SetFullSizeContent(w, true)` | content extends under a transparent title bar (macOS); leave about 28 points at the top for the traffic lights |
| `look.SetRoundedCorners(w, true)` | force rounded or square window corners on Windows 11 |
| `look.SetDarkContent(app, true)` | Windows dark content, see above |
| `look.Classic()` | before `Run`: ask macOS 26 for the pre-Liquid-Glass look |

On macOS 26 the system chrome is Liquid Glass without any call, because Go 1.27 records SDK 26.2 in the binary.

`look.Classic()` was not confirmed to work for a binary without an app bundle. The reliable way is building with `-ldflags=-macsdk=15.0`.

## Known gaps

- On Windows the material shows only where the client area is black, so `SetBackdrop` turns the shell background black. Text drawn in pure black becomes transparent: use a near-black foreground. Widgets with their own opaque background are unaffected.
- Mica, Acrylic and rounded corners were never checked on a real Windows 11 (CrossOver has no DWM).
- Windows per-monitor DPI changes while running (`WM_DPICHANGED`) are not handled.
- Not available on macOS: `NSGlassEffectContainerView`, `NSBackgroundExtensionView`, a unified tool bar.
